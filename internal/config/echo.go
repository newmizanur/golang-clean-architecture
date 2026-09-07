package config

import (
	"net/http"

	"golang-clean-architecture/internal/apperror"
	"golang-clean-architecture/internal/delivery/http/response"

	"github.com/labstack/echo/v5"
	"github.com/spf13/viper"
)

func NewEcho(config *viper.Viper) *echo.Echo {
	_ = config
	app := echo.New()
	app.HTTPErrorHandler = NewErrorHandler()

	return app
}

func NewErrorHandler() echo.HTTPErrorHandler {
	return func(ctx *echo.Context, err error) {
		if resp, ok := ctx.Response().(*echo.Response); ok && resp.Committed {
			return
		}

		if appErr, ok := err.(*apperror.AppError); ok {
			if sendErr := response.NewErrorBuilder(appErr).Send(ctx); sendErr != nil {
				ctx.Logger().Error(sendErr.Error())
			}
			return
		}

		if httpErr, ok := err.(*echo.HTTPError); ok {
			message := httpErr.Message
			if message == "" {
				message = "Internal Server Error"
			}

			fallback := apperror.NewAppError(httpErr.Code, httpErr.Code, message)
			if sendErr := response.NewErrorBuilder(fallback).Send(ctx); sendErr != nil {
				ctx.Logger().Error(sendErr.Error())
			}
			return
		}

		fallback := apperror.NewAppError(http.StatusInternalServerError, http.StatusInternalServerError, "internal server error")
		if sendErr := response.NewErrorBuilder(fallback).Send(ctx); sendErr != nil {
			ctx.Logger().Error(sendErr.Error())
		}
	}
}
