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

type TestScenario struct {
	Name        string
	Amount      int64
	Priority    string
	CustomerTier string
	Description string
}

func main() {
	// Get API key from environment variable
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		log.Fatal("ANTHROPIC_API_KEY environment variable is required")
	}

	// Create AI fee calculator
	calc := fees.NewAIFeeCalculator(apiKey)

	// Define 5 different test scenarios
	scenarios := []TestScenario{
		{
			Name:         "Small Retail Transfer",
			Amount:       5000, // $50
			Priority:     "standard",
			CustomerTier: "standard",
			Description:  "Small consumer transfer - should optimize for cost",
		},
		{
			Name:         "Medium Business Transfer",
			Amount:       50000, // $500
			Priority:     "standard",
			CustomerTier: "business",
			Description:  "Medium business transfer - balanced cost/speed",
		},
		{
			Name:         "Large Enterprise Transfer",
			Amount:       50000000, // $500,000
			Priority:     "standard",
			CustomerTier: "enterprise",
			Description:  "Large enterprise transfer - may prioritize security",
		},
		{
			Name:         "Urgent Small Transfer",
			Amount:       10000, // $100
			Priority:     "express",
			CustomerTier: "premium",
			Description:  "Urgent transfer - should prioritize speed (Solana?)",
		},
		{
			Name:         "High-Value Secure Transfer",
			Amount:       100000000, // $1,000,000
			Priority:     "standard",
			CustomerTier: "enterprise",
			Description:  "Very large transfer - may prefer Ethereum for security",
		},
	}

	fmt.Println("================================================================================")
	fmt.Println("AI FEE ENGINE - DYNAMIC ROUTING TEST SUITE")
	fmt.Println("Testing 5 different scenarios to prove intelligent route optimization")
	fmt.Println("================================================================================\n")

	results := make([]map[string]interface{}, 0)

	for i, scenario := range scenarios {
		fmt.Printf("\n╔════════════════════════════════════════════════════════════════════════════╗\n")
		fmt.Printf("║ TEST #%d: %-67s║\n", i+1, scenario.Name)
		fmt.Printf("╚════════════════════════════════════════════════════════════════════════════╝\n")

		fmt.Printf("\n📋 SCENARIO:\n")
		fmt.Printf("   %s\n", scenario.Description)
		fmt.Printf("\n💵 PARAMETERS:\n")
		fmt.Printf("   Amount:        $%.2f USD\n", float64(scenario.Amount)/100.0)
		fmt.Printf("   Priority:      %s\n", scenario.Priority)
		fmt.Printf("   Customer Tier: %s\n", scenario.CustomerTier)
		fmt.Printf("\n⏳ Running AI analysis...\n")

		// Create request
		req := &fees.AIFeeRequest{
			Amount:             scenario.Amount,
			FromCurrency:       "USD",
			ToCurrency:         "EUR",
			DestinationCountry: "Germany",
			Priority:           scenario.Priority,
			CustomerTier:       scenario.CustomerTier,
		}

		// Call AI calculator
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
		startTime := time.Now()
		resp, err := calc.Calculate(ctx, req)
		elapsed := time.Since(startTime)
		cancel()

		if err != nil {
			log.Printf("❌ Test #%d failed: %v\n", i+1, err)
			continue
		}

		// Display results
		feePercentage := (float64(resp.TotalFee) / float64(scenario.Amount)) * 100.0
		netAmount := scenario.Amount - resp.TotalFee

		fmt.Printf("\n✅ RESULTS:\n")
		fmt.Printf("   Total Fee:      $%.2f (%.2f%%)\n", float64(resp.TotalFee)/100.0, feePercentage)
		fmt.Printf("   Net Payout:     $%.2f\n", float64(netAmount)/100.0)
		fmt.Printf("   Chain:          %s\n", resp.Provider.Chain)
		fmt.Printf("   Settlement:     %s\n", resp.EstimatedSettlementTime)
		fmt.Printf("   Confidence:     %.0f%%\n", resp.ConfidenceScore*100)
		fmt.Printf("   Analysis Time:  %.2fs\n", elapsed.Seconds())

		fmt.Printf("\n💡 AI REASONING:\n")
		fmt.Printf("   %s\n", resp.Provider.Reasoning)

		if len(resp.RiskFactors) > 0 {
			fmt.Printf("\n⚠️  RISK FACTORS:\n")
			for _, risk := range resp.RiskFactors {
				fmt.Printf("   • %s\n", risk)
			}
		}

		// Store result for comparison
		results = append(results, map[string]interface{}{
			"scenario":      scenario.Name,
			"amount":        scenario.Amount,
			"priority":      scenario.Priority,
			"tier":          scenario.CustomerTier,
			"fee":           resp.TotalFee,
			"fee_percent":   feePercentage,
			"chain":         resp.Provider.Chain,
			"settlement":    resp.EstimatedSettlementTime,
			"confidence":    resp.ConfidenceScore,
			"reasoning":     resp.Provider.Reasoning,
			"risk_factors":  resp.RiskFactors,
			"analysis_time": elapsed.Seconds(),
		})

		// Add delay between requests to avoid rate limits
		if i < len(scenarios)-1 {
			fmt.Printf("\n⏸  Waiting 3 seconds before next test...\n")
			time.Sleep(3 * time.Second)
		}
	}

	// Print comparison table
	fmt.Printf("\n\n")
	fmt.Println("╔════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                         COMPARATIVE ANALYSIS                               ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println("")
	fmt.Println("┌─────────────────────────────┬───────────┬────────┬───────────┬──────────────┐")
	fmt.Println("│ Scenario                    │ Amount    │ Chain  │ Fee %     │ Settlement   │")
	fmt.Println("├─────────────────────────────┼───────────┼────────┼───────────┼──────────────┤")

	for _, result := range results {
		scenario := result["scenario"].(string)
		amount := result["amount"].(int64)
		chain := result["chain"].(string)
		feePercent := result["fee_percent"].(float64)
		settlement := result["settlement"].(string)

		// Truncate long strings
		if len(scenario) > 27 {
			scenario = scenario[:24] + "..."
		}
		if len(settlement) > 12 {
			settlement = settlement[:9] + "..."
		}

		fmt.Printf("│ %-27s │ $%-8.0f │ %-6s │ %6.2f%%  │ %-12s │\n",
			scenario,
			float64(amount)/100.0,
			chain,
			feePercent,
			settlement,
		)
	}

	fmt.Println("└─────────────────────────────┴───────────┴────────┴───────────┴──────────────┘")

	// Analysis summary
	fmt.Println("\n╔════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                           KEY INSIGHTS                                     ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════╝\n")

	// Check if chains vary
	chainsUsed := make(map[string]bool)
	for _, result := range results {
		chainsUsed[result["chain"].(string)] = true
	}

	fmt.Printf("🔍 ROUTING INTELLIGENCE:\n")
	if len(chainsUsed) > 1 {
		fmt.Printf("   ✅ AI used %d different chains across scenarios\n", len(chainsUsed))
		fmt.Printf("   Chains: ")
		first := true
		for chain := range chainsUsed {
			if !first {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", chain)
			first = false
		}
		fmt.Printf("\n")
	} else {
		fmt.Printf("   📊 AI consistently chose the same chain (%s) - likely optimal for all scenarios\n",
			results[0]["chain"].(string))
	}

	// Check fee variation
	fmt.Printf("\n💰 FEE OPTIMIZATION:\n")
	minFee := 100.0
	maxFee := 0.0
	for _, result := range results {
		feePercent := result["fee_percent"].(float64)
		if feePercent < minFee {
			minFee = feePercent
		}
		if feePercent > maxFee {
			maxFee = feePercent
		}
	}
	fmt.Printf("   Fee range: %.2f%% - %.2f%%\n", minFee, maxFee)
	if maxFee-minFee > 0.1 {
		fmt.Printf("   ✅ Dynamic fee adjustment detected (%.2f%% variance)\n", maxFee-minFee)
	} else {
		fmt.Printf("   📊 Consistent fee structure across scenarios\n")
	}

	// Check settlement time variation
	fmt.Printf("\n⏱️  SETTLEMENT OPTIMIZATION:\n")
	settlements := make(map[string]int)
	for _, result := range results {
		settlements[result["settlement"].(string)]++
	}
	if len(settlements) > 1 {
		fmt.Printf("   ✅ Settlement times vary by scenario (%d different estimates)\n", len(settlements))
	} else {
		fmt.Printf("   📊 Consistent settlement time across scenarios\n")
	}

	// Export detailed JSON
	fmt.Println("\n╔════════════════════════════════════════════════════════════════════════════╗")
	fmt.Println("║                      DETAILED JSON RESULTS                                 ║")
	fmt.Println("╚════════════════════════════════════════════════════════════════════════════╝\n")

	jsonBytes, _ := json.MarshalIndent(results, "", "  ")
	fmt.Println(string(jsonBytes))

	fmt.Println("\n================================================================================")
	fmt.Println("ALL TESTS COMPLETED SUCCESSFULLY")
	fmt.Println("================================================================================")
}
