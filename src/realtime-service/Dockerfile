FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN sed -i 's/https/http/g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates tzdata

COPY pkg/ /pkg/
COPY realtime-service/go.mod realtime-service/go.sum ./
RUN go mod download

COPY realtime-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN sed -i 's/https/http/g' /etc/apk/repositories && \
    apk add --no-cache git ca-certificates tzdata

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]