# B2B Aggregator API – Production deployment (Med client)

**Audience:** Developers and release owners deploying the Go API to the **Med** production Azure App Service.

**Related guides**

- UrMediConnect / shared prod guide: [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md)
- Staging: [DEPLOYMENT.md](./DEPLOYMENT.md)

This guide targets the **Med** production container App Service. Replace ACR and database names with your organisation’s Med production values where noted.

| Azure resource | Name |
|----------------|------|
| **App Service** | `med-prod-api` |
| **Resource group** | `med-prod-rg-appservice` |
| **App Service plan** | `med-prod-appservice-plan-linux` |
| **Location** | **South India** (`southindia`) — App Service plan, App Service, and ACR must all use this region |
| **Publishing model** | **Container** (Linux) |
| **Public URL** | `https://med-prod-api.azurewebsites.net` |

**Production expectations**

- Deploy only from **reviewed, released** commits (tag or release branch).
- Prefer **versioned image tags** (e.g. `v1.2.3`) so you can roll back by changing the container tag.
- Treat **secrets** (`DB_PASSWORD`, `JWT_SECRET`, `LOGIN_ENC_KEY`, `LOGIN_ENC_SALT`, `AZURE_STORAGE_KEY` / connection string) as Med-production-only; do not reuse staging or other-client secrets.
- Coordinate **maintenance windows** if the API has downtime during container restarts.

**Primary path:** Create or update the Linux container App Service, then deploy via WSL (Parts 1–9). **Alternative:** Binary / Zip only if the site is not Container-based (Part 10).

---

## Table of contents

1. [Prerequisites](#prerequisites)
2. [Part 1: Docker in WSL](#part-1-docker-in-wsl)
3. [Part 2: Azure CLI (`az`) in WSL](#part-2-azure-cli-az-in-wsl)
4. [Part 3: Log in to Azure](#part-3-log-in-to-azure)
5. [Part 4: App Service plan and App Service (create or update)](#part-4-app-service-plan-and-app-service-create-or-update)
6. [Part 5: Azure Container Registry (ACR)](#part-5-azure-container-registry-acr)
7. [Part 6: Deploy the container](#part-6-deploy-the-container)
8. [Part 7: Configure Application settings](#part-7-configure-application-settings)
9. [Part 8: Verify deployment](#part-8-verify-deployment)
10. [Part 9: Deploying updates later](#part-9-deploying-updates-later)
11. [Part 10: Binary / Zip deployment (non-container)](#part-10-binary--zip-deployment-non-container)
12. [Troubleshooting](#troubleshooting)
13. [Quick reference](#quick-reference)
14. [Deployment checklist (quick)](#deployment-checklist-quick)

---

## Prerequisites

- **Windows PC** with **WSL**, Docker, and Azure CLI (or any host with Docker + `az`).
- **Project** cloned (`B2B-Aggregator-APIs`); repo root = folder with `Dockerfile` and `src/`.
- **Azure subscription** with rights on `med-prod-rg-appservice`, plan `med-prod-appservice-plan-linux`, and App Service `med-prod-api`.
- **Access** to Med production SQL, JWT secrets, login encryption values, and (if MoU / diagnostic report features are used) Azure Storage credentials.

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

- If `docker: command not found`: install Docker Engine in WSL or Docker Desktop with WSL 2 integration.

### 1.3 (If needed) Fix Docker permission

```bash
sudo usermod -aG docker $USER
```

Log out of WSL and open it again, then retry `docker run hello-world`.

---

## Part 2: Azure CLI (`az`) in WSL

### 2.1 Install Azure CLI (one-time)

```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
az --version
```

### 2.2 (Optional) Package manager install (Ubuntu/Debian)

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

```bash
az login
az account list --output table
az account set --subscription "Your production subscription"
```

Confirm the resource group:

```bash
az group show --name med-prod-rg-appservice --output table
az appservice plan list --resource-group med-prod-rg-appservice --output table
az webapp list --resource-group med-prod-rg-appservice --output table
```

You should see plan **`med-prod-appservice-plan-linux`** and/or app **`med-prod-api`** (depending on whether they already exist).

---

## Part 4: App Service plan and App Service (create or update)

Use this part when the plan or App Service does **not** exist yet, or when you need to **re-point** an existing app to the Linux container plan.

### 4.1 Create App Service plan (if missing)

```bash
az appservice plan create \
  --name med-prod-appservice-plan-linux \
  --resource-group med-prod-rg-appservice \
  --is-linux \
  --sku P1V3 \
  --location southindia
```

Med production App Services run in **South India**. Keep `--location southindia` so the plan matches the App Service region. Adjust `--sku` only if your standards differ. If the plan already exists, skip this step.

Verify:

```bash
az appservice plan show \
  --name med-prod-appservice-plan-linux \
  --resource-group med-prod-rg-appservice \
  --query "{name:name,sku:sku.name,kind:kind,reserved:reserved}" -o table
```

### 4.2 Create App Service as Linux container (if missing)

```bash
az webapp create \
  --resource-group med-prod-rg-appservice \
  --plan med-prod-appservice-plan-linux \
  --name med-prod-api \
  --deployment-container-image-name mcr.microsoft.com/appsvc/staticsite:latest
```

- `--deployment-container-image-name` is a temporary placeholder; Part 6 replaces it with your ACR image.
- App name **`med-prod-api`** must be globally unique in `*.azurewebsites.net`.

### 4.3 Update existing App Service (plan / container mode)

If **`med-prod-api` already exists** but is on the wrong plan or not configured for containers:

**Move to the Med Linux plan**

```bash
az webapp update \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --plan med-prod-appservice-plan-linux
```

**Ensure Linux container configuration** (image is set properly in Part 6):

```bash
az webapp config set \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --linux-fx-version "DOCKER|mcr.microsoft.com/appsvc/staticsite:latest"
```

**Confirm App Service ↔ plan**

```bash
az webapp show \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --query "{name:name,state:state,appServicePlanId:appServicePlanId,kind:kind,defaultHostName:defaultHostName}" -o json
```

`appServicePlanId` should end with `/serverfarms/med-prod-appservice-plan-linux`.

### 4.4 Recommended platform settings for this container

The API listens on **`PORT` default `8080`** (`src/internal/config/config.go`). Set Azure to map traffic to that port:

```bash
az webapp config appsettings set \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --settings WEBSITES_PORT=8080 PORT=8080
```

Optional (HTTPS only, always on where SKU allows):

```bash
az webapp update \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --https-only true

az webapp config set \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --always-on true
```

---

## Part 5: Azure Container Registry (ACR)

Do this only if you **do not yet have** a registry the Med App Service can pull from. You may **reuse** an existing prod ACR in `med-prod-rg-appservice`, or create a Med-specific one.

**Region rule:** Create (or reuse) ACR in **South India** (`southindia`) — the same location as `med-prod-appservice-plan-linux` / `med-prod-api`. Do not place Med prod ACR in Central India or another region.

### 5.0 Register Container Registry provider (one-time per subscription)

If `az acr create` fails with `MissingSubscriptionRegistration` / namespace `Microsoft.ContainerRegistry`, register the provider on the **current** subscription, wait until it is `Registered`, then create ACR again:

```bash
az account show --query "{name:name,id:id}" -o table

az provider register --namespace Microsoft.ContainerRegistry

# Wait until RegistrationState is Registered (can take 1–2 minutes)
az provider show --namespace Microsoft.ContainerRegistry --query "registrationState" -o tsv
```

Optional poll until registered:

```bash
while [ "$(az provider show --namespace Microsoft.ContainerRegistry --query registrationState -o tsv)" != "Registered" ]; do
  echo "Waiting for Microsoft.ContainerRegistry registration..."
  sleep 10
done
echo "Registered."
```

You need permission to register resource providers on that subscription (typically Owner or Contributor).

### 5.1 Create ACR (example name)

ACR names must be globally unique (lowercase alphanumeric, 5–50 chars):

```bash
az acr create \
  --name medproductionacr \
  --resource-group med-prod-rg-appservice \
  --sku Basic \
  --admin-enabled true \
  --location southindia
```

Confirm location after create:

```bash
az acr show \
  --name medproductionacr \
  --resource-group med-prod-rg-appservice \
  --query "{name:name,location:location,loginServer:loginServer}" -o table
```

Expected `location`: `southindia`.

For production, your team may use Premium SKU, private endpoints, or managed identity instead of admin user; adjust Part 6 / Portal container settings accordingly.

### 5.2 Enable admin user (if using admin pull)

```bash
az acr update \
  --name medproductionacr \
  --resource-group med-prod-rg-appservice \
  --admin-enabled true
```

---

## Part 6: Deploy the container

All commands below: **WSL**, from the **project root** (folder with `Dockerfile` and `src/`).

### 6.1 Go to project root

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
```

### 6.2 Set Med production ACR, App Service, and resource group

The deploy script defaults to **staging** names. For Med production you **must** export:

```bash
export ACR_NAME=medproductionacr
export APP_NAME=med-prod-api
export RESOURCE_GROUP=med-prod-rg-appservice
```

Optional:

```bash
# export IMAGE_TAG=v1.0.0
# export IMAGE_NAME=b2b-aggregator-api
# export ACR_RESOURCE_GROUP=med-prod-rg-appservice   # if ACR lives in another RG
```

If you reuse a shared prod ACR (e.g. `umproductionacr`), set `ACR_NAME` to that registry instead.

### 6.3 Run the deploy script

```bash
chmod +x scripts/deploy-container.sh
./scripts/deploy-container.sh
```

### 6.4 What the deploy script does

1. **Build** the Docker image from the project `Dockerfile` (Go 1.27 → Alpine; binary `./api`; `EXPOSE 8080`).
2. **Log in** to ACR (`az acr login`).
3. **Tag** and **push** (e.g. `medproductionacr.azurecr.io/b2b-aggregator-api:latest`).
4. **Enable** ACR admin when applicable; retrieve credentials.
5. **Configure** App Service **`med-prod-api`** to pull that image.
6. **Restart** the App Service.

`.env` is **not** baked into the image (excluded by `.dockerignore`). All runtime config must be Application settings (Part 7).

### 6.5 Manual deploy (without the script)

```bash
docker build -t b2b-aggregator-api:latest .
az acr login --name medproductionacr
ACR_LOGIN_SERVER=$(az acr show --name medproductionacr --resource-group med-prod-rg-appservice --query loginServer -o tsv)
docker tag b2b-aggregator-api:latest "${ACR_LOGIN_SERVER}/b2b-aggregator-api:latest"
docker push "${ACR_LOGIN_SERVER}/b2b-aggregator-api:latest"

az webapp config container set \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --container-image-name "${ACR_LOGIN_SERVER}/b2b-aggregator-api:latest" \
  --container-registry-url "https://${ACR_LOGIN_SERVER}" \
  --container-registry-user "$(az acr credential show -n medproductionacr -g med-prod-rg-appservice --query username -o tsv)" \
  --container-registry-password "$(az acr credential show -n medproductionacr -g med-prod-rg-appservice --query 'passwords[0].value' -o tsv)"

az webapp restart --name med-prod-api --resource-group med-prod-rg-appservice
```

Or in Portal: **med-prod-api** → **Deployment Center** / **Container settings** → Azure Container Registry → image `b2b-aggregator-api`, tag `latest` (or version tag).

---

## Part 7: Configure Application settings

Values below are derived from `src/internal/config/config.go`, `src/pkg/utils/encryption_utils.go`, and related loaders. The container does **not** read a `.env` file in Azure; set these on **med-prod-api**.

### 7.1 Where to set them

1. [Azure Portal](https://portal.azure.com) → **med-prod-api**.
2. **Settings** → **Configuration** / **Environment variables** → **Application settings**.
3. Add each name/value → **Save** → restart when prompted.

Or via CLI (example pattern):

```bash
az webapp config appsettings set \
  --name med-prod-api \
  --resource-group med-prod-rg-appservice \
  --settings \
    PORT=8080 \
    WEBSITES_PORT=8080 \
    ENVIRONMENT=prod \
    DB_ENCRYPT=true \
    DB_TRUST_SERVER_CERT=false
```

Secrets should be pasted in Portal or sourced from Key Vault references—not committed to git.

### 7.2 Required / core API settings

| Name | Value (Med production examples) | Required | Source in code |
|------|--------------------------------|----------|----------------|
| `PORT` | `8080` | Yes | `config.go` (default `8080`) |
| `WEBSITES_PORT` | `8080` | Recommended for Linux containers | Azure platform (maps inbound traffic to container) |
| `ENVIRONMENT` | `prod` | Recommended | `config.go`; shown in `/ping` |
| `DB_SERVER` | `med-prod-server-database.database.windows.net:1433` | **Yes** | `config.go` — use your Med SQL FQDN + port |
| `DB_DATABASE_NAME` | `med-prod-DB` | **Yes** | `config.go` |
| `DB_SCHEMA` | *Med SQL schema name* | **Yes** for Med | `config.go` — SQL Server schema prefix for all tables (default `MediAdmin`; set to Med’s schema, not UrMedi’s) |
| `DB_USER` | *Med prod SQL user* | **Yes** | `config.go` |
| `DB_PASSWORD` | *Med prod SQL password* | **Yes** (secret) | `config.go` |
| `DB_ENCRYPT` | `true` | Yes | `config.go` (default `true`) |
| `DB_TRUST_SERVER_CERT` | `false` | Yes (typical for Azure SQL) | `config.go` (default `false`) |
| `JWT_SECRET` | *Strong secret unique to Med prod* | **Yes** (secret) | `config.go` |
| `JWT_EXPIRES_IN` | `24h` | Optional | `config.go` (default `24h`) |
| `JWT_REFRESH_EXPIRES_IN` | `7d` | Optional | `config.go` (default `7d`) |
| `LOGIN_ENC_KEY` | *Med prod key* | **Yes** for login (secret) | `encryption_utils.go` — must match how passwords were encrypted |
| `LOGIN_ENC_SALT` | *Med prod salt* | **Yes** for login (secret) | same |

Without valid `DB_*`, the process may start but APIs that need the database fail (often `503 Database unavailable`). Without `LOGIN_ENC_*`, login returns credential/encryption errors.

### 7.3 Optional pool / logging tuning

| Name | Example / default | Required | Notes |
|------|-------------------|----------|-------|
| `DB_POOL_MAX` | `25` | No | Default in code: `25` |
| `DB_POOL_MIN` | `5` | No | Default: `5` (max idle) |
| `DB_IDLE_TIMEOUT` | `30000` | No | Idle timeout ms |
| `DB_CONN_MAX_LIFETIME_MS` | `3600000` | No | `0` = no limit |
| `LOG_DIR` | `logs` | No | File logger; falls back to stdout if create fails |
| `LOG_RETENTION_HOURS` | `24` | No | |

### 7.4 Azure Blob (MoU + diagnostic reports)

Needed when MoU PDF or diagnostic report upload/download is used. Auth: **either** `AZURE_STORAGE_CONNECTION_STRING` **or** `AZURE_STORAGE_ACCOUNT` + `AZURE_STORAGE_KEY`.

| Name | Example / default | Required for blob features | Notes |
|------|-------------------|----------------------------|-------|
| `AZURE_STORAGE_CONNECTION_STRING` | *Full connection string* | One of auth options | Prefer Key Vault / App settings secret |
| `AZURE_STORAGE_ACCOUNT` | e.g. `medprodstorage` | Alt auth | |
| `AZURE_STORAGE_KEY` | *Account key* | Alt auth (secret) | |
| `AZURE_STORAGE_ENDPOINT_SUFFIX` | `core.windows.net` | No | Default in code |
| `AZURE_LEGAL_CONTAINER_NAME` | `legal-documents` | No | MoU PDFs (default) |
| `MOU_MAX_UPLOAD_BYTES` | `5242880` | No | 5 MiB default |
| `MOU_UPLOAD_TIMEOUT_SECONDS` | `60` | No | |
| `MOU_SAS_TTL_MINUTES` | `15` | No | Max `1440` |
| `AZURE_DIAGNOSTIC_REPORTS_CONTAINER` | `diagnostic-reports` | No | Default in code |
| `DIAGNOSTIC_REPORTS_MAX_UPLOAD_BYTES` | `5242880` | No | Falls back to MoU max |
| `DIAGNOSTIC_REPORTS_UPLOAD_TIMEOUT_SECONDS` | `60` | No | Falls back to MoU timeout |
| `DIAGNOSTIC_REPORTS_SAS_TTL_MINUTES` | `15` | No | Falls back to MoU SAS; max `1440` |

If blob auth is unset, the API still starts; MoU / report blob features stay disabled.

### 7.5 Loaded but not required for container runtime

These are read into config (`CLIENT_DOMAIN_URL`, `EMPLOYEE_DOMAIN_URL`, `LAB_DOMAIN_URL`, `DOMAIN`) but login domain resolution is hardcoded in code today. You may omit them on App Service unless you later wire them up.

`LAST_BUILD_PUSHED` / `LATEST_COMMIT` appear in older deploy docs; current `/ping` does **not** read them from the environment (values are hardcoded in the handler). Setting them is harmless but has no effect until the handler is updated.

### 7.6 Docker registry application settings (`DOCKER_REGISTRY_*`)

Some Linux container App Services also store pull credentials as application settings:

```bash
az acr show --name medproductionacr --resource-group med-prod-rg-appservice --query loginServer -o tsv
# → set DOCKER_REGISTRY_SERVER_URL=https://<that-login-server>

az acr update -n medproductionacr -g med-prod-rg-appservice --admin-enabled true
az acr credential show -n medproductionacr -g med-prod-rg-appservice --query username -o tsv
az acr credential show -n medproductionacr -g med-prod-rg-appservice --query "passwords[0].value" -o tsv
```

| Name | Value |
|------|--------|
| `DOCKER_REGISTRY_SERVER_URL` | `https://medproductionacr.azurecr.io` (example) |
| `DOCKER_REGISTRY_SERVER_USERNAME` | ACR admin username |
| `DOCKER_REGISTRY_SERVER_PASSWORD` | ACR admin password (secret) |

The deploy script already passes registry credentials via `az webapp config container set`; keep Portal/App settings in sync if you edit them manually.

### 7.7 Not for this App Service

Do **not** put fitness-worker or email-worker variables (`FITNESS_CERT_*`, `ACS_CONNECTION_STRING`, `EMAIL_WORKER_*`, `CHROMIUM_PATH`, `WEBSITE_SKIP_RUNNING_KUDUAGENT`) on **`med-prod-api`**. Those belong on a separate Code/WebJob App Service (Alpine custom containers do not support Linux WebJobs). See worker prod docs under `docs/`.

---

## Part 8: Verify deployment

1. **Health**
   ```bash
   curl https://med-prod-api.azurewebsites.net/ping
   ```
   Expect a healthy `pong`-style response (and `ENVIRONMENT=prod` metadata if configured).

2. **API base**  
   `https://med-prod-api.azurewebsites.net/api/v1/...`

3. **Log stream**  
   Portal → **med-prod-api** → **Log stream** — watch for DB connect errors or missing secrets.

4. **Plan confirmation**
   ```bash
   az webapp show -n med-prod-api -g med-prod-rg-appservice --query appServicePlanId -o tsv
   ```
   Should reference `med-prod-appservice-plan-linux`.

---

## Part 9: Deploying updates later

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=medproductionacr
export APP_NAME=med-prod-api
export RESOURCE_GROUP=med-prod-rg-appservice
export IMAGE_TAG=v1.2.3
./scripts/deploy-container.sh
```

Confirm Portal → **med-prod-api** → container **Tag** is `v1.2.3`. To roll back, point the tag at a previous image still in ACR (no rebuild required).

---

## Part 10: Binary / Zip deployment (non-container)

Use only if **`med-prod-api` is not** Publishing model: Container.

### 10.1 Build Linux binary

```bash
cd src
GOOS=linux GOARCH=amd64 go build -o b2b-aggregator ./cmd/api
```

### 10.2 Package

```bash
mkdir -p deploy
cp b2b-aggregator deploy/
# Prefer Portal Application settings for secrets; do not ship staging .env
```

Optional `deploy/start.sh`:

```bash
#!/usr/bin/env sh
set -e
./b2b-aggregator
```

### 10.3 Zip deploy

```bash
cd src/deploy
zip -r ../deploy.zip .
cd ..
az webapp deployment source config-zip \
  --resource-group med-prod-rg-appservice \
  --name med-prod-api \
  --src deploy.zip
```

Set **Startup Command** to `./b2b-aggregator`. Mirror Part 7 Application settings (including `PORT` / `WEBSITES_PORT` as appropriate for Code hosting).

---

## Troubleshooting

### 503 – Database unavailable

App is up but cannot reach SQL. Check Med prod `DB_*` on **med-prod-api** and Azure SQL firewall / networking for this App Service outbound IPs (or “Allow Azure services”).

### Image pull / wrong plan

| Issue | What to do |
|-------|------------|
| Wrong App Service updated | You forgot `export APP_NAME=med-prod-api` / `RESOURCE_GROUP=med-prod-rg-appservice`; script defaults to staging. |
| App not on Med plan | `az webapp update -n med-prod-api -g med-prod-rg-appservice --plan med-prod-appservice-plan-linux` |
| Site never reaches API | Set `WEBSITES_PORT=8080` and `PORT=8080`; confirm container listens on 8080. |
| "enable admin first" | `az acr update -n medproductionacr -g med-prod-rg-appservice --admin-enabled true` |
| Image pull fails | ACR Access keys + App Service container registry URL/user/password. |
| App keeps restarting | Log stream — often missing `DB_PASSWORD` or bad `DB_*`. |
| 500 validating credentials | Med prod `LOGIN_ENC_KEY` / `LOGIN_ENC_SALT` mismatch. |
| MoU / report upload fails | Set Azure Storage auth + containers (Part 7.4). |
| Docker build fails (Go version) | `src/go.mod` must match Dockerfile Go version; build from repo root. |

---

## Quick reference

```bash
az login
az account set --subscription "Your production subscription"

# Create plan + app (only if they do not exist)
az appservice plan create \
  --name med-prod-appservice-plan-linux \
  --resource-group med-prod-rg-appservice \
  --is-linux \
  --sku P1V3 \
  --location southindia

# ACR must be South India as well (same region as App Service)
az acr create \
  --name medproductionacr \
  --resource-group med-prod-rg-appservice \
  --sku Basic \
  --admin-enabled true \
  --location southindia

az webapp create \
  --resource-group med-prod-rg-appservice \
  --plan med-prod-appservice-plan-linux \
  --name med-prod-api \
  --deployment-container-image-name mcr.microsoft.com/appsvc/staticsite:latest

# Deploy container
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=medproductionacr
export APP_NAME=med-prod-api
export RESOURCE_GROUP=med-prod-rg-appservice
export IMAGE_TAG=v1.0.0
chmod +x scripts/deploy-container.sh
./scripts/deploy-container.sh
```

Then set Application settings on **med-prod-api** (Part 7): `DB_*`, `JWT_*`, `LOGIN_ENC_*`, `ENVIRONMENT=prod`, `PORT` / `WEBSITES_PORT`, and Azure Blob vars if needed.

---

## Deployment checklist (quick)

1. **Plan / app:** `med-prod-appservice-plan-linux` + `med-prod-api` in `med-prod-rg-appservice` (create or update per Part 4).
2. **Platform:** `WEBSITES_PORT=8080`, `PORT=8080`, container publishing model.
3. **From project root:**
   ```bash
   export ACR_NAME=medproductionacr
   export APP_NAME=med-prod-api
   export RESOURCE_GROUP=med-prod-rg-appservice
   export IMAGE_TAG=<release-tag>
   ./scripts/deploy-container.sh
   ```
4. **Application settings:** Med prod `DB_*`, `JWT_SECRET`, `LOGIN_ENC_KEY`, `LOGIN_ENC_SALT`, `ENVIRONMENT=prod`; blob settings if MoU/reports are in scope.
5. **Verify:** `https://med-prod-api.azurewebsites.net/ping` and Log stream.

For the UrMediConnect prod App Service (`um-prod-api`), use [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md). For staging, use [DEPLOYMENT.md](./DEPLOYMENT.md).
