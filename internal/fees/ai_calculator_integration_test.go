package fees

import (
	"context"
	"testing"
	"time"
)

// TestAICalculatorIntegration tests the integration with RealDataProvider
// This test DOES NOT call the Anthropic API (would require API key)
// It verifies that the RealDataProvider integration works correctly
func TestAICalculatorIntegration(t *testing.T) {
	// Create AI calculator (without API key, so it will use fallback)
	calc := NewAIFeeCalculator("")

	// Verify RealDataProvider is initialized
	if calc.realData == nil {
		t.Fatal("RealDataProvider not initialized")
	}

	// Test that we can gather context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	marketCtx, err := calc.realData.GatherContext(ctx)
	if err != nil {
		t.Fatalf("Failed to gather market context: %v", err)
	}

	// Verify market context has real data
	if marketCtx.FXRate == 0 {
		t.Error("FX rate is zero - expected real USD/EUR rate")
	}

	if marketCtx.ETHPriceUSD == 0 {
		t.Error("ETH price is zero - expected real ETH/USD price")
	}

	// Verify we have gas costs for all 5 chains
	expectedChains := []string{"base", "polygon", "arbitrum", "solana", "ethereum"}
	for _, chain := range expectedChains {
		gasCost, exists := marketCtx.GasCosts[chain]
		if !exists {
			t.Errorf("Missing gas cost for chain: %s", chain)
			continue
		}
		if gasCost.Chain != chain {
			t.Errorf("Chain mismatch: expected %s, got %s", chain, gasCost.Chain)
		}
	}

	// Verify provider status is present
	if len(marketCtx.ProviderStatuses) == 0 {
		t.Error("No provider statuses found")
	}

	t.Logf("Market context successfully gathered:")
	t.Logf("  FX Rate (USD/EUR): %.4f", marketCtx.FXRate)
	t.Logf("  ETH Price: $%.2f", marketCtx.ETHPriceUSD)
	t.Logf("  Gas Costs:")
	for chain, cost := range marketCtx.GasCosts {
		t.Logf("    %s: $%.4f (%s)", chain, cost.EstimatedCostUSD, cost.Status)
	}
	t.Logf("  Provider Statuses:")
	for provider, status := range marketCtx.ProviderStatuses {
		t.Logf("    %s: %s", provider, status.Status)
	}
}

// TestAICalculatorFallback tests that fallback works when API key is missing
func TestAICalculatorFallback(t *testing.T) {
	// Create calculator without API key
	calc := NewAIFeeCalculator("")

	ctx := context.Background()
	req := &AIFeeRequest{
		Amount:             100000, // $1000.00
		FromCurrency:       "USD",
		ToCurrency:         "EUR",
		DestinationCountry: "Germany",
		Priority:           "standard",
		CustomerTier:       "standard",
	}

	resp, err := calc.Calculate(ctx, req)
	if err != nil {
		t.Fatalf("Calculate failed: %v", err)
	}

	// Verify fallback response structure
	if resp.TotalFee == 0 {
		t.Error("Total fee is zero")
	}

	if resp.FeeBreakdown.PlatformFee == 0 {
		t.Error("Platform fee is zero")
	}

	if resp.Provider.Chain != "Base" {
		t.Errorf("Expected default chain 'Base', got %s", resp.Provider.Chain)
	}

	if resp.Provider.Onramp != "Circle" {
		t.Errorf("Expected on-ramp 'Circle', got %s", resp.Provider.Onramp)
	}

	if resp.Provider.Offramp != "Circle" {
		t.Errorf("Expected off-ramp 'Circle', got %s", resp.Provider.Offramp)
	}

	// Check that fee breakdown matches expected structure
	expectedTotal := resp.FeeBreakdown.PlatformFee +
		resp.FeeBreakdown.OnrampFee +
		resp.FeeBreakdown.OfframpFee +
		resp.FeeBreakdown.GasCost +
		resp.FeeBreakdown.RiskPremium

	if resp.TotalFee != expectedTotal {
		t.Errorf("Total fee mismatch: expected %d, got %d", expectedTotal, resp.TotalFee)
	}

	t.Logf("Fallback response:")
	t.Logf("  Total Fee: $%.2f", float64(resp.TotalFee)/100)
	t.Logf("  Platform Fee: $%.2f", float64(resp.FeeBreakdown.PlatformFee)/100)
	t.Logf("  On-ramp Fee: $%.2f", float64(resp.FeeBreakdown.OnrampFee)/100)
	t.Logf("  Off-ramp Fee: $%.2f", float64(resp.FeeBreakdown.OfframpFee)/100)
	t.Logf("  Gas Cost: $%.2f", float64(resp.FeeBreakdown.GasCost)/100)
	t.Logf("  Chain: %s", resp.Provider.Chain)
	t.Logf("  Confidence: %.2f", resp.ConfidenceScore)
}

// TestPromptStructure tests that the prompt is built correctly with RealMarketContext
func TestPromptStructure(t *testing.T) {
	calc := NewAIFeeCalculator("")

	// Create real market context
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	marketCtx, err := calc.realData.GatherContext(ctx)
	if err != nil {
		t.Fatalf("Failed to gather market context: %v", err)
	}

	req := &AIFeeRequest{
		Amount:             100000,
		FromCurrency:       "USD",
		ToCurrency:         "EUR",
		DestinationCountry: "Germany",
		Priority:           "standard",
		CustomerTier:       "standard",
	}

	prompt := calc.buildPrompt(req, marketCtx)

	// Verify prompt contains key elements
	if prompt == "" {
		t.Fatal("Prompt is empty")
	}

	// Check for USD→EUR specific content
	if !containsString(prompt, "USD→EUR") {
		t.Error("Prompt does not mention USD→EUR routing")
	}

	// Check for Circle provider
	if !containsString(prompt, "Circle") {
		t.Error("Prompt does not mention Circle")
	}

	// Check for all 5 chains
	chains := []string{"Base", "Polygon", "Arbitrum", "Solana", "Ethereum"}
	for _, chain := range chains {
		if !containsString(prompt, chain) {
			t.Errorf("Prompt does not mention chain: %s", chain)
		}
	}

	// Check for real-time data
	if !containsString(prompt, "REAL-TIME") {
		t.Error("Prompt does not emphasize real-time data")
	}

	t.Logf("Prompt successfully built with %d characters", len(prompt))
	previewLen := 500
	if len(prompt) < previewLen {
		previewLen = len(prompt)
	}
	t.Logf("Prompt preview (first 500 chars):\n%s...", prompt[:previewLen])
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || containsString(s[1:], substr)))
}
