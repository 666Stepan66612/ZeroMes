# GitHub Actions Workflows

This directory contains CI/CD workflows for ZeroMes project.

## Workflows

| Workflow | Trigger | Duration | Purpose |
|----------|---------|----------|---------|
| `backend.yml` | Push/PR to main | ~3-5 min | Run Go tests, check formatting, build |
| `frontend.yml` | Push/PR to main | ~2-3 min | ESLint, TypeScript, build frontend |
| `docker.yml` | Push/PR to main | ~5-8 min | Build all Docker images |
| `code-quality.yml` | PR to main | ~2-4 min | Linting, dependency checks |
| `integration.yml` | PR to main | ~30 sec | Validate configs, check secrets |

## Status

View workflow runs: https://github.com/666Stepan66612/ZeroMes/actions

## Documentation

- [CI Overview](../CI.md) - How CI works
- [Deployment Guide](../DEPLOYMENT.md) - How to deploy
