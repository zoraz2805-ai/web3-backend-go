package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type healthResponse struct {
	Status   string            `json:"status"`
	Env      string            `json:"env"`
	Services map[string]string `json:"services"`
}

func healthHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		services := map[string]string{
			"postgres": "ok",
			"redis":    "ok",
			"evm":      "disabled",
		}

		ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
		defer cancel()

		if deps.DB == nil {
			services["postgres"] = "disabled"
		} else if err := deps.DB.Ping(ctx); err != nil {
			services["postgres"] = "error"
		}

		if deps.Redis == nil {
			services["redis"] = "disabled"
		} else if err := deps.Redis.Ping(ctx).Err(); err != nil {
			services["redis"] = "error"
		}

		if client := ethClient(deps.EVM); client != nil {
			if _, err := client.ChainID(ctx); err != nil {
				services["evm"] = "error"
			} else {
				services["evm"] = "ok"
			}
		}

		status := "ok"
		if services["postgres"] == "error" || services["redis"] == "error" {
			status = "error"
		}

		response := healthResponse{
			Status:   status,
			Env:      deps.Config.AppEnv,
			Services: services,
		}

		if status != "ok" {
			c.JSON(http.StatusServiceUnavailable, responseWrapper{
				Code: http.StatusServiceUnavailable,
				Msg:  "service unavailable",
				Data: response,
			})
			return
		}

		writeSuccess(c, response)
	}
}

func statusHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		writeSuccess(c, gin.H{
			"service": "web3-backend",
			"env":     deps.Config.AppEnv,
		})
	}
}
