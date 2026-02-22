# Stage 1: BUILDER
FROM golang:1.26.0-alpine3.23 AS builder

LABEL stage=gobuilder
WORKDIR /build

# Install dependencies
RUN apk update && apk add --no-cache protoc git
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Download Go dependencies
COPY bots/go.mod ./bots/go.sum .
RUN go mod download

# Copy source code
COPY bots .

# Generate code
RUN mkdir -p history/pb
RUN protoc --go_out=. --go_opt=module=telellama history/history.proto

# Build
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o telellama-app .

# Compression (optional)
# RUN apk add --no-cache upx && upx --best --lzma main

# Stage 2: IMAGE
FROM alpine:3.23

WORKDIR /app

# Install dependencies
RUN apk update --no-cache && apk add --no-cache ca-certificates

# Copy the binary from builder stage
COPY --from=builder /telellama-app /app/telellama-app .

ENTRYPOINT ["/app/telellama-app"]
