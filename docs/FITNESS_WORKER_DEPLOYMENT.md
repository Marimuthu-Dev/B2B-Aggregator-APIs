# B2B Fitness Worker – Setup and Deployment to Azure

**Audience:** Developers (including juniors) deploying the Go fitness worker to Azure App Service.

This guide is written to be as **copy/paste ready** as possible using the staging values already used in this repo:

- **App Service:** `um-staging-fitness-worker`
- **ACR:** `umstagingacr`
- **Resource group:** `um-staging-rg-appservice`
- **Location:** `centralindia`
- **Image name:** `fitness-worker`

**Primary path:** Container deployment using WSL and Azure Container Registry.

---

## Table of contents

1. [Prerequisites](#prerequisites)
2. [Part 1: Docker in WSL](#part-1-docker-in-wsl)
3. [Part 2: Azure CLI (`az`) in WSL](#part-2-azure-cli-az-in-wsl)
4. [Part 3: Log in to Azure](#part-3-log-in-to-azure)
5. [Part 4: Create Dockerfile for the worker](#part-4-create-dockerfile-for-the-worker)
6. [Part 5: Build and push the worker container](#part-5-build-and-push-the-worker-container)
7. [Part 6: Create the new Azure App Service](#part-6-create-the-new-azure-app-service)
8. [Part 7: Configure Application settings in Azure Portal](#part-7-configure-application-settings-in-azure-portal)
9. [Part 8: Verify deployment](#part-8-verify-deployment)
10. [Part 9: Deploying updates later](#part-9-deploying-updates-later)
11. [Troubleshooting](#troubleshooting)
12. [Quick reference](#quick-reference)
13. [Deployment checklist (quick)](#deployment-checklist-quick)

---

## Prerequisites

- **Windows PC** with **WSL** for the easiest copy/paste flow.
- **Docker** installed and working in WSL.
- **Azure CLI** installed and working in WSL.
- **Azure subscription** access to the resource group `um-staging-rg-appservice`.
- **Repo root** = the folder that contains `Dockerfile`, `docs/`, and `src/`.

This worker is deployed separately from the API because:

- it is a long-running background process,
- it polls the database continuously,
- it needs Chromium for HTML-to-PDF generation,
- it should not share the API app's lifecycle.

---

## Part 1: Docker in WSL

### 1.1 Open WSL

```bash
wsl
```

### 1.2 Check Docker

```bash
docker --version
docker run hello-world
```

- If both commands succeed, continue to [Part 2](#part-2-azure-cli-az-in-wsl).
- If Docker is missing, install Docker Desktop with WSL integration or install Docker Engine inside WSL.

### 1.3 If Docker says permission denied

```bash
sudo usermod -aG docker $USER
```

Then close WSL, open it again, and run:

```bash
docker run hello-world
```

---

## Part 2: Azure CLI (`az`) in WSL

### 2.1 Install Azure CLI

```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

Verify:

```bash
az --version
```

### 2.2 If the install script fails

```bash
sudo apt-get update
sudo apt-get install -y ca-certificates curl apt-transport-https lsb-release gnupg
curl -sL https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor | sudo tee /usr/share/keyrings/microsoft-prod.gpg > /dev/null
echo "deb [arch=amd64 signed-by=/usr/share/keyrings/microsoft-prod.gpg] https://packages.microsoft.com/repos/azure-cli/ $(lsb_release -cs) main" | sudo tee /etc/apt/sources.list.d/azure-cli.list
sudo apt-get update
sudo apt-get install -y azure-cli
az --version
```

---

## Part 3: Log in to Azure

### 3.1 Sign in

```bash
az login
```

### 3.2 Choose the correct subscription if needed

```bash
az account list --output table
az account set --subscription "Azure subscription 1"
```

Replace `"Azure subscription 1"` with the real subscription name or ID if needed.

### 3.3 Confirm ACR is visible

```bash
az acr list --resource-group um-staging-rg-appservice --output table
```

You should see `umstagingacr`.

---

## Part 4: Create Dockerfile for the worker

The current repo `Dockerfile` builds only the API.  
For the worker, create a new file in the repo root named `Dockerfile.fitness-worker`.

### 4.1 Go to the repo root

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
```

### 4.2 Create `Dockerfile.fitness-worker`

Run this exact command:

```bash
cat > Dockerfile.fitness-worker <<'EOF'
FROM golang:1.25-bookworm AS builder

WORKDIR /app

COPY src/go.mod src/go.sum ./
RUN go mod download

COPY src/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /fitness-worker ./cmd/fitness-worker

FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    chromium \
    chromium-common \
    fonts-dejavu-core \
    fonts-liberation \
    fontconfig \
    libasound2 \
    libatk-bridge2.0-0 \
    libatk1.0-0 \
    libc6 \
    libcairo2 \
    libcups2 \
    libdbus-1-3 \
    libdrm2 \
    libexpat1 \
    libfontconfig1 \
    libgbm1 \
    libgcc-s1 \
    libglib2.0-0 \
    libgtk-3-0 \
    libnspr4 \
    libnss3 \
    libpango-1.0-0 \
    libpangocairo-1.0-0 \
    libstdc++6 \
    libx11-6 \
    libx11-xcb1 \
    libxcb1 \
    libxcomposite1 \
    libxcursor1 \
    libxdamage1 \
    libxext6 \
    libxfixes3 \
    libxi6 \
    libxrandr2 \
    libxrender1 \
    libxshmfence1 \
    libxss1 \
    libxtst6 \
    xdg-utils \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /app

COPY --from=builder /fitness-worker /app/fitness-worker
COPY src/cmd/fitness-worker/templates /app/templates

ENV CHROMIUM_PATH=/usr/bin/chromium

CMD ["/app/fitness-worker"]
EOF
```

### 4.3 Why this Dockerfile is needed

This worker uses headless Chromium through `chromedp` to generate PDFs.

The installed dependencies include:

- `chromium`
- `chromium-common`
- fonts packages
- `fontconfig`
- required Chromium shared libraries

Without these, PDF generation usually fails in Azure with missing library or browser startup errors.

---

## Part 5: Build and push the worker container

All commands below are run from the repo root.

### 5.1 Go to the repo root

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
```

### 5.2 Set the exact deployment values

```bash
export ACR_NAME=umstagingacr
export RESOURCE_GROUP=um-staging-rg-appservice
export APP_NAME=um-staging-fitness-worker
export IMAGE_NAME=fitness-worker
export IMAGE_TAG=latest
```

### 5.3 Enable ACR admin user if needed

```bash
az acr update --name umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true
```

### 5.4 Build the worker image

```bash
docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .
```

### 5.5 Log in to ACR

```bash
az acr login --name umstagingacr
```

### 5.6 Get the ACR login server

```bash
export ACR_LOGIN_SERVER=$(az acr show --name umstagingacr --resource-group um-staging-rg-appservice --query loginServer -o tsv)
echo "$ACR_LOGIN_SERVER"
```

Expected output is usually:

```text
umstagingacr.azurecr.io
```

### 5.7 Tag and push the image

```bash
docker tag fitness-worker:latest "${ACR_LOGIN_SERVER}/fitness-worker:latest"
docker push "${ACR_LOGIN_SERVER}/fitness-worker:latest"
```

---

## Part 6: Create the new Azure App Service

Create a **new** App Service for the worker. Do not reuse `um-staging-api`.

### 6.1 Create from Azure Portal

1. Open [Azure Portal](https://portal.azure.com).
2. Go to **Create a resource** → **Web App**.
3. Set **Resource Group** = `um-staging-rg-appservice`.
4. Set **Name** = `um-staging-fitness-worker`.
5. Set **Publish** = `Container`.
6. Set **Operating System** = `Linux`.
7. Set **Region** = `Central India`.
8. Choose an existing Linux App Service Plan or create a new Linux plan.
9. In the container section, choose:
   - **Image Source** = `Azure Container Registry`
   - **Registry** = `umstagingacr`
   - **Image** = `fitness-worker`
   - **Tag** = `latest`
10. Create the app.

### 6.2 If the container image is not pulling

Run:

```bash
az acr credential show --name umstagingacr --resource-group um-staging-rg-appservice
```

Then in Azure Portal → `um-staging-fitness-worker` → **Deployment Center** or **Container settings**, use the ACR admin username and password from that command.

### 6.3 Turn on Always On

In Azure Portal:

1. Open `um-staging-fitness-worker`
2. Go to **Configuration** or **General settings**
3. Enable **Always On**
4. Save

This worker runs continuously, so **Always On** should be enabled.

### 6.4 Keep scale-out at 1 instance

Because this worker polls the database in a loop, start with **1 instance** only.

If you scale to multiple instances, more than one worker may process the same pending work at the same time.

---

## Part 7: Configure Application settings in Azure Portal

The worker reads environment variables from Azure App Service settings.  
It does **not** use the local `src/.env` file inside Azure unless you explicitly package one, which this deployment does not do.

### 7.1 Open Configuration

1. Open [Azure Portal](https://portal.azure.com).
2. Search for `um-staging-fitness-worker`.
3. Open the App Service.
4. In the left menu, click **Configuration** or **Environment variables**.

### 7.2 Add the worker settings

Add these application settings exactly.

### Database settings

| Name | Value | Required |
|------|-------|----------|
| `DB_SERVER` | `um-staging-server-database.database.windows.net:1433` | Yes |
| `DB_DATABASE_NAME` | `um-staging-DB` | Yes |
| `DB_USER` | `dev_admin` | Yes |
| `DB_PASSWORD` | *Your real database password* | Yes |
| `DB_ENCRYPT` | `true` | Yes |
| `DB_TRUST_SERVER_CERT` | `false` | Yes |

### Azure Blob settings

| Name | Value | Required |
|------|-------|----------|
| `AZURE_STORAGE_ACCOUNT` | *Your storage account name* | Yes |
| `AZURE_STORAGE_KEY` | *Your storage key* | Yes if not using connection string |
| `AZURE_STORAGE_CONNECTION_STRING` | *Optional fallback connection string* | Optional |
| `AZURE_STORAGE_ENDPOINT_SUFFIX` | `core.windows.net` | Yes |
| `AZURE_LEGAL_CONTAINER_NAME` | `legal-documents` | Optional |
| `MOU_MAX_UPLOAD_BYTES` | `5242880` | Optional |
| `MOU_UPLOAD_TIMEOUT_SECONDS` | `60` | Optional |
| `MOU_SAS_TTL_MINUTES` | `15` | Optional |
| `AZURE_DIAGNOSTIC_REPORTS_CONTAINER` | `diagnostic-reports` | Yes |
| `DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES` | `5242880` | Optional |
| `DIAGNOSTIC_REPORTS_UPLOAD_TIMEOUT_SECONDS` | `60` | Optional |
| `DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES` | `15` | Optional |

### Fitness worker settings

| Name | Value | Required |
|------|-------|----------|
| `FITNESS_CERT_TEMPLATE_DIR` | `/app/templates` | Yes |
| `FITNESS_CERT_PENDING_LEAD_STATUS_ID` | `9` | Yes |
| `FITNESS_CERT_DONE_LEAD_STATUS_ID` | `10` | Yes |
| `FITNESS_CERT_POLL_INTERVAL_SECONDS` | `120` | Yes |
| `FITNESS_CERT_BATCH_SIZE` | `10` | Yes |
| `FITNESS_CERT_WORKER_RUN_ONCE` | `false` | Yes |
| `FITNESS_CERT_WORKER_USER_ID` | `1` | Yes, only if user ID `1` is valid in your DB |
| `CHROMIUM_PATH` | `/usr/bin/chromium` | Yes |

### 7.3 Important note for `FITNESS_CERT_WORKER_USER_ID`

The worker fails on startup if `FITNESS_CERT_WORKER_USER_ID` is missing or invalid.

If user ID `1` is not correct in your database, replace it with the correct positive integer user ID used for audit fields.

### 7.4 Save the settings

After adding the settings:

1. Click **Save**
2. Click **Continue** if Azure asks to restart the app

---

## Part 8: Verify deployment

### 8.1 Open log stream

In Azure Portal:

1. Open `um-staging-fitness-worker`
2. Open **Log stream**

### 8.2 Restart the app once

```bash
az webapp restart --name um-staging-fitness-worker --resource-group um-staging-rg-appservice
```

### 8.3 Watch for startup errors

Check for errors such as:

- invalid `FITNESS_CERT_WORKER_USER_ID`
- database connection failure
- Azure Blob initialization failure
- Chromium not found
- missing Chromium libraries
- template path not found

### 8.4 Safe first run option

For the first deployment, you can temporarily change this setting:

```text
FITNESS_CERT_WORKER_RUN_ONCE=true
```

Then restart:

```bash
az webapp restart --name um-staging-fitness-worker --resource-group um-staging-rg-appservice
```

This makes the worker process one batch and exit, which is safer for initial validation.

After verification, set it back to:

```text
FITNESS_CERT_WORKER_RUN_ONCE=false
```

and restart again.

---

## Part 9: Deploying updates later

When worker code changes:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=umstagingacr
export RESOURCE_GROUP=um-staging-rg-appservice
export APP_NAME=um-staging-fitness-worker
export IMAGE_NAME=fitness-worker
export IMAGE_TAG=latest
export ACR_LOGIN_SERVER=$(az acr show --name umstagingacr --resource-group um-staging-rg-appservice --query loginServer -o tsv)
docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .
docker tag fitness-worker:latest "${ACR_LOGIN_SERVER}/fitness-worker:latest"
docker push "${ACR_LOGIN_SERVER}/fitness-worker:latest"
az webapp restart --name um-staging-fitness-worker --resource-group um-staging-rg-appservice
```

---

## Troubleshooting

| Issue | What to do |
|-------|------------|
| `az: command not found` | Reopen WSL or run `source ~/.bashrc`, then run `az --version`. |
| Docker permission denied | Run `sudo usermod -aG docker $USER`, then reopen WSL. |
| ACR login fails | Run `az login`, then `az acr login --name umstagingacr`. |
| App Service cannot pull image | Run `az acr update --name umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true`, then use ACR admin credentials in Deployment Center. |
| Chromium not found | Confirm `Dockerfile.fitness-worker` installed `chromium` and set `CHROMIUM_PATH=/usr/bin/chromium`. |
| Missing `.so` library errors | Rebuild using the exact package list in `Dockerfile.fitness-worker`. |
| Template files not found | Confirm `COPY src/cmd/fitness-worker/templates /app/templates` exists in `Dockerfile.fitness-worker` and `FITNESS_CERT_TEMPLATE_DIR=/app/templates` is set. |
| Worker exits immediately | Check if `FITNESS_CERT_WORKER_RUN_ONCE=true`, or if startup failed due to DB / Blob / Chromium config. |
| Startup fails with worker user ID error | Set a valid positive integer in `FITNESS_CERT_WORKER_USER_ID`. |
| Duplicate processing risk | Keep App Service scale-out at `1` instance unless multi-instance processing is intentionally designed. |

---

## Quick reference

Copy-paste in WSL:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs

cat > Dockerfile.fitness-worker <<'EOF'
FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY src/go.mod src/go.sum ./
RUN go mod download
COPY src/ .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /fitness-worker ./cmd/fitness-worker
FROM debian:bookworm-slim
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates chromium chromium-common fonts-dejavu-core fonts-liberation fontconfig \
    libasound2 libatk-bridge2.0-0 libatk1.0-0 libc6 libcairo2 libcups2 libdbus-1-3 libdrm2 \
    libexpat1 libfontconfig1 libgbm1 libgcc-s1 libglib2.0-0 libgtk-3-0 libnspr4 libnss3 \
    libpango-1.0-0 libpangocairo-1.0-0 libstdc++6 libx11-6 libx11-xcb1 libxcb1 libxcomposite1 \
    libxcursor1 libxdamage1 libxext6 libxfixes3 libxi6 libxrandr2 libxrender1 libxshmfence1 \
    libxss1 libxtst6 xdg-utils && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /fitness-worker /app/fitness-worker
COPY src/cmd/fitness-worker/templates /app/templates
ENV CHROMIUM_PATH=/usr/bin/chromium
CMD ["/app/fitness-worker"]
EOF

az login
az acr update --name umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true
az acr login --name umstagingacr
export ACR_LOGIN_SERVER=$(az acr show --name umstagingacr --resource-group um-staging-rg-appservice --query loginServer -o tsv)
docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .
docker tag fitness-worker:latest "${ACR_LOGIN_SERVER}/fitness-worker:latest"
docker push "${ACR_LOGIN_SERVER}/fitness-worker:latest"
az webapp restart --name um-staging-fitness-worker --resource-group um-staging-rg-appservice
```

After that, add the worker app settings in Azure Portal for:

- DB
- Azure Blob
- `FITNESS_CERT_*`
- `CHROMIUM_PATH`

---

## Deployment checklist (quick)

1. Create `Dockerfile.fitness-worker`.
2. Build image: `docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .`
3. Push image to `umstagingacr.azurecr.io/fitness-worker:latest`.
4. Create App Service `um-staging-fitness-worker` as **Linux + Container**.
5. Point it to image `fitness-worker:latest` in `umstagingacr`.
6. Add all DB, Blob, and worker app settings.
7. Set `FITNESS_CERT_TEMPLATE_DIR=/app/templates`.
8. Set `CHROMIUM_PATH=/usr/bin/chromium`.
9. Enable **Always On**.
10. Keep scale-out at `1`.
11. Restart the app and verify in **Log stream**.
