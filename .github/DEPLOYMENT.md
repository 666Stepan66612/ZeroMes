# Deployment Guide

## CI/CD Pipeline Overview

After each push to GitHub, checks run automatically:

```
Push to GitHub
     ↓
┌────────────────────────────────────────┐
│  GitHub Actions (Parallel Execution)   │
├────────────────────────────────────────┤
│ ✅ Backend Tests (4 jobs)              │
│    - auth-service tests                │
│    - message-service tests             │
│    - realtime-service build            │
│    - api-gateway build                 │
│                                        │
│ ✅ Frontend Tests                      │
│    - ESLint                            │
│    - TypeScript check                  │
│    - Production build                  │
│                                        │
│ ✅ Docker Build (5 images)             │
│                                        │
│ ✅ Code Quality                        │
│    - golangci-lint                     │
│    - Dependency checks                 │
│                                        │
│ ✅ Integration Checks                  │
│    - docker-compose validation         │
│    - Security scan                     │
└────────────────────────────────────────┘
     ↓
All checks pass ✅
     ↓
Ready to merge/deploy
```

## Workflows Explained

### 1. backend.yml
**Runs on:** Changes to Go code  
**Duration:** ~3-5 minutes  
**What it does:**
- Installs Go 1.23
- Downloads dependencies (with caching)
- Runs tests with race detector
- Checks code formatting
- Builds binaries

**Example output:**
```
✅ test-message-service
   Running tests...
   PASS: TestSendMessage_Success (0.01s)
   PASS: TestGetMessages_Forbidden (0.00s)
   ...
   coverage: 87.3% of statements
   ok      message-service/internal/messaging/service      2.145s
```

### 2. frontend.yml
**Runs on:** Changes to frontend code  
**Duration:** ~2-3 minutes  
**What it does:**
- Installs Node.js 20
- Runs ESLint
- Checks TypeScript
- Builds production bundle
- Shows bundle size

**Example output:**
```
✅ Frontend Build & Lint
   ESLint: 0 errors, 0 warnings
   TypeScript: No errors
   Build output: 1.2MB
   dist/assets/index-a1b2c3d4.js  450KB
   dist/assets/index-e5f6g7h8.css 120KB
```

### 3. docker.yml
**Runs on:** Push to main or PR  
**Duration:** ~5-8 minutes  
**What it does:**
- Builds Docker images for all services
- Uses layer caching for speed
- Verifies images build without errors

### 4. code-quality.yml
**Runs on:** Pull request  
**Duration:** ~2-4 minutes  
**What it does:**
- Runs golangci-lint (checks code style)
- Verifies `go.mod` consistency
- Scans npm packages for vulnerabilities

### 5. integration.yml
**Runs on:** Pull request  
**Duration:** ~30 seconds  
**What it does:**
- Checks docker-compose.yml syntax
- Verifies migrations exist
- Scans code for hardcoded secrets
- Checks .env.example exists

## How to Use CI

### Local Development

**Before commit:**
```bash
# Run tests
cd message-service
go test ./...

# Check formatting
gofmt -w .

# Check frontend
cd frontend
npm run lint
npm run build
```

**Creating a PR:**
1. Create branch: `git checkout -b feature/my-feature`
2. Make changes
3. Commit: `git commit -m "Add feature"`
4. Push: `git push origin feature/my-feature`
5. Open PR on GitHub

**GitHub automatically:**
- Runs all checks
- Shows status in PR:
  ```
  ✅ Backend Tests — Passed in 3m 45s
  ✅ Frontend Tests — Passed in 2m 10s
  ✅ Docker Build — Passed in 6m 30s
  ✅ Code Quality — Passed in 1m 50s
  ✅ Integration Check — Passed in 25s
  ```

### If Tests Fail

**Example error:**
```
❌ test-message-service
   FAIL: TestGetMessages_Forbidden (0.00s)
       Expected error, got nil
```

**What to do:**
1. Check logs in GitHub Actions
2. Reproduce locally:
   ```bash
   cd message-service
   go test -v ./internal/messaging/service -run TestGetMessages_Forbidden
   ```
3. Fix the code
4. Push fix — CI runs automatically

## Production Deployment

### Manual Deployment

```bash
# 1. Pull latest code
git pull origin main

# 2. Build frontend
cd frontend
npm run build

# 3. Rebuild and restart services
cd ..
docker compose down
docker compose up -d --build

# 4. Check health
docker compose ps
docker compose logs -f
```

### Automated Deployment (Future)

You can add automatic deployment after successful merge to main:

```yaml
# .github/workflows/deploy.yml
name: Deploy to Production

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - name: Deploy to server
      uses: appleboy/ssh-action@master
      with:
        host: ${{ secrets.SERVER_HOST }}
        username: ${{ secrets.SERVER_USER }}
        key: ${{ secrets.SSH_PRIVATE_KEY }}
        script: |
          cd /opt/zeromes
          git pull origin main
          docker compose up -d --build
```

## Monitoring CI

### GitHub Actions Dashboard
https://github.com/666Stepan66612/ZeroMes/actions

Here you can see:
- All workflow runs
- Execution time
- Logs for each step
- Success/failure history

### Status Badges

README.md displays badges:
- ![Backend Tests](https://github.com/666Stepan66612/ZeroMes/actions/workflows/backend.yml/badge.svg)
- Green = all tests passed
- Red = errors found

## Troubleshooting

### "Tests pass locally but fail in CI"

**Cause:** Race conditions  
**Solution:** Run locally with `-race`:
```bash
go test -race ./...
```

### "Docker build fails in CI"

**Cause:** Missing file in build context  
**Solution:** Check `.dockerignore`:
```bash
# Make sure required files aren't excluded
cat .dockerignore
```

### "Frontend build fails with memory error"

**Cause:** Not enough memory for Vite  
**Solution:** Add to workflow:
```yaml
- name: Build
  run: NODE_OPTIONS="--max-old-space-size=4096" npm run build
```

### "golangci-lint timeout"

**Cause:** Check takes too long  
**Solution:** Increase timeout in workflow:
```yaml
args: --timeout=10m
```

## Best Practices

1. **Always run tests locally before push**
   ```bash
   make test  # or go test ./...
   ```

2. **Check CI status before merge**
   - All checkmarks should be green ✅

3. **Don't ignore warnings**
   - Even if tests pass, fix warnings

4. **Write tests for new features**
   - CI will verify they work

5. **Use draft PR for WIP**
   - CI runs, but PR isn't ready to merge

## Metrics

After setting up CI you'll see improvements:

**Before CI:**
- 🐛 Bugs reached production
- ⏰ Manual testing took time
- 🤷 Unclear if code works

**After CI:**
- ✅ Bugs caught before merge
- ⚡ Automatic checks in 5 minutes
- 📊 Test coverage visible
- 🚀 Confidence in deployment

## Next Steps

1. **Add integration tests**
   - Test service interactions
   - Use testcontainers

2. **Add E2E tests**
   - Playwright for frontend
   - Test critical flows

3. **Set up auto-deploy**
   - After merge to main → deploy to staging
   - After tag → deploy to production

4. **Add monitoring**
   - Sentry for errors
   - Prometheus for metrics
   - Grafana for dashboards

5. **Security scanning**
   - Dependabot for updates
   - Snyk for vulnerabilities
   - CodeQL for static analysis
