# CI/CD Documentation

## GitHub Actions Workflows

This project uses GitHub Actions for continuous integration. All workflows are located in `.github/workflows/`.

### Workflows

#### 1. Backend Tests (`backend.yml`)
Runs on every push to `main`/`develop` and on pull requests.

**Jobs:**
- `test-auth-service` - Auth service unit tests + build
- `test-message-service` - Message service unit tests + build
- `test-realtime-service` - Realtime service build check
- `test-api-gateway` - API gateway build check
- `test-pkg` - Shared package (JWT) tests

**What it checks:**
- ✅ All unit tests pass
- ✅ Code compiles without errors
- ✅ Code is properly formatted (`gofmt`)
- ✅ Test coverage report

**Triggers:**
- Push to `main` or `develop`
- Pull request to `main`
- Changes in Go service directories

#### 2. Frontend Tests (`frontend.yml`)
Runs on every push/PR that touches frontend code.

**What it checks:**
- ✅ ESLint passes (no linting errors)
- ✅ TypeScript compiles without errors
- ✅ Production build succeeds
- ✅ Build size report

**Triggers:**
- Push to `main` or `develop`
- Pull request to `main`
- Changes in `frontend/` directory

#### 3. Docker Build (`docker.yml`)
Verifies all Docker images build successfully.

**What it checks:**
- ✅ All backend service Dockerfiles build
- ✅ Frontend Dockerfile builds
- ✅ Uses build cache for faster builds

**Triggers:**
- Push to `main`
- Pull request to `main`

#### 4. Code Quality (`code-quality.yml`)
Runs linters and dependency checks on pull requests.

**What it checks:**
- ✅ golangci-lint passes
- ✅ Go dependencies are up to date
- ✅ No npm vulnerabilities

**Triggers:**
- Pull request to `main`

#### 5. Integration Check (`integration.yml`)
Runs additional checks on pull requests.

**What it checks:**
- ✅ `docker-compose.yml` syntax is valid
- ✅ Database migrations exist
- ✅ Proto files are present
- ✅ No hardcoded secrets in code
- ✅ `.env.example` exists

**Triggers:**
- Pull request to `main`

## Running Tests Locally

### Backend (Go)
```bash
# Auth service
cd auth-service
go test -v -race ./...

# Message service
cd message-service
go test -v -race ./...

# JWT package
cd pkg/jwt
go test -v ./...
```

### Frontend
```bash
cd frontend
npm install
npm run lint
npm run build
```

### Docker
```bash
# Build all services
docker compose build

# Or build specific service
docker compose build auth-service
```

## CI Status Badges

Add to README.md:
```markdown
![Backend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/backend.yml/badge.svg)
![Frontend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/frontend.yml/badge.svg)
![Docker Build](https://github.com/666Stepan66612/ZeroMes/actions/workflows/docker.yml/badge.svg)
```

## Troubleshooting

### Tests fail locally but pass in CI
- Check Go version: `go version` (should be 1.23)
- Check Node version: `node --version` (should be 20.x)
- Clean cache: `go clean -cache` or `npm ci`

### Tests pass locally but fail in CI
- Race conditions: CI runs with `-race` flag
- Missing dependencies in `go.mod` or `package.json`
- Environment-specific issues

### Docker build fails
- Check Dockerfile syntax
- Verify all files referenced in Dockerfile exist
- Check `.dockerignore` isn't excluding required files

## Coverage Reports

Test coverage is reported in CI logs:
```
auth-service/internal/auth/service/service.go:26:    Register        85.7%
message-service/internal/messaging/service/service.go:31:    SendMessage     92.3%
```

To generate coverage HTML locally:
```bash
cd message-service
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

## Best Practices

1. **Always run tests before pushing:**
   ```bash
   make test  # or go test ./...
   ```

2. **Check formatting:**
   ```bash
   gofmt -w .
   ```

3. **Verify build:**
   ```bash
   go build ./...
   ```

4. **For frontend:**
   ```bash
   npm run lint
   npm run build
   ```

## Future Improvements

- [ ] Add integration tests with testcontainers
- [ ] Add E2E tests for critical flows
- [ ] Deploy preview environments for PRs
- [ ] Add performance benchmarks
- [ ] Automated security scanning (Dependabot, Snyk)
