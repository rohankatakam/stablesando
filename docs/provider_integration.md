# Provider Integration Guide

## Overview

The payment system is designed with **provider abstraction** in mind, allowing easy integration of multiple on-ramp and off-ramp providers without changing core business logic.

## Current Architecture

### Mock Provider Implementation

Currently, the system uses stateful mock providers that simulate real provider behavior:

```go
// internal/payment/mock_providers.go

type StatefulOnRampProvider struct {
    transactions map[string]*OnRampTransaction
    mu           sync.RWMutex
}

type StatefulOffRampProvider struct {
    transactions map[string]*OffRampTransaction
    mu           sync.RWMutex
}
```

These mocks demonstrate the expected provider behavior:
- **InitiateTransfer**: Creates a transaction and returns a provider-specific ID
- **GetTransferStatus**: Polls transaction status (simulates async settlement)
- **Stateful simulation**: Transactions transition from `PENDING` → `COMPLETED` over time

## Provider Interface Design

### Recommended Interface Pattern

```go
// internal/providers/interface.go

package providers

import (
    "context"
    "time"
)

// OnRampProvider handles fiat → stablecoin conversion
type OnRampProvider interface {
    // Name returns the provider identifier (e.g., "Circle", "Bridge")
    Name() string

    // InitiateTransfer starts a fiat → stablecoin conversion
    // Returns provider-specific transaction ID
    InitiateTransfer(ctx context.Context, req *OnRampRequest) (*OnRampResponse, error)

    // GetStatus polls the current status of a transaction
    GetStatus(ctx context.Context, txID string) (*TransferStatus, error)

    // HealthCheck verifies the provider is operational
    HealthCheck(ctx context.Context) error

    // SupportedCurrencies returns list of supported fiat currencies
    SupportedCurrencies() []string

    // SupportedChains returns list of supported blockchain networks
    SupportedChains() []string
}

// OffRampProvider handles stablecoin → fiat conversion
type OffRampProvider interface {
    // Name returns the provider identifier (e.g., "Circle", "Bridge")
    Name() string

    // InitiateTransfer starts a stablecoin → fiat conversion
    InitiateTransfer(ctx context.Context, req *OffRampRequest) (*OffRampResponse, error)

    // GetStatus polls the current status of a transaction
    GetStatus(ctx context.Context, txID string) (*TransferStatus, error)

    // HealthCheck verifies the provider is operational
    HealthCheck(ctx context.Context) error

    // SupportedCurrencies returns list of supported fiat currencies
    SupportedCurrencies() []string

    // SupportedChains returns list of supported blockchain networks
    SupportedChains() []string
}

// Request/Response types
type OnRampRequest struct {
    Amount          int64  `json:"amount"`
    Currency        string `json:"currency"`
    SourceAccount   string `json:"source_account"`
    Chain           string `json:"chain"`            // Target blockchain
    IdempotencyKey  string `json:"idempotency_key"`
}

type OnRampResponse struct {
    TransactionID   string    `json:"transaction_id"`
    Status          string    `json:"status"`
    Amount          int64     `json:"amount"`
    Currency        string    `json:"currency"`
    EstimatedTime   string    `json:"estimated_time"`
    CreatedAt       time.Time `json:"created_at"`
}

type OffRampRequest struct {
    Amount             int64  `json:"amount"`
    Currency           string `json:"currency"`
    DestinationAccount string `json:"destination_account"`
    Chain              string `json:"chain"`            // Source blockchain
    IdempotencyKey     string `json:"idempotency_key"`
}

type OffRampResponse struct {
    TransactionID   string    `json:"transaction_id"`
    Status          string    `json:"status"`
    Amount          int64     `json:"amount"`
    Currency        string    `json:"currency"`
    EstimatedTime   string    `json:"estimated_time"`
    CreatedAt       time.Time `json:"created_at"`
}

type TransferStatus struct {
    TransactionID   string    `json:"transaction_id"`
    Status          string    `json:"status"` // PENDING, PROCESSING, COMPLETED, FAILED
    ErrorMessage    string    `json:"error_message,omitempty"`
    CompletedAt     *time.Time `json:"completed_at,omitempty"`
}
```

## Provider Registry Pattern

### Multi-Provider Support

```go
// internal/providers/registry.go

package providers

import (
    "context"
    "fmt"
    "sync"
)

// ProviderRegistry manages multiple provider implementations
type ProviderRegistry struct {
    onrampProviders  map[string]OnRampProvider
    offrampProviders map[string]OffRampProvider
    mu               sync.RWMutex
}

func NewProviderRegistry() *ProviderRegistry {
    return &ProviderRegistry{
        onrampProviders:  make(map[string]OnRampProvider),
        offrampProviders: make(map[string]OffRampProvider),
    }
}

// RegisterOnRamp adds an on-ramp provider to the registry
func (r *ProviderRegistry) RegisterOnRamp(provider OnRampProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.onrampProviders[provider.Name()] = provider
}

// RegisterOffRamp adds an off-ramp provider to the registry
func (r *ProviderRegistry) RegisterOffRamp(provider OffRampProvider) {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.offrampProviders[provider.Name()] = provider
}

// GetOnRamp retrieves a specific on-ramp provider
func (r *ProviderRegistry) GetOnRamp(name string) (OnRampProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    provider, exists := r.onrampProviders[name]
    if !exists {
        return nil, fmt.Errorf("on-ramp provider %s not found", name)
    }
    return provider, nil
}

// GetOffRamp retrieves a specific off-ramp provider
func (r *ProviderRegistry) GetOffRamp(name string) (OffRampProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    provider, exists := r.offrampProviders[name]
    if !exists {
        return nil, fmt.Errorf("off-ramp provider %s not found", name)
    }
    return provider, nil
}

// SelectBestOnRamp chooses optimal provider based on criteria
func (r *ProviderRegistry) SelectBestOnRamp(ctx context.Context, req *OnRampRequest) (OnRampProvider, error) {
    r.mu.RLock()
    defer r.mu.RUnlock()

    // Simple strategy: use first available provider
    // Production: Implement intelligent routing based on:
    // - Provider health status
    // - Fee comparison
    // - Success rate
    // - Estimated settlement time
    // - Currency/chain support

    for _, provider := range r.onrampProviders {
        if err := provider.HealthCheck(ctx); err == nil {
            // Check if provider supports required currency and chain
            if contains(provider.SupportedCurrencies(), req.Currency) &&
               contains(provider.SupportedChains(), req.Chain) {
                return provider, nil
            }
        }
    }

    return nil, fmt.Errorf("no available on-ramp provider")
}

func contains(slice []string, item string) bool {
    for _, s := range slice {
        if s == item {
            return true
        }
    }
    return false
}
```

## Real Provider Implementations

### Circle Provider Example

```go
// internal/providers/circle/onramp.go

package circle

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "crypto_conversion/internal/providers"
)

type CircleOnRampProvider struct {
    apiKey     string
    baseURL    string
    httpClient *http.Client
}

func NewCircleOnRampProvider(apiKey string) *CircleOnRampProvider {
    return &CircleOnRampProvider{
        apiKey:  apiKey,
        baseURL: "https://api.circle.com/v1",
        httpClient: &http.Client{
            Timeout: 30 * time.Second,
        },
    }
}

func (c *CircleOnRampProvider) Name() string {
    return "Circle"
}

func (c *CircleOnRampProvider) InitiateTransfer(ctx context.Context, req *providers.OnRampRequest) (*providers.OnRampResponse, error) {
    // Implementation example:
    // 1. Create Circle account deposit request
    // 2. POST to /v1/businessAccount/deposits
    // 3. Return Circle transaction ID

    endpoint := fmt.Sprintf("%s/businessAccount/deposits", c.baseURL)

    circleReq := map[string]interface{}{
        "idempotencyKey": req.IdempotencyKey,
        "amount": map[string]interface{}{
            "amount":   fmt.Sprintf("%.2f", float64(req.Amount)/100),
            "currency": req.Currency,
        },
        "source": map[string]interface{}{
            "type": "ach",
            "id":   req.SourceAccount,
        },
        "destination": map[string]interface{}{
            "type":  "blockchain",
            "chain": req.Chain,
        },
    }

    // Make HTTP request to Circle API
    // Parse response
    // Return OnRampResponse

    return &providers.OnRampResponse{
        TransactionID: "circle_tx_123",
        Status:        "PENDING",
        Amount:        req.Amount,
        Currency:      req.Currency,
        EstimatedTime: "2-5 minutes",
        CreatedAt:     time.Now(),
    }, nil
}

func (c *CircleOnRampProvider) GetStatus(ctx context.Context, txID string) (*providers.TransferStatus, error) {
    // Poll Circle API for transaction status
    endpoint := fmt.Sprintf("%s/businessAccount/deposits/%s", c.baseURL, txID)

    // Make HTTP GET request
    // Parse status from Circle response
    // Map to our status enum

    return &providers.TransferStatus{
        TransactionID: txID,
        Status:        "COMPLETED",
        CompletedAt:   timePtr(time.Now()),
    }, nil
}

func (c *CircleOnRampProvider) HealthCheck(ctx context.Context) error {
    endpoint := fmt.Sprintf("%s/health", c.baseURL)

    req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
    if err != nil {
        return err
    }

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("circle health check failed: %d", resp.StatusCode)
    }

    return nil
}

func (c *CircleOnRampProvider) SupportedCurrencies() []string {
    return []string{"USD", "EUR", "GBP"}
}

func (c *CircleOnRampProvider) SupportedChains() []string {
    return []string{"ethereum", "polygon", "avalanche", "base", "arbitrum"}
}

func timePtr(t time.Time) *time.Time {
    return &t
}
```

### Bridge Provider Example

```go
// internal/providers/bridge/onramp.go

package bridge

import (
    "context"
    "crypto_conversion/internal/providers"
    "time"
)

type BridgeOnRampProvider struct {
    apiKey  string
    baseURL string
}

func NewBridgeOnRampProvider(apiKey string) *BridgeOnRampProvider {
    return &BridgeOnRampProvider{
        apiKey:  apiKey,
        baseURL: "https://api.bridge.xyz/v0",
    }
}

func (b *BridgeOnRampProvider) Name() string {
    return "Bridge"
}

func (b *BridgeOnRampProvider) InitiateTransfer(ctx context.Context, req *providers.OnRampRequest) (*providers.OnRampResponse, error) {
    // Bridge-specific implementation
    // POST to /v0/external_accounts/{id}/receive
    return &providers.OnRampResponse{
        TransactionID: "bridge_tx_456",
        Status:        "PENDING",
        Amount:        req.Amount,
        Currency:      req.Currency,
        EstimatedTime: "1-3 minutes",
        CreatedAt:     time.Now(),
    }, nil
}

// ... implement remaining interface methods
```

## Integration with State Machine

### Updating State Handlers

```go
// internal/payment/state_handlers.go (modifications)

import (
    "crypto_conversion/internal/providers"
)

type StateHandler struct {
    providerRegistry *providers.ProviderRegistry
    // ... other fields
}

func NewStateHandler(registry *providers.ProviderRegistry, ...) *StateHandler {
    return &StateHandler{
        providerRegistry: registry,
        // ...
    }
}

func (h *StateHandler) handlePending(ctx context.Context, payment *models.Payment) error {
    // Get recommended provider from AI fee engine or config
    providerName := payment.ProviderOnRamp // e.g., "Circle" or "Bridge"

    provider, err := h.providerRegistry.GetOnRamp(providerName)
    if err != nil {
        return fmt.Errorf("failed to get provider: %w", err)
    }

    // Use provider interface
    resp, err := provider.InitiateTransfer(ctx, &providers.OnRampRequest{
        Amount:         payment.Amount,
        Currency:       payment.Currency,
        SourceAccount:  payment.SourceAccount,
        Chain:          payment.Chain,
        IdempotencyKey: payment.IdempotencyKey,
    })

    if err != nil {
        return fmt.Errorf("failed to initiate on-ramp: %w", err)
    }

    // Update payment with provider transaction ID
    payment.OnRampTxID = resp.TransactionID
    payment.Status = "ONRAMP_PENDING"

    // Save to database...
    return nil
}
```

## Webhook vs Polling Tradeoffs

### Current Implementation: Polling

**Advantages:**
- Simple to implement
- No webhook infrastructure needed
- Works with any provider
- Easy to test and debug

**Disadvantages:**
- Higher latency (poll every 10-30 seconds)
- More API calls (cost and rate limiting)
- Wastes resources polling completed transactions

### Webhook-Based Implementation

**Advantages:**
- Lower latency (instant notifications)
- Fewer API calls (provider pushes updates)
- More efficient resource usage

**Disadvantages:**
- Requires public webhook endpoint
- Need to handle webhook authentication
- More complex error handling (missed webhooks)
- Provider-specific webhook formats

### Hybrid Approach (Recommended for Production)

```go
// Use webhooks as primary mechanism, polling as fallback

type HybridStateHandler struct {
    providerRegistry *providers.ProviderRegistry
    webhookTimeout   time.Duration // e.g., 5 minutes
}

func (h *HybridStateHandler) handleOnRampPending(ctx context.Context, payment *models.Payment) error {
    // 1. Register webhook with provider (if supported)
    provider, _ := h.providerRegistry.GetOnRamp(payment.ProviderOnRamp)

    if webhookProvider, ok := provider.(providers.WebhookProvider); ok {
        webhookProvider.RegisterWebhook(ctx, payment.OnRampTxID, h.webhookURL)
    }

    // 2. Wait for webhook with timeout
    select {
    case <-h.webhookReceived(payment.OnRampTxID):
        // Webhook received, transition to next state
        return h.handleOnRampComplete(ctx, payment)

    case <-time.After(h.webhookTimeout):
        // Timeout - fall back to polling
        status, err := provider.GetStatus(ctx, payment.OnRampTxID)
        if err != nil {
            return err
        }

        if status.Status == "COMPLETED" {
            return h.handleOnRampComplete(ctx, payment)
        }

        // Re-enqueue for next poll
        return h.requeueWithDelay(payment, 10*time.Second)
    }
}
```

## Provider Selection Strategy

### AI-Powered Routing (Current Implementation)

The AI fee engine already provides provider recommendations:

```json
{
  "recommended_provider": {
    "onramp": "Circle",
    "offramp": "Circle",
    "chain": "base",
    "reasoning": "Base offers zero gas costs..."
  }
}
```

### Extending with Provider Registry

```go
// Store AI recommendation in payment record
payment.ProviderOnRamp = feeResponse.Provider.Onramp   // "Circle"
payment.ProviderOffRamp = feeResponse.Provider.Offramp // "Circle"
payment.Chain = feeResponse.Provider.Chain             // "base"

// State machine uses these fields to select provider
provider, err := providerRegistry.GetOnRamp(payment.ProviderOnRamp)
```

## Testing Strategy

### Unit Tests with Mock Providers

```go
// tests/providers_test.go

func TestCircleProvider(t *testing.T) {
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Mock Circle API responses
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(map[string]interface{}{
            "id": "circle_tx_123",
            "status": "pending",
        })
    }))
    defer mockServer.Close()

    provider := circle.NewCircleOnRampProvider("test_key")
    provider.SetBaseURL(mockServer.URL) // For testing

    resp, err := provider.InitiateTransfer(context.Background(), &providers.OnRampRequest{
        Amount:   100000,
        Currency: "USD",
        Chain:    "base",
    })

    assert.NoError(t, err)
    assert.Equal(t, "circle_tx_123", resp.TransactionID)
}
```

### Integration Tests with Provider Registry

```go
func TestProviderRegistry(t *testing.T) {
    registry := providers.NewProviderRegistry()

    // Register multiple providers
    registry.RegisterOnRamp(circle.NewCircleOnRampProvider("key1"))
    registry.RegisterOnRamp(bridge.NewBridgeOnRampProvider("key2"))

    // Test provider selection
    provider, err := registry.GetOnRamp("Circle")
    assert.NoError(t, err)
    assert.Equal(t, "Circle", provider.Name())

    // Test intelligent routing
    provider, err = registry.SelectBestOnRamp(context.Background(), &providers.OnRampRequest{
        Currency: "USD",
        Chain:    "base",
    })
    assert.NoError(t, err)
    assert.NotNil(t, provider)
}
```

## Migration Path

### Phase 1: Add Provider Interface (No Breaking Changes)

1. Create `internal/providers/interface.go`
2. Create `internal/providers/registry.go`
3. Refactor existing mocks to implement new interface
4. Update state handlers to use registry (with mock providers)

**Result:** Same functionality, better architecture

### Phase 2: Add Real Provider (Circle)

1. Create `internal/providers/circle/onramp.go`
2. Create `internal/providers/circle/offramp.go`
3. Add Circle API credentials to environment
4. Register Circle provider in registry
5. Test in development environment

**Result:** Production-ready provider alongside mocks

### Phase 3: Multi-Provider Support

1. Add Bridge provider
2. Add Coinbase provider
3. Implement intelligent routing in `SelectBestOnRamp`
4. Add failover logic (try provider B if provider A fails)
5. Add provider health monitoring

**Result:** Redundancy and optimal routing

## Production Considerations

### Error Handling

```go
// Retry with exponential backoff
func (h *StateHandler) initiateOnRampWithRetry(ctx context.Context, payment *models.Payment) error {
    provider, _ := h.providerRegistry.GetOnRamp(payment.ProviderOnRamp)

    var resp *providers.OnRampResponse
    var err error

    backoff := time.Second
    maxRetries := 3

    for i := 0; i < maxRetries; i++ {
        resp, err = provider.InitiateTransfer(ctx, &providers.OnRampRequest{...})
        if err == nil {
            break
        }

        // Check if error is retryable
        if !isRetryableError(err) {
            return err
        }

        time.Sleep(backoff)
        backoff *= 2 // Exponential backoff
    }

    if err != nil {
        return fmt.Errorf("failed after %d retries: %w", maxRetries, err)
    }

    return nil
}
```

### Monitoring and Alerting

```go
// Log provider metrics
h.metrics.RecordProviderLatency(provider.Name(), duration)
h.metrics.RecordProviderSuccess(provider.Name())
h.metrics.RecordProviderFailure(provider.Name(), err)

// Alert on high failure rates
if h.metrics.GetProviderFailureRate(provider.Name()) > 0.05 {
    h.alerting.SendAlert("High failure rate for provider: " + provider.Name())
}
```

### Rate Limiting

```go
type RateLimitedProvider struct {
    provider OnRampProvider
    limiter  *rate.Limiter
}

func (r *RateLimitedProvider) InitiateTransfer(ctx context.Context, req *OnRampRequest) (*OnRampResponse, error) {
    if err := r.limiter.Wait(ctx); err != nil {
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }

    return r.provider.InitiateTransfer(ctx, req)
}
```

## Summary

**Current State:**
- Mock providers demonstrate expected behavior
- Polling-based settlement tracking
- Single-provider architecture

**Recommended Next Steps:**
1. Implement provider interface pattern (1-2 hours)
2. Add Circle provider implementation (4-6 hours)
3. Test with real Circle sandbox environment (2 hours)
4. Add provider registry and routing logic (2-3 hours)
5. Implement webhook support for Circle (3-4 hours)
6. Add monitoring and alerting (2-3 hours)

**Total Effort:** ~15-20 hours to production-ready multi-provider system

**Key Benefits:**
- Easy to add new providers (Bridge, Coinbase, etc.)
- Intelligent routing and failover
- Provider-agnostic state machine
- Testable with mocks or real providers
