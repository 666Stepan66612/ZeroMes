FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk update && apk add git ca-certificates tzdata && rm -rf /var/cache/apk/*

COPY pkg/ /pkg/
COPY api-gateway/go.mod api-gateway/go.sum ./
RUN go mod download

COPY api-gateway/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN apk update && apk add git ca-certificates tzdata && rm -rf /var/cache/apk/*

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]