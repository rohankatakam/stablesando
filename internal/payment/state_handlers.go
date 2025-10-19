package payment

import (
	"context"
	"fmt"
	"time"

	"crypto-conversion/internal/logger"
	"crypto-conversion/internal/models"
)

// StateMachine represents the payment state machine orchestrator
type StateMachine struct {
	onRampClient  *StatefulOnRampClient
	offRampClient *StatefulOffRampClient
	dbClient      DatabaseClient
	queueClient   QueueClient
}

// DatabaseClient interface for payment database operations
type DatabaseClient interface {
	UpdatePayment(ctx context.Context, payment *models.Payment) error
	GetPaymentByID(ctx context.Context, paymentID string) (*models.Payment, error)
}

// QueueClient interface for re-enqueuing jobs
type QueueClient interface {
	EnqueuePaymentWithDelay(ctx context.Context, job *models.PaymentJob, delaySeconds int) error
}

// NewStateMachine creates a new state machine orchestrator
func NewStateMachine(onRamp *StatefulOnRampClient, offRamp *StatefulOffRampClient, db DatabaseClient, queue QueueClient) *StateMachine {
	return &StateMachine{
		onRampClient:  onRamp,
		offRampClient: offRamp,
		dbClient:      db,
		queueClient:   queue,
	}
}

// ProcessPayment processes a payment based on its current state
func (sm *StateMachine) ProcessPayment(ctx context.Context, job *models.PaymentJob) error {
	// Fetch current payment state
	payment, err := sm.dbClient.GetPaymentByID(ctx, job.PaymentID)
	if err != nil {
		return fmt.Errorf("failed to fetch payment: %w", err)
	}

	logger.Info("Processing payment in state machine", logger.Fields{
		"payment_id": payment.PaymentID,
		"status":     payment.Status,
	})

	// Route to appropriate handler based on current state
	switch payment.Status {
	case models.StatusPending:
		return sm.handlePending(ctx, job, payment)
	case models.StatusOnrampPending:
		return sm.handleOnrampPending(ctx, job, payment)
	case models.StatusOnrampComplete:
		return sm.handleOnrampComplete(ctx, job, payment)
	case models.StatusOfframpPending:
		return sm.handleOfframpPending(ctx, job, payment)
	case models.StatusCompleted, models.StatusFailed:
		logger.Info("Payment already in terminal state", logger.Fields{
			"payment_id": payment.PaymentID,
			"status":     payment.Status,
		})
		return nil
	default:
		return fmt.Errorf("unexpected payment status: %s", payment.Status)
	}
}

// handlePending initiates the onramp transfer
func (sm *StateMachine) handlePending(ctx context.Context, job *models.PaymentJob, payment *models.Payment) error {
	logger.Info("Handling PENDING state - initiating onramp", logger.Fields{
		"payment_id": payment.PaymentID,
	})

	// Initiate onramp transfer
	txID, err := sm.onRampClient.InitiateTransfer(ctx, payment.Amount, payment.Currency)
	if err != nil {
		// Mark as failed
		sm.transitionState(payment, models.StatusFailed, fmt.Sprintf("Onramp initiation failed: %s", err.Error()))
		payment.ErrorMessage = err.Error()
		sm.dbClient.UpdatePayment(ctx, payment)
		return fmt.Errorf("onramp initiation failed: %w", err)
	}

	// Update payment state
	payment.OnRampTxID = txID
	sm.transitionState(payment, models.StatusOnrampPending, "Onramp transfer initiated")

	if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Re-enqueue with 30-second delay to poll onramp status
	if err := sm.queueClient.EnqueuePaymentWithDelay(ctx, job, 30); err != nil {
		return fmt.Errorf("failed to re-enqueue payment: %w", err)
	}

	logger.Info("Onramp initiated, re-enqueued for polling", logger.Fields{
		"payment_id":    payment.PaymentID,
		"on_ramp_tx_id": txID,
		"delay_seconds": 30,
	})

	return nil
}

// handleOnrampPending polls onramp status
func (sm *StateMachine) handleOnrampPending(ctx context.Context, job *models.PaymentJob, payment *models.Payment) error {
	logger.Info("Handling ONRAMP_PENDING state - polling status", logger.Fields{
		"payment_id":    payment.PaymentID,
		"on_ramp_tx_id": payment.OnRampTxID,
		"poll_count":    payment.OnRampPollCount,
	})

	// Poll onramp status
	transfer, err := sm.onRampClient.GetTransferStatus(ctx, payment.OnRampTxID)
	if err != nil {
		return fmt.Errorf("failed to poll onramp status: %w", err)
	}

	payment.OnRampPollCount = transfer.PollCount

	switch transfer.Status {
	case TransferStatusSettled:
		// Onramp complete, move to next stage
		sm.transitionState(payment, models.StatusOnrampComplete, "Onramp settled, USDC received")

		if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		// Immediately process offramp (no delay)
		if err := sm.queueClient.EnqueuePaymentWithDelay(ctx, job, 0); err != nil {
			return fmt.Errorf("failed to re-enqueue payment: %w", err)
		}

		logger.Info("Onramp settled, proceeding to offramp", logger.Fields{
			"payment_id": payment.PaymentID,
			"poll_count": payment.OnRampPollCount,
		})

	case TransferStatusFailed:
		// Mark payment as failed
		sm.transitionState(payment, models.StatusFailed, "Onramp transfer failed")
		payment.ErrorMessage = "Onramp settlement failed"
		sm.dbClient.UpdatePayment(ctx, payment)

		logger.Error("Onramp transfer failed", logger.Fields{
			"payment_id": payment.PaymentID,
			"tx_id":      payment.OnRampTxID,
		})

	case TransferStatusPending:
		// Still pending, check again in 30 seconds
		if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		if err := sm.queueClient.EnqueuePaymentWithDelay(ctx, job, 30); err != nil {
			return fmt.Errorf("failed to re-enqueue payment: %w", err)
		}

		logger.Info("Onramp still pending, will poll again", logger.Fields{
			"payment_id":   payment.PaymentID,
			"poll_count":   payment.OnRampPollCount,
			"delay_seconds": 30,
		})
	}

	return nil
}

// handleOnrampComplete initiates the offramp transfer
func (sm *StateMachine) handleOnrampComplete(ctx context.Context, job *models.PaymentJob, payment *models.Payment) error {
	logger.Info("Handling ONRAMP_COMPLETE state - initiating offramp", logger.Fields{
		"payment_id": payment.PaymentID,
	})

	// Determine amount to send to offramp
	// Use guaranteed payout if quote was used, otherwise use payment amount
	amountToConvert := payment.GuaranteedPayoutAmount
	if amountToConvert == 0 {
		amountToConvert = payment.Amount
	}

	// Initiate offramp transfer
	txID, err := sm.offRampClient.InitiateTransfer(ctx, amountToConvert, payment.Currency)
	if err != nil {
		// Mark as failed
		sm.transitionState(payment, models.StatusFailed, fmt.Sprintf("Offramp initiation failed: %s", err.Error()))
		payment.ErrorMessage = err.Error()
		sm.dbClient.UpdatePayment(ctx, payment)
		return fmt.Errorf("offramp initiation failed: %w", err)
	}

	// Update payment state
	payment.OffRampTxID = txID
	sm.transitionState(payment, models.StatusOfframpPending, "Offramp transfer initiated")

	if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
		return fmt.Errorf("failed to update payment: %w", err)
	}

	// Re-enqueue with 30-second delay to poll offramp status
	if err := sm.queueClient.EnqueuePaymentWithDelay(ctx, job, 30); err != nil {
		return fmt.Errorf("failed to re-enqueue payment: %w", err)
	}

	logger.Info("Offramp initiated, re-enqueued for polling", logger.Fields{
		"payment_id":     payment.PaymentID,
		"off_ramp_tx_id": txID,
		"delay_seconds":  30,
	})

	return nil
}

// handleOfframpPending polls offramp status
func (sm *StateMachine) handleOfframpPending(ctx context.Context, job *models.PaymentJob, payment *models.Payment) error {
	logger.Info("Handling OFFRAMP_PENDING state - polling status", logger.Fields{
		"payment_id":     payment.PaymentID,
		"off_ramp_tx_id": payment.OffRampTxID,
		"poll_count":     payment.OffRampPollCount,
	})

	// Poll offramp status
	transfer, err := sm.offRampClient.GetTransferStatus(ctx, payment.OffRampTxID)
	if err != nil {
		return fmt.Errorf("failed to poll offramp status: %w", err)
	}

	payment.OffRampPollCount = transfer.PollCount

	switch transfer.Status {
	case TransferStatusSettled:
		// Payment complete!
		sm.transitionState(payment, models.StatusCompleted, "Offramp settled, funds delivered")
		now := time.Now()
		payment.ProcessedAt = &now

		if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		logger.Info("Payment completed successfully", logger.Fields{
			"payment_id":         payment.PaymentID,
			"onramp_poll_count":  payment.OnRampPollCount,
			"offramp_poll_count": payment.OffRampPollCount,
			"total_time":         time.Since(payment.CreatedAt).String(),
		})

	case TransferStatusFailed:
		// Mark payment as failed
		sm.transitionState(payment, models.StatusFailed, "Offramp transfer failed")
		payment.ErrorMessage = "Offramp settlement failed"
		sm.dbClient.UpdatePayment(ctx, payment)

		logger.Error("Offramp transfer failed", logger.Fields{
			"payment_id": payment.PaymentID,
			"tx_id":      payment.OffRampTxID,
		})

	case TransferStatusPending:
		// Still pending, check again in 30 seconds
		if err := sm.dbClient.UpdatePayment(ctx, payment); err != nil {
			return fmt.Errorf("failed to update payment: %w", err)
		}

		if err := sm.queueClient.EnqueuePaymentWithDelay(ctx, job, 30); err != nil {
			return fmt.Errorf("failed to re-enqueue payment: %w", err)
		}

		logger.Info("Offramp still pending, will poll again", logger.Fields{
			"payment_id":    payment.PaymentID,
			"poll_count":    payment.OffRampPollCount,
			"delay_seconds": 30,
		})
	}

	return nil
}

// transitionState records a state transition
func (sm *StateMachine) transitionState(payment *models.Payment, newStatus models.PaymentStatus, message string) {
	transition := models.StateTransition{
		FromStatus: payment.Status,
		ToStatus:   newStatus,
		Timestamp:  time.Now(),
		Message:    message,
	}

	if payment.StateHistory == nil {
		payment.StateHistory = []models.StateTransition{}
	}
	payment.StateHistory = append(payment.StateHistory, transition)
	payment.Status = newStatus
	payment.UpdatedAt = time.Now()

	logger.Info("State transition", logger.Fields{
		"payment_id": payment.PaymentID,
		"from":       transition.FromStatus,
		"to":         transition.ToStatus,
		"message":    message,
	})
}
