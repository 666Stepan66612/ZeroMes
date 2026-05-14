FROM golang:1.26-alpine AS builder

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY src/pkg/ /pkg/
COPY src/message-service/go.mod src/message-service/go.sum ./
RUN go mod download

COPY src/message-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /build/app \
    ./cmd/app/main.go

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

COPY --from=builder /build/app /app

ENTRYPOINT ["/app"]