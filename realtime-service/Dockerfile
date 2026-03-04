FROM golang:1.26-bookworm

WORKDIR /app

COPY pkg/ /pkg/
COPY realtime-service/go.mod realtime-service/go.sum ./
RUN go mod download

COPY realtime-service/ .
RUN go build -o /app/exe ./cmd/app/main.go

CMD ["/app/exe"]
