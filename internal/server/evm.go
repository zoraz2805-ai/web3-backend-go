package server

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"web3-backend/internal/evm"
)

const maxBatchBalanceAddresses = 50

func evmNetworkHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.EVM == nil {
			writeError(c, http.StatusServiceUnavailable, "evm rpc is not configured")
			return
		}

		status, err := deps.EVM.NetworkStatus(c.Request.Context())
		if err != nil {
			writeEVMError(c, err)
			return
		}

		writeSuccess(c, status)
	}
}

func evmNativeBalanceHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.EVM == nil {
			writeError(c, http.StatusServiceUnavailable, "evm rpc is not configured")
			return
		}

		balance, err := deps.EVM.NativeBalance(c.Request.Context(), c.Param("address"))
		if err != nil {
			writeEVMError(c, err)
			return
		}

		writeSuccess(c, balance)
	}
}

func evmTokenBalanceHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.EVM == nil {
			writeError(c, http.StatusServiceUnavailable, "evm rpc is not configured")
			return
		}

		balance, err := deps.EVM.TokenBalance(
			c.Request.Context(),
			c.Param("address"),
			c.Param("tokenAddress"),
		)
		if err != nil {
			writeEVMError(c, err)
			return
		}

		writeSuccess(c, balance)
	}
}

func evmBatchBalancesHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.EVM == nil {
			writeError(c, http.StatusServiceUnavailable, "evm rpc is not configured")
			return
		}

		addresses := parseAddressCSV(c.Query("addresses"))
		if len(addresses) == 0 {
			writeError(c, http.StatusBadRequest, "addresses is required")
			return
		}
		if len(addresses) > maxBatchBalanceAddresses {
			writeError(c, http.StatusBadRequest, "too many addresses")
			return
		}

		balances, err := deps.EVM.BatchBalances(c.Request.Context(), evm.BatchBalanceRequest{
			Addresses:    addresses,
			Asset:        c.DefaultQuery("asset", "native"),
			TokenAddress: c.Query("tokenAddress"),
		})
		if err != nil {
			writeEVMError(c, err)
			return
		}

		writeSuccess(c, balances)
	}
}

func writeEVMError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, evm.ErrClientDisabled):
		writeError(c, http.StatusServiceUnavailable, "evm rpc is not configured")
	case errors.Is(err, evm.ErrInvalidAddress):
		writeError(c, http.StatusBadRequest, "invalid evm address")
	case errors.Is(err, evm.ErrInvalidTokenAddress):
		writeError(c, http.StatusBadRequest, "invalid evm token address")
	case errors.Is(err, evm.ErrUnsupportedAsset):
		writeError(c, http.StatusBadRequest, "unsupported evm balance asset")
	default:
		writeError(c, http.StatusBadGateway, "evm rpc request failed")
	}
}

func parseAddressCSV(value string) []string {
	parts := strings.Split(value, ",")
	addresses := make([]string, 0, len(parts))
	for _, part := range parts {
		address := strings.TrimSpace(part)
		if address == "" {
			continue
		}
		addresses = append(addresses, address)
	}

	return addresses
}
