# Pre-Push Checklist

Before pushing to GitHub, verify everything is ready:

## ✅ Files Created

### Workflows (5 files)
- [ ] `.github/workflows/backend.yml`
- [ ] `.github/workflows/frontend.yml`
- [ ] `.github/workflows/docker.yml`
- [ ] `.github/workflows/code-quality.yml`
- [ ] `.github/workflows/integration.yml`

### Documentation (6 files)
- [ ] `.github/CI.md`
- [ ] `.github/DEPLOYMENT.md`
- [ ] `.github/QUICKSTART.md`
- [ ] `.github/CHEATSHEET.md`
- [ ] `.github/SUMMARY.md`
- [ ] `.github/workflows/README.md`

### Project Files
- [ ] `README.md` (updated with badges)
- [ ] `TESTING.md` (testing guide)

## ✅ Tests Written

### message-service
- [ ] `internal/messaging/service/service_test.go` (service layer tests)
- [ ] `internal/messaging/service/slog_test.go` (logging tests)
- [ ] `internal/cores/outbox-worker/worker_test.go` (outbox worker tests)

### auth-service
- [ ] `internal/auth/service/service_test.go` (auth tests)

### pkg/jwt
- [ ] `token_service_test.go` (JWT validation tests)

## ✅ Local Verification

Run these commands to verify everything works:

```bash
# 1. Check all tests pass
cd message-service && go test ./... && cd ..
cd auth-service && go test ./... && cd ..
cd pkg/jwt && go test ./... && cd ..

# 2. Check formatting
find . -name "*.go" -not -path "*/vendor/*" | xargs gofmt -l
# Should return nothing

# 3. Check frontend
cd frontend && npm run lint && npm run build && cd ..

# 4. Verify docker-compose
docker compose config > /dev/null
# Should not error

# 5. Check for secrets
grep -r "password.*=.*\"" --include="*.go" --include="*.ts" . || echo "No secrets found"
```

## ✅ Git Status

```bash
# Check what will be committed
git status

# Should see:
# - .github/ (new directory)
# - README.md (modified)
# - TESTING.md (new file)
# - Test files (new)
```

## ✅ Ready to Push

If all checks pass:

```bash
# Stage files
git add .github/ README.md TESTING.md
git add message-service/internal/messaging/service/*_test.go
git add message-service/internal/cores/outbox-worker/worker_test.go
git add auth-service/internal/auth/service/service_test.go
git add pkg/jwt/token_service_test.go

# Commit
git commit -m "Add CI/CD pipelines and comprehensive unit tests

- Add GitHub Actions workflows (backend, frontend, docker, code-quality, integration)
- Add comprehensive documentation (CI.md, DEPLOYMENT.md, TESTING.md)
- Add unit tests for message-service (service layer, outbox worker, logging)
- Add unit tests for auth-service
- Add unit tests for pkg/jwt
- Update README with CI badges and testing info"

# Push
git push origin main
```

## ✅ After Push

1. Go to: https://github.com/666Stepan66612/ZeroMes/actions
2. Watch workflows run (should take ~5-10 minutes total)
3. Verify all pass with green checkmarks ✅
4. Check README badges are green

## 🐛 If Something Fails

### Backend Tests Fail
```bash
# Check logs in GitHub Actions
# Reproduce locally:
cd message-service
go test -v -race ./...
```

### Frontend Build Fails
```bash
cd frontend
npm run build
# Fix errors and push again
```

### Docker Build Fails
```bash
docker compose build
# Check Dockerfile and .dockerignore
```

## 📊 Expected CI Results

After successful push:
```
✅ Backend Tests (auth-service)      1m 30s
✅ Backend Tests (message-service)   2m 15s
✅ Backend Tests (realtime-service)  45s
✅ Backend Tests (api-gateway)       50s
✅ Backend Tests (pkg)               30s
✅ Frontend Build & Lint             2m 10s
✅ Docker Build (all services)       6m 30s
✅ Code Quality                      1m 50s
✅ Integration Check                 25s
```

## 🎉 Success!

Once all workflows pass:
- ✅ CI/CD is fully operational
- ✅ Tests run automatically on every push
- ✅ Code quality is enforced
- ✅ Team can develop with confidence

---

**Last updated:** 2026-05-08
