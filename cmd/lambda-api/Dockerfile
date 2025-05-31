# Multi-stage build
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set work directory
WORKDIR /build

# Copy source code from the build context root
COPY . .

RUN pwd
RUN ls -la

# Build the binary
RUN GOOS=linux GOARCH=arm64 go build -C cmd/lambda-api -o bootstrap -tags lambda.norpc

RUN ls -la
RUN ls -la cmd/lambda-api

# Final stage - use AWS Lambda base image
FROM alpine:3

# Install poppler-utils
RUN apk add --no-cache poppler-utils

# Copy the binary from builder stage and make it executable
COPY --from=builder /build/cmd/lambda-api/bootstrap /main
RUN chmod +x /main

RUN pwd
RUN ls -la

# Set the CMD to the handler
ENTRYPOINT [ "/main" ]
