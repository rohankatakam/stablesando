package fees

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestRealDataProvider_GatherContext(t *testing.T) {
	provider := NewRealDataProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	marketCtx, err := provider.GatherContext(ctx)
	if err != nil {
		t.Fatalf("Failed to gather market context: %v", err)
	}

	// Print the full context for inspection
	jsonData, _ := json.MarshalIndent(marketCtx, "", "  ")
	t.Logf("Market Context:\n%s", string(jsonData))

	// Verify FX rate is reasonable (USD/EUR typically 0.85-1.10)
	if marketCtx.FXRate < 0.80 || marketCtx.FXRate > 1.20 {
		t.Errorf("Unexpected FX rate: %.4f (expected 0.80-1.20)", marketCtx.FXRate)
	}

	// Verify ETH price is reasonable ($1000-$10000 typical range)
	if marketCtx.ETHPriceUSD < 1000 || marketCtx.ETHPriceUSD > 10000 {
		t.Errorf("Unexpected ETH price: $%.2f (expected $1000-$10000)", marketCtx.ETHPriceUSD)
	}

	// Verify we have gas costs for all 5 optimal USD→EUR chains
	expectedChains := []string{"base", "polygon", "arbitrum", "solana", "ethereum"}
	for _, chain := range expectedChains {
		if _, ok := marketCtx.GasCosts[chain]; !ok {
			t.Errorf("Missing gas cost data for chain: %s", chain)
		}
	}

	// Verify we have Circle provider status (primary for USD→EUR)
	if _, ok := marketCtx.ProviderStatuses["circle"]; !ok {
		t.Errorf("Missing provider status for: circle")
	}

	// Log gas costs
	t.Log("\nGas Costs:")
	for chain, cost := range marketCtx.GasCosts {
		t.Logf("  %s: %.2f gwei ($%.4f USD) - %s",
			chain, cost.GasPrice, cost.EstimatedCostUSD, cost.Status)
	}

	// Log provider statuses
	t.Log("\nProvider Statuses:")
	for provider, health := range marketCtx.ProviderStatuses {
		t.Logf("  %s: %s (operational: %v)",
			provider, health.Status, health.IsOperational)
		if len(health.Issues) > 0 {
			t.Logf("    Issues: %v", health.Issues)
		}
	}
}

func TestRealDataProvider_CalculateOptimalRoute(t *testing.T) {
	provider := NewRealDataProvider()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	route, err := provider.CalculateOptimalRoute(ctx, 100000) // $1000
	if err != nil {
		t.Fatalf("Failed to calculate optimal route: %v", err)
	}

	t.Logf("Optimal Route for $1000 USD→EUR transfer:")
	t.Logf("  Chain: %s", route.Chain)
	t.Logf("  Provider: %s", route.Provider)
	t.Logf("  Gas Cost: $%.4f", route.GasCostUSD)
	t.Logf("  Reasoning: %s", route.Reasoning)

	// Verify route makes sense
	validChains := map[string]bool{"ethereum": true, "base": true, "polygon": true}
	if !validChains[route.Chain] {
		t.Errorf("Invalid chain selected: %s", route.Chain)
	}

	validProviders := map[string]bool{"coinbase": true, "circle": true}
	if !validProviders[route.Provider] {
		t.Errorf("Invalid provider selected: %s", route.Provider)
	}

	// Gas cost should be reasonable (< $100)
	if route.GasCostUSD < 0 || route.GasCostUSD > 100 {
		t.Errorf("Unexpected gas cost: $%.2f", route.GasCostUSD)
	}
}

func TestIndividualDataSources(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("FX Rate Source", func(t *testing.T) {
		source := NewFXRateSource("USD")
		data, err := source.Fetch(ctx)
		if err != nil {
			t.Fatalf("FX rate fetch failed: %v", err)
		}

		response := data.(*FXRateResponse)
		t.Logf("FX Rates from %s:", response.Base)
		t.Logf("  EUR: %.4f", response.Rates["EUR"])
		t.Logf("  GBP: %.4f", response.Rates["GBP"])
		t.Logf("  Date: %s", response.Date)
	})

	t.Run("Gas Price Sources", func(t *testing.T) {
		chains := []string{"base", "polygon", "arbitrum", "solana", "ethereum"}
		for _, chain := range chains {
			source := NewGasPriceSource(chain)
			data, err := source.Fetch(ctx)
			if err != nil {
				t.Logf("Warning: %s gas price fetch failed: %v", chain, err)
				continue
			}

			response := data.(*GasOracleResponse)
			t.Logf("%s Gas Prices:", chain)
			t.Logf("  Slow: %.2f gwei", float64(response.Data.Slow)/1e9)
			t.Logf("  Standard: %.2f gwei", float64(response.Data.Standard)/1e9)
			t.Logf("  Fast: %.2f gwei", float64(response.Data.Fast)/1e9)
			t.Logf("  Rapid: %.2f gwei", float64(response.Data.Rapid)/1e9)
		}
	})

	t.Run("Provider Status Sources", func(t *testing.T) {
		providers := []string{"coinbase", "circle"}
		for _, provider := range providers {
			source := NewProviderStatusSource(provider)
			data, err := source.Fetch(ctx)
			if err != nil {
				t.Logf("Warning: %s status fetch failed: %v", provider, err)
				continue
			}

			response := data.(*StatusPageResponse)
			t.Logf("%s Status:", provider)
			t.Logf("  Overall: %s - %s", response.Status.Indicator, response.Status.Description)
			t.Logf("  Components: %d", len(response.Components))
			for _, comp := range response.Components[:min(3, len(response.Components))] {
				t.Logf("    - %s: %s", comp.Name, comp.Status)
			}
		}
	})

	t.Run("ETH Price Source", func(t *testing.T) {
		source := NewETHPriceSource()
		data, err := source.Fetch(ctx)
		if err != nil {
			t.Fatalf("ETH price fetch failed: %v", err)
		}

		response := data.(*CoinGeckoResponse)
		t.Logf("ETH Price:")
		t.Logf("  USD: $%.2f", response.Ethereum.USD)
		t.Logf("  EUR: €%.2f", response.Ethereum.EUR)
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
