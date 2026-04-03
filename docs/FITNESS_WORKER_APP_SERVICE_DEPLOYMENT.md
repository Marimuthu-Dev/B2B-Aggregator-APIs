# Fitness Worker Deployment to a New Azure App Service

This guide explains how to deploy `src/cmd/fitness-worker` as a separate Azure App Service.

Use a separate App Service because this worker is a long-running background process with its own runtime needs:

- it polls the database continuously,
- it connects to Azure Blob Storage,
- it renders HTML to PDF using headless Chromium,
- it should not share lifecycle or scaling settings with the API app.

## Recommended Deployment Model

Deploy the fitness worker as a **Linux App Service running a custom container**.

This is the safest option for this worker because Chromium needs OS packages that are easier to control in a container image.

Do **not** deploy this worker as a plain zip/code App Service unless you already have a proven way to install and maintain Chromium and its Linux libraries on startup.

## What This Worker Needs

From the current code, the worker:

- starts from `src/cmd/fitness-worker/main.go`,
- loads `.env` values plus worker-specific environment variables,
- reads certificate templates from `src/cmd/fitness-worker/templates`,
- needs access to the database,
- needs access to Azure Blob Storage,
- needs Chromium available in `PATH` or through `CHROMIUM_PATH`.

Relevant worker configuration comes from:

- `src/internal/config/fitness_worker.go`
- `src/.env.example`

## Step 1: Create a Worker-Specific Dockerfile

Create a new file in the repo root named `Dockerfile.fitness-worker`.

Use this image instead of the current `Dockerfile`, because the current one builds the API only and does not install Chromium.

```dockerfile
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

# Optional explicit path. The code also works if chromium is already in PATH.
ENV CHROMIUM_PATH=/usr/bin/chromium

CMD ["/app/fitness-worker"]
```

## Step 2: Why These Dependencies Are Installed

The worker uses `chromedp` and launches Chromium in headless mode to generate PDFs.

The most important packages are:

- `chromium`
- `chromium-common`
- `fontconfig`
- `fonts-dejavu-core`
- `fonts-liberation`
- `libnss3`
- `libnspr4`
- `libatk1.0-0`
- `libatk-bridge2.0-0`
- `libgbm1`
- `libgtk-3-0`
- `libx11-6`
- `libx11-xcb1`
- `libxcb1`
- `libxcomposite1`
- `libxdamage1`
- `libxfixes3`
- `libxrandr2`
- `libxshmfence1`
- `libxss1`
- `libasound2`

Without these, Chromium commonly fails at runtime with errors related to missing shared libraries, fonts, or headless browser startup.

## Step 3: Build the Worker Image Locally

Run these commands from the repo root:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .
```

## Step 4: Test the Image Locally Before Azure

Before deploying, confirm the container starts correctly.

Example:

```bash
docker run --rm \
  -e DB_SERVER="<your-db-server>" \
  -e DB_DATABASE_NAME="<your-db-name>" \
  -e DB_USER="<your-db-user>" \
  -e DB_PASSWORD="<your-db-password>" \
  -e DB_ENCRYPT=true \
  -e DB_TRUST_SERVER_CERT=false \
  -e AZURE_STORAGE_ACCOUNT="<your-storage-account>" \
  -e AZURE_STORAGE_KEY="<your-storage-key>" \
  -e AZURE_STORAGE_ENDPOINT_SUFFIX=core.windows.net \
  -e AZURE_DIAGNOSTIC_REPORTS_CONTAINER=diagnostic-reports \
  -e FITNESS_CERT_TEMPLATE_DIR=/app/templates \
  -e FITNESS_CERT_PENDING_LEAD_STATUS_ID=9 \
  -e FITNESS_CERT_DONE_LEAD_STATUS_ID=10 \
  -e FITNESS_CERT_POLL_INTERVAL_SECONDS=120 \
  -e FITNESS_CERT_BATCH_SIZE=10 \
  -e FITNESS_CERT_WORKER_RUN_ONCE=true \
  -e FITNESS_CERT_WORKER_USER_ID=1 \
  -e CHROMIUM_PATH=/usr/bin/chromium \
  fitness-worker:latest
```

Use `FITNESS_CERT_WORKER_RUN_ONCE=true` for local verification so the worker processes one batch and exits.

## Step 5: Push the Image to Azure Container Registry

Set your values first:

```bash
export ACR_NAME=umstagingacr
export ACR_RESOURCE_GROUP=um-staging-rg-appservice
export IMAGE_NAME=fitness-worker
export IMAGE_TAG=latest
```

Login to Azure and ACR:

```bash
az login
az acr login --name "$ACR_NAME"
```

Get the ACR login server:

```bash
export ACR_LOGIN_SERVER=$(az acr show --name "$ACR_NAME" --resource-group "$ACR_RESOURCE_GROUP" --query loginServer -o tsv)
```

Tag and push:

```bash
docker tag fitness-worker:latest "${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}"
docker push "${ACR_LOGIN_SERVER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

## Step 6: Create a New Azure App Service for the Worker

Create a **new** App Service instead of reusing the API app.

Recommended example names:

- App Service name: `um-staging-fitness-worker`
- Plan: reuse an existing Linux App Service Plan if sizing is enough, or create a separate one
- Runtime model: **Container**

If you are creating it from Azure Portal:

1. Open [Azure Portal](https://portal.azure.com).
2. Create a new **Web App**.
3. Choose **Publish = Container**.
4. Choose **Operating System = Linux**.
5. Select your subscription, resource group, region, and App Service Plan.
6. Set the app name, for example `um-staging-fitness-worker`.
7. In the container section, choose **Azure Container Registry**.
8. Select your registry, image `fitness-worker`, and tag `latest`.
9. Finish creation.

## Step 7: Configure Container Pull Access

If the App Service cannot pull from ACR, enable ACR admin credentials:

```bash
az acr update --name "$ACR_NAME" --resource-group "$ACR_RESOURCE_GROUP" --admin-enabled true
```

Then retrieve credentials:

```bash
az acr credential show --name "$ACR_NAME" --resource-group "$ACR_RESOURCE_GROUP"
```

Use those values in the App Service container configuration if Azure does not wire them automatically.

## Step 8: Add App Settings for the Worker

In Azure Portal:

1. Open the new worker App Service.
2. Go to **Configuration**.
3. Add the required application settings.
4. Save and allow the app to restart.

### Required Core Database Settings

```text
DB_SERVER
DB_DATABASE_NAME
DB_USER
DB_PASSWORD
DB_ENCRYPT=true
DB_TRUST_SERVER_CERT=false
```

### Required Azure Blob Settings

At minimum, add the values used by your storage configuration:

```text
AZURE_STORAGE_ACCOUNT
AZURE_STORAGE_KEY
AZURE_STORAGE_CONNECTION_STRING
AZURE_STORAGE_ENDPOINT_SUFFIX=core.windows.net
AZURE_DIAGNOSTIC_REPORTS_CONTAINER=diagnostic-reports
```

If your environment also depends on these defaults, add them too:

```text
MOU_MAX_UPLOAD_BYTES=5242880
MOU_UPLOAD_TIMEOUT_SECONDS=60
MOU_SAS_TTL_MINUTES=15
DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES=5242880
DIAGNOSTIC_REPORTS_UPLOAD_TIMEOUT_SECONDS=60
DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES=15
AZURE_LEGAL_CONTAINER_NAME=legal-documents
```

### Required Worker Settings

```text
FITNESS_CERT_TEMPLATE_DIR=/app/templates
FITNESS_CERT_PENDING_LEAD_STATUS_ID=9
FITNESS_CERT_DONE_LEAD_STATUS_ID=10
FITNESS_CERT_POLL_INTERVAL_SECONDS=120
FITNESS_CERT_BATCH_SIZE=10
FITNESS_CERT_WORKER_RUN_ONCE=false
FITNESS_CERT_WORKER_USER_ID=<required positive integer>
CHROMIUM_PATH=/usr/bin/chromium
```

Important notes:

- `FITNESS_CERT_WORKER_USER_ID` is mandatory. The worker fails on startup if it is missing or invalid.
- `FITNESS_CERT_TEMPLATE_DIR` should point to the folder copied into the image.
- `CHROMIUM_PATH` should match the installed Chromium binary path in the container.

## Step 9: App Service Settings Recommended for Background Workers

Set these in the worker App Service:

1. Turn on **Always On**.
2. Keep it as a **single dedicated worker app**, separate from the API app.
3. Start with **1 instance** unless you intentionally want parallel worker execution.

Important:

- this worker polls the database in a loop,
- running multiple instances may process the same work unless your database rules fully prevent overlap,
- if you want one logical worker only, keep scale-out at `1`.

## Step 10: Verify Startup in Azure

After deployment:

1. Open the worker App Service in Azure Portal.
2. Open **Log stream**.
3. Restart the app once.
4. Watch logs for startup errors.

Look for failures such as:

- invalid `FITNESS_CERT_WORKER_USER_ID`
- database connection failure
- Azure Blob authentication failure
- Chromium executable not found
- missing shared libraries for Chromium
- template directory not found

## Step 11: Safe First Run

For the first production-style validation, you can temporarily set:

```text
FITNESS_CERT_WORKER_RUN_ONCE=true
```

Then restart the App Service.

This lets the worker run one batch and exit, which is safer for first-time verification.

Once you confirm logs and output are correct, set it back to:

```text
FITNESS_CERT_WORKER_RUN_ONCE=false
```

and restart again so it runs continuously.

## Step 12: Deploy Future Updates

When the worker code changes:

1. Rebuild the worker image.
2. Push the new image to ACR.
3. Restart the worker App Service.

Example:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
docker build -f Dockerfile.fitness-worker -t fitness-worker:latest .
docker tag fitness-worker:latest "${ACR_LOGIN_SERVER}/fitness-worker:latest"
docker push "${ACR_LOGIN_SERVER}/fitness-worker:latest"
az webapp restart --name um-staging-fitness-worker --resource-group um-staging-rg-appservice
```

## Troubleshooting

### Chromium Not Found

If logs show Chromium cannot be found:

- confirm the image installed `chromium`,
- confirm `CHROMIUM_PATH=/usr/bin/chromium`,
- open a shell in the container and verify the binary path.

### Shared Library Errors

If Chromium fails with missing `.so` library errors:

- rebuild the image with the dependency list from Step 1,
- do not remove the font and X11-related libraries,
- redeploy the container.

### Template Path Errors

If logs show template file errors:

- confirm `src/cmd/fitness-worker/templates` was copied into the image,
- confirm `FITNESS_CERT_TEMPLATE_DIR=/app/templates`,
- confirm files like `certificate_1.html` and `certificate_2.html` exist in that path.

### Worker Exits Immediately

Possible reasons:

- `FITNESS_CERT_WORKER_RUN_ONCE=true`
- invalid `FITNESS_CERT_WORKER_USER_ID`
- failed DB connection
- failed Azure Blob initialization

### Multiple Certificates Generated Unexpectedly

If you scale the App Service to more than one instance, more than one worker process may poll at the same time.

Keep the worker App Service at **1 instance** unless you intentionally design for multi-instance processing.

## Minimal Settings Checklist

Before considering the deployment complete, verify all of these are true:

- `Dockerfile.fitness-worker` exists and builds successfully
- the image includes Chromium and its Linux libraries
- templates are copied to `/app/templates`
- `CHROMIUM_PATH=/usr/bin/chromium`
- `FITNESS_CERT_TEMPLATE_DIR=/app/templates`
- `FITNESS_CERT_WORKER_USER_ID` is set
- DB settings are present
- Azure Blob settings are present
- the worker App Service is separate from the API App Service
- **Always On** is enabled
- scale-out is set to `1` unless intentionally changed
