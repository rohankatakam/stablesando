package fees

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"
)

// RealDataProvider fetches live market data for fee optimization
type RealDataProvider struct {
	// Data sources
	gasSources       map[string]*GasPriceSource
	fxSource         *FXRateSource
	providerSources  map[string]*ProviderStatusSource
	ethPriceSource   *ETHPriceSource

	// Caching
	cache            *DataCache
	cacheDuration    time.Duration
}

// DataCache stores fetched data with timestamps
type DataCache struct {
	mu               sync.RWMutex
	gasData          map[string]*CachedGasData
	fxData           *CachedFXData
	providerData     map[string]*CachedProviderData
	ethPrice         *CachedETHPrice
}

type CachedGasData struct {
	Data      *GasOracleResponse
	FetchedAt time.Time
}

type CachedFXData struct {
	Data      *FXRateResponse
	FetchedAt time.Time
}

type CachedProviderData struct {
	Data      *StatusPageResponse
	FetchedAt time.Time
}

type CachedETHPrice struct {
	PriceUSD  float64
	FetchedAt time.Time
}

// NewRealDataProvider creates a new real-time data provider
// Optimized for USD→EUR transfers only
func NewRealDataProvider() *RealDataProvider {
	return &RealDataProvider{
		gasSources: map[string]*GasPriceSource{
			// Optimal 5 chains for USD→EUR transfers (ordered by typical preference)
			"base":     NewGasPriceSource("base"),     // #1: Lowest cost (~$0.00), EVM L2, Coinbase-backed
			"polygon":  NewGasPriceSource("polygon"),  // #2: Very low cost (~$0.001), popular sidechain
			"arbitrum": NewGasPriceSource("arbitrum"), // #3: Low cost (~$0.01), popular EVM L2
			"solana":   NewGasPriceSource("solana"),   // #4: Extremely fast & cheap (~$0.0002), non-EVM
			"ethereum": NewGasPriceSource("ethereum"), // #5: High security, variable cost, most liquid
		},
		fxSource: NewFXRateSource("USD"),
		providerSources: map[string]*ProviderStatusSource{
			// Only providers that support USD→EUR
			"circle": NewProviderStatusSource("circle"),
			// Coinbase removed for now - Circle is primary provider
		},
		ethPriceSource: NewETHPriceSource(),
		cache: &DataCache{
			gasData:      make(map[string]*CachedGasData),
			providerData: make(map[string]*CachedProviderData),
		},
		cacheDuration: 2 * time.Minute, // Cache data for 2 minutes to avoid rate limits
	}
}

// RealMarketContext contains real-time market data for USD→EUR transfers
// Only includes data that directly affects fee calculation
type RealMarketContext struct {
	Timestamp         time.Time                    `json:"timestamp"`
	FXRate            float64                      `json:"fx_rate_usd_eur"`       // Current USD/EUR exchange rate
	ETHPriceUSD       float64                      `json:"eth_price_usd"`         // ETH price for gas cost calculation
	GasCosts          map[string]GasCostEstimate   `json:"gas_costs"`             // Gas costs per chain (Ethereum, Base)
	ProviderStatuses  map[string]ProviderHealth    `json:"provider_statuses"`     // Circle operational status
}

// GasCostEstimate shows the cost to transfer on each chain
type GasCostEstimate struct {
	Chain            string  `json:"chain"`
	GasPrice         float64 `json:"gas_price_gwei"`
	EstimatedCostUSD float64 `json:"estimated_cost_usd"`
	Status           string  `json:"status"` // "low", "medium", "high", "very_high"
}

// ProviderHealth shows operational status of payment providers
type ProviderHealth struct {
	Provider      string   `json:"provider"`
	Status        string   `json:"status"` // "operational", "degraded", "outage"
	IsOperational bool     `json:"is_operational"`
	Issues        []string `json:"issues,omitempty"`
}

// GatherContext fetches all real-time data needed for USD→EUR fee calculation
func (r *RealDataProvider) GatherContext(ctx context.Context) (*RealMarketContext, error) {
	// Use errgroup for concurrent fetching
	var (
		fxRate       float64
		ethPrice     float64
		gasCosts     map[string]GasCostEstimate
		providerStats map[string]ProviderHealth
		err          error
	)

	// Fetch data concurrently
	var wg sync.WaitGroup
	errChan := make(chan error, 4)

	// Fetch FX rate
	wg.Add(1)
	go func() {
		defer wg.Done()
		rate, fetchErr := r.getFXRate(ctx)
		if fetchErr != nil {
			errChan <- fmt.Errorf("FX rate fetch failed: %w", fetchErr)
			return
		}
		fxRate = rate
	}()

	// Fetch ETH price
	wg.Add(1)
	go func() {
		defer wg.Done()
		price, fetchErr := r.getETHPrice(ctx)
		if fetchErr != nil {
			errChan <- fmt.Errorf("ETH price fetch failed: %w", fetchErr)
			return
		}
		ethPrice = price
	}()

	// Fetch gas costs (depends on ETH price, so we'll do it after)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Wait a tiny bit for ETH price to be available
		time.Sleep(100 * time.Millisecond)
		costs, fetchErr := r.getGasCosts(ctx, ethPrice)
		if fetchErr != nil {
			errChan <- fmt.Errorf("gas costs fetch failed: %w", fetchErr)
			return
		}
		gasCosts = costs
	}()

	// Fetch provider statuses
	wg.Add(1)
	go func() {
		defer wg.Done()
		stats, fetchErr := r.getProviderStatuses(ctx)
		if fetchErr != nil {
			errChan <- fmt.Errorf("provider status fetch failed: %w", fetchErr)
			return
		}
		providerStats = stats
	}()

	wg.Wait()
	close(errChan)

	// Check for errors
	for e := range errChan {
		if e != nil {
			err = e
			break
		}
	}

	if err != nil {
		return nil, err
	}

	return &RealMarketContext{
		Timestamp:        time.Now(),
		FXRate:           fxRate,
		ETHPriceUSD:      ethPrice,
		GasCosts:         gasCosts,
		ProviderStatuses: providerStats,
	}, nil
}

// getFXRate fetches current USD/EUR exchange rate
func (r *RealDataProvider) getFXRate(ctx context.Context) (float64, error) {
	// Check cache first
	r.cache.mu.RLock()
	if r.cache.fxData != nil && time.Since(r.cache.fxData.FetchedAt) < r.cacheDuration {
		rate := r.cache.fxData.Data.Rates["EUR"]
		r.cache.mu.RUnlock()
		return rate, nil
	}
	r.cache.mu.RUnlock()

	// Fetch fresh data
	data, err := r.fxSource.Fetch(ctx)
	if err != nil {
		return 0, err
	}

	response := data.(*FXRateResponse)

	// Cache the result
	r.cache.mu.Lock()
	r.cache.fxData = &CachedFXData{
		Data:      response,
		FetchedAt: time.Now(),
	}
	r.cache.mu.Unlock()

	return response.Rates["EUR"], nil
}

// getETHPrice fetches current ETH price in USD
func (r *RealDataProvider) getETHPrice(ctx context.Context) (float64, error) {
	// Check cache first
	r.cache.mu.RLock()
	if r.cache.ethPrice != nil && time.Since(r.cache.ethPrice.FetchedAt) < r.cacheDuration {
		price := r.cache.ethPrice.PriceUSD
		r.cache.mu.RUnlock()
		return price, nil
	}
	r.cache.mu.RUnlock()

	// Fetch fresh data
	data, err := r.ethPriceSource.Fetch(ctx)
	if err != nil {
		return 0, err
	}

	response := data.(*CoinGeckoResponse)

	// Cache the result
	r.cache.mu.Lock()
	r.cache.ethPrice = &CachedETHPrice{
		PriceUSD:  response.Ethereum.USD,
		FetchedAt: time.Now(),
	}
	r.cache.mu.Unlock()

	return response.Ethereum.USD, nil
}

// getGasCosts fetches gas prices and calculates USD costs for each chain
func (r *RealDataProvider) getGasCosts(ctx context.Context, ethPriceUSD float64) (map[string]GasCostEstimate, error) {
	if ethPriceUSD == 0 {
		// Fallback if ETH price fetch failed
		ethPriceUSD = 2000.0
	}

	costs := make(map[string]GasCostEstimate)

	for chain, source := range r.gasSources {
		// Check cache
		r.cache.mu.RLock()
		if cached, ok := r.cache.gasData[chain]; ok && time.Since(cached.FetchedAt) < r.cacheDuration {
			var gasPrice float64
			var costUSD float64

			if chain == "solana" {
				lamports := cached.Data.Data.Standard
				gasPrice = lamportsToSOL(lamports)
				costUSD = calculateSolanaGasCostUSD(lamports, 180.0)
			} else {
				gasPrice = weiToGwei(cached.Data.Data.Standard)
				costUSD = calculateGasCostUSD(gasPrice, ethPriceUSD)
			}

			costs[chain] = GasCostEstimate{
				Chain:            chain,
				GasPrice:         gasPrice,
				EstimatedCostUSD: costUSD,
				Status:           classifyGasPrice(gasPrice, chain),
			}
			r.cache.mu.RUnlock()
			continue
		}
		r.cache.mu.RUnlock()

		// Fetch fresh data
		data, err := source.Fetch(ctx)
		if err != nil {
			// If fetch fails, use fallback
			costs[chain] = GasCostEstimate{
				Chain:            chain,
				GasPrice:         getFallbackGasPrice(chain),
				EstimatedCostUSD: 1.0,
				Status:           "unknown",
			}
			continue
		}

		response := data.(*GasOracleResponse)

		// Cache the result
		r.cache.mu.Lock()
		r.cache.gasData[chain] = &CachedGasData{
			Data:      response,
			FetchedAt: time.Now(),
		}
		r.cache.mu.Unlock()

		var gasPrice float64
		var costUSD float64

		if chain == "solana" {
			// Solana uses lamports, different calculation
			lamports := response.Data.Standard
			gasPrice = lamportsToSOL(lamports) // Convert to SOL for display
			costUSD = calculateSolanaGasCostUSD(lamports, 180.0) // Assume $180 SOL price
		} else {
			// EVM chains use gwei
			gasPrice = weiToGwei(response.Data.Standard)
			costUSD = calculateGasCostUSD(gasPrice, ethPriceUSD)
		}

		costs[chain] = GasCostEstimate{
			Chain:            chain,
			GasPrice:         gasPrice,
			EstimatedCostUSD: costUSD,
			Status:           classifyGasPrice(gasPrice, chain),
		}
	}

	return costs, nil
}

// getProviderStatuses fetches operational status of payment providers
func (r *RealDataProvider) getProviderStatuses(ctx context.Context) (map[string]ProviderHealth, error) {
	statuses := make(map[string]ProviderHealth)

	for provider, source := range r.providerSources {
		// Check cache
		r.cache.mu.RLock()
		if cached, ok := r.cache.providerData[provider]; ok && time.Since(cached.FetchedAt) < r.cacheDuration {
			statuses[provider] = parseProviderHealth(provider, cached.Data)
			r.cache.mu.RUnlock()
			continue
		}
		r.cache.mu.RUnlock()

		// Fetch fresh data
		data, err := source.Fetch(ctx)
		if err != nil {
			// If fetch fails, assume operational (optimistic)
			statuses[provider] = ProviderHealth{
				Provider:      provider,
				Status:        "unknown",
				IsOperational: true,
				Issues:        []string{"Unable to fetch status"},
			}
			continue
		}

		response := data.(*StatusPageResponse)

		// Cache the result
		r.cache.mu.Lock()
		r.cache.providerData[provider] = &CachedProviderData{
			Data:      response,
			FetchedAt: time.Now(),
		}
		r.cache.mu.Unlock()

		statuses[provider] = parseProviderHealth(provider, response)
	}

	return statuses, nil
}

// Helper functions

func weiToGwei(wei int64) float64 {
	// 1 gwei = 1e9 wei
	return float64(wei) / 1e9
}

func lamportsToSOL(lamports int64) float64 {
	// 1 SOL = 1e9 lamports
	return float64(lamports) / 1e9
}

func calculateGasCostUSD(gasPriceGwei, ethPriceUSD float64) float64 {
	// Standard USDC transfer uses ~65,000 gas
	// ERC-20 transfer gas limit
	gasLimit := 65000.0

	// Convert gwei to ETH: 1 ETH = 1e9 gwei
	gasInETH := (gasPriceGwei * gasLimit) / 1e9

	// Convert to USD
	return gasInETH * ethPriceUSD
}

func calculateSolanaGasCostUSD(lamports int64, solPriceUSD float64) float64 {
	// Solana USDC transfer typically costs ~5000 lamports (0.000005 SOL)
	// This is a fixed cost, not variable like EVM
	baseTransferCost := 5000.0

	// Convert lamports to SOL
	costInSOL := (float64(lamports) + baseTransferCost) / 1e9

	// Convert to USD (SOL price ~$150-200 typically)
	if solPriceUSD == 0 {
		solPriceUSD = 180.0 // Fallback SOL price
	}
	return costInSOL * solPriceUSD
}

func classifyGasPrice(gasPrice float64, chain string) string {
	// Classification thresholds vary by chain
	switch chain {
	case "ethereum":
		if gasPrice < 20 {
			return "low"
		} else if gasPrice < 50 {
			return "medium"
		} else if gasPrice < 100 {
			return "high"
		}
		return "very_high"
	case "base":
		// Base is subsidized, almost always free
		if gasPrice < 1 {
			return "low"
		} else if gasPrice < 5 {
			return "medium"
		}
		return "high"
	case "polygon":
		// Polygon uses POL token
		if gasPrice < 30 {
			return "low"
		} else if gasPrice < 80 {
			return "medium"
		} else if gasPrice < 150 {
			return "high"
		}
		return "very_high"
	case "arbitrum":
		// Arbitrum L2, typically very cheap
		if gasPrice < 0.5 {
			return "low"
		} else if gasPrice < 2 {
			return "medium"
		} else if gasPrice < 5 {
			return "high"
		}
		return "very_high"
	case "solana":
		// Solana measures in lamports, extremely cheap
		if gasPrice < 0.001 {
			return "low"
		} else if gasPrice < 0.01 {
			return "medium"
		}
		return "high"
	default:
		return "unknown"
	}
}

func getFallbackGasPrice(chain string) float64 {
	fallbacks := map[string]float64{
		"ethereum": 30.0,  // 30 gwei typical
		"base":     0.5,   // Very low, subsidized
		"polygon":  50.0,  // 50 gwei typical
		"arbitrum": 0.1,   // Very low L2
		"solana":   0.001, // Extremely low
	}
	if price, ok := fallbacks[chain]; ok {
		return price
	}
	return 10.0
}

func parseProviderHealth(provider string, status *StatusPageResponse) ProviderHealth {
	health := ProviderHealth{
		Provider:      provider,
		Status:        "operational",
		IsOperational: true,
		Issues:        []string{},
	}

	// Define critical components for USD→EUR transfers (all 5 optimal chains)
	criticalComponents := map[string][]string{
		"circle": {
			"Circle Mint APIs",
			"USDC",
			"USDC - BASE - Minting",     // Base (L2)
			"USDC - BASE - Redeeming",
			"USDC - POLY - Minting",     // Polygon (Sidechain)
			"USDC - POLY - Redeeming",
			"USDC - ARB - Minting",      // Arbitrum (L2)
			"USDC - ARB - Redeeming",
			"USDC - SOL - Minting",      // Solana (L1)
			"USDC - SOL - Redeeming",
			"USDC - ETH - Minting",      // Ethereum (L1)
			"USDC - ETH - Redeeming",
		},
		"coinbase": {
			"Coinbase - Website",
			"Bitcoin", // Proxy for overall blockchain operations
			"APIs",
		},
	}

	relevantComponents, ok := criticalComponents[provider]
	if !ok {
		// If provider not defined, check all components
		relevantComponents = nil
	}

	// Check individual components
	criticalIssues := []string{}
	minorIssues := []string{}

	for _, component := range status.Components {
		// If we have a whitelist, only check those components
		isRelevant := relevantComponents == nil
		if relevantComponents != nil {
			for _, critical := range relevantComponents {
				if component.Name == critical ||
				   (len(component.Name) > len(critical) && component.Name[:len(critical)] == critical) {
					isRelevant = true
					break
				}
			}
		}

		if !isRelevant {
			continue
		}

		// Check component status
		if component.Status == "major_outage" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("%s: %s", component.Name, component.Status))
			health.Status = "outage"
			health.IsOperational = false
		} else if component.Status == "partial_outage" {
			criticalIssues = append(criticalIssues, fmt.Sprintf("%s: %s", component.Name, component.Status))
			if health.Status == "operational" {
				health.Status = "degraded"
			}
		} else if component.Status == "degraded_performance" {
			minorIssues = append(minorIssues, fmt.Sprintf("%s: %s", component.Name, component.Status))
			// Don't change operational status for non-critical degradations
		}
	}

	// Only include critical issues - don't pollute with non-critical degradations
	health.Issues = criticalIssues

	// Only add overall status if there are actual operational issues with critical components
	if len(criticalIssues) == 0 && status.Status.Indicator == "major" {
		health.Status = "degraded"
		health.Issues = append(health.Issues, status.Status.Description)
	}

	return health
}

// CalculateOptimalRoute determines the best routing based on real market data
func (r *RealDataProvider) CalculateOptimalRoute(ctx context.Context, amountUSD int64) (*RouteRecommendation, error) {
	marketCtx, err := r.GatherContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to gather market context: %w", err)
	}

	// Find cheapest gas chain
	cheapestChain := "base"
	lowestGasCost := math.MaxFloat64
	for chain, gasCost := range marketCtx.GasCosts {
		if gasCost.EstimatedCostUSD < lowestGasCost {
			lowestGasCost = gasCost.EstimatedCostUSD
			cheapestChain = chain
		}
	}

	// Find best provider (prefer operational over degraded)
	bestProvider := "circle"
	for provider, health := range marketCtx.ProviderStatuses {
		if health.IsOperational && health.Status == "operational" {
			bestProvider = provider
			break
		}
	}

	return &RouteRecommendation{
		Chain:     cheapestChain,
		Provider:  bestProvider,
		GasCostUSD: lowestGasCost,
		Reasoning: fmt.Sprintf("Selected %s chain (gas: $%.2f) with %s provider (status: %s)",
			cheapestChain, lowestGasCost, bestProvider, marketCtx.ProviderStatuses[bestProvider].Status),
	}, nil
}

// RouteRecommendation represents the optimal routing decision
type RouteRecommendation struct {
	Chain      string  `json:"chain"`
	Provider   string  `json:"provider"`
	GasCostUSD float64 `json:"gas_cost_usd"`
	Reasoning  string  `json:"reasoning"`
}
