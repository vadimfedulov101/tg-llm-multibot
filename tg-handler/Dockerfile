# Stage 1: Compile Go application
FROM golang:1.25.4-alpine3.22 AS builder

LABEL stage=gobuilder

WORKDIR /build

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY ./tg-handler/go.mod ./tg-handler/go.sum .
RUN go mod download

COPY ./tg-handler .

RUN apk update --no-cache && apk add --no-cache upx

RUN go build -ldflags="-s -w" main.go && upx --best --lzma main

# Stage 2: Create a lightweight image
FROM alpine

RUN apk update --no-cache && apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /build/main .

CMD ["./main"]
