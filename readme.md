## Setup

Install Taskfile CLI:
```
  go install github.com/go-task/task/v3/cmd/task@latest
```

Start PostgreSQL (Docker):
```
  task postgres:docker
```

Install Goose tool:
```
  task tools:install
```

Run migrations:
```
  task goose:up
```

Run the API:
```
  task run
```

Build:
```
  task build
```

Build for Amazon Linux (static):
```
  task build:amazon-linux
```

## Testing

Run unit tests only (mocked dependencies, no Postgres required):
```
  task test:unit
  # or
  make test-unit
```

Run unit tests with a per-function coverage report:
```
  task test:cov
  # or
  make test-cov
```

Run everything, including integration tests that hit a real Postgres (needs `postgres-docker` + `goose-up` first):
```
  task test
  # or
  make test
```

## Config

Update `config.json` for database and JWT settings.
