# B2B Aggregator APIs – Documentation

This folder contains all project documentation.

## Documentation index

- **DEPLOYMENT.md** – Setup and deployment to Azure (WSL, Docker, Azure CLI, ACR, container deploy, Application settings, binary/zip deploy, troubleshooting). Use this for deploying the Go API to Azure App Service.
- **GoLang_Technical_Handbook.md** – Go language and architecture guide for the project (concurrency, patterns, testing, performance).
- **SONARQUBE.md** – How to run the SonarQube (or SonarCloud) scanner and optional coverage for the Go project.

## Local setup and environment variables

Key parameters are read from environment variables (and from a `.env` file when present). Copy `src/.env.example` to `src/.env` and set the required values (DB_*, JWT_SECRET, LOGIN_ENC_KEY, LOGIN_ENC_SALT). Never commit `.env` with real secrets.

## Update dependencies

Run these from the `src/` folder (project root is the parent of `src/`):

```bash
cd src
go get -u ./...
go mod tidy
```

## Quick links

- Deploy (container): See DEPLOYMENT.md Part 5 and Quick reference.
- Application settings: See DEPLOYMENT.md Part 6.
