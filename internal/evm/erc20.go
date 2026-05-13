package evm

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

const erc20ABIJSON = `[
	{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"type":"function"},
	{"constant":true,"inputs":[],"name":"symbol","outputs":[{"name":"","type":"string"}],"type":"function"}
]`

var erc20ABI = mustParseERC20ABI()

type TokenBalance struct {
	Owner        string `json:"owner"`
	TokenAddress string `json:"tokenAddress"`
	Symbol       string `json:"symbol"`
	Decimals     uint8  `json:"decimals"`
	Raw          string `json:"raw"`
	Formatted    string `json:"formatted"`
}

type balanceAsset struct {
	Asset        string
	Type         string
	Symbol       string
	Decimals     uint8
	TokenAddress string
}

var stablecoinContracts = map[string]map[string]string{
	"1": {
		"USDT": "0xdAC17F958D2ee523a2206206994597C13D831ec7",
		"USDC": "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
	},
	"56": {
		"USDT": "0x55d398326f99059fF775485246999027B3197955",
		"USDC": "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d",
	},
}

func (c *Client) TokenBalance(ctx context.Context, ownerAddress string, tokenAddress string) (TokenBalance, error) {
	if c == nil || c.Client == nil {
		return TokenBalance{}, ErrClientDisabled
	}
	if !common.IsHexAddress(ownerAddress) {
		return TokenBalance{}, ErrInvalidAddress
	}
	if !common.IsHexAddress(tokenAddress) {
		return TokenBalance{}, ErrInvalidTokenAddress
	}

	owner := common.HexToAddress(ownerAddress)
	token := common.HexToAddress(tokenAddress)

	raw, err := c.callUint256(ctx, token, "balanceOf", owner)
	if err != nil {
		return TokenBalance{}, fmt.Errorf("call balanceOf: %w", err)
	}

	decimals, err := c.callUint8(ctx, token, "decimals")
	if err != nil {
		return TokenBalance{}, fmt.Errorf("call decimals: %w", err)
	}

	symbol, err := c.callString(ctx, token, "symbol")
	if err != nil {
		return TokenBalance{}, fmt.Errorf("call symbol: %w", err)
	}

	return TokenBalance{
		Owner:        owner.Hex(),
		TokenAddress: token.Hex(),
		Symbol:       symbol,
		Decimals:     decimals,
		Raw:          raw.String(),
		Formatted:    formatTokenAmount(raw, decimals),
	}, nil
}

func (c *Client) resolveBalanceAsset(ctx context.Context, chainID string, asset string, tokenAddress string) (balanceAsset, error) {
	asset = strings.ToUpper(strings.TrimSpace(asset))
	tokenAddress = strings.TrimSpace(tokenAddress)

	if tokenAddress != "" {
		if !common.IsHexAddress(tokenAddress) {
			return balanceAsset{}, ErrInvalidTokenAddress
		}

		token := common.HexToAddress(tokenAddress)
		symbol, err := c.callString(ctx, token, "symbol")
		if err != nil {
			return balanceAsset{}, fmt.Errorf("call symbol: %w", err)
		}
		decimals, err := c.callUint8(ctx, token, "decimals")
		if err != nil {
			return balanceAsset{}, fmt.Errorf("call decimals: %w", err)
		}

		return balanceAsset{
			Asset:        symbol,
			Type:         "token",
			Symbol:       symbol,
			Decimals:     decimals,
			TokenAddress: token.Hex(),
		}, nil
	}

	if asset == "" || asset == "NATIVE" || asset == "ETH" || asset == "BNB" || asset == "BSC" {
		return balanceAsset{
			Asset:    nativeAssetName(chainID, asset),
			Type:     "native",
			Symbol:   nativeAssetSymbol(chainID, asset),
			Decimals: 18,
		}, nil
	}

	contracts, ok := stablecoinContracts[chainID]
	if !ok {
		return balanceAsset{}, ErrUnsupportedAsset
	}

	address, ok := contracts[asset]
	if !ok {
		return balanceAsset{}, ErrUnsupportedAsset
	}

	token := common.HexToAddress(address)
	decimals, err := c.callUint8(ctx, token, "decimals")
	if err != nil {
		return balanceAsset{}, fmt.Errorf("call decimals: %w", err)
	}
	symbol, err := c.callString(ctx, token, "symbol")
	if err != nil {
		return balanceAsset{}, fmt.Errorf("call symbol: %w", err)
	}

	return balanceAsset{
		Asset:        asset,
		Type:         "token",
		Symbol:       symbol,
		Decimals:     decimals,
		TokenAddress: token.Hex(),
	}, nil
}

func (c *Client) balanceForAsset(ctx context.Context, owner common.Address, asset balanceAsset) (*big.Int, error) {
	if asset.Type == "native" {
		return c.BalanceAt(ctx, owner, nil)
	}

	return c.callUint256(ctx, common.HexToAddress(asset.TokenAddress), "balanceOf", owner)
}

func (c *Client) callUint256(ctx context.Context, contract common.Address, method string, args ...any) (*big.Int, error) {
	values, err := c.callContractMethod(ctx, contract, method, args...)
	if err != nil {
		return nil, err
	}

	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected %s return type", method)
	}

	return value, nil
}

func (c *Client) callUint8(ctx context.Context, contract common.Address, method string, args ...any) (uint8, error) {
	values, err := c.callContractMethod(ctx, contract, method, args...)
	if err != nil {
		return 0, err
	}

	value, ok := values[0].(uint8)
	if !ok {
		return 0, fmt.Errorf("unexpected %s return type", method)
	}

	return value, nil
}

func (c *Client) callString(ctx context.Context, contract common.Address, method string, args ...any) (string, error) {
	values, err := c.callContractMethod(ctx, contract, method, args...)
	if err != nil {
		return "", err
	}

	value, ok := values[0].(string)
	if !ok {
		return "", fmt.Errorf("unexpected %s return type", method)
	}

	return value, nil
}

func (c *Client) callContractMethod(ctx context.Context, contract common.Address, method string, args ...any) ([]any, error) {
	data, err := erc20ABI.Pack(method, args...)
	if err != nil {
		return nil, err
	}

	output, err := c.CallContract(ctx, ethereum.CallMsg{
		To:   &contract,
		Data: data,
	}, nil)
	if err != nil {
		return nil, err
	}

	return erc20ABI.Unpack(method, output)
}

func formatTokenAmount(raw *big.Int, decimals uint8) string {
	if raw == nil {
		return "0"
	}
	if decimals == 0 {
		return raw.String()
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat := new(big.Rat).SetFrac(raw, scale)
	formatted := rat.FloatString(int(decimals))

	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func mustParseERC20ABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		panic(err)
	}

	return parsed
}

func nativeAssetName(chainID string, requested string) string {
	symbol := nativeAssetSymbol(chainID, requested)
	if symbol == "BNB" {
		return "BSC"
	}

	return symbol
}

func nativeAssetSymbol(chainID string, requested string) string {
	switch chainID {
	case "56":
		return "BNB"
	default:
		if requested != "" && requested != "NATIVE" && requested != "BSC" {
			return requested
		}
		return "ETH"
	}
}
