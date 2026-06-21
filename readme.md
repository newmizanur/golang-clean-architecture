# golang-clean-architecture-minimal

A minimal Go clean architecture example with three delivery layers — REST (Echo), gRPC, and GraphQL — all sitting on top of the same unchanged usecase and repository layer.

## Architecture

```
REST client          GraphQL client (future SvelteKit)
     |                        |
     | HTTP/JSON              | GraphQL query
     v                        v
cmd/web              cmd/graphql  (port 4000)
internal/delivery/http/      internal/delivery/graphql/
     |                        |
     |                        | gRPC call (JWT in metadata)
     |                        v
     |               cmd/grpc  (port 50051)
     |               internal/delivery/grpc/
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

The GraphQL layer is a BFF (Backend-for-Frontend): it aggregates data from the gRPC service and enriches it with additional lookups (e.g. currency display info). It never calls the usecase directly.

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
  "web":     { "port": 3000  },
  "grpc":    { "port": 50051 },
  "graphql": { "port": 4000  },
  "jwt":     { "secret": "change-me", "ttl_minutes": 1440 }
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

### GraphQL server
```sh
task run:graphql
# or
make run-graphql
# or
go run ./cmd/graphql
```

GraphQL playground is available at http://localhost:4000/ when the server is running.

## Building

```sh
# All binaries
task build:all
make build-all

# Individual
task build            # REST — bin/web
task build:grpc       # gRPC — bin/grpc
task build:graphql    # GraphQL — bin/graphql

# Amazon Linux static binary (REST)
task build:amazon-linux
make build-amazon-linux
```

## Proto / gqlgen codegen

After editing `internal/delivery/grpc/proto/item.proto`:
```sh
task proto:gen
# or
make proto-gen
```

After editing `internal/delivery/graphql/schema/item.graphqls`:
```sh
task graphql:gen
# or
make graphql-gen
```

## GraphQL queries and mutations

All queries and mutations are in [`graphql.http`](graphql.http), compatible with the **REST Client** extension (VS Code), **IntelliJ HTTP Client**, and **Bruno**.

Get a token from `api.http → Login`, paste it into `graphql.http` at `@token`, then run any request directly from the file.

The GraphiQL UI is available at http://localhost:4000/ when `cmd/graphql` is running. Add the `Authorization` header in the **Headers** panel (bottom-left in GraphiQL):

```
Authorization: Bearer <token>
```

### Query examples (for curl / reference)

**List items:**
```sh
curl -s -X POST http://localhost:4000/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query":"{ items(filter:{page:1,size:5}){ items{id name stock currency{code symbol}} total } }"}' | jq .
```

**Get item by ID:**
```sh
curl -s -X POST http://localhost:4000/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query":"{ item(id:\"1\"){ id name sku stock currency{code symbol decimalPlaces} } }"}' | jq .
```

**Create item:**
```sh
curl -s -X POST http://localhost:4000/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <TOKEN>" \
  -d '{"query":"mutation { createItem(input:{name:\"Keyboard\",sku:\"KB-001\",currency:\"USD\",stock:10}){ id name currency{code symbol} } }"}' | jq .
```

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

## N+1 demo (GraphQL)

The `Item.currency` field demonstrates the N+1 problem and its fix with DataLoader:

- **Naive (before DataLoader):** querying 20 items fires 20 separate `GetCurrency` calls (+1 `ListItems`).
- **Fixed (DataLoader):** same query fires 1 `ListItems` + 1 batched `GetCurrencies`, regardless of page size. Duplicate currencies are de-duplicated before the batch call.

The `currencyservice` package in `internal/delivery/graphql/` is a stand-in for what would be a real currency/FX microservice in production. It lives in the delivery layer (not the domain) because enriching items with currency display data is a BFF/presentation concern, not business logic.
