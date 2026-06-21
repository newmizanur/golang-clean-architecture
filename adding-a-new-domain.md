# Adding a New Domain to gRPC and GraphQL

This guide walks through adding a new domain (e.g. `Product`) to the gRPC and GraphQL delivery layers. It assumes the usecase and repository already exist — we only touch the delivery layers.

The `Item` domain is the reference implementation. Every step below mirrors what exists for `Item` and uses `Product` as the example.

---

## Overview

11 steps across two parts, using `Product` as the example domain.

**Part 1 — gRPC (Steps 1–4):** expose the existing usecase over the network as a gRPC service.

| Step | What | File |
|------|------|------|
| 1 | Define the proto contract | `internal/delivery/grpc/proto/product.proto` |
| 2 | Generate Go code | `internal/delivery/grpc/pb/` (via `make proto-gen`) |
| 3 | Write the gRPC server adapter | `internal/delivery/grpc/product_server.go` |
| 4 | Register the service in the entrypoint | `cmd/grpc/main.go` |

**Part 2 — GraphQL (Steps 5–11):** expose the gRPC service as GraphQL fields. The GraphQL layer calls gRPC — never the usecase directly.

| Step | What | File |
|------|------|------|
| 5 | Add schema types and operations | `internal/delivery/graphql/schema/product.graphqls` |
| 6 | Add Go model structs | `internal/delivery/graphql/graph/model/models.go` |
| 7 | Bind models in gqlgen config | `internal/delivery/graphql/gqlgen.yml` |
| 8 | Re-run codegen | `graph/generated.go` + resolver stubs (via `make graphql-gen`) |
| 9 | Add the gRPC client field to the existing root Resolver struct | `internal/delivery/graphql/resolver/resolver.go` |
| 10 | Implement the resolver stubs | `internal/delivery/graphql/resolver/product.resolvers.go` |
| 11 | Wire the client in the GraphQL entrypoint | `cmd/graphql/main.go` |

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

## Part 2 — GraphQL

### Step 5: Add types to the schema

Add to `internal/delivery/graphql/schema/item.graphqls`, or create a new file `internal/delivery/graphql/schema/product.graphqls` (gqlgen picks up all `*.graphqls` files in the schema directory):

```graphql
type Product {
  id:        ID!
  name:      String!
  sku:       String!
  price:     Int!
  createdAt: String
  updatedAt: String
}

type ProductPage {
  products: [Product!]!
  total:    Int!
}

input ListProductsInput {
  name: String
  page: Int!
  size: Int!
}

input CreateProductInput {
  name:  String!
  sku:   String!
  price: Int!
}

extend type Query {
  product(id: ID!):                  Product
  products(filter: ListProductsInput!): ProductPage!
}

extend type Mutation {
  createProduct(input: CreateProductInput!): Product!
}
```

> Use `extend type Query` / `extend type Mutation` when adding to an existing schema split across multiple files.

### Step 6: Add the Go model

Add to `internal/delivery/graphql/graph/model/models.go` (hand-written, not generated):

```go
type Product struct {
    ID        string
    Name      string
    Sku       string
    Price     int
    CreatedAt *string
    UpdatedAt *string
}

type ProductPage struct {
    Products []*Product
    Total    int
}

type ListProductsInput struct {
    Name *string
    Page int
    Size int
}

type CreateProductInput struct {
    Name  string
    Sku   string
    Price int
}
```

### Step 7: Bind the model in gqlgen.yml

Open `internal/delivery/graphql/gqlgen.yml` and add entries under `models:`:

```yaml
models:
  # existing Item entries...
  Product:
    model: golang-clean-architecture/internal/delivery/graphql/graph/model.Product
  ProductPage:
    model: golang-clean-architecture/internal/delivery/graphql/graph/model.ProductPage
  ListProductsInput:
    model: golang-clean-architecture/internal/delivery/graphql/graph/model.ListProductsInput
  CreateProductInput:
    model: golang-clean-architecture/internal/delivery/graphql/graph/model.CreateProductInput
```

Without this, gqlgen regenerates the struct in `models_gen.go` and you lose the ability to add custom fields.

### Step 8: Re-run gqlgen codegen

```sh
make graphql-gen
# or
task graphql:gen
```

gqlgen will:
- Regenerate `graph/generated.go` with the new Product resolver interfaces
- Create `resolver/product.resolvers.go` with stub methods

### Step 9: Add ProductClient to the existing root Resolver struct

`resolver.go` already exists with `ItemClient` — `Item` was the first domain so its
field was created together with the struct. For every domain added after that, you
extend the existing struct here.

Open `internal/delivery/graphql/resolver/resolver.go` and add the new field:

```go
type Resolver struct {
    ItemClient    pb.ItemServiceClient
    ProductClient pb.ProductServiceClient  // add this
}
```

### Step 10: Implement the product resolvers

Fill in the stubs in `internal/delivery/graphql/resolver/product.resolvers.go`:

```go
package resolver

import (
    "context"
    "fmt"
    "strconv"

    "golang-clean-architecture/internal/delivery/graphql/graph/model"
    pb "golang-clean-architecture/internal/delivery/grpc/pb"
)

func (r *queryResolver) Product(ctx context.Context, id string) (*model.Product, error) {
    idInt, err := strconv.ParseInt(id, 10, 64)
    if err != nil {
        return nil, fmt.Errorf("invalid id: %s", id)
    }
    resp, err := r.ProductClient.GetProduct(grpcCtx(ctx), &pb.GetProductRequest{Id: idInt})
    if err != nil {
        return nil, err
    }
    return pbToProduct(resp), nil
}

func (r *queryResolver) Products(ctx context.Context, filter model.ListProductsInput) (*model.ProductPage, error) {
    req := &pb.ListProductsRequest{
        Page: int32(filter.Page),
        Size: int32(filter.Size),
    }
    if filter.Name != nil { req.Name = *filter.Name }

    resp, err := r.ProductClient.ListProducts(grpcCtx(ctx), req)
    if err != nil {
        return nil, err
    }

    products := make([]*model.Product, len(resp.Products))
    for i, p := range resp.Products {
        products[i] = pbToProduct(p)
    }
    return &model.ProductPage{Products: products, Total: int(resp.Total)}, nil
}

func (r *mutationResolver) CreateProduct(ctx context.Context, input model.CreateProductInput) (*model.Product, error) {
    resp, err := r.ProductClient.CreateProduct(grpcCtx(ctx), &pb.CreateProductRequest{
        Name:  input.Name,
        Sku:   input.Sku,
        Price: int32(input.Price),
    })
    if err != nil {
        return nil, err
    }
    return pbToProduct(resp), nil
}

func pbToProduct(r *pb.ProductResponse) *model.Product {
    return &model.Product{
        ID:    fmt.Sprintf("%d", r.Id),
        Name:  r.Name,
        Sku:   r.Sku,
        Price: int(r.Price),
    }
}
```

> `grpcCtx(ctx)` is defined in `item.resolvers.go` — it forwards the JWT to gRPC metadata. It is shared across all resolvers because they live in the same package.

### Step 11: Wire ProductClient in cmd/graphql/main.go

```go
productClient := pb.NewProductServiceClient(conn)

rootResolver := &resolver.Resolver{
    ItemClient:    itemClient,
    ProductClient: productClient,  // add this
}
```

Verify the whole project builds:
```sh
go build ./...
```

---

## Summary checklist

```
gRPC
  [ ] internal/delivery/grpc/proto/product.proto
  [ ] make proto-gen  →  pb/product.pb.go + pb/product_grpc.pb.go
  [ ] internal/delivery/grpc/product_server.go
  [ ] cmd/grpc/main.go  →  register ProductService

GraphQL
  [ ] internal/delivery/graphql/schema/product.graphqls
  [ ] internal/delivery/graphql/graph/model/models.go  →  add Product structs
  [ ] internal/delivery/graphql/gqlgen.yml  →  bind Product models
  [ ] make graphql-gen  →  regenerate graph/generated.go + resolver stubs
  [ ] internal/delivery/graphql/resolver/resolver.go  →  add ProductClient field
  [ ] internal/delivery/graphql/resolver/product.resolvers.go  →  implement stubs
  [ ] cmd/graphql/main.go  →  wire ProductClient into Resolver
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
