# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ZeroMes is an end-to-end encrypted messenger built with a microservices architecture. The system uses client-side encryption (AES-256-GCM) with ECDH key exchange, ensuring zero-knowledge architecture where the server never sees plaintext messages.

## Architecture

### Backend Services (Go)

Four microservices communicate via gRPC and Kafka:

- **auth-service** (port 8080): User authentication, JWT tokens, password management. Uses PostgreSQL + Redis for session storage.
- **message-service** (gRPC port 50051): Message persistence, chat management, encrypted key storage. Uses PostgreSQL and publishes to Kafka.
- **realtime-service** (port 8082): WebSocket connections for real-time message delivery. Consumes from Kafka, uses Redis for connection tracking.
- **api-gateway** (port 8083): HTTP/WebSocket entry point. Routes requests to backend services, handles JWT validation.

### Frontend (React + TypeScript)

- **Crypto layer** (`frontend/src/lib/crypto/`): Client-side encryption using Web Crypto API. ECDH for key exchange, AES-256-GCM for messages.
- **API layer** (`frontend/src/lib/api/`): REST client and WebSocket manager with auto-reconnection.
- **Pages**: Register, Login, ChangePassword, Chats (main messaging UI).
- **Components**: VirtualizedMessageList and VirtualizedChatList use react-window for performance.

### Shared Package (`pkg/`)

- **proto definitions**: `message.proto` and `realtime.proto` define gRPC contracts.
- **Generated code**: `pkg/gen/messagepb/` contains protobuf-generated Go code.
- **JWT utilities**: `pkg/jwt/` for token validation shared across services.

### Infrastructure

- **PostgreSQL**: Two databases (auth_db, messages_db) with separate instances.
- **Redis**: Session storage and WebSocket connection registry.
- **Kafka**: Message event bus (topic: `messages.sent`) for decoupling message-service from realtime-service.
- **Caddy**: Reverse proxy with automatic HTTPS, serves frontend and routes /api/* and /ws.

## Development Commands

### Full Stack

```bash
# Build and run all services
make all

# Or manually:
cd frontend && npm run build
docker compose up -d --build
```

### Frontend Only

```bash
cd frontend
npm install
npm run dev          # Development server (Vite)
npm run build        # Production build
npm run lint         # ESLint
```

### Backend Services

Each service follows the same structure:

```bash
# Run locally (requires .env configuration)
cd auth-service
go run cmd/app/main.go

# Build
go build -o bin/auth-service cmd/app/main.go
```

### Protocol Buffers

When modifying `.proto` files in `pkg/proto/`:

```bash
cd pkg
protoc --go_out=. --go-grpc_out=. proto/message.proto
protoc --go_out=. --go-grpc_out=. proto/realtime.proto
```

### Database Migrations

Migrations run automatically via docker-compose (`migrate-auth`, `migrate-messages` services). Manual execution:

```bash
migrate -path=./auth-service/migrations -database="postgres://..." up
migrate -path=./message-service/migrations -database="postgres://..." up
```

## Code Organization

### Backend Service Structure

Each Go service follows this pattern:

```
service-name/
├── cmd/app/main.go           # Entry point
├── internal/
│   ├── <domain>/             # Business logic (e.g., auth/, messaging/)
│   │   ├── repository/       # Database layer
│   │   ├── service/          # Business logic
│   │   └── transport/        # HTTP/gRPC handlers
│   └── cores/                # Shared utilities (config, logger)
├── migrations/               # SQL migrations
└── Dockerfile
```

### Frontend Structure

```
frontend/src/
├── lib/
│   ├── api/                  # HTTP client, WebSocket, auth API
│   └── crypto/               # Encryption, key management, ECDH
├── components/               # Reusable UI (ChatWindow, VirtualizedLists)
├── pages/                    # Route pages
├── types/                    # TypeScript definitions
└── hooks/                    # React hooks
```

## Key Technical Details

### Encryption Flow

1. User registers → generates ECDH keypair (secp256k1), stores private key locally
2. Starting chat → compute shared secret via ECDH, derive AES-256 key
3. Send message → encrypt with AES-GCM, send ciphertext + nonce to server
4. Receive message → decrypt using locally stored chat key

### Message Flow

1. Client → API Gateway (HTTP POST /api/messages)
2. API Gateway → message-service (gRPC SendMessage)
3. message-service → PostgreSQL (store encrypted message) + Kafka (publish event)
4. realtime-service (Kafka consumer) → checks Redis for recipient's WebSocket
5. If online → push via WebSocket, update status to DELIVERED

### Authentication

- JWT with access (short-lived) and refresh (long-lived) tokens
- Access tokens validated by api-gateway via Redis cache
- Password changes invalidate all sessions via Redis flush

## Environment Configuration

Copy `.env` and configure:
- PostgreSQL credentials (two databases)
- Redis password
- JWT secrets (use `openssl rand -hex 32`)
- Kafka cluster ID
- Service ports

## Testing

Frontend uses Vite's dev server for hot reload. Backend services have health checks in docker-compose. Use `test-scripts/` for manual API testing.

## Common Patterns

- All services use graceful shutdown (SIGTERM/SIGINT handling)
- gRPC services use connection pooling
- WebSocket implements exponential backoff reconnection
- Frontend stores sensitive keys in memory only (no localStorage for private keys)
