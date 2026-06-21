package grpc

import (
	"context"

	"golang-clean-architecture/internal/apperror"
	pb "golang-clean-architecture/internal/delivery/grpc/pb"
	"golang-clean-architecture/internal/dto"
	"golang-clean-architecture/internal/usecase"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ItemUseCasePort is the subset of ItemUseCase methods used by ItemGRPCServer.
type ItemUseCasePort interface {
	Create(ctx context.Context, req *dto.CreateItemRequest) (*dto.CreateItemResponse, error)
	Search(ctx context.Context, req *dto.SearchItemRequest) ([]dto.CreateItemResponse, int64, error)
	Get(ctx context.Context, req *dto.GetItemRequest) (*dto.CreateItemResponse, error)
	Update(ctx context.Context, req *dto.UpdateItemRequest) (*dto.CreateItemResponse, error)
	Delete(ctx context.Context, req *dto.DeleteItemRequest) error
}

type ItemGRPCServer struct {
	pb.UnimplementedItemServiceServer
	ItemUseCase ItemUseCasePort
}

func NewItemGRPCServer(uc *usecase.ItemUseCase) *ItemGRPCServer {
	return &ItemGRPCServer{ItemUseCase: uc}
}

func (s *ItemGRPCServer) GetItem(ctx context.Context, req *pb.GetItemRequest) (*pb.ItemResponse, error) {
	resp, err := s.ItemUseCase.Get(ctx, &dto.GetItemRequest{ID: req.Id})
	if err != nil {
		return nil, mapItemError(err)
	}
	return itemResponseToPB(resp), nil
}

func (s *ItemGRPCServer) ListItems(ctx context.Context, req *pb.ListItemsRequest) (*pb.ListItemsResponse, error) {
	items, total, err := s.ItemUseCase.Search(ctx, &dto.SearchItemRequest{
		Name: req.Name,
		SKU:  req.Sku,
		Sort: req.Sort,
		Page: int(req.Page),
		Size: int(req.Size),
	})
	if err != nil {
		return nil, mapItemError(err)
	}

	pbItems := make([]*pb.ItemResponse, len(items))
	for i := range items {
		pbItems[i] = itemResponseToPB(&items[i])
	}
	return &pb.ListItemsResponse{Items: pbItems, Total: total}, nil
}

func (s *ItemGRPCServer) CreateItem(ctx context.Context, req *pb.CreateItemRequest) (*pb.ItemResponse, error) {
	resp, err := s.ItemUseCase.Create(ctx, &dto.CreateItemRequest{
		Name:     req.Name,
		SKU:      req.Sku,
		Currency: req.Currency,
		Stock:    req.Stock,
	})
	if err != nil {
		return nil, mapItemError(err)
	}
	return itemResponseToPB(resp), nil
}

func (s *ItemGRPCServer) UpdateItem(ctx context.Context, req *pb.UpdateItemRequest) (*pb.ItemResponse, error) {
	resp, err := s.ItemUseCase.Update(ctx, &dto.UpdateItemRequest{
		ID:       req.Id,
		Name:     req.Name,
		SKU:      req.Sku,
		Currency: req.Currency,
		Stock:    req.Stock,
	})
	if err != nil {
		return nil, mapItemError(err)
	}
	return itemResponseToPB(resp), nil
}

func (s *ItemGRPCServer) DeleteItem(ctx context.Context, req *pb.DeleteItemRequest) (*pb.DeleteItemResponse, error) {
	if err := s.ItemUseCase.Delete(ctx, &dto.DeleteItemRequest{ID: req.Id}); err != nil {
		return nil, mapItemError(err)
	}
	return &pb.DeleteItemResponse{Success: true}, nil
}

func itemResponseToPB(r *dto.CreateItemResponse) *pb.ItemResponse {
	var createdAt, updatedAt int64
	if r.CreatedAt != nil {
		createdAt = r.CreatedAt.Unix()
	}
	if r.UpdatedAt != nil {
		updatedAt = r.UpdatedAt.Unix()
	}
	return &pb.ItemResponse{
		Id:            r.ID,
		Name:          r.Name,
		Sku:           r.SKU,
		Currency:      r.Currency,
		Stock:         r.Stock,
		CreatedAtUnix: createdAt,
		UpdatedAtUnix: updatedAt,
	}
}

func mapItemError(err error) error {
	appErr, ok := err.(*apperror.AppError)
	if !ok {
		return status.Error(codes.Internal, err.Error())
	}
	switch appErr {
	case apperror.ItemErrors.NotFound:
		return status.Error(codes.NotFound, appErr.Message)
	case apperror.ItemErrors.InvalidRequest:
		return status.Error(codes.InvalidArgument, appErr.Message)
	default:
		return status.Error(codes.Internal, appErr.Message)
	}
}
