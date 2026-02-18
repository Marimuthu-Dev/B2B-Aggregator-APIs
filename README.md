# B2B Aggregator APIs

Go API for the B2B Diagnostic Aggregator. Built with Gin, GORM, and SQL Server.

## Documentation

All documentation is in the **[docs/](docs/)** folder:

| Document | Description |
|----------|-------------|
| [docs/README.md](docs/README.md) | Documentation index and project info. |
| [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md) | **Setup and deployment to Azure** (WSL, Docker, Azure CLI, container and binary deploy, troubleshooting). |
| [docs/GoLang_Technical_Handbook.md](docs/GoLang_Technical_Handbook.md) | Go language and architecture guide for this project. |

## Quick start

- **Run locally:** From `src/`: `go run ./cmd/api` (ensure `.env` or env vars are set).
- **Update dependencies:** From `src/`: `go get -u ./...` then `go mod tidy`.
- **Deploy to Azure:** See [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md).

## Project layout

- `src/` – Go module (cmd, internal, pkg)
- `scripts/` – Deploy and utility scripts (e.g. `deploy-container.sh`)
- `docs/` – All markdown documentation
- `Dockerfile` – Container build for Azure App Service
