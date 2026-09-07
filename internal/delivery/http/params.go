package http

import (
	"strconv"

	"github.com/labstack/echo/v5"
)

// IntParam parses a query parameter as int, returning fallback when missing or invalid.
func IntParam(ctx *echo.Context, key string, fallback int) int {
	value, err := strconv.Atoi(ctx.QueryParam(key))
	if err != nil {
		return fallback
	}
	return value
}
