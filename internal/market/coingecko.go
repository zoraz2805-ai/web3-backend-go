package market

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CoinGeckoClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

type TokenMarketData struct {
	PriceUSD     string
	MarketCapUSD float64
	VolumeUSD24H float64
}

type simplePriceResponse map[string]struct {
	USD          float64 `json:"usd"`
	USDMarketCap float64 `json:"usd_market_cap"`
	USD24HVol    float64 `json:"usd_24h_vol"`
}

func NewCoinGeckoClient(baseURL string, apiKey string, timeout time.Duration) *CoinGeckoClient {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = "https://api.coingecko.com/api/v3"
	}
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &CoinGeckoClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *CoinGeckoClient) SimplePrice(ctx context.Context, ids []string) (map[string]TokenMarketData, error) {
	ids = uniqueNonEmpty(ids)
	if len(ids) == 0 {
		return map[string]TokenMarketData{}, nil
	}

	endpoint, err := url.Parse(c.baseURL + "/simple/price")
	if err != nil {
		return nil, err
	}

	query := endpoint.Query()
	query.Set("ids", strings.Join(ids, ","))
	query.Set("vs_currencies", "usd")
	query.Set("include_market_cap", "true")
	query.Set("include_24hr_vol", "true")
	endpoint.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, err
	}
	if c.apiKey != "" {
		req.Header.Set("x-cg-demo-api-key", c.apiKey)
		req.Header.Set("x-cg-pro-api-key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("coingecko status: %d", resp.StatusCode)
	}

	var payload simplePriceResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	result := make(map[string]TokenMarketData, len(payload))
	for id, data := range payload {
		result[id] = TokenMarketData{
			PriceUSD:     formatUSDPrice(data.USD),
			MarketCapUSD: data.USDMarketCap,
			VolumeUSD24H: data.USD24HVol,
		}
	}

	return result, nil
}

func uniqueNonEmpty(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		normalized := strings.TrimSpace(value)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}

	return result
}

func formatUSDPrice(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.12f", value), "0"), ".")
}
