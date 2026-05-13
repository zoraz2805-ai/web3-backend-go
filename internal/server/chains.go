package server

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func chainsListHandler(deps Dependencies) gin.HandlerFunc {
	return func(c *gin.Context) {
		if deps.Chains == nil {
			writeError(c, http.StatusServiceUnavailable, "chains repository is disabled")
			return
		}

		page := parsePositiveIntQuery(c, "page", 1)
		size := parsePositiveIntQuery(c, "size", 30)
		key := c.Query("key")

		list, total, err := deps.Chains.List(c.Request.Context(), page, size, key)
		if err != nil {
			writeError(c, http.StatusInternalServerError, "failed to list chains")
			return
		}

		writePageSuccess(c, page, size, &total, list)
	}
}

func parsePositiveIntQuery(c *gin.Context, key string, fallback int) int {
	value := c.Query(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}

	return parsed
}
