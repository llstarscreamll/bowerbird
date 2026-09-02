FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ .
RUN go build -o bin/api ./cmd/api && \
    go build -o bin/worker ./cmd/worker && \
    go build -o bin/migrate ./cmd/migrate

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /src/bin ./bin
COPY --from=builder /src/migrations ./migrations
EXPOSE 8080
