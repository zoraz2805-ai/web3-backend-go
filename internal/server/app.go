package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"web3-backend/internal/alchemy"
	"web3-backend/internal/appapi"
)

type appBroadcastRequest struct {
	ChainCode     string          `json:"chain_code"`
	RawTx         string          `json:"raw_tx"`
	Encoding      string          `json:"encoding"`
	MaxFeeRate    string          `json:"max_fee_rate"`
	SkipPreflight *bool           `json:"skip_preflight"`
	TxObject      json.RawMessage `json:"tx_object"`
}

func appAvailableChainsHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeSuccess(c, gin.H{
			"list": deps.AppCatalog.Chains(parseBoolQuery(c, "include_disabled")),
		})
	}
}

func appAvailableCoinsHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeSuccess(c, gin.H{
			"list": deps.AppCatalog.Coins(c.Query("chain"), parseBoolQuery(c, "include_disabled")),
		})
	}
}

func appBindHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request appapi.BindRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid bind request")
			return
		}
		if strings.TrimSpace(request.Address) == "" {
			writeError(c, http.StatusBadRequest, "addr is required")
			return
		}

		writeSuccess(c, appapi.BindResponse{
			UserID: appapi.UserIDForWallet(request.Address),
		})
	}
}

func appAddressBookListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		walletAddress, ok := requireWalletAddress(c)
		if !ok {
			return
		}

		writeSuccess(c, gin.H{
			"list": deps.AddressBook.List(walletAddress, c.Query("chain")),
		})
	}
}

func appAddressBookCreateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		walletAddress, ok := requireWalletAddress(c)
		if !ok {
			return
		}

		var request appapi.AddressBookCreateRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid address book request")
			return
		}

		item, err := deps.AddressBook.Create(walletAddress, request)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		writeSuccess(c, item)
	}
}

func appAddressBookUpdateHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		walletAddress, ok := requireWalletAddress(c)
		if !ok {
			return
		}

		var request appapi.AddressBookUpdateRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid address book request")
			return
		}

		item, err := deps.AddressBook.Update(walletAddress, request)
		if err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		writeSuccess(c, item)
	}
}

func appAddressBookDeleteHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		walletAddress, ok := requireWalletAddress(c)
		if !ok {
			return
		}

		var request appapi.AddressBookDeleteRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid address book request")
			return
		}

		if err := deps.AddressBook.Delete(walletAddress, request.ID); err != nil {
			writeError(c, http.StatusBadRequest, err.Error())
			return
		}

		writeSuccess(c, gin.H{})
	}
}

func appBroadcastHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		var request appBroadcastRequest
		if err := c.ShouldBindJSON(&request); err != nil {
			writeError(c, http.StatusBadRequest, "invalid broadcast request")
			return
		}
		if strings.TrimSpace(request.ChainCode) == "" {
			writeError(c, http.StatusBadRequest, "chain_code is required")
			return
		}

		response, err := deps.Alchemy.Broadcast(c.Request.Context(), alchemy.BroadcastRequest{
			ChainCode:     request.ChainCode,
			RawTx:         request.RawTx,
			Encoding:      request.Encoding,
			MaxFeeRate:    request.MaxFeeRate,
			SkipPreflight: request.SkipPreflight,
			TxObject:      request.TxObject,
		})
		if err != nil {
			if errors.Is(err, alchemy.ErrInvalidRequest) || errors.Is(err, alchemy.ErrUnsupportedChain) {
				writeError(c, http.StatusBadRequest, err.Error())
				return
			}
			writeError(c, http.StatusBadGateway, err.Error())
			return
		}

		writeSuccess(c, response)
	}
}

func parseBoolQuery(c *gin.Context, key string) bool {
	value := strings.TrimSpace(c.Query(key))
	if value == "" {
		return false
	}

	parsed, err := strconv.ParseBool(value)
	return err == nil && parsed
}

func requireWalletAddress(c *gin.Context) (string, bool) {
	walletAddress := strings.TrimSpace(c.GetHeader("X-Wallet-Address"))
	if walletAddress == "" {
		writeError(c, http.StatusBadRequest, "X-Wallet-Address is required")
		return "", false
	}

	return walletAddress, true
}
