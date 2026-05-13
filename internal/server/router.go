package server

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"web3-backend/internal/alchemy"
	"web3-backend/internal/appapi"
	"web3-backend/internal/chains"
	"web3-backend/internal/config"
	"web3-backend/internal/evm"
	"web3-backend/internal/tokens"
)

type Dependencies struct {
	Config config.Config
	DB     *pgxpool.Pool
	Redis  *redis.Client
	EVM    *evm.Client
	Chains *chains.Repository
	Tokens *tokens.Service

	AppCatalog  *appapi.Catalog
	AddressBook *appapi.AddressBookStore
	Alchemy     *alchemy.Client
}

func NewRouter(deps Dependencies) *gin.Engine {
	if deps.Config.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(requestLogger())

	router.GET("/healthz", healthHandler(deps))

	app := router.Group("/app")
	app.GET("/available_chains", appAvailableChainsHandler(deps))
	app.GET("/available_coins", appAvailableCoinsHandler(deps))
	app.POST("/bind", appBindHandler(deps))
	app.GET("/address_book", appAddressBookListHandler(deps))
	app.POST("/address_book", appAddressBookCreateHandler(deps))
	app.PUT("/address_book", appAddressBookUpdateHandler(deps))
	app.DELETE("/address_book", appAddressBookDeleteHandler(deps))
	app.POST("/quicknode/broadcast", appBroadcastHandler(deps))

	api := router.Group("/api/v1")
	api.GET("/status", statusHandler(deps))
	api.GET("/chains", chainsListHandler(deps))
	api.GET("/tokens", tokensListHandler(deps))

	evmAPI := api.Group("/evm")
	evmAPI.GET("/network", evmNetworkHandler(deps))
	evmAPI.GET("/balances", evmBatchBalancesHandler(deps))
	evmAPI.GET("/balances/:address/native", evmNativeBalanceHandler(deps))
	evmAPI.GET("/balances/:address/tokens/:tokenAddress", evmTokenBalanceHandler(deps))

	return router
}

func ethClient(client *evm.Client) *ethclient.Client {
	if client == nil {
		return nil
	}

	return client.Client
}
