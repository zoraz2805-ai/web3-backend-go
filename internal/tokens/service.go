package tokens

import (
	"context"
	"fmt"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"

	"web3-backend/internal/market"
)

type MarketClient interface {
	SimplePrice(ctx context.Context, ids []string) (map[string]market.TokenMarketData, error)
}

type Service struct {
	market MarketClient
}

type Token struct {
	Address                     string   `json:"address"`
	ChainID                     int64    `json:"chainId"`
	Symbol                      string   `json:"symbol"`
	Decimals                    int      `json:"decimals"`
	Name                        string   `json:"name"`
	CoinKey                     string   `json:"coinKey"`
	LogoURI                     string   `json:"logoURI"`
	PriceUSD                    string   `json:"priceUSD"`
	MarketCapUSD                float64  `json:"marketCapUSD"`
	VolumeUSD24H                float64  `json:"volumeUSD24H"`
	Tags                        []string `json:"tags"`
	VerificationStatus          string   `json:"verificationStatus"`
	VerificationStatusBreakdown []string `json:"verificationStatusBreakdown"`
	Balance                     string   `json:"balance"`
	EqualUsdtBalance            string   `json:"equalUsdtBalance"`

	marketID string
}

type Response struct {
	Tokens map[string][]Token `json:"tokens"`
}

var baseTokens = map[string][]Token{
	"1": {
		{
			Address:                     "0x0000000000000000000000000000000000000000",
			ChainID:                     1,
			Symbol:                      "ETH",
			Decimals:                    18,
			Name:                        "ETH",
			CoinKey:                     "ETH",
			LogoURI:                     "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png",
			Tags:                        []string{"major_asset"},
			VerificationStatus:          "unverified",
			VerificationStatusBreakdown: []string{},
			marketID:                    "ethereum",
		},
		{
			Address:                     "0xdAC17F958D2ee523a2206206994597C13D831ec7",
			ChainID:                     1,
			Symbol:                      "USDT",
			Decimals:                    6,
			Name:                        "USDT",
			CoinKey:                     "USDT",
			LogoURI:                     "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xdAC17F958D2ee523a2206206994597C13D831ec7/logo.png",
			Tags:                        []string{"stablecoin"},
			VerificationStatus:          "unverified",
			VerificationStatusBreakdown: []string{},
			marketID:                    "tether",
		},
	},
	"56": {
		{
			Address:                     "0x0000000000000000000000000000000000000000",
			ChainID:                     56,
			Symbol:                      "BNB",
			Decimals:                    18,
			Name:                        "BNB",
			CoinKey:                     "BNB",
			LogoURI:                     "https://assets.coingecko.com/coins/images/825/small/binance-coin-logo.png?1547034615",
			Tags:                        []string{"major_asset"},
			VerificationStatus:          "unverified",
			VerificationStatusBreakdown: []string{},
			marketID:                    "binancecoin",
		},
		{
			Address:                     "0x55d398326f99059fF775485246999027B3197955",
			ChainID:                     56,
			Symbol:                      "USDT",
			Decimals:                    18,
			Name:                        "USDT",
			CoinKey:                     "USDT",
			LogoURI:                     "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xdAC17F958D2ee523a2206206994597C13D831ec7/logo.png",
			Tags:                        []string{"stablecoin"},
			VerificationStatus:          "unverified",
			VerificationStatusBreakdown: []string{},
			marketID:                    "tether",
		},
	},
}

const erc20ABIJSON = `[
	{"constant":true,"inputs":[{"name":"account","type":"address"}],"name":"balanceOf","outputs":[{"name":"","type":"uint256"}],"type":"function"}
]`

var (
	erc20ABI  = mustParseERC20ABI()
	chainRPCs = map[string]string{
		"1":  "https://ethereum-rpc.publicnode.com",
		"56": "https://bsc-dataseed2.binance.org",
	}
)

func NewService(market MarketClient) *Service {
	return &Service{market: market}
}

func (s *Service) List(ctx context.Context, chainIDs []string, walletAddress string) (Response, error) {
	result := cloneTokens(chainIDs)
	marketIDs := collectMarketIDs(result)

	marketData, err := s.market.SimplePrice(ctx, marketIDs)
	if err != nil {
		return Response{}, err
	}

	for chainID, list := range result {
		for index := range list {
			data, ok := marketData[list[index].marketID]
			if !ok {
				continue
			}
			list[index].PriceUSD = data.PriceUSD
			list[index].MarketCapUSD = data.MarketCapUSD
			list[index].VolumeUSD24H = data.VolumeUSD24H
		}
		result[chainID] = list
	}

	if strings.TrimSpace(walletAddress) != "" {
		if !common.IsHexAddress(walletAddress) {
			return Response{}, fmt.Errorf("invalid wallet address")
		}
		if err := fillBalances(ctx, result, common.HexToAddress(walletAddress)); err != nil {
			return Response{}, err
		}
	}

	return Response{Tokens: result}, nil
}

func ParseChainIDs(value string) []string {
	parts := strings.Split(value, ",")
	chainIDs := make([]string, 0, len(parts))
	for _, part := range parts {
		chainID := strings.TrimSpace(part)
		if chainID == "" {
			continue
		}
		if _, err := strconv.ParseInt(chainID, 10, 64); err != nil {
			continue
		}
		chainIDs = append(chainIDs, chainID)
	}

	return chainIDs
}

func cloneTokens(chainIDs []string) map[string][]Token {
	selected := chainIDs
	if len(selected) == 0 {
		selected = []string{"1", "56"}
	}

	result := make(map[string][]Token, len(selected))
	for _, chainID := range selected {
		list, ok := baseTokens[chainID]
		if !ok {
			continue
		}
		result[chainID] = slices.Clone(list)
	}

	return result
}

func collectMarketIDs(tokens map[string][]Token) []string {
	ids := make([]string, 0)
	for _, list := range tokens {
		for _, token := range list {
			ids = append(ids, token.marketID)
		}
	}

	return ids
}

func fillBalances(ctx context.Context, tokensByChain map[string][]Token, wallet common.Address) error {
	for chainID, list := range tokensByChain {
		rpc, ok := chainRPCs[chainID]
		if !ok {
			continue
		}

		client, err := ethclient.DialContext(ctx, rpc)
		if err != nil {
			return err
		}

		for index := range list {
			rawBalance, err := queryTokenRawBalance(ctx, client, wallet, list[index].Address)
			if err != nil {
				client.Close()
				return err
			}

			balance := formatTokenAmount(rawBalance, list[index].Decimals)
			list[index].Balance = balance
			list[index].EqualUsdtBalance = calculateEqualUSDT(balance, list[index].PriceUSD)
		}

		client.Close()
		tokensByChain[chainID] = list
	}

	return nil
}

func queryTokenRawBalance(ctx context.Context, client *ethclient.Client, wallet common.Address, tokenAddress string) (*big.Int, error) {
	if tokenAddress == "0x0000000000000000000000000000000000000000" {
		return client.BalanceAt(ctx, wallet, nil)
	}

	contract := common.HexToAddress(tokenAddress)
	data, err := erc20ABI.Pack("balanceOf", wallet)
	if err != nil {
		return nil, err
	}

	output, err := client.CallContract(ctx, ethereum.CallMsg{
		To:   &contract,
		Data: data,
	}, nil)
	if err != nil {
		return nil, err
	}

	values, err := erc20ABI.Unpack("balanceOf", output)
	if err != nil {
		return nil, err
	}

	value, ok := values[0].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("unexpected balance type")
	}

	return value, nil
}

func formatTokenAmount(raw *big.Int, decimals int) string {
	if raw == nil {
		return "0"
	}
	if decimals <= 0 {
		return raw.String()
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	rat := new(big.Rat).SetFrac(raw, scale)
	formatted := rat.FloatString(decimals)

	return strings.TrimRight(strings.TrimRight(formatted, "0"), ".")
}

func calculateEqualUSDT(balance string, priceUSD string) string {
	if strings.TrimSpace(balance) == "" || strings.TrimSpace(priceUSD) == "" {
		return "0"
	}

	balanceValue := new(big.Float).SetPrec(256)
	if _, ok := balanceValue.SetString(balance); !ok {
		return "0"
	}
	priceValue := new(big.Float).SetPrec(256)
	if _, ok := priceValue.SetString(priceUSD); !ok {
		return "0"
	}

	value := new(big.Float).Mul(balanceValue, priceValue)
	return strings.TrimRight(strings.TrimRight(value.Text('f', 8), "0"), ".")
}

func mustParseERC20ABI() abi.ABI {
	parsed, err := abi.JSON(strings.NewReader(erc20ABIJSON))
	if err != nil {
		panic(err)
	}

	return parsed
}
