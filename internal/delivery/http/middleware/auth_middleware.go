package middleware

import (
	"strings"

	"golang-clean-architecture/internal/apperror"
	"golang-clean-architecture/internal/auth"
	"golang-clean-architecture/internal/delivery/http/response"
	"golang-clean-architecture/internal/dto"

	"github.com/labstack/echo/v5"
)

func NewAuth(jwtSecret string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx *echo.Context) error {
			header := ctx.Request().Header.Get("Authorization")
			if header == "" {
				return response.NewErrorBuilder(apperror.AuthErrors.MissingToken).Send(ctx)
			}

			parts := strings.SplitN(header, " ", 2)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				return response.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
			}

			userID, err := auth.ParseToken(parts[1], jwtSecret)
			if err != nil {
				return response.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
			}

			ctx.Set("auth", &dto.Auth{ID: userID})
			return next(ctx)
		}
	}
}

// GetUser returns the authenticated user from context.
// The second return is false if the auth middleware did not set the user (e.g. route not protected).
func GetUser(ctx *echo.Context) (*dto.Auth, bool) {
	v := ctx.Get("auth")
	if v == nil {
		return nil, false
	}
	auth, ok := v.(*dto.Auth)
	return auth, ok
}
