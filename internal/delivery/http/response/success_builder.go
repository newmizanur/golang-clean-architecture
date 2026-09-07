package response

import (
	"net/http"

	"golang-clean-architecture/internal/dto"

	"github.com/labstack/echo/v5"
)

type SuccessResponse[T any] struct {
	Data   T
	Paging *dto.PageMetadata
}

func SuccessBuilder[T any](data T) *SuccessResponse[T] {
	return &SuccessResponse[T]{Data: data}
}

func (r *SuccessResponse[T]) WithPaging(paging *dto.PageMetadata) *SuccessResponse[T] {
	r.Paging = paging
	return r
}

func (r *SuccessResponse[T]) Send(ctx *echo.Context) error {
	return ctx.JSON(http.StatusOK, dto.WebResponse[T]{Data: r.Data, Paging: r.Paging})
}
