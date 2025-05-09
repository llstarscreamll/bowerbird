FROM golang:1.24-alpine

RUN apk add zip

WORKDIR /app

RUN mkdir -p dist

CMD GOARCH=arm64 GOOS=linux go build -C cmd/lambda-api -o ../../dist/bootstrap -tags netgo \
    && zip -j /app/dist/api-server.zip /app/dist/bootstrap
