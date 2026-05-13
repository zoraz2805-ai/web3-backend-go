package evm

import (
	"context"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

var (
	ErrClientDisabled      = errors.New("evm client is disabled")
	ErrInvalidAddress      = errors.New("invalid evm address")
	ErrInvalidTokenAddress = errors.New("invalid evm token address")
	ErrUnsupportedAsset    = errors.New("unsupported evm balance asset")
)

type NetworkStatus struct {
	ChainID     string `json:"chainId"`
	BlockNumber uint64 `json:"blockNumber"`
}

type NativeBalance struct {
	Address string `json:"address"`
	Wei     string `json:"wei"`
	Ether   string `json:"ether"`
}

type BatchBalanceRequest struct {
	Addresses    []string
	Asset        string
	TokenAddress string
}

type BatchBalanceResponse struct {
	ChainID      string               `json:"chainId"`
	Asset        string               `json:"asset"`
	Type         string               `json:"type"`
	Symbol       string               `json:"symbol"`
	Decimals     uint8                `json:"decimals"`
	TokenAddress string               `json:"tokenAddress,omitempty"`
	List         []BatchBalanceResult `json:"list"`
}

type BatchBalanceResult struct {
	Address   string `json:"address"`
	Raw       string `json:"raw"`
	Formatted string `json:"formatted"`
}

func (c *Client) NetworkStatus(ctx context.Context) (NetworkStatus, error) {
	if c == nil || c.Client == nil {
		return NetworkStatus{}, ErrClientDisabled
	}

	chainID, err := c.ChainID(ctx)
	if err != nil {
		return NetworkStatus{}, err
	}

	blockNumber, err := c.BlockNumber(ctx)
	if err != nil {
		return NetworkStatus{}, err
	}

	return NetworkStatus{
		ChainID:     chainID.String(),
		BlockNumber: blockNumber,
	}, nil
}

func (c *Client) NativeBalance(ctx context.Context, address string) (NativeBalance, error) {
	if c == nil || c.Client == nil {
		return NativeBalance{}, ErrClientDisabled
	}
	if !common.IsHexAddress(address) {
		return NativeBalance{}, ErrInvalidAddress
	}

	account := common.HexToAddress(address)
	balance, err := c.BalanceAt(ctx, account, nil)
	if err != nil {
		return NativeBalance{}, err
	}

	return NativeBalance{
		Address: account.Hex(),
		Wei:     balance.String(),
		Ether:   weiToEther(balance),
	}, nil
}

func (c *Client) BatchBalances(ctx context.Context, req BatchBalanceRequest) (BatchBalanceResponse, error) {
	if c == nil || c.Client == nil {
		return BatchBalanceResponse{}, ErrClientDisabled
	}

	addresses := make([]common.Address, 0, len(req.Addresses))
	for _, address := range req.Addresses {
		trimmed := strings.TrimSpace(address)
		if !common.IsHexAddress(trimmed) {
			return BatchBalanceResponse{}, ErrInvalidAddress
		}
		addresses = append(addresses, common.HexToAddress(trimmed))
	}

	chainID, err := c.ChainID(ctx)
	if err != nil {
		return BatchBalanceResponse{}, err
	}

	asset, err := c.resolveBalanceAsset(ctx, chainID.String(), req.Asset, req.TokenAddress)
	if err != nil {
		return BatchBalanceResponse{}, err
	}

	response := BatchBalanceResponse{
		ChainID:      chainID.String(),
		Asset:        asset.Asset,
		Type:         asset.Type,
		Symbol:       asset.Symbol,
		Decimals:     asset.Decimals,
		TokenAddress: asset.TokenAddress,
		List:         make([]BatchBalanceResult, 0, len(addresses)),
	}

	for _, address := range addresses {
		raw, err := c.balanceForAsset(ctx, address, asset)
		if err != nil {
			return BatchBalanceResponse{}, err
		}

		response.List = append(response.List, BatchBalanceResult{
			Address:   address.Hex(),
			Raw:       raw.String(),
			Formatted: formatTokenAmount(raw, asset.Decimals),
		})
	}

	return response, nil
}

func weiToEther(wei *big.Int) string {
	if wei == nil {
		return "0"
	}

	value := new(big.Float).SetPrec(256).SetInt(wei)
	ether := new(big.Float).Quo(value, big.NewFloat(1e18))

	return ether.Text('f', 18)
}
