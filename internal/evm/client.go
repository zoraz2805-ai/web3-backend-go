package evm

import (
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/ethclient"
)

type Client struct {
	*ethclient.Client
}

func NewClient(rpcURL string) (*Client, error) {
	if strings.TrimSpace(rpcURL) == "" {
		return nil, errors.New("EVM_RPC_URL is not configured")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, err
	}

	return &Client{Client: client}, nil
}
