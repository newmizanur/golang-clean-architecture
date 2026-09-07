# Adding a New Domain to gRPC

This guide walks through adding a new domain (e.g. `Product`) to the gRPC delivery layer. It assumes the usecase and repository already exist — we only touch the delivery layer.

The `Item` domain is the reference implementation. Every step below mirrors what exists for `Item` and uses `Product` as the example.

---

## Overview

4 steps, using `Product` as the example domain: expose the existing usecase over the network as a gRPC service.

| Step | What | File |
|------|------|------|
| 1 | Define the proto contract | `internal/delivery/grpc/proto/product.proto` |
| 2 | Generate Go code | `internal/delivery/grpc/pb/` (via `make proto-gen`) |
| 3 | Write the gRPC server adapter | `internal/delivery/grpc/product_server.go` |
| 4 | Register the service in the entrypoint | `cmd/grpc/main.go` |

---

## Prerequisites

- Usecase exists at `internal/usecase/product_usecase.go` with a concrete struct, e.g. `ProductUseCase`
- DTOs exist at `internal/dto/product_model.go`
- Error constants exist at `internal/apperror/error_codes.go` under `ProductErrors`

---

## Part 1 — gRPC

### Step 1: Define the proto

Create `internal/delivery/grpc/proto/product.proto`:

```protobuf
syntax = "proto3";

package product;
option go_package = "golang-clean-architecture/internal/delivery/grpc/pb";

service ProductService {
  rpc GetProduct(GetProductRequest)       returns (ProductResponse);
  rpc ListProducts(ListProductsRequest)   returns (ListProductsResponse);
  rpc CreateProduct(CreateProductRequest) returns (ProductResponse);
  rpc UpdateProduct(UpdateProductRequest) returns (ProductResponse);
  rpc DeleteProduct(DeleteProductRequest) returns (DeleteProductResponse);
}

message GetProductRequest  { int64 id = 1; }

message ListProductsRequest {
  string name = 1;
  int32  page = 2;
  int32  size = 3;
}

message ListProductsResponse {
  repeated ProductResponse products = 1;
  int64                    total    = 2;
}

message CreateProductRequest {
  string name  = 1;
  string sku   = 2;
  int32  price = 3;
}

message UpdateProductRequest {
  int64  id    = 1;
  string name  = 2;
  string sku   = 3;
  int32  price = 4;
}

message DeleteProductRequest  { int64 id = 1; }
message DeleteProductResponse { bool success = 1; }

message ProductResponse {
  int64  id             = 1;
  string name           = 2;
  string sku            = 3;
  int32  price          = 4;
  int64  created_at_unix = 5;
  int64  updated_at_unix = 6;
}
```

### Step 2: Generate Go code

```sh
make proto-gen
# or
task proto:gen
```

This produces `internal/delivery/grpc/pb/product.pb.go` and `product_grpc.pb.go`.

> If you have multiple proto files, `make proto-gen` runs protoc once per file. You may need to extend the Makefile target to list both protos, or run protoc manually targeting `product.proto`.

### Step 3: Write the gRPC server adapter

Create `internal/delivery/grpc/product_server.go`:

```go
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

type ProductGRPCServer struct {
    pb.UnimplementedProductServiceServer
    ProductUseCase *usecase.ProductUseCase
}

func NewProductGRPCServer(uc *usecase.ProductUseCase) *ProductGRPCServer {
    return &ProductGRPCServer{ProductUseCase: uc}
}

func (s *ProductGRPCServer) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.ProductResponse, error) {
    resp, err := s.ProductUseCase.Get(ctx, &dto.GetProductRequest{ID: req.Id})
    if err != nil {
        return nil, mapProductError(err)
    }
    return productResponseToPB(resp), nil
}

// ... ListProducts, CreateProduct, UpdateProduct, DeleteProduct follow the
// same pattern as item_server.go — map pb request → dto → usecase → pb response.

func productResponseToPB(r *dto.ProductResponse) *pb.ProductResponse {
    var createdAt, updatedAt int64
    if r.CreatedAt != nil { createdAt = r.CreatedAt.Unix() }
    if r.UpdatedAt != nil { updatedAt = r.UpdatedAt.Unix() }
    return &pb.ProductResponse{
        Id:            r.ID,
        Name:          r.Name,
        Sku:           r.SKU,
        Price:         r.Price,
        CreatedAtUnix: createdAt,
        UpdatedAtUnix: updatedAt,
    }
}

func mapProductError(err error) error {
    appErr, ok := err.(*apperror.AppError)
    if !ok {
        return status.Error(codes.Internal, err.Error())
    }
    switch appErr {
    case apperror.ProductErrors.NotFound:
        return status.Error(codes.NotFound, appErr.Message)
    case apperror.ProductErrors.InvalidRequest:
        return status.Error(codes.InvalidArgument, appErr.Message)
    default:
        return status.Error(codes.Internal, appErr.Message)
    }
}
```

**Rule:** one `mapXxxError` function per domain — each domain has its own error sentinel values.

### Step 4: Register the new server in cmd/grpc/main.go

```go
// existing
productRepository := repository.NewProductRepository(db, log)
productUseCase    := usecase.NewProductUseCase(db, log, validate, productRepository)

// existing registration line for item stays — add the new one below it
pb.RegisterItemServiceServer(grpcServer, grpcdelivery.NewItemGRPCServer(itemUseCase))
pb.RegisterProductServiceServer(grpcServer, grpcdelivery.NewProductGRPCServer(productUseCase))
```

Verify:
```sh
go build ./cmd/grpc/...
```

---

## Summary checklist

```
gRPC
  [ ] internal/delivery/grpc/proto/product.proto
  [ ] make proto-gen  →  pb/product.pb.go + pb/product_grpc.pb.go
  [ ] internal/delivery/grpc/product_server.go
  [ ] cmd/grpc/main.go  →  register ProductService
```

---

## What you never touch

These are owned by the domain layer and must not change when adding a delivery layer:

| Path | Owner |
|------|-------|
| `internal/usecase/` | Business logic |
| `internal/repository/` | Data access |
| `internal/persistence/model/` | DB models |
| `internal/dto/` | Already exists — delivery just reads from it |
| `db/migrations/` | Schema — separate concern |
