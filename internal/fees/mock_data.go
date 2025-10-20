package fees

import "time"

// MockDataProvider provides simulated market data for AI fee calculation
type MockDataProvider struct{}

// NewMockDataProvider creates a new mock data provider
func NewMockDataProvider() *MockDataProvider {
	return &MockDataProvider{}
}

// ProviderStatus represents the operational status of a payment provider
type ProviderStatus struct {
	Name              string  `json:"name"`
	Status            string  `json:"status"`
	Uptime24h         float64 `json:"uptime_24h"`
	AvgSettlementTime string  `json:"avg_settlement_time"`
	BaseFee           float64 `json:"base_fee"`
	SupportedChains   []string `json:"supported_chains"`
}

// GasPrice represents blockchain gas prices
type GasPrice struct {
	Chain  string  `json:"chain"`
	Price  float64 `json:"price"`
	Unit   string  `json:"unit"`
	Status string  `json:"status"`
}

// FXVolatility represents foreign exchange volatility data
type FXVolatility struct {
	Pair          string  `json:"pair"`
	CurrentRate   float64 `json:"current_rate"`
	Volatility1h  float64 `json:"volatility_1h"`
	Volatility24h float64 `json:"volatility_24h"`
	Status        string  `json:"status"`
}

// CountryRisk represents destination country risk scores
type CountryRisk struct {
	Country   string  `json:"country"`
	RiskScore float64 `json:"risk_score"`
	Tier      string  `json:"tier"`
}

// LiquidityDepth represents available liquidity for currency pairs
type LiquidityDepth struct {
	Provider string `json:"provider"`
	Currency string `json:"currency"`
	Depth    int64  `json:"depth"`
}

// MarketContext aggregates all market data for AI analysis
type MarketContext struct {
	Timestamp      time.Time            `json:"timestamp"`
	Providers      []ProviderStatus     `json:"providers"`
	GasPrices      []GasPrice           `json:"gas_prices"`
	FXVolatility   []FXVolatility       `json:"fx_volatility"`
	CountryRisks   []CountryRisk        `json:"country_risks"`
	LiquidityDepth []LiquidityDepth     `json:"liquidity_depth"`
}

// GetProviderStatus returns mock provider status data
func (m *MockDataProvider) GetProviderStatus() []ProviderStatus {
	return []ProviderStatus{
		{
			Name:              "Circle",
			Status:            "operational",
			Uptime24h:         99.9,
			AvgSettlementTime: "11.2min",
			BaseFee:           0.007,
			SupportedChains:   []string{"Ethereum", "Base", "Polygon"},
		},
		{
			Name:              "Bridge",
			Status:            "investigating_delays",
			Uptime24h:         88.2,
			AvgSettlementTime: "18.5min",
			BaseFee:           0.005,
			SupportedChains:   []string{"Ethereum", "Polygon", "Solana"},
		},
		{
			Name:              "Coinbase",
			Status:            "operational",
			Uptime24h:         99.5,
			AvgSettlementTime: "9.8min",
			BaseFee:           0.008,
			SupportedChains:   []string{"Ethereum", "Base"},
		},
	}
}

// GetGasPrices returns mock gas price data
func (m *MockDataProvider) GetGasPrices() []GasPrice {
	return []GasPrice{
		{Chain: "Ethereum", Price: 85.5, Unit: "gwei", Status: "high"},
		{Chain: "Base", Price: 0.8, Unit: "gwei", Status: "low"},
		{Chain: "Polygon", Price: 120.3, Unit: "gwei", Status: "very_high"},
		{Chain: "Solana", Price: 0.005, Unit: "SOL", Status: "low"},
	}
}

// GetFXVolatility returns mock FX volatility data
func (m *MockDataProvider) GetFXVolatility(pair string) FXVolatility {
	volatilityData := map[string]FXVolatility{
		"USD/EUR": {
			Pair:          "USD/EUR",
			CurrentRate:   0.9205,
			Volatility1h:  0.0075,
			Volatility24h: 0.0121,
			Status:        "elevated",
		},
		"USD/BRL": {
			Pair:          "USD/BRL",
			CurrentRate:   5.1234,
			Volatility1h:  0.0155,
			Volatility24h: 0.0310,
			Status:        "high",
		},
		"USD/GBP": {
			Pair:          "USD/GBP",
			CurrentRate:   0.7891,
			Volatility1h:  0.0045,
			Volatility24h: 0.0089,
			Status:        "normal",
		},
		"EUR/USD": {
			Pair:          "EUR/USD",
			CurrentRate:   1.0864,
			Volatility1h:  0.0075,
			Volatility24h: 0.0121,
			Status:        "elevated",
		},
	}

	if vol, ok := volatilityData[pair]; ok {
		return vol
	}

	// Default for unknown pairs
	return FXVolatility{
		Pair:          pair,
		CurrentRate:   1.0,
		Volatility1h:  0.005,
		Volatility24h: 0.01,
		Status:        "normal",
	}
}

// GetCountryRisk returns mock country risk data
func (m *MockDataProvider) GetCountryRisk(country string) CountryRisk {
	riskData := map[string]CountryRisk{
		"Germany":   {Country: "Germany", RiskScore: 1.0, Tier: "low"},
		"Brazil":    {Country: "Brazil", RiskScore: 4.5, Tier: "medium-high"},
		"Nigeria":   {Country: "Nigeria", RiskScore: 6.2, Tier: "high"},
		"Singapore": {Country: "Singapore", RiskScore: 1.2, Tier: "low"},
		"USA":       {Country: "USA", RiskScore: 1.1, Tier: "low"},
		"UK":        {Country: "UK", RiskScore: 1.3, Tier: "low"},
	}

	if risk, ok := riskData[country]; ok {
		return risk
	}

	// Default for unknown countries
	return CountryRisk{
		Country:   country,
		RiskScore: 3.0,
		Tier:      "medium",
	}
}

// GetLiquidityDepth returns mock liquidity data
func (m *MockDataProvider) GetLiquidityDepth() []LiquidityDepth {
	return []LiquidityDepth{
		{Provider: "Circle", Currency: "EUR", Depth: 5000000},
		{Provider: "Bridge", Currency: "EUR", Depth: 12000000},
		{Provider: "Circle", Currency: "BRL", Depth: 800000},
		{Provider: "Bridge", Currency: "BRL", Depth: 3500000},
		{Provider: "Circle", Currency: "GBP", Depth: 4200000},
		{Provider: "Coinbase", Currency: "EUR", Depth: 8500000},
		{Provider: "Coinbase", Currency: "GBP", Depth: 6100000},
	}
}

// GatherContext collects all market data for AI analysis
func (m *MockDataProvider) GatherContext(fromCurrency, toCurrency, destinationCountry string) *MarketContext {
	// Build currency pair
	pair := fromCurrency + "/" + toCurrency

	return &MarketContext{
		Timestamp:      time.Now(),
		Providers:      m.GetProviderStatus(),
		GasPrices:      m.GetGasPrices(),
		FXVolatility:   []FXVolatility{m.GetFXVolatility(pair)},
		CountryRisks:   []CountryRisk{m.GetCountryRisk(destinationCountry)},
		LiquidityDepth: m.GetLiquidityDepth(),
	}
}
