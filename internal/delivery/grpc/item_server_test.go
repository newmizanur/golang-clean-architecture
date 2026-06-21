package grpc

import (
	"context"
	"testing"

	"golang-clean-architecture/internal/apperror"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"
	"golang-clean-architecture/internal/dto"

	"github.com/stretchr/testify/assert"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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

func newTestServer(uc ItemUseCasePort) *ItemGRPCServer {
	return &ItemGRPCServer{ItemUseCase: uc}
}

// --- GetItem ---

func TestGetItem_Success(t *testing.T) {
	mock := &mockItemUseCase{
		getFunc: func(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: 1, Name: "Widget", SKU: "W1", Currency: "USD", Stock: 10}, nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.GetItem(context.Background(), &pb.GetItemRequest{Id: 1})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Widget", resp.Name)
	assert.Equal(t, "W1", resp.Sku)
	assert.Equal(t, "USD", resp.Currency)
	assert.Equal(t, int32(10), resp.Stock)
}

func TestGetItem_NotFound(t *testing.T) {
	mock := &mockItemUseCase{
		getFunc: func(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error) {
			return nil, apperror.ItemErrors.NotFound
		},
	}
	srv := newTestServer(mock)
	_, err := srv.GetItem(context.Background(), &pb.GetItemRequest{Id: 99})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// --- ListItems ---

func TestListItems_Success(t *testing.T) {
	mock := &mockItemUseCase{
		searchFunc: func(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error) {
			return []dto.CreateItemResponse{
				{ID: 1, Name: "A", SKU: "A1", Currency: "USD", Stock: 5},
				{ID: 2, Name: "B", SKU: "B1", Currency: "EUR", Stock: 3},
			}, 2, nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.ListItems(context.Background(), &pb.ListItemsRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 2)
	assert.Equal(t, int64(2), resp.Total)
}

func TestListItems_Empty(t *testing.T) {
	mock := &mockItemUseCase{
		searchFunc: func(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error) {
			return []dto.CreateItemResponse{}, 0, nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.ListItems(context.Background(), &pb.ListItemsRequest{})
	assert.NoError(t, err)
	assert.Len(t, resp.Items, 0)
	assert.Equal(t, int64(0), resp.Total)
}

// --- CreateItem ---

func TestCreateItem_Success(t *testing.T) {
	mock := &mockItemUseCase{
		createFunc: func(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: 42, Name: req.Name, SKU: req.SKU, Currency: req.Currency, Stock: req.Stock}, nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.CreateItem(context.Background(), &pb.CreateItemRequest{
		Name: "Gadget", Sku: "G1", Currency: "USD", Stock: 7,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(42), resp.Id)
	assert.Equal(t, "Gadget", resp.Name)
}

func TestCreateItem_InvalidRequest(t *testing.T) {
	mock := &mockItemUseCase{
		createFunc: func(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error) {
			return nil, apperror.ItemErrors.InvalidRequest
		},
	}
	srv := newTestServer(mock)
	_, err := srv.CreateItem(context.Background(), &pb.CreateItemRequest{})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.InvalidArgument, st.Code())
}

// --- UpdateItem ---

func TestUpdateItem_Success(t *testing.T) {
	mock := &mockItemUseCase{
		updateFunc: func(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error) {
			return &dto.CreateItemResponse{ID: req.ID, Name: req.Name, SKU: req.SKU, Currency: req.Currency, Stock: req.Stock}, nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.UpdateItem(context.Background(), &pb.UpdateItemRequest{
		Id: 1, Name: "Updated", Sku: "U1", Currency: "GBP", Stock: 20,
	})
	assert.NoError(t, err)
	assert.Equal(t, int64(1), resp.Id)
	assert.Equal(t, "Updated", resp.Name)
}

func TestUpdateItem_NotFound(t *testing.T) {
	mock := &mockItemUseCase{
		updateFunc: func(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error) {
			return nil, apperror.ItemErrors.NotFound
		},
	}
	srv := newTestServer(mock)
	_, err := srv.UpdateItem(context.Background(), &pb.UpdateItemRequest{Id: 999})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.NotFound, st.Code())
}

// --- DeleteItem ---

func TestDeleteItem_Success(t *testing.T) {
	mock := &mockItemUseCase{
		deleteFunc: func(ctx context.Context, req *dto.DeleteItemRequest) error {
			return nil
		},
	}
	srv := newTestServer(mock)
	resp, err := srv.DeleteItem(context.Background(), &pb.DeleteItemRequest{Id: 1})
	assert.NoError(t, err)
	assert.True(t, resp.Success)
}

func TestDeleteItem_Error(t *testing.T) {
	mock := &mockItemUseCase{
		deleteFunc: func(ctx context.Context, req *dto.DeleteItemRequest) error {
			return apperror.ItemErrors.FailedToDelete
		},
	}
	srv := newTestServer(mock)
	_, err := srv.DeleteItem(context.Background(), &pb.DeleteItemRequest{Id: 1})
	assert.Error(t, err)
	st, ok := status.FromError(err)
	assert.True(t, ok)
	assert.Equal(t, codes.Internal, st.Code())
}
