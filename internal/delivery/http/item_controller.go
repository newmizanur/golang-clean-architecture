package http

import (
	"context"
	"golang-clean-architecture/internal/apperror"
	httpresponse "golang-clean-architecture/internal/delivery/http/response"
	"golang-clean-architecture/internal/dto"
	"math"
	"strconv"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

// ItemUseCasePort is the subset of ItemUseCase methods used by ItemController.
type ItemUseCasePort interface {
	Create(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error)
	Search(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error)
	Get(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error)
	Update(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error)
	Delete(ctx context.Context, req *dto.DeleteItemRequest) error
}

type ItemController struct {
	ItemUseCase ItemUseCasePort
	Log         *logrus.Logger
}

func NewItemController(useCase ItemUseCasePort, log *logrus.Logger) *ItemController {
	return &ItemController{
		ItemUseCase: useCase,
		Log:         log,
	}
}

func (c *ItemController) Create(ctx *echo.Context) error {
	request := new(dto.CreateItemRequest)
	if err := ctx.Bind(&request); err != nil {
		c.Log.WithError(err).Error("error parsing request body")
		return httpresponse.NewErrorBuilder(apperror.ItemErrors.InvalidRequest).Send(ctx)
	}

	response, err := c.ItemUseCase.Create(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error on item create")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ItemController) List(ctx *echo.Context) error {
	request := &dto.SearchItemRequest{
		Name: ctx.QueryParam("name"),
		SKU:  ctx.QueryParam("sku"),
		Sort: ctx.QueryParam("sort"),
		Page: IntParam(ctx, "page", 1),
		Size: IntParam(ctx, "size", 10),
	}

	response, total, err := c.ItemUseCase.Search(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error on search")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	paging := &dto.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return httpresponse.SuccessBuilder(response).WithPaging(paging).Send(ctx)
}

func (c *ItemController) Get(ctx *echo.Context) error {
	itemID, err := itemIDParam(ctx)
	if err != nil {
		return httpresponse.NewErrorBuilder(apperror.ItemErrors.InvalidRequest).Send(ctx)
	}

	request := &dto.GetItemRequest{ID: itemID}
	response, err := c.ItemUseCase.Get(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error on get item")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ItemController) Update(ctx *echo.Context) error {
	itemID, err := itemIDParam(ctx)
	if err != nil {
		return httpresponse.NewErrorBuilder(apperror.ItemErrors.InvalidRequest).Send(ctx)
	}

	request := new(dto.UpdateItemRequest)
	if err := ctx.Bind(&request); err != nil {
		c.Log.WithError(err).Error("error parsing request body")
		return httpresponse.NewErrorBuilder(apperror.ItemErrors.InvalidRequest).Send(ctx)
	}
	request.ID = itemID

	response, err := c.ItemUseCase.Update(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error on update item")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ItemController) Delete(ctx *echo.Context) error {
	itemID, err := itemIDParam(ctx)
	if err != nil {
		return httpresponse.NewErrorBuilder(apperror.ItemErrors.InvalidRequest).Send(ctx)
	}

	request := &dto.DeleteItemRequest{ID: itemID}
	if err := c.ItemUseCase.Delete(ctx.Request().Context(), request); err != nil {
		c.Log.WithError(err).Error("error on delete item")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(true).Send(ctx)
}

func itemIDParam(ctx *echo.Context) (int64, error) {
	return strconv.ParseInt(ctx.Param("itemId"), 10, 64)
}
