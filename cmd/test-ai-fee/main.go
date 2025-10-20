package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"crypto-conversion/internal/fees"
)

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create AI fee calculator
	calc := fees.NewAIFeeCalculator(apiKey)

	// Create test request for $1000 USD -> EUR
	req := &fees.AIFeeRequest{
		Amount:             100000, // $1000.00 in cents
		FromCurrency:       "USD",
		ToCurrency:         "EUR",
		DestinationCountry: "Germany",
		Priority:           "standard",
		CustomerTier:       "standard",
	}

	fmt.Println("================================================================================")
	fmt.Println("AI FEE ENGINE TEST - USD → EUR Transfer")
	fmt.Println("================================================================================")
	fmt.Printf("\nTransaction Details:\n")
	fmt.Printf("  Amount: $%.2f USD\n", float64(req.Amount)/100.0)
	fmt.Printf("  Destination: %s (%s)\n", req.ToCurrency, req.DestinationCountry)
	fmt.Printf("  Priority: %s\n", req.Priority)
	fmt.Printf("  Customer Tier: %s\n", req.CustomerTier)
	fmt.Println("\nGathering real-time market data...")

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Call AI fee calculator
	startTime := time.Now()
	resp, err := calc.Calculate(ctx, req)
	elapsed := time.Since(startTime)

	if err != nil {
		log.Fatalf("AI fee calculation failed: %v", err)
	}

	// Display results
	fmt.Println("\n================================================================================")
	fmt.Println("AI ANALYSIS RESULTS")
	fmt.Println("================================================================================")

	fmt.Printf("\n💰 TOTAL FEE: $%.2f (%.2f%%)\n",
		float64(resp.TotalFee)/100.0,
		(float64(resp.TotalFee)/float64(req.Amount))*100.0)

	fmt.Printf("\n📊 FEE BREAKDOWN:\n")
	fmt.Printf("  Platform Fee:    $%.2f (%.2f%%)\n",
		float64(resp.FeeBreakdown.PlatformFee)/100.0,
		(float64(resp.FeeBreakdown.PlatformFee)/float64(req.Amount))*100.0)
	fmt.Printf("  On-ramp Fee:     $%.2f (%.2f%%)\n",
		float64(resp.FeeBreakdown.OnrampFee)/100.0,
		(float64(resp.FeeBreakdown.OnrampFee)/float64(req.Amount))*100.0)
	fmt.Printf("  Off-ramp Fee:    $%.2f (%.2f%%)\n",
		float64(resp.FeeBreakdown.OfframpFee)/100.0,
		(float64(resp.FeeBreakdown.OfframpFee)/float64(req.Amount))*100.0)
	fmt.Printf("  Gas Cost:        $%.2f\n",
		float64(resp.FeeBreakdown.GasCost)/100.0)
	fmt.Printf("  Risk Premium:    $%.2f\n",
		float64(resp.FeeBreakdown.RiskPremium)/100.0)

	fmt.Printf("\n🚀 RECOMMENDED ROUTING:\n")
	fmt.Printf("  On-ramp Provider:  %s\n", resp.Provider.Onramp)
	fmt.Printf("  Blockchain Chain:  %s\n", resp.Provider.Chain)
	fmt.Printf("  Off-ramp Provider: %s\n", resp.Provider.Offramp)

	fmt.Printf("\n💡 ROUTING REASONING:\n")
	fmt.Printf("  %s\n", resp.Provider.Reasoning)

	fmt.Printf("\n📝 FEE EXPLANATION:\n")
	fmt.Printf("  %s\n", resp.FeeExplanation)

	fmt.Printf("\n⏱️  ESTIMATED SETTLEMENT TIME:\n")
	fmt.Printf("  %s\n", resp.EstimatedSettlementTime)

	fmt.Printf("\n🎯 CONFIDENCE SCORE:\n")
	fmt.Printf("  %.1f%% (%.2f/1.0)\n", resp.ConfidenceScore*100, resp.ConfidenceScore)

	if len(resp.RiskFactors) > 0 {
		fmt.Printf("\n⚠️  RISK FACTORS:\n")
		for _, risk := range resp.RiskFactors {
			fmt.Printf("  • %s\n", risk)
		}
	}

	// Calculate final payout
	finalPayout := req.Amount - resp.TotalFee
	fmt.Printf("\n💵 FINAL PAYOUT:\n")
	fmt.Printf("  $%.2f USD sent\n", float64(req.Amount)/100.0)
	fmt.Printf("  - $%.2f USD fees\n", float64(resp.TotalFee)/100.0)
	fmt.Printf("  = $%.2f USD net\n", float64(finalPayout)/100.0)

	fmt.Printf("\n⚡ PERFORMANCE:\n")
	fmt.Printf("  AI analysis completed in %.2f seconds\n", elapsed.Seconds())

	// Print raw JSON for debugging
	fmt.Println("\n================================================================================")
	fmt.Println("RAW JSON RESPONSE:")
	fmt.Println("================================================================================\n")
	jsonBytes, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(jsonBytes))

	fmt.Println("\n================================================================================")
	fmt.Println("TEST COMPLETED SUCCESSFULLY")
	fmt.Println("================================================================================")
}
