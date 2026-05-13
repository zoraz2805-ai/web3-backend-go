package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"web3-backend/internal/alchemy"
	"web3-backend/internal/appapi"
	"web3-backend/internal/chains"
	"web3-backend/internal/config"
	"web3-backend/internal/database"
	"web3-backend/internal/evm"
	"web3-backend/internal/market"
	"web3-backend/internal/server"
	"web3-backend/internal/tokens"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx := context.Background()

	db, err := database.NewPostgres(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Printf("postgres disabled: %v", err)
	}
	if db != nil {
		defer db.Close()
	}

	cache, err := database.NewRedis(ctx, cfg.RedisURL)
	if err != nil {
		log.Printf("redis disabled: %v", err)
	}
	if cache != nil {
		defer func() {
			if err := cache.Close(); err != nil {
				log.Printf("close redis: %v", err)
			}
		}()
	}

	evmClient, err := evm.NewClient(cfg.EVMRPCURL)
	if err != nil {
		log.Printf("evm client disabled: %v", err)
	}
	if evmClient != nil {
		defer evmClient.Close()
	}

	var chainsRepo *chains.Repository
	if db != nil {
		chainsRepo = chains.NewRepository(db)
		if imported, err := chainsRepo.ImportFromFile(ctx, cfg.ChainsPath); err != nil {
			log.Printf("import chains: %v", err)
		} else if imported > 0 {
			log.Printf("imported chains: %d", imported)
		}
	}

	marketClient := market.NewCoinGeckoClient(cfg.MarketURL, cfg.MarketAPIKey, cfg.RequestTimeout)
	tokensService := tokens.NewService(marketClient)
	alchemyClient := alchemy.NewClient(cfg.AlchemyAPIKey, cfg.RequestTimeout)

	router := server.NewRouter(server.Dependencies{
		Config:      cfg,
		DB:          db,
		Redis:       cache,
		EVM:         evmClient,
		Chains:      chainsRepo,
		Tokens:      tokensService,
		AppCatalog:  appapi.NewCatalog(),
		AddressBook: appapi.NewAddressBookStore(),
		Alchemy:     alchemyClient,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("api listening on :%s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown server: %v", err)
	}
}
