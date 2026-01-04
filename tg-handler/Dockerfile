# Stage 1: BUILDER
FROM golang:alpine AS builder

LABEL stage=gobuilder

WORKDIR /build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

# Dependencies
RUN apk update && apk add --no-cache protoc git
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

# Modules
COPY ./tg-handler/go.mod ./tg-handler/go.sum .
RUN go mod download

# Source code copy
COPY ./tg-handler .

# Code generation
RUN mkdir -p history/pb
RUN protoc --go_out=. --go_opt=module=tg-handler history/history.proto

# Build
RUN go build -ldflags="-s -w" main.go

# Compression (optional)
# RUN apk add --no-cache upx && upx --best --lzma main

# Stage 2: IMAGE
FROM alpine

# Dependencies
RUN apk update --no-cache && apk add --no-cache ca-certificates

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /build/main .

ENTRYPOINT ["./main"]
