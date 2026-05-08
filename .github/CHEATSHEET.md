# CI/CD Cheat Sheet

## 🚀 Quick Commands

### Run Tests Locally
```bash
# All backend tests
for service in auth-service message-service realtime-service api-gateway; do
  cd $service && go test ./... && cd ..
done

# Specific service
cd message-service && go test -v -race ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Frontend
cd frontend && npm run lint && npm run build
```

### Check Before Push
```bash
# Format Go code
find . -name "*.go" -not -path "*/vendor/*" | xargs gofmt -w

# Verify builds
docker compose build

# Check for secrets
grep -r "password.*=.*\"" --include="*.go" --include="*.ts" .
```

## 📊 CI Workflows

| Workflow | When | Time | What |
|----------|------|------|------|
| backend.yml | Push/PR | 3-5m | Tests + build |
| frontend.yml | Push/PR | 2-3m | Lint + build |
| docker.yml | Push/PR | 5-8m | Image builds |
| code-quality.yml | PR only | 2-4m | Linters |
| integration.yml | PR only | 30s | Validation |

## ✅ PR Checklist

Before creating PR:
- [ ] Tests pass locally: `go test ./...`
- [ ] Code formatted: `gofmt -w .`
- [ ] No linting errors: `npm run lint`
- [ ] Builds successfully: `docker compose build`
- [ ] Commit message is clear

After creating PR:
- [ ] All CI checks pass (green ✅)
- [ ] No merge conflicts
- [ ] Code reviewed
- [ ] Ready to merge

## 🐛 Common Issues

### Race Condition Detected
```bash
# Run with race detector
go test -race ./...

# Fix: Add proper mutex locks
```

### Import Cycle
```bash
# Check dependencies
go mod graph | grep your-package

# Fix: Refactor to break cycle
```

### Docker Build Fails
```bash
# Check locally
docker compose build service-name

# Common fixes:
# - Add missing files
# - Update .dockerignore
# - Check Dockerfile syntax
```

### ESLint Errors
```bash
# Auto-fix
npm run lint --fix

# Check specific file
npx eslint src/path/to/file.ts
```

## 📈 Coverage Goals

| Component | Target | Current |
|-----------|--------|---------|
| message-service | >85% | ~87% |
| auth-service | >80% | ~85% |
| pkg/jwt | 100% | 100% |

## 🔗 Links

- Actions: https://github.com/666Stepan66612/ZeroMes/actions
- Docs: `.github/CI.md`
- Deploy: `.github/DEPLOYMENT.md`

## 💡 Tips

1. **Use draft PRs** for work in progress
2. **Run tests locally** before pushing
3. **Check CI logs** if tests fail
4. **Keep PRs small** for faster review
5. **Write descriptive commits**

## 🎯 Goals

- ✅ All tests pass
- ✅ No linting errors
- ✅ Code coverage >80%
- ✅ Docker builds succeed
- ✅ No security issues
