# Stage 1: BUILDER
FROM golang:1.26.0-alpine3.23 AS builder
LABEL stage=gobuilder
WORKDIR /build

ENV CGO_ENABLED=0
ENV GOOS=linux

# Dependencies
RUN apk update && apk add --no-cache protoc git
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Modules
COPY ./bots/go.mod ./bots/go.sum .
RUN go mod download

# Copy source code
COPY ./bots .

# Code generation
RUN mkdir -p history/pb
RUN protoc --go_out=. --go_opt=module=telellama history/history.proto

# Build
RUN go build -ldflags="-s -w" main.go

# Compression (optional)
# RUN apk add --no-cache upx && upx --best --lzma main

# Stage 2: IMAGE
FROM alpine:3.23
WORKDIR /app

# Dependencies
RUN apk update --no-cache && apk add --no-cache ca-certificates

# Copy the binary from builder stage
COPY --from=builder /build/telellama-app .

ENTRYPOINT ["/app/telellama-app"]
