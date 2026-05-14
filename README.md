# ZeroMes

![Backend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/backend.yml/badge.svg)
![Frontend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/frontend.yml/badge.svg)
![Docker Build](https://github.com/666Stepan66612/ZeroMes/actions/workflows/docker.yml/badge.svg)
![Code Quality](https://github.com/666Stepan66612/ZeroMes/actions/workflows/code-quality.yml/badge.svg)

End-to-end encrypted messenger with zero-knowledge architecture. The server never sees plaintext messages.

## Features

- 🔐 **End-to-End Encryption** - AES-256-GCM with ECDH key exchange
- 🚀 **Real-time messaging** - WebSocket connections
- 🔑 **Zero-knowledge** - Server stores only encrypted data
- 🏗️ **Microservices** - Scalable architecture with gRPC
- 📦 **Event-driven** - Kafka for async communication
- ✅ **Well-tested** - Comprehensive unit test coverage

## Architecture

```
┌─────────────┐
│   Frontend  │ (React + TypeScript)
│  (Vite SPA) │
└──────┬──────┘
       │ HTTPS/REST :443
       ▼
┌─────────────┐
│    Caddy    │ (Reverse Proxy)
└──────┬──────┘
       │
       ├─────► /api/*  ──────┐
       │                     │
       └─────► /ws     ──────┤
                             ▼
                    ┌────────────────┐
                    │  API Gateway   │ :8080
                    └────────┬───────┘
                             │ gRPC
        ┌────────────────────┼────────────────────┐
        │                    │                    │
        ▼                    ▼                    ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│Auth Service  │    │Message       │    │Realtime      │
│(JWT, Users)  │    │Service       │    │Service       │
│    :50051    │    │(Messages)    │    │(WebSocket)   │
└──────┬───────┘    │    :50052    │    │    :50053    │
       │            └───┬──────┬───┘    └───────┬──────┘
       │ pgx :5432      │      │                │
       ▼                │      │                │
  PostgreSQL            │      │ Kafka :9092    │
   (auth_db)            │      └───────┐        │
                        │              ▼        │
                        │           Kafka       │
                        │       (message events)│
                        │              │        │
                        │ pgx :5432    │ Kafka  │
                        ▼              └────────┤
                   PostgreSQL                   │
                  (messages_db)                 │ Redis :6379
                                                ▼
                                             Redis
                                         (connections)
```

## Tech Stack

### Backend
- **Go 1.25.5** - All microservices
- **gRPC** - Inter-service communication
- **PostgreSQL** - Persistent storage
- **Redis** - Session storage & WebSocket registry
- **Kafka** - Event streaming
- **Caddy** - Reverse proxy with auto-HTTPS

### Frontend
- **React 18** - UI framework
- **TypeScript** - Type safety
- **Vite** - Build tool
- **Web Crypto API** - Client-side encryption
- **@noble/secp256k1** - ECDH key exchange

## Quick Start

### Prerequisites
- Docker & Docker Compose
- Node.js 20+ (for frontend development)
- Go 1.25.5+ (for backend development)

### Run Full Stack

```bash
# Build and start all services
make all

# Or manually:
cd frontend && npm run build
docker compose up -d
```

Services will be available at:
- Frontend: https://localhost
- API: https://localhost/api
- WebSocket: wss://localhost/ws

### Development

**Backend (Go):**
```bash
cd auth-service
go run cmd/app/main.go
```

**Frontend (React):**
```bash
cd frontend
npm install
npm run dev  # http://localhost:5173
```

**Run Tests:**
```bash
# Backend tests
cd message-service
go test -v ./...

# Frontend
cd frontend
npm run lint
npm run build
```

## Project Structure

```
ZeroMes/
├── auth-service/          # User authentication & JWT
│   ├── internal/
│   │   └── auth/
│   │       ├── service/   # Business logic + tests
│   │       ├── repository/
│   │       └── transport/
│   └── migrations/
├── message-service/       # Message persistence
│   ├── internal/
│   │   ├── messaging/
│   │   │   ├── service/   # Business logic + tests
│   │   │   ├── repository/
│   │   │   └── transport/
│   │   └── cores/
│   │       └── outbox-worker/  # Kafka retry logic
│   └── migrations/
├── realtime-service/      # WebSocket connections
├── api-gateway/           # HTTP/WS entry point
├── frontend/              # React SPA
│   ├── src/
│   │   ├── lib/
│   │   │   ├── crypto/    # E2EE implementation
│   │   │   └── api/
│   │   ├── components/
│   │   └── pages/
│   └── public/
├── pkg/                   # Shared Go packages
│   ├── jwt/               # JWT validation
│   └── proto/             # gRPC definitions
└── .github/
    └── workflows/         # CI/CD pipelines
```

## Encryption Flow

1. **Registration:**
   - User enters password
   - Client derives private key: `PBKDF2(password, 1M iterations)`
   - Client generates public key: `secp256k1.getPublicKey(privateKey)`
   - Server stores public key (never sees private key)

2. **Starting a chat:**
   - Client computes shared secret: `ECDH(myPrivateKey, theirPublicKey)`
   - Derives AES-256 key: `SHA256(sharedSecret)`
   - Key stored in memory only (never sent to server)

3. **Sending a message:**
   - Client encrypts: `AES-256-GCM(message, chatKey, nonce)`
   - Sends `{ciphertext, nonce}` to server
   - Server stores encrypted data (can't decrypt)

4. **Receiving a message:**
   - Client receives `{ciphertext, nonce}`
   - Decrypts: `AES-256-GCM.decrypt(ciphertext, chatKey, nonce)`

## Testing

The project has comprehensive unit test coverage for critical components:

- **message-service** - Service layer, outbox worker, logging
- **auth-service** - Registration, login, password change, JWT validation
- **pkg/jwt** - Token validation, security checks

**Run all tests:**
```bash
# Backend
go test ./... -v -race -coverprofile=coverage.out

# Frontend
npm test
```

## CI/CD

GitHub Actions automatically runs on every push/PR:

- ✅ **Backend Tests** - All Go services unit tests
- ✅ **Frontend Tests** - ESLint, TypeScript, build
- ✅ **Docker Build** - Verify all images build
- ✅ **Code Quality** - golangci-lint, dependency checks
- ✅ **Integration** - docker-compose validation, security checks

See [CI Documentation](.github/CI.md) for details.

## Environment Variables

Copy `.env.example` to `.env` and configure:

```bash
# PostgreSQL
POSTGRES_USER=zeromes
POSTGRES_PASSWORD=<generate-strong-password>

# Redis
REDIS_PASSWORD=<generate-strong-password>

# JWT Secrets (use: openssl rand -hex 32)
JWT_ACCESS_SECRET=<random-32-byte-hex>
JWT_REFRESH_SECRET=<random-32-byte-hex>

# Kafka
KAFKA_CLUSTER_ID=<random-uuid>
```

## Security

- ✅ End-to-end encryption (AES-256-GCM)
- ✅ ECDH key exchange (secp256k1)
- ✅ Password hashing (bcrypt + server salt)
- ✅ JWT with access/refresh tokens
- ✅ Token blacklist in Redis
- ✅ HTTPS only (Caddy auto-TLS)
- ✅ No plaintext storage
- ✅ Algorithm confusion protection

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Make changes and add tests
4. Run tests: `go test ./...` and `npm test`
5. Commit: `git commit -m "Add my feature"`
6. Push: `git push origin feature/my-feature`
7. Open a Pull Request

CI will automatically run tests on your PR.

## License

MIT

## Author

Stepan Lozovoy ([@666Stepan66612](https://github.com/666Stepan66612))
