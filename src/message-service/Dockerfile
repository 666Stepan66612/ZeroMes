FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN sed -i 's/https/http/g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates tzdata

COPY pkg/ /pkg/
COPY message-service/go.mod message-service/go.sum ./
RUN go mod download

COPY message-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN sed -i 's/https/http/g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates tzdata

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]