FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk update && apk add git ca-certificates tzdata && rm -rf /var/cache/apk/*

COPY pkg/ /pkg/
COPY message-service/go.mod message-service/go.sum ./
RUN go mod download

COPY message-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN apk update && apk add git ca-certificates tzdata && rm -rf /var/cache/apk/*

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]