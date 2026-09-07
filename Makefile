GOOSE_FLAGS = GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://postgres:postgres@127.0.0.1:5432/db?sslmode=disable"

.PHONY: run run-grpc \
        build build-grpc build-all build-amazon-linux \
        proto-gen \
        test test-grpc test-http \
        postgres-docker tools-install \
        goose-up goose-down goose-create

# ── Run ────────────────────────────────────────────────────────────────────────

## Run the REST API (Echo)
run:
	go run ./cmd/web

## Run the gRPC server
run-grpc:
	go run ./cmd/grpc

# ── Build ──────────────────────────────────────────────────────────────────────

## Build REST API binary → bin/web
build:
	go build -o bin/web ./cmd/web

## Build gRPC server binary → bin/grpc
build-grpc:
	go build -o bin/grpc ./cmd/grpc

## Build both binaries
build-all: build build-grpc

## Static build for Amazon Linux (linux/amd64) → bin/web-linux-amd64
build-amazon-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/web-linux-amd64 ./cmd/web

# ── Test ───────────────────────────────────────────────────────────────────────

## Run all unit tests
test:
	go test ./internal/... -v

## Run gRPC server unit tests only
test-grpc:
	go test ./internal/delivery/grpc/... -v

## Run HTTP controller unit tests only
test-http:
	go test ./internal/delivery/http/... -v

# ── Codegen ────────────────────────────────────────────────────────────────────

## Re-generate gRPC Go code from internal/delivery/grpc/proto/item.proto
proto-gen:
	protoc \
		--go_out=internal/delivery/grpc/pb \
		--go_opt=paths=import \
		--go-grpc_out=internal/delivery/grpc/pb \
		--go-grpc_opt=paths=import \
		--proto_path=internal/delivery/grpc/proto \
		internal/delivery/grpc/proto/item.proto
	@# protoc puts files in a nested subdirectory matching go_package; move them up
	@find internal/delivery/grpc/pb -name '*.go' -not -path 'internal/delivery/grpc/pb/*.go' \
		-exec mv {} internal/delivery/grpc/pb/ \;
	@find internal/delivery/grpc/pb -mindepth 1 -type d -empty -delete

# ── Infrastructure ─────────────────────────────────────────────────────────────

## Start a local PostgreSQL container
postgres-docker:
	docker run --name postgres -p 5432:5432 \
		-e POSTGRES_PASSWORD=postgres \
		-e POSTGRES_USER=postgres \
		-e POSTGRES_DB=db \
		-d postgres:16

## Install CLI tools: goose, protoc-gen-go, protoc-gen-go-grpc
tools-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# ── Migrations ─────────────────────────────────────────────────────────────────

## Apply all pending migrations
goose-up:
	$(GOOSE_FLAGS) goose -dir ./db/migrations up

## Roll back one migration step
goose-down:
	$(GOOSE_FLAGS) goose -dir ./db/migrations down

## Create a new migration (usage: make goose-create NAME=create_table_foo)
goose-create:
ifndef NAME
	$(error NAME is required. Example: make goose-create NAME=create_table_foo)
endif
	goose -dir ./db/migrations create "$(NAME)" sql
