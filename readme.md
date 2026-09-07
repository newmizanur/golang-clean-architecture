# golang-clean-architecture-minimal

A minimal Go clean architecture example with two delivery layers — REST (Echo) and gRPC — sitting on top of the same unchanged usecase and repository layer.

## Architecture

```
REST client                gRPC client
     |                        |
     | HTTP/JSON              | gRPC call (JWT in metadata)
     v                        v
cmd/web               cmd/grpc  (port 50051)
internal/delivery/http/      internal/delivery/grpc/
     |                        |
     +------------------------+
                              | direct Go call
                              v
              internal/usecase/      (business logic — untouched)
                              |
                              v
              internal/repository/   (data access — untouched)
                              |
                              v
                         PostgreSQL
```

## Setup

### 1. Install prerequisites

```sh
# Taskfile CLI
go install github.com/go-task/task/v3/cmd/task@latest

# Goose (migrations) and protoc plugins
task tools:install
```

Or with make:
```sh
make tools-install
```

### 2. Start PostgreSQL

```sh
task postgres:docker
# or
make postgres-docker
```

### 3. Run migrations

```sh
task goose:up
# or
make goose-up
```

### 4. Config

Edit `config.json` for database DSN, JWT secret, and server ports:

```json
{
  "web":  { "port": 3000  },
  "grpc": { "port": 50051 },
  "jwt":  { "secret": "change-me", "ttl_minutes": 1440 }
}
```

## Running the servers

### REST API (Echo)
```sh
task run
# or
make run
# or
go run ./cmd/web
```

### gRPC server
```sh
task run:grpc
# or
make run-grpc
# or
go run ./cmd/grpc
```

## Testing

Run unit tests only (mocked dependencies, no Postgres required):
```sh
task test:unit
# or
make test-unit
```

Run unit tests with a per-function coverage report:
```sh
task test:cov
# or
make test-cov
```

Run everything, including integration tests that hit a real Postgres (needs `postgres-docker` + `goose-up` first):
```sh
task test
# or
make test
```

Domain-specific test targets are still available: `test:grpc`/`test-grpc` and `test:http`/`test-http`.

## Building

```sh
# Both binaries
task build:all
make build-all

# Individual
task build       # REST — bin/web
task build:grpc  # gRPC — bin/grpc

# Amazon Linux static binary (REST)
task build:amazon-linux
make build-amazon-linux
```

## Proto codegen

After editing `internal/delivery/grpc/proto/item.proto`:
```sh
task proto:gen
# or
make proto-gen
```

## Adding a new gRPC domain/feature

Full walkthrough: [`adding-a-new-domain.md`](adding-a-new-domain.md). Short version, using `Product` as the example (assumes `internal/usecase/product_usecase.go` and `internal/repository/product_repository.go` already exist):

1. **Write the proto contract** at `internal/delivery/grpc/proto/product.proto` — define the service, request/response messages (see `item.proto` next to it for the pattern).
2. **Generate Go code**:
   ```sh
   task proto:gen
   # or
   make proto-gen
   ```
   This produces `internal/delivery/grpc/pb/product.pb.go` and `product_grpc.pb.go`.
3. **Write the gRPC server adapter** at `internal/delivery/grpc/product_server.go` — a thin struct implementing the generated `pb.UnimplementedProductServiceServer`, mapping pb request → dto → usecase → pb response (mirror `item_server.go`).
4. **Register the new service** in `cmd/grpc/main.go`: construct the repository/usecase, then `pb.RegisterProductServiceServer(grpcServer, grpcdelivery.NewProductGRPCServer(productUseCase))` alongside the existing `Item` registration.
5. **Verify**: `go build ./cmd/grpc/...` and add unit tests under `internal/delivery/grpc/product_server_test.go` (see `item_server_test.go` for the pattern) — run with `task test:grpc` / `make test-grpc`.

## Verifying the gRPC server (grpcurl)

Install grpcurl: `brew install grpcurl`

Get a JWT first:
```sh
curl -s -X POST http://localhost:3000/api/users/login \
  -H "Content-Type: application/json" \
  -d '{"username":"<user>","password":"<pass>"}' | jq .data.token
```

Then call gRPC:
```sh
grpcurl -plaintext \
  -H "authorization: Bearer <TOKEN>" \
  -d '{"page":1,"size":5}' \
  localhost:50051 item.ItemService/ListItems
```

## Database migrations

```sh
task goose:up                          # apply all pending
task goose:down                        # roll back one step
task goose:create -- create_table_foo  # new migration file
```
