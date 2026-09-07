package http

import (
	"golang-clean-architecture/internal/apperror"
	"golang-clean-architecture/internal/delivery/http/middleware"
	httpresponse "golang-clean-architecture/internal/delivery/http/response"
	"golang-clean-architecture/internal/dto"
	"golang-clean-architecture/internal/usecase"
	"math"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
)

type ContactController struct {
	UseCase *usecase.ContactUseCase
	Log     *logrus.Logger
}

func NewContactController(useCase *usecase.ContactUseCase, log *logrus.Logger) *ContactController {
	return &ContactController{
		UseCase: useCase,
		Log:     log,
	}
}

func (c *ContactController) Create(ctx *echo.Context) error {
	auth, ok := middleware.GetUser(ctx)
	if !ok {
		return httpresponse.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
	}

	request := new(dto.CreateContactRequest)
	if err := ctx.Bind(request); err != nil {
		c.Log.WithError(err).Error("error parsing request body")
		return httpresponse.NewErrorBuilder(apperror.ContactErrors.InvalidRequest).Send(ctx)
	}
	request.UserId = auth.ID

	response, err := c.UseCase.Create(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error creating contact")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ContactController) List(ctx *echo.Context) error {
	auth, ok := middleware.GetUser(ctx)
	if !ok {
		return httpresponse.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
	}

	request := &dto.SearchContactRequest{
		UserId: auth.ID,
		Name:   ctx.QueryParam("name"),
		Email:  ctx.QueryParam("email"),
		Phone:  ctx.QueryParam("phone"),
		Page:   IntParam(ctx, "page", 1),
		Size:   IntParam(ctx, "size", 10),
	}

	responses, total, err := c.UseCase.Search(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error searching contact")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	paging := &dto.PageMetadata{
		Page:      request.Page,
		Size:      request.Size,
		TotalItem: total,
		TotalPage: int64(math.Ceil(float64(total) / float64(request.Size))),
	}

	return httpresponse.SuccessBuilder(responses).WithPaging(paging).Send(ctx)
}

func (c *ContactController) Get(ctx *echo.Context) error {
	auth, ok := middleware.GetUser(ctx)
	if !ok {
		return httpresponse.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
	}

	request := &dto.GetContactRequest{
		UserId: auth.ID,
		ID:     ctx.Param("contactId"),
	}

	response, err := c.UseCase.Get(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error getting contact")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ContactController) Update(ctx *echo.Context) error {
	auth, ok := middleware.GetUser(ctx)
	if !ok {
		return httpresponse.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
	}

	request := new(dto.UpdateContactRequest)
	if err := ctx.Bind(request); err != nil {
		c.Log.WithError(err).Error("error parsing request body")
		return httpresponse.NewErrorBuilder(apperror.ContactErrors.InvalidRequest).Send(ctx)
	}

	request.UserId = auth.ID
	request.ID = ctx.Param("contactId")

	response, err := c.UseCase.Update(ctx.Request().Context(), request)
	if err != nil {
		c.Log.WithError(err).Error("error updating contact")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(response).Send(ctx)
}

func (c *ContactController) Delete(ctx *echo.Context) error {
	auth, ok := middleware.GetUser(ctx)
	if !ok {
		return httpresponse.NewErrorBuilder(apperror.AuthErrors.Unauthorized).Send(ctx)
	}
	contactId := ctx.Param("contactId")

	request := &dto.DeleteContactRequest{
		UserId: auth.ID,
		ID:     contactId,
	}

	if err := c.UseCase.Delete(ctx.Request().Context(), request); err != nil {
		c.Log.WithError(err).Error("error deleting contact")
		return httpresponse.NewErrorBuilder(err).Send(ctx)
	}

	return httpresponse.SuccessBuilder(true).Send(ctx)
}
