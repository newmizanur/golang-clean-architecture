package resolver

import (
	"context"
	"testing"

	"golang-clean-architecture/internal/delivery/graphql/currencyservice"
	"golang-clean-architecture/internal/delivery/graphql/graph/model"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// mockItemServiceClient is a hand-written mock of pb.ItemServiceClient.
type mockItemServiceClient struct {
	getItemFunc    func(ctx context.Context, in *pb.GetItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error)
	listItemsFunc  func(ctx context.Context, in *pb.ListItemsRequest, opts ...grpc.CallOption) (*pb.ListItemsResponse, error)
	createItemFunc func(ctx context.Context, in *pb.CreateItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error)
}

func (m *mockItemServiceClient) GetItem(ctx context.Context, in *pb.GetItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
	if m.getItemFunc != nil {
		return m.getItemFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (m *mockItemServiceClient) ListItems(ctx context.Context, in *pb.ListItemsRequest, opts ...grpc.CallOption) (*pb.ListItemsResponse, error) {
	if m.listItemsFunc != nil {
		return m.listItemsFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (m *mockItemServiceClient) CreateItem(ctx context.Context, in *pb.CreateItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
	if m.createItemFunc != nil {
		return m.createItemFunc(ctx, in, opts...)
	}
	return nil, nil
}

func (m *mockItemServiceClient) UpdateItem(ctx context.Context, in *pb.UpdateItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
	return nil, nil
}

func (m *mockItemServiceClient) DeleteItem(ctx context.Context, in *pb.DeleteItemRequest, opts ...grpc.CallOption) (*pb.DeleteItemResponse, error) {
	return nil, nil
}

// ptr is a generic helper to get a pointer to any value.
func ptr[T any](v T) *T { return &v }

func newResolver(mock *mockItemServiceClient) *Resolver {
	return &Resolver{ItemClient: mock}
}

// --- queryResolver.Item ---

func TestQueryResolver_Item_Success(t *testing.T) {
	mock := &mockItemServiceClient{
		getItemFunc: func(ctx context.Context, in *pb.GetItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
			return &pb.ItemResponse{Id: 1, Name: "Keyboard", Sku: "KB1", Currency: "USD", Stock: 5}, nil
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	item, err := r.Query().Item(ctx, "1")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ID != "1" {
		t.Errorf("expected ID=1, got %s", item.ID)
	}
	if item.Name != "Keyboard" {
		t.Errorf("expected Name=Keyboard, got %s", item.Name)
	}
	if item.CurrencyCode != "USD" {
		t.Errorf("expected CurrencyCode=USD, got %s", item.CurrencyCode)
	}
}

func TestQueryResolver_Item_GrpcError(t *testing.T) {
	mock := &mockItemServiceClient{
		getItemFunc: func(ctx context.Context, in *pb.GetItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
			return nil, status.Error(codes.NotFound, "not found")
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	_, err := r.Query().Item(ctx, "1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- queryResolver.Items ---

func TestQueryResolver_Items_Success(t *testing.T) {
	mock := &mockItemServiceClient{
		listItemsFunc: func(ctx context.Context, in *pb.ListItemsRequest, opts ...grpc.CallOption) (*pb.ListItemsResponse, error) {
			return &pb.ListItemsResponse{
				Items: []*pb.ItemResponse{
					{Id: 1, Name: "Item1", Sku: "S1", Currency: "USD", Stock: 1},
					{Id: 2, Name: "Item2", Sku: "S2", Currency: "SGD", Stock: 2},
				},
				Total: 2,
			}, nil
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	page, err := r.Query().Items(ctx, model.ListItemsInput{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(page.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(page.Items))
	}
	if page.Total != 2 {
		t.Errorf("expected Total=2, got %d", page.Total)
	}
}

func TestQueryResolver_Items_WithNameFilter(t *testing.T) {
	var capturedReq *pb.ListItemsRequest
	mock := &mockItemServiceClient{
		listItemsFunc: func(ctx context.Context, in *pb.ListItemsRequest, opts ...grpc.CallOption) (*pb.ListItemsResponse, error) {
			capturedReq = in
			return &pb.ListItemsResponse{Items: []*pb.ItemResponse{}, Total: 0}, nil
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	_, err := r.Query().Items(ctx, model.ListItemsInput{Name: ptr("key"), Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if capturedReq == nil {
		t.Fatal("mock was not called")
	}
	if capturedReq.Name != "key" {
		t.Errorf("expected req.Name=key, got %s", capturedReq.Name)
	}
}

func TestQueryResolver_Items_Empty(t *testing.T) {
	mock := &mockItemServiceClient{
		listItemsFunc: func(ctx context.Context, in *pb.ListItemsRequest, opts ...grpc.CallOption) (*pb.ListItemsResponse, error) {
			return &pb.ListItemsResponse{Items: []*pb.ItemResponse{}, Total: 0}, nil
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	page, err := r.Query().Items(ctx, model.ListItemsInput{Page: 1, Size: 10})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(page.Items) != 0 {
		t.Errorf("expected 0 items, got %d", len(page.Items))
	}
	if page.Total != 0 {
		t.Errorf("expected Total=0, got %d", page.Total)
	}
}

// --- mutationResolver.CreateItem ---

func TestMutationResolver_CreateItem_Success(t *testing.T) {
	mock := &mockItemServiceClient{
		createItemFunc: func(ctx context.Context, in *pb.CreateItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
			return &pb.ItemResponse{Id: 99, Name: "New", Sku: "N1", Currency: "SGD", Stock: 3}, nil
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	item, err := r.Mutation().CreateItem(ctx, model.CreateItemInput{Name: "New", Sku: "N1", Currency: "SGD", Stock: 3})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if item.ID != "99" {
		t.Errorf("expected ID=99, got %s", item.ID)
	}
	if item.CurrencyCode != "SGD" {
		t.Errorf("expected CurrencyCode=SGD, got %s", item.CurrencyCode)
	}
}

func TestMutationResolver_CreateItem_GrpcError(t *testing.T) {
	mock := &mockItemServiceClient{
		createItemFunc: func(ctx context.Context, in *pb.CreateItemRequest, opts ...grpc.CallOption) (*pb.ItemResponse, error) {
			return nil, status.Error(codes.InvalidArgument, "bad")
		},
	}
	r := newResolver(mock)
	ctx := ContextWithJWT(context.Background(), "test-token")
	_, err := r.Mutation().CreateItem(ctx, model.CreateItemInput{Name: "New", Sku: "N1", Currency: "SGD", Stock: 3})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

// --- itemResolver.Currency (naive, no gRPC mock needed) ---

func TestItemResolver_Currency_KnownCode(t *testing.T) {
	currencyservice.CallCount.Store(0)
	r := newResolver(&mockItemServiceClient{})
	ctx := ContextWithJWT(context.Background(), "test-token")
	obj := &model.Item{CurrencyCode: "USD"}
	currency, err := r.Item().Currency(ctx, obj)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if currency.Code != "USD" {
		t.Errorf("expected Code=USD, got %s", currency.Code)
	}
	if currency.Symbol != "$" {
		t.Errorf("expected Symbol=$, got %s", currency.Symbol)
	}
	if currency.DecimalPlaces != 2 {
		t.Errorf("expected DecimalPlaces=2, got %d", currency.DecimalPlaces)
	}
}

func TestItemResolver_Currency_UnknownCode(t *testing.T) {
	currencyservice.CallCount.Store(0)
	r := newResolver(&mockItemServiceClient{})
	ctx := ContextWithJWT(context.Background(), "test-token")
	obj := &model.Item{CurrencyCode: "XYZ"}
	currency, err := r.Item().Currency(ctx, obj)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if currency.Code != "XYZ" {
		t.Errorf("expected Code=XYZ, got %s", currency.Code)
	}
	if currency.Symbol != "?" {
		t.Errorf("expected Symbol=?, got %s", currency.Symbol)
	}
}
