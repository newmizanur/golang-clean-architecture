package response

import (
	"errors"
	"net/http"

	"golang-clean-architecture/internal/apperror"

	"github.com/labstack/echo/v5"
)

type FailedResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type ErrorBuilder struct {
	err error
}

func NewErrorBuilder(err error) *ErrorBuilder {
	return &ErrorBuilder{err: err}
}

func (b *ErrorBuilder) Send(ctx *echo.Context) error {
	status, payload := buildErrorResponse(b.err)
	return ctx.JSON(status, payload)
}

func buildErrorResponse(err error) (int, FailedResponse) {
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		return appErr.Status, FailedResponse{
			Code:    appErr.Code,
			Message: appErr.Message,
		}
	}

	return http.StatusInternalServerError, FailedResponse{
		Code:    http.StatusInternalServerError,
		Message: http.StatusText(http.StatusInternalServerError),
	}
}
