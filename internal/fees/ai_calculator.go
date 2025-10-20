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

// AIFeeCalculator uses Claude API for intelligent fee calculation
type AIFeeCalculator struct {
	apiKey       string
	realData     *RealDataProvider
	httpClient   *http.Client
	cacheEnabled bool
}

// NewAIFeeCalculator creates a new AI-powered fee calculator
func NewAIFeeCalculator(apiKey string) *AIFeeCalculator {
	return &AIFeeCalculator{
		apiKey:   apiKey,
		realData: NewRealDataProvider(),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheEnabled: true,
	}
}

// AIFeeRequest represents the request for AI fee calculation
type AIFeeRequest struct {
	Amount              int64  `json:"amount"`
	FromCurrency        string `json:"from_currency"`
	ToCurrency          string `json:"to_currency"`
	DestinationCountry  string `json:"destination_country"`
	Priority            string `json:"priority"`
	CustomerTier        string `json:"customer_tier"`
}

// AIFeeResponse represents the AI-generated fee recommendation
type AIFeeResponse struct {
	TotalFee     int64        `json:"total_fee"`
	FeeBreakdown FeeBreakdown `json:"fee_breakdown"`
	Provider     ProviderRecommendation `json:"recommended_provider"`
	FeeExplanation          string   `json:"fee_explanation"`
	EstimatedSettlementTime string   `json:"estimated_settlement_time"`
	ConfidenceScore         float64  `json:"confidence_score"`
	RiskFactors             []string `json:"risk_factors"`
}

// FeeBreakdown shows component-level fee structure
type FeeBreakdown struct {
	PlatformFee int64 `json:"platform_fee"`
	OnrampFee   int64 `json:"onramp_fee"`
	OfframpFee  int64 `json:"offramp_fee"`
	GasCost     int64 `json:"gas_cost"`
	RiskPremium int64 `json:"risk_premium"`
}

// ProviderRecommendation suggests optimal provider routing
type ProviderRecommendation struct {
	Onramp    string `json:"onramp"`
	Offramp   string `json:"offramp"`
	Chain     string `json:"chain"`
	Reasoning string `json:"reasoning"`
}

// ClaudeRequest represents the API request to Claude
type ClaudeRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
}

// ClaudeMessage represents a message in the conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents the API response from Claude
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// Calculate performs AI-powered fee calculation
func (a *AIFeeCalculator) Calculate(ctx context.Context, req *AIFeeRequest) (*AIFeeResponse, error) {
	// If API key is missing, return fallback response
	if a.apiKey == "" {
		return a.fallbackResponse(req), nil
	}

	// Gather real-time market context
	marketCtx, err := a.realData.GatherContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to gather market context: %w", err)
	}

	// Build prompts for Claude
	systemPrompt, userPrompt := a.buildPrompt(req, marketCtx)

	// Call Claude API
	claudeResp, err := a.callClaudeAPI(ctx, systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("claude API call failed: %w", err)
	}

	// Parse JSON response from Claude
	feeResp, err := a.parseClaudeResponse(claudeResp)
	if err != nil {
		// Return fallback response if parsing fails
		return a.fallbackResponse(req), nil
	}

	return feeResp, nil
}

// buildPrompt constructs the LLM prompt with context
// Returns (systemPrompt, userPrompt)
func (a *AIFeeCalculator) buildPrompt(req *AIFeeRequest, ctx *RealMarketContext) (string, string) {
	systemPrompt := `You are an expert payment orchestration engine for USD→EUR stablecoin transfers. Your role is to analyze real-time market data and optimize routing decisions.

ROUTING FLOW (3 steps):
1. ON-RAMP: USD → USDC (Circle Mint API)
2. BLOCKCHAIN: Move USDC on chain (or cross-chain if needed)
3. OFF-RAMP: USDC → EUR (Circle Redemption API)

You will receive REAL-TIME data:
1. FX Rate: Live USD/EUR exchange rate
2. Gas Costs: Actual gas prices for 5 chains (Base, Polygon, Arbitrum, Solana, Ethereum)
3. Provider Status: Circle operational status for USDC minting/redeeming
4. ETH Price: For accurate gas cost calculation in USD

SUPPORTED CHAINS (all support Circle USDC):
- Base (L2): ~$0.00 gas - DEFAULT CHOICE
- Polygon (Sidechain): ~$0.001 gas - Backup L2
- Arbitrum (L2): ~$0.01 gas - Popular L2
- Solana (L1): ~$0.0009 gas - Fastest settlement
- Ethereum (L1): Variable gas - Maximum security for large transfers

OPTIMIZATION FACTORS:
1. Gas Costs: Minimize blockchain fees (Base is almost always optimal)
2. Provider Status: Verify Circle operational for chosen chain
3. Transfer Amount: Large transfers (>$100K) may justify Ethereum security
4. Speed: Solana for fastest settlement if needed

SETTLEMENT TIME EXPECTATIONS:
- Base L2: 3-5 minutes (L2 finality is very fast)
- Polygon: 5-8 minutes (sidechain with fast blocks)
- Arbitrum L2: 4-6 minutes (L2 optimistic rollup)
- Solana: 2-4 minutes (fastest finality)
- Ethereum L1: 10-15 minutes (L1 finality takes longer)

Note: These times include on-ramp (USD→USDC), blockchain settlement, and off-ramp (USDC→EUR).

FEE STRUCTURE:
- Platform Fee: 2% (our revenue)
- On-ramp Fee: ~0.7% (Circle USD→USDC minting)
- Off-ramp Fee: ~0.5% (Circle USDC→EUR redemption)
- Gas Cost: Chain-specific (real-time)
- Total: ~3.2% + gas

Return ONLY valid JSON with this exact structure:
{
  "total_fee": <number in cents>,
  "fee_breakdown": {
    "platform_fee": <number>,
    "onramp_fee": <number>,
    "offramp_fee": <number>,
    "gas_cost": <number>,
    "risk_premium": <number>
  },
  "recommended_provider": {
    "onramp": "Circle",
    "offramp": "Circle",
    "chain": "<blockchain>",
    "reasoning": "<2-3 sentences explaining why this chain is optimal>"
  },
  "fee_explanation": "<2-3 sentences explaining total fee calculation>",
  "estimated_settlement_time": "<human readable time>",
  "confidence_score": <0.0 to 1.0>,
  "risk_factors": ["<factor1>", "<factor2>"]
}`

	// Marshal context to JSON
	ctxJSON, _ := json.MarshalIndent(ctx, "", "  ")

	userPrompt := fmt.Sprintf(`Payment Request:
- Amount: $%.2f %s → %s
- Customer Tier: %s
- Priority: %s

Real-Time Market Data:
%s

Additional Context:
- Current time: %s
- Target: Minimize total cost while ensuring reliable settlement
- Circle is primary provider for both on-ramp and off-ramp

Calculate optimal fees and routing strategy based on real market data. Return ONLY the JSON response, no other text.`,
		float64(req.Amount)/100.0,
		req.FromCurrency,
		req.ToCurrency,
		req.CustomerTier,
		req.Priority,
		string(ctxJSON),
		time.Now().Format(time.RFC3339),
	)

	return systemPrompt, userPrompt
}

// callClaudeAPI makes the HTTP request to Claude API
func (a *AIFeeCalculator) callClaudeAPI(ctx context.Context, systemPrompt, userPrompt string) (*ClaudeResponse, error) {
	reqBody := ClaudeRequest{
		Model:     "claude-sonnet-4-20250514",
		MaxTokens: 2048,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{
				Role:    "user",
				Content: userPrompt,
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", a.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var claudeResp ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&claudeResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &claudeResp, nil
}

// parseClaudeResponse extracts fee response from Claude's output
func (a *AIFeeCalculator) parseClaudeResponse(claudeResp *ClaudeResponse) (*AIFeeResponse, error) {
	if len(claudeResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	text := claudeResp.Content[0].Text

	// Try to extract JSON from the response
	// Claude might include markdown code blocks, so we need to clean it
	text = cleanJSONResponse(text)

	var feeResp AIFeeResponse
	if err := json.Unmarshal([]byte(text), &feeResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return &feeResp, nil
}

// cleanJSONResponse removes markdown code blocks and extra text
func cleanJSONResponse(text string) string {
	// Remove ```json and ``` markers
	text = bytes.NewBuffer([]byte(text)).String()

	// Find first { and last }
	start := bytes.IndexByte([]byte(text), '{')
	end := bytes.LastIndexByte([]byte(text), '}')

	if start >= 0 && end >= 0 && end > start {
		return text[start : end+1]
	}

	return text
}

// fallbackResponse provides a default response if AI fails
func (a *AIFeeCalculator) fallbackResponse(req *AIFeeRequest) *AIFeeResponse {
	// Calculate basic fee (2% platform fee)
	platformFee := req.Amount * 2 / 100
	onrampFee := req.Amount * 7 / 1000   // 0.7%
	offrampFee := req.Amount * 5 / 1000  // 0.5%
	gasCost := int64(0)                  // Base has ~$0.00 gas
	totalFee := platformFee + onrampFee + offrampFee + gasCost

	return &AIFeeResponse{
		TotalFee: totalFee,
		FeeBreakdown: FeeBreakdown{
			PlatformFee: platformFee,
			OnrampFee:   onrampFee,
			OfframpFee:  offrampFee,
			GasCost:     gasCost,
			RiskPremium: 0,
		},
		Provider: ProviderRecommendation{
			Onramp:    "Circle",
			Offramp:   "Circle",
			Chain:     "Base",
			Reasoning: "Default routing using Circle for both on-ramp and off-ramp with Base chain for minimal gas fees.",
		},
		FeeExplanation:          "Standard 3.2% fee (2% platform + 0.7% on-ramp + 0.5% off-ramp) with negligible gas costs on Base L2.",
		EstimatedSettlementTime: "3-5 minutes",
		ConfidenceScore:         0.75,
		RiskFactors:             []string{"Using fallback calculation - AI analysis unavailable"},
	}
}
