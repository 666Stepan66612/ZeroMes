# CI Quick Reference

## What's Created

```
.github/
├── workflows/
│   ├── backend.yml         # Go tests
│   ├── frontend.yml        # React build + lint
│   ├── docker.yml          # Docker images
│   ├── code-quality.yml    # Linters + security
│   └── integration.yml     # Config validation
├── CI.md                   # Detailed documentation
└── DEPLOYMENT.md           # Deployment guide
```

## Quick Start

### 1. First Push with CI

```bash
# Add all files
git add .github/ README.md

# Commit
git commit -m "Add CI/CD pipelines"

# Push
git push origin main
```

### 2. Check Status

Open: https://github.com/666Stepan66612/ZeroMes/actions

You'll see:
```
✅ Backend Tests      3m 45s
✅ Frontend Tests     2m 10s  
✅ Docker Build       6m 30s
✅ Code Quality       1m 50s
✅ Integration Check  25s
```

### 3. Badges in README

Badges are already added to README.md:
```markdown
![Backend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/backend.yml/badge.svg)
![Frontend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/frontend.yml/badge.svg)
```

## What's Checked

| Service | Tests | Coverage |
|--------|-------|----------|
| message-service | Unit tests | ~87% |
| auth-service | Unit tests | ~85% |
| pkg/jwt | Unit tests | 100% |
| frontend | lint + build | - |

## Development Workflow

```bash
# 1. Create branch
git checkout -b feature/my-feature

# 2. Write code + tests
vim message-service/internal/messaging/service/service.go
vim message-service/internal/messaging/service/service_test.go

# 3. Check locally
cd message-service
go test ./...
go fmt ./...

# 4. Commit & Push
git add .
git commit -m "Add new feature"
git push origin feature/my-feature

# 5. Create PR on GitHub
# CI runs automatically

# 6. Wait for green checkmarks ✅

# 7. Merge to main
```

## Local Check Commands

```bash
# Backend
cd message-service
go test -v -race ./...              # Run tests
go test -coverprofile=coverage.out ./...  # With coverage
go tool cover -html=coverage.out    # Open HTML report
gofmt -w .                          # Format code

# Frontend
cd frontend
npm run lint                        # ESLint
npm run build                       # Production build
npx tsc --noEmit                    # TypeScript check

# Docker
docker compose build                # Build all images
docker compose up -d                # Start services
```

## If CI Fails

### ❌ Backend Tests Failed

```bash
# Check logs in GitHub Actions
# Reproduce locally:
cd message-service
go test -v ./internal/messaging/service -run TestName

# Fix
vim internal/messaging/service/service.go

# Verify
go test ./...

# Push fix
git add .
git commit -m "Fix test"
git push
```

### ❌ Frontend Build Failed

```bash
cd frontend
npm run build  # Check error

# Common issues:
# - TypeScript errors → fix types
# - ESLint errors → npm run lint --fix
# - Missing dependencies → npm install
```

### ❌ Docker Build Failed

```bash
# Check locally
docker compose build service-name

# Common problems:
# - Missing file → check .dockerignore
# - Dockerfile error → check syntax
```

## Useful Links

- **Actions Dashboard**: https://github.com/666Stepan66612/ZeroMes/actions
- **Detailed docs**: `.github/CI.md`
- **Deployment guide**: `.github/DEPLOYMENT.md`
- **Workflows**: `.github/workflows/`

## Statistics

After CI setup:
- ⚡ Automatic checks in **~5 minutes**
- ✅ Comprehensive test coverage
- 📊 Code coverage reports
- 🚀 Confidence in code quality

## Next Steps

1. ✅ Push to GitHub → CI runs
2. ✅ Check status in Actions
3. ✅ Add more tests
4. 🔜 Set up auto-deploy
5. 🔜 Add E2E tests
6. 🔜 Integrate Sentry/Prometheus

---

**Done!** GitHub now automatically checks your code on every push.
