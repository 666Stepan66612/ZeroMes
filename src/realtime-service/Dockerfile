FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY pkg/ /pkg/
COPY realtime-service/go.mod realtime-service/go.sum ./
RUN go mod download

COPY realtime-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]