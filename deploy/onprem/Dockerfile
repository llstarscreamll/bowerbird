FROM golang:1.25-alpine AS builder
WORKDIR /src
COPY apps/backend/go.mod apps/backend/go.sum ./
RUN go mod download
COPY apps/backend/ .
RUN go build -o bin/api ./cmd/onprem/api && \
    go build -o bin/relay ./cmd/onprem/relay && \
    go build -o bin/events-consumer ./cmd/onprem/events-consumer && \
    go build -o bin/jobs-consumer ./cmd/onprem/jobs-consumer && \
    go build -o bin/scheduler ./cmd/onprem/scheduler && \
    go build -o bin/migrate ./cmd/onprem/migrate

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /src/bin ./bin
COPY --from=builder /src/migrations ./migrations
EXPOSE 8080
