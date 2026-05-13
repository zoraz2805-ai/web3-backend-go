package appapi

import "strings"

type AvailableChain struct {
	ID         int    `json:"id"`
	ChainName  string `json:"chain_name"`
	IsDisabled bool   `json:"is_is_disabled"`
	IconURL    string `json:"icon_url"`
}

type AvailableCoin struct {
	ID           int    `json:"id"`
	Chain        string `json:"chain"`
	CoinName     string `json:"coin_name"`
	ContractAddr string `json:"contract_addr"`
	Decimal      int    `json:"decimal"`
	IconURL      string `json:"icon_url"`
	IsDisabled   bool   `json:"is_disabled"`
}

type Catalog struct {
	chains []AvailableChain
	coins  []AvailableCoin
}

func NewCatalog() *Catalog {
	return &Catalog{
		chains: []AvailableChain{
			{
				ID:        1,
				ChainName: "BSC",
				IconURL:   "https://assets.coingecko.com/coins/images/825/small/binance-coin-logo.png?1547034615",
			},
			{
				ID:        2,
				ChainName: "ETH",
				IconURL:   "https://assets.coingecko.com/coins/images/279/small/ethereum.png?1595348880",
			},
			{
				ID:        3,
				ChainName: "SOLANA",
				IconURL:   "https://assets.coingecko.com/coins/images/4128/small/solana.png?1718769756",
			},
		},
		coins: []AvailableCoin{
			{
				ID:       1,
				Chain:    "BSC",
				CoinName: "BNB",
				Decimal:  18,
				IconURL:  "https://assets.coingecko.com/coins/images/825/small/binance-coin-logo.png?1547034615",
			},
			{
				ID:           2,
				Chain:        "BSC",
				CoinName:     "USDT",
				ContractAddr: "0x55d398326f99059fF775485246999027B3197955",
				Decimal:      18,
				IconURL:      "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/smartchain/assets/0x55d398326f99059fF775485246999027B3197955/logo.png",
			},
			{
				ID:       3,
				Chain:    "ETH",
				CoinName: "ETH",
				Decimal:  18,
				IconURL:  "https://assets.coingecko.com/coins/images/279/small/ethereum.png?1595348880",
			},
			{
				ID:           4,
				Chain:        "ETH",
				CoinName:     "USDT",
				ContractAddr: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
				Decimal:      6,
				IconURL:      "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xdAC17F958D2ee523a2206206994597C13D831ec7/logo.png",
			},
			{
				ID:       5,
				Chain:    "SOLANA",
				CoinName: "SOL",
				Decimal:  9,
				IconURL:  "https://assets.coingecko.com/coins/images/4128/small/solana.png?1718769756",
			},
		},
	}
}

func (c *Catalog) Chains(includeDisabled bool) []AvailableChain {
	list := make([]AvailableChain, 0, len(c.chains))
	for _, chain := range c.chains {
		if chain.IsDisabled && !includeDisabled {
			continue
		}
		list = append(list, chain)
	}

	return list
}

func (c *Catalog) Coins(chain string, includeDisabled bool) []AvailableCoin {
	chain = normalizeChain(chain)
	list := make([]AvailableCoin, 0, len(c.coins))
	for _, coin := range c.coins {
		if coin.IsDisabled && !includeDisabled {
			continue
		}
		if chain != "" && normalizeChain(coin.Chain) != chain {
			continue
		}
		list = append(list, coin)
	}

	return list
}

func normalizeChain(chain string) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "":
		return ""
	case "bsc", "bnb", "bnb smart chain", "binance smart chain", "smartchain", "bnb-mainnet":
		return "bsc"
	case "eth", "ethereum", "ethereum mainnet", "eth-mainnet":
		return "eth"
	case "sol", "solana", "solana mainnet", "solana-mainnet":
		return "solana"
	default:
		return strings.ToLower(strings.TrimSpace(chain))
	}
}
