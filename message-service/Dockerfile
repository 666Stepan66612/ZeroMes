FROM golang:1.26-bookworm

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o /app/exe ./cmd/app/main.go

CMD ["/app/exe"]
