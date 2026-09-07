GOOSE_FLAGS = GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://postgres:postgres@127.0.0.1:5432/db?sslmode=disable"

.PHONY: run build build-amazon-linux \
        test test-unit test-cov \
        postgres-docker tools-install goose-up goose-down goose-create

## Run the API locally
run:
	go run ./cmd/web

## Build for local OS
build:
	go build -o bin/web ./cmd/web

## Static build for Amazon Linux (linux/amd64)
build-amazon-linux:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o bin/web-linux-amd64 ./cmd/web

# ── Test ───────────────────────────────────────────────────────────────────────

## Run all tests: unit + integration (integration needs Postgres — see postgres-docker/goose-up)
test:
	go test -tags=integration ./internal/... -v

## Run unit tests only (mocked dependencies, no external services required)
test-unit:
	go test ./internal/... -v

## Run unit tests and display a per-function coverage report
test-cov:
	go test ./internal/... -coverprofile=coverage.out
	go tool cover -func=coverage.out

## Start a local PostgreSQL container
postgres-docker:
	docker run --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=postgres -e POSTGRES_USER=postgres -e POSTGRES_DB=db -d postgres:16

## Install goose CLI tool
tools-install:
	go install github.com/pressly/goose/v3/cmd/goose@latest

## Run Goose migrations
goose-up:
	$(GOOSE_FLAGS) goose -dir ./db/migrations up

## Run Goose migrations down
goose-down:
	$(GOOSE_FLAGS) goose -dir ./db/migrations down

## Create a migration (usage: make goose-create NAME=create_table_items)
goose-create:
ifndef NAME
	$(error NAME is required. Example: make goose-create NAME=create_table_items)
endif
	goose -dir ./db/migrations create "$(NAME)" sql
