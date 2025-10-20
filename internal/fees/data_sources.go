package fees

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// DataSource is a generic interface for fetching real-time market data
type DataSource interface {
	Fetch(ctx context.Context) (interface{}, error)
	GetName() string
}

// HTTPDataSource is a reusable HTTP client for data sources
type HTTPDataSource struct {
	client  *http.Client
	name    string
	baseURL string
}

// NewHTTPDataSource creates a new HTTP-based data source
func NewHTTPDataSource(name, baseURL string, timeout time.Duration) *HTTPDataSource {
	return &HTTPDataSource{
		name:    name,
		baseURL: baseURL,
		client: &http.Client{
			Timeout: timeout,
		},
	}
}

func (h *HTTPDataSource) GetName() string {
	return h.name
}

// FetchJSON is a helper to fetch and parse JSON from an API
func (h *HTTPDataSource) FetchJSON(ctx context.Context, endpoint string, result interface{}) error {
	url := h.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := h.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	return nil
}

// GasPriceSource fetches gas prices from blockchain explorers
type GasPriceSource struct {
	*HTTPDataSource
	chain string
}

// NewGasPriceSource creates a gas price data source
func NewGasPriceSource(chain string) *GasPriceSource {
	var baseURL string
	switch chain {
	case "ethereum":
		baseURL = "https://beaconcha.in"
	case "base":
		baseURL = "https://base.blockscout.com"
	case "polygon":
		baseURL = "https://polygon.blockscout.com"
	case "arbitrum":
		baseURL = "https://arbitrum.blockscout.com"
	case "solana":
		baseURL = "https://solana.blockscout.com"
	default:
		baseURL = "https://beaconcha.in"
	}

	return &GasPriceSource{
		HTTPDataSource: NewHTTPDataSource(fmt.Sprintf("%s-gas", chain), baseURL, 10*time.Second),
		chain:          chain,
	}
}

// GasOracleResponse represents the response from gas price APIs
type GasOracleResponse struct {
	Code int `json:"code"`
	Data struct {
		Rapid     int64   `json:"rapid"`      // fastest (wei)
		Fast      int64   `json:"fast"`       // fast (wei)
		Standard  int64   `json:"standard"`   // standard (wei)
		Slow      int64   `json:"slow"`       // slow (wei)
		Timestamp int64   `json:"timestamp"`
		Price     float64 `json:"price"`      // ETH price in USD
		PriceUSD  float64 `json:"priceUSD"`
	} `json:"data"`
}

// Fetch retrieves current gas prices
func (g *GasPriceSource) Fetch(ctx context.Context) (interface{}, error) {
	// Solana uses RPC API, different from EVM chains
	if g.chain == "solana" {
		return g.fetchSolanaGas(ctx)
	}

	var endpoint string
	if g.chain == "ethereum" {
		endpoint = "/api/v1/execution/gasnow"
	} else {
		// For Base/Polygon/Arbitrum, use Blockscout stats
		endpoint = "/api/v2/stats"
	}

	var response GasOracleResponse
	err := g.FetchJSON(ctx, endpoint, &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// fetchSolanaGas fetches Solana prioritization fees via RPC
func (g *GasPriceSource) fetchSolanaGas(ctx context.Context) (interface{}, error) {
	// Solana RPC request for recent prioritization fees
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "getRecentPrioritizationFees",
		"params":  []interface{}{[]string{}},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal Solana RPC request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mainnet-beta.solana.com", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create Solana RPC request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Solana RPC request failed: %w", err)
	}
	defer resp.Body.Close()

	var rpcResp struct {
		Result []struct {
			PrioritizationFee int64 `json:"prioritizationFee"`
			Slot              int64 `json:"slot"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&rpcResp); err != nil {
		return nil, fmt.Errorf("failed to decode Solana RPC response: %w", err)
	}

	// Calculate average fee in lamports (1 SOL = 1e9 lamports)
	// Convert to our standard response format
	avgFee := int64(0)
	if len(rpcResp.Result) > 0 {
		sum := int64(0)
		for _, fee := range rpcResp.Result {
			sum += fee.PrioritizationFee
		}
		avgFee = sum / int64(len(rpcResp.Result))
	}

	// Return in our standard format (convert lamports to "gwei-equivalent" for consistency)
	return &GasOracleResponse{
		Code: 200,
		Data: struct {
			Rapid     int64   `json:"rapid"`
			Fast      int64   `json:"fast"`
			Standard  int64   `json:"standard"`
			Slow      int64   `json:"slow"`
			Timestamp int64   `json:"timestamp"`
			Price     float64 `json:"price"`
			PriceUSD  float64 `json:"priceUSD"`
		}{
			Standard:  avgFee,      // Solana fee in lamports
			Fast:      avgFee,
			Rapid:     avgFee,
			Slow:      avgFee,
			Timestamp: time.Now().Unix(),
			Price:     0,
			PriceUSD:  0,
		},
	}, nil
}

// FXRateSource fetches foreign exchange rates
type FXRateSource struct {
	*HTTPDataSource
	baseCurrency string
}

// NewFXRateSource creates an FX rate data source
func NewFXRateSource(baseCurrency string) *FXRateSource {
	return &FXRateSource{
		HTTPDataSource: NewHTTPDataSource("fx-rates", "https://api.exchangerate-api.com", 10*time.Second),
		baseCurrency:   baseCurrency,
	}
}

// FXRateResponse represents the response from exchangerate-api.com
type FXRateResponse struct {
	Provider         string             `json:"provider"`
	Base             string             `json:"base"`
	Date             string             `json:"date"`
	TimeLastUpdated  int64              `json:"time_last_updated"`
	Rates            map[string]float64 `json:"rates"`
}

// Fetch retrieves current FX rates
func (f *FXRateSource) Fetch(ctx context.Context) (interface{}, error) {
	var response FXRateResponse
	endpoint := fmt.Sprintf("/v4/latest/%s", f.baseCurrency)
	err := f.FetchJSON(ctx, endpoint, &response)
	if err != nil {
		return nil, err
	}

	if response.Base == "" {
		return nil, fmt.Errorf("FX API error: invalid response")
	}

	return &response, nil
}

// ProviderStatusSource fetches operational status from status pages
type ProviderStatusSource struct {
	*HTTPDataSource
	provider string
}

// NewProviderStatusSource creates a provider status data source
func NewProviderStatusSource(provider string) *ProviderStatusSource {
	var baseURL string
	switch provider {
	case "coinbase":
		baseURL = "https://status.coinbase.com"
	case "circle":
		baseURL = "https://status.circle.com"
	default:
		baseURL = "https://status.coinbase.com"
	}

	return &ProviderStatusSource{
		HTTPDataSource: NewHTTPDataSource(fmt.Sprintf("%s-status", provider), baseURL, 10*time.Second),
		provider:       provider,
	}
}

// StatusPageResponse represents Atlassian Statuspage API response
type StatusPageResponse struct {
	Page struct {
		ID        string    `json:"id"`
		Name      string    `json:"name"`
		URL       string    `json:"url"`
		UpdatedAt time.Time `json:"updated_at"`
	} `json:"page"`
	Status struct {
		Indicator   string `json:"indicator"`   // "none", "minor", "major", "critical"
		Description string `json:"description"` // "All Systems Operational"
	} `json:"status"`
	Components []struct {
		ID          string    `json:"id"`
		Name        string    `json:"name"`
		Status      string    `json:"status"` // "operational", "degraded_performance", "partial_outage", "major_outage"
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Position    int       `json:"position"`
		Description string    `json:"description"`
		OnlyShowIf  bool      `json:"only_show_if_degraded"`
	} `json:"components"`
}

// Fetch retrieves provider status
func (p *ProviderStatusSource) Fetch(ctx context.Context) (interface{}, error) {
	var response StatusPageResponse
	err := p.FetchJSON(ctx, "/api/v2/summary.json", &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

// ETHPriceSource fetches current ETH price (needed for gas cost calculation)
type ETHPriceSource struct {
	*HTTPDataSource
}

// NewETHPriceSource creates an ETH price data source
func NewETHPriceSource() *ETHPriceSource {
	return &ETHPriceSource{
		HTTPDataSource: NewHTTPDataSource("eth-price", "https://api.coingecko.com", 10*time.Second),
	}
}

// CoinGeckoResponse represents CoinGecko API response
type CoinGeckoResponse struct {
	Ethereum struct {
		USD float64 `json:"usd"`
		EUR float64 `json:"eur"`
	} `json:"ethereum"`
}

// Fetch retrieves current ETH price
func (e *ETHPriceSource) Fetch(ctx context.Context) (interface{}, error) {
	var response CoinGeckoResponse
	err := e.FetchJSON(ctx, "/api/v3/simple/price?ids=ethereum&vs_currencies=usd,eur", &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
