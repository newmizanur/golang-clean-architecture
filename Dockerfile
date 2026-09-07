FROM golang:1.27-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o web ./cmd/web

FROM alpine:3.20

RUN apk add --no-cache tzdata

WORKDIR /app

COPY --from=builder /app/web /app/web
COPY --from=builder /app/config.json /app/config.json
COPY --from=builder /app/.env /app/.env
COPY --from=builder /app/db/migrations /app/db/migrations

EXPOSE 3000

CMD ["/app/web"]
