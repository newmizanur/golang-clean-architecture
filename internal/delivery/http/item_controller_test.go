package http

import (
	"context"
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"golang-clean-architecture/internal/apperror"
	"golang-clean-architecture/internal/dto"

	"github.com/labstack/echo/v5"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// mockItemUseCase is a hand-written mock implementing ItemUseCasePort.
type mockItemUseCase struct {
	createFunc func(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error)
	searchFunc func(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error)
	getFunc    func(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error)
	updateFunc func(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error)
	deleteFunc func(ctx context.Context, req *dto.DeleteItemRequest) error
}

func (m *mockItemUseCase) Create(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error) {
	return m.createFunc(ctx, req)
}
func (m *mockItemUseCase) Search(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error) {
	return m.searchFunc(ctx, req)
}
func (m *mockItemUseCase) Get(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error) {
	return m.getFunc(ctx, req)
}
func (m *mockItemUseCase) Update(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error) {
	return m.updateFunc(ctx, req)
}
func (m *mockItemUseCase) Delete(ctx context.Context, req *dto.DeleteItemRequest) error {
	return m.deleteFunc(ctx, req)
}

func newTestController(mock *mockItemUseCase) *ItemController {
	log := logrus.New()
	log.SetOutput(io.Discard)
	return NewItemController(mock, log)
}

func setupEcho(method, path, body string) (*echo.Context, *httptest.ResponseRecorder) {
	e := echo.New()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	return c, rec
}

// ---- Create tests ----

func TestItemController_Create_Success(t *testing.T) {
	mock := &mockItemUseCase{
		createFunc: func(_ context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: 1, Name: "Widget", SKU: "W-001", Stock: 10}, nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("POST", "/api/items", `{"name":"Widget","sku":"W-001","currency":"USD","stock":10}`)
	err := ctrl.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"id":1`)
	assert.Contains(t, body, `"name":"Widget"`)
	assert.Contains(t, body, `"sku":"W-001"`)
	assert.Contains(t, body, `"stock":10`)
}

func TestItemController_Create_BindError(t *testing.T) {
	ctrl := newTestController(&mockItemUseCase{})

	c, rec := setupEcho("POST", "/api/items", `{invalid json}`)
	err := ctrl.Create(c)
	assert.NoError(t, err)
	assert.Equal(t, 400, rec.Code)
}

// ---- List tests ----

func TestItemController_List_Success(t *testing.T) {
	items := []dto.CreateItemResponse{
		{ID: 1, Name: "A", SKU: "A-1"},
		{ID: 2, Name: "B", SKU: "B-1"},
	}
	mock := &mockItemUseCase{
		searchFunc: func(_ context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error) {
			return items, 2, nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("GET", "/api/items?page=1&size=10", "")
	err := ctrl.List(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"id":1`)
	assert.Contains(t, body, `"id":2`)
	assert.Contains(t, body, `"totalItem":2`)
}

func TestItemController_List_Empty(t *testing.T) {
	mock := &mockItemUseCase{
		searchFunc: func(_ context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error) {
			return []dto.CreateItemResponse{}, 0, nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("GET", "/api/items?page=1&size=10", "")
	err := ctrl.List(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, `"totalItem":0`)
}

// ---- Get tests ----

func TestItemController_Get_Success(t *testing.T) {
	mock := &mockItemUseCase{
		getFunc: func(_ context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: 1, Name: "Widget", SKU: "W-001"}, nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("GET", "/api/items/1", "")
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Get(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), `"id":1`)
}

func TestItemController_Get_NotFound(t *testing.T) {
	mock := &mockItemUseCase{
		getFunc: func(_ context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error) {
			return nil, apperror.ItemErrors.NotFound
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("GET", "/api/items/1", "")
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Get(c)
	assert.NoError(t, err)
	assert.Equal(t, 404, rec.Code)
}

func TestItemController_Get_InvalidID(t *testing.T) {
	ctrl := newTestController(&mockItemUseCase{})

	c, rec := setupEcho("GET", "/api/items/abc", "")
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "abc"}})
	err := ctrl.Get(c)
	assert.NoError(t, err)
	assert.Equal(t, 400, rec.Code)
}

// ---- Update tests ----

func TestItemController_Update_Success(t *testing.T) {
	mock := &mockItemUseCase{
		updateFunc: func(_ context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: 1, Name: "Updated"}, nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("PUT", "/api/items/1", `{"name":"Updated"}`)
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
	assert.Contains(t, rec.Body.String(), `"name":"Updated"`)
}

func TestItemController_Update_NotFound(t *testing.T) {
	mock := &mockItemUseCase{
		updateFunc: func(_ context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error) {
			return nil, apperror.ItemErrors.NotFound
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("PUT", "/api/items/1", `{"name":"Updated"}`)
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Update(c)
	assert.NoError(t, err)
	assert.Equal(t, 404, rec.Code)
}

// ---- Delete tests ----

func TestItemController_Delete_Success(t *testing.T) {
	mock := &mockItemUseCase{
		deleteFunc: func(_ context.Context, req *dto.DeleteItemRequest) error {
			return nil
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("DELETE", "/api/items/1", "")
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Delete(c)
	assert.NoError(t, err)
	assert.Equal(t, 200, rec.Code)
}

func TestItemController_Delete_NotFound(t *testing.T) {
	mock := &mockItemUseCase{
		deleteFunc: func(_ context.Context, req *dto.DeleteItemRequest) error {
			return apperror.ItemErrors.NotFound
		},
	}
	ctrl := newTestController(mock)

	c, rec := setupEcho("DELETE", "/api/items/1", "")
	c.SetPathValues(echo.PathValues{{Name: "itemId", Value: "1"}})
	err := ctrl.Delete(c)
	assert.NoError(t, err)
	assert.Equal(t, 404, rec.Code)
}
