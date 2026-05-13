package alchemy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

var (
	ErrInvalidRequest   = errors.New("invalid broadcast request")
	ErrMissingAPIKey    = errors.New("alchemy api key is required")
	ErrUnsupportedChain = errors.New("unsupported chain for alchemy broadcast")
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

type BroadcastRequest struct {
	ChainCode     string
	RawTx         string
	Encoding      string
	MaxFeeRate    string
	SkipPreflight *bool
	TxObject      json.RawMessage
}

type BroadcastResponse struct {
	ChainCode string `json:"chain_code"`
	TxHash    string `json:"tx_hash"`
}

type rpcRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  []any  `json:"params"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id"`
	Result  json.RawMessage `json:"result"`
	Error   *rpcError       `json:"error"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type tronBroadcastResponse struct {
	Result  *bool  `json:"result"`
	TxID    string `json:"txid"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func NewClient(apiKey string, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	return &Client{
		apiKey: strings.TrimSpace(apiKey),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Broadcast(ctx context.Context, request BroadcastRequest) (BroadcastResponse, error) {
	chainCode := normalizeChainCode(request.ChainCode)
	if chainCode == "" {
		return BroadcastResponse{}, fmt.Errorf("%w: chain_code is required", ErrInvalidRequest)
	}
	if c == nil || c.apiKey == "" {
		return BroadcastResponse{}, ErrMissingAPIKey
	}

	switch chainCode {
	case "eth":
		return c.broadcastJSONRPC(ctx, chainCode, "eth-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "bsc":
		return c.broadcastJSONRPC(ctx, chainCode, "bnb-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "polygon":
		return c.broadcastJSONRPC(ctx, chainCode, "polygon-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "base":
		return c.broadcastJSONRPC(ctx, chainCode, "base-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "arbitrum":
		return c.broadcastJSONRPC(ctx, chainCode, "arb-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "optimism":
		return c.broadcastJSONRPC(ctx, chainCode, "opt-mainnet", "eth_sendRawTransaction", requiredRawTxParams(request.RawTx))
	case "solana":
		return c.broadcastSolana(ctx, request)
	case "btc":
		return c.broadcastBitcoin(ctx, request)
	case "tron":
		return c.broadcastTron(ctx, request)
	case "ltc", "ton":
		return BroadcastResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedChain, chainCode)
	default:
		return BroadcastResponse{}, fmt.Errorf("%w: %s", ErrUnsupportedChain, chainCode)
	}
}

func requiredRawTxParams(rawTx string) []any {
	rawTx = strings.TrimSpace(rawTx)
	if rawTx == "" {
		return nil
	}

	return []any{rawTx}
}

func (c *Client) broadcastSolana(ctx context.Context, request BroadcastRequest) (BroadcastResponse, error) {
	rawTx := strings.TrimSpace(request.RawTx)
	if rawTx == "" {
		return BroadcastResponse{}, fmt.Errorf("%w: raw_tx is required", ErrInvalidRequest)
	}

	encoding := strings.TrimSpace(request.Encoding)
	if encoding == "" {
		encoding = "base64"
	}
	config := map[string]any{
		"encoding": encoding,
	}
	if request.SkipPreflight != nil {
		config["skipPreflight"] = *request.SkipPreflight
	}

	return c.broadcastJSONRPC(ctx, "solana", "solana-mainnet", "sendTransaction", []any{rawTx, config})
}

func (c *Client) broadcastBitcoin(ctx context.Context, request BroadcastRequest) (BroadcastResponse, error) {
	rawTx := strings.TrimSpace(request.RawTx)
	if rawTx == "" {
		return BroadcastResponse{}, fmt.Errorf("%w: raw_tx is required", ErrInvalidRequest)
	}

	params := []any{rawTx}
	if maxFeeRate := strings.TrimSpace(request.MaxFeeRate); maxFeeRate != "" {
		params = append(params, maxFeeRate)
	}

	return c.broadcastJSONRPC(ctx, "btc", "bitcoin-mainnet", "sendrawtransaction", params)
}

func (c *Client) broadcastTron(ctx context.Context, request BroadcastRequest) (BroadcastResponse, error) {
	rawTx := strings.TrimSpace(request.RawTx)
	if len(request.TxObject) > 0 && string(request.TxObject) != "null" {
		var payload any
		if err := json.Unmarshal(request.TxObject, &payload); err != nil {
			return BroadcastResponse{}, fmt.Errorf("%w: invalid tx_object", ErrInvalidRequest)
		}

		txID, err := c.broadcastTronPayload(ctx, "broadcasttransaction", payload)
		if err != nil {
			return BroadcastResponse{}, err
		}

		return BroadcastResponse{ChainCode: "tron", TxHash: txID}, nil
	}
	if rawTx == "" {
		return BroadcastResponse{}, fmt.Errorf("%w: raw_tx or tx_object is required", ErrInvalidRequest)
	}

	txID, err := c.broadcastTronPayload(ctx, "broadcasthex", map[string]any{"transaction": rawTx})
	if err != nil {
		return BroadcastResponse{}, err
	}

	return BroadcastResponse{ChainCode: "tron", TxHash: txID}, nil
}

func (c *Client) broadcastJSONRPC(ctx context.Context, chainCode string, network string, method string, params []any) (BroadcastResponse, error) {
	if len(params) == 0 {
		return BroadcastResponse{}, fmt.Errorf("%w: raw_tx is required", ErrInvalidRequest)
	}

	var response rpcResponse
	if err := c.postJSON(ctx, fmt.Sprintf("https://%s.g.alchemy.com/v2/%s", network, c.apiKey), rpcRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	}, &response); err != nil {
		return BroadcastResponse{}, err
	}
	if response.Error != nil {
		return BroadcastResponse{}, fmt.Errorf("alchemy rpc error %d: %s", response.Error.Code, response.Error.Message)
	}

	txHash, err := stringResult(response.Result)
	if err != nil {
		return BroadcastResponse{}, err
	}

	return BroadcastResponse{ChainCode: chainCode, TxHash: txHash}, nil
}

func (c *Client) broadcastTronPayload(ctx context.Context, method string, payload any) (string, error) {
	var response tronBroadcastResponse
	if err := c.postJSON(
		ctx,
		fmt.Sprintf("https://tron-mainnet.g.alchemy.com/v2/%s/wallet/%s", c.apiKey, method),
		payload,
		&response,
	); err != nil {
		return "", err
	}

	if response.TxID == "" {
		if response.Code != "" || response.Message != "" {
			return "", fmt.Errorf("alchemy tron error %s: %s", response.Code, response.Message)
		}
		return "", errors.New("alchemy tron response missing txid")
	}
	if response.Result != nil && !*response.Result {
		return "", fmt.Errorf("alchemy tron broadcast rejected transaction: %s", response.TxID)
	}
	if response.Code != "" {
		return "", fmt.Errorf("alchemy tron error %s: %s", response.Code, response.Message)
	}

	return response.TxID, nil
}

func (c *Client) postJSON(ctx context.Context, url string, payload any, output any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		errorBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("alchemy http %d: %s", resp.StatusCode, strings.TrimSpace(string(errorBody)))
	}

	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return err
	}

	return nil
}

func stringResult(raw json.RawMessage) (string, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", errors.New("alchemy response missing result")
	}

	var result string
	if err := json.Unmarshal(raw, &result); err != nil {
		return "", fmt.Errorf("alchemy result is not a string: %w", err)
	}
	if strings.TrimSpace(result) == "" {
		return "", errors.New("alchemy response result is empty")
	}

	return result, nil
}

func normalizeChainCode(chainCode string) string {
	switch strings.ToLower(strings.TrimSpace(chainCode)) {
	case "eth", "ethereum", "ethereum-mainnet", "eth-mainnet":
		return "eth"
	case "bsc", "bnb", "bnb smart chain", "binance smart chain", "bnb-mainnet":
		return "bsc"
	case "polygon", "matic", "polygon-mainnet":
		return "polygon"
	case "base", "base-mainnet":
		return "base"
	case "arb", "arbitrum", "arbitrum-one", "arb-mainnet":
		return "arbitrum"
	case "op", "optimism", "opt", "opt-mainnet":
		return "optimism"
	case "sol", "solana", "solana-mainnet":
		return "solana"
	case "btc", "bitcoin", "bitcoin-mainnet":
		return "btc"
	case "trx", "tron", "tron-mainnet":
		return "tron"
	case "ltc", "litecoin":
		return "ltc"
	case "ton":
		return "ton"
	default:
		return strings.ToLower(strings.TrimSpace(chainCode))
	}
}
