package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"web3-backend/internal/tokens"
)

func tokensListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		response, err := deps.Tokens.List(
			c.Request.Context(),
			tokens.ParseChainIDs(c.Query("chainIds")),
			c.Query("address"),
		)
		if err != nil {
			writeError(c, http.StatusBadGateway, "failed to fetch token market data")
			return
		}

		writeSuccess(c, response)
	}
}
