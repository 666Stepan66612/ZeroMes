FROM golang:1.26-bookworm

WORKDIR /app

COPY pkg/ /pkg/
COPY message-service/go.mod message-service/go.sum ./
RUN go mod download

COPY message-service/ .
RUN go build -o /app/exe ./cmd/app/main.go

CMD ["/app/exe"]
