# B2B Aggregator API – Production deployment to Azure

**Audience:** Developers and release owners deploying the Go API to **production** Azure App Service.

**Staging counterpart:** For non-production environments, see [DEPLOYMENT.md](./DEPLOYMENT.md).

This guide is **reusable**: replace placeholder values (ACR name, resource group, App Service name, database endpoints) with your organisation’s **production** names. Example placeholders used below: **um-prod-api** (App Service), **umprodacr** (ACR), **um-prod-rg-appservice** (resource group).

**Production expectations**

- Deploy only from **reviewed, released** commits (tag or release branch), not experimental branches.
- Prefer **versioned image tags** (e.g. `v1.2.3`) for production so you can roll back in Azure Portal by changing the container tag.
- Treat **secrets** (`DB_PASSWORD`, `JWT_SECRET`, `LOGIN_ENC_KEY`, `LOGIN_ENC_SALT`) as production-only: rotate separately from staging; never reuse staging secrets in prod.
- Coordinate **maintenance or deploy windows** with stakeholders if the API has downtime during container restarts.

**Primary path:** Container deployment using WSL (Parts 1–8). **Alternative path:** Binary / Zip deployment when the App Service is not set to Container (Part 9).

---

## Table of contents

1. [Prerequisites](#prerequisites)
2. [Part 1: Docker in WSL](#part-1-docker-in-wsl)
3. [Part 2: Azure CLI (`az`) in WSL](#part-2-azure-cli-az-in-wsl)
4. [Part 3: Log in to Azure](#part-3-log-in-to-azure)
5. [Part 4: Create Azure Container Registry (ACR)](#part-4-create-azure-container-registry-acr)
6. [Part 5: Deploy the container](#part-5-deploy-the-container)
7. [Part 6: Configure Application settings in Azure Portal](#part-6-configure-application-settings-in-azure-portal)
8. [Part 7: Verify deployment](#part-7-verify-deployment)
9. [Part 8: Deploying updates later](#part-8-deploying-updates-later)
10. [Part 9: Binary / Zip deployment (non-container)](#part-9-binary--zip-deployment-non-container)
11. [Troubleshooting](#troubleshooting)
12. [Quick reference](#quick-reference)
13. [Deployment checklist (quick)](#deployment-checklist-quick)

---

## Prerequisites

- **Windows PC** with **WSL** (Windows Subsystem for Linux) for container deploy; or any environment with Docker and Azure CLI for manual steps.
- **Project** cloned (e.g. `B2B-Aggregator-APIs`) and available in WSL or locally.
- **Azure subscription** with permission to deploy to **production** App Service and push to the **production** Container Registry (often a different subscription or resource group than staging).
- **Repo root** = folder that contains `Dockerfile` and the `src/` directory.
- **Access** to production database, JWT secrets, and login encryption values used by live users (`LOGIN_ENC_KEY` / `LOGIN_ENC_SALT` must match how passwords were encrypted).

---

## Part 1: Docker in WSL

### 1.1 Open WSL

- Open **Windows Terminal** or **Command Prompt** and run:
  ```bash
  wsl
  ```
- Or start **Ubuntu** (or your Linux distro) from the Start menu.

### 1.2 Check Docker

```bash
docker --version
docker run hello-world
```

- If both succeed, Docker is ready. Go to [Part 2](#part-2-azure-cli-az-in-wsl).
- If `docker: command not found`:
  - **Option A:** Install Docker Engine inside WSL: [Install Docker Engine on Ubuntu](https://docs.docker.com/engine/install/ubuntu/).
  - **Option B:** Install **Docker Desktop for Windows**, enable **"Use the WSL 2 based engine"** and **"Integrate with my default WSL distro"**. Then `docker` in WSL will use Docker Desktop.

### 1.3 (If needed) Fix Docker permission

If you see "permission denied" when running `docker`:

```bash
sudo usermod -aG docker $USER
```

Log out of WSL and open it again, then try `docker run hello-world` again.

---

## Part 2: Azure CLI (`az`) in WSL

### 2.1 Install Azure CLI (one-time)

Run in WSL:

```bash
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

Verify:

```bash
az --version
```

### 2.2 (Optional) Install via package manager (Ubuntu/Debian)

If the script above fails:

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

In WSL:

```bash
az login
```

- A browser window will open (or you get a URL and code). Sign in with an account that can deploy to **production**.

### 3.2 Set the correct subscription (if you have more than one)

```bash
az account list --output table
az account set --subscription "Azure subscription 1"
```

Use the exact subscription name or ID from the list.

### 3.3 Confirm you can see your resources

```bash
az acr list --resource-group um-prod-rg-appservice --output table
```

You should see your production ACR (e.g. `umprodacr`). If your resource group name is different, replace it in the command.

---

## Part 4: Create Azure Container Registry (ACR)

Do this only if you **do not yet have** a **production** container registry.

### 4.1 Create ACR (Azure CLI)

Replace `umproductionacr` with a **unique** name (lowercase letters and numbers, 5–50 characters). It must be globally unique across Azure.

```bash
az acr create --name umproductionacr --resource-group um-prod-rg-appservice --sku Basic --admin-enabled true --location centralindia
```

- `--admin-enabled true` is required so the App Service can pull images using username/password.
- For production, your team may use **Premium** SKU, **private endpoints**, or **managed identity** instead of admin user; if so, follow your organisation’s pattern and adjust Part 5 / Portal container settings accordingly.

### 4.2 Create ACR (Azure Portal)

1. Go to [Azure Portal](https://portal.azure.com) → **Create a resource** → **Container Registry**.
2. **Basics:** Subscription, resource group (e.g. `um-prod-rg-appservice`), registry name (e.g. `umproductionacr`), location (e.g. Central India), SKU per your standards.
3. **Domain name label scope:** Choose **Subscription Reuse** (or **Resource Group Reuse**). Do **not** use **Unsecure**.
4. **Role assignment permissions mode:** **RBAC Registry Permissions** (unless your org specifies otherwise).
5. Create the resource. After deployment, go to the registry → **Settings** → **Access keys** → set **Admin user** to **Enable** if you use admin credentials for App Service pull.

---

## Part 5: Deploy the container

All commands in this part are run in **WSL**, from the **project root** (the folder that contains `Dockerfile` and `src/`).

### 5.1 Go to project root

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
```

If your repo is elsewhere (e.g. `~/projects/B2B-Aggregator-APIs`), use that path.

### 5.2 Set production ACR, App Service, and resource group

The deploy script defaults to **staging** names. For production you **must** export all of the following in the same shell session:

```bash
export ACR_NAME=umproductionacr
export APP_NAME=um-prod-api
export RESOURCE_GROUP=um-prod-rg-appservice
```

Optional overrides:

```bash
# export IMAGE_TAG=v1.0.0
# export ACR_RESOURCE_GROUP=um-prod-rg-appservice   # if ACR lives in another RG
```

### 5.3 Enable ACR admin user (if you see "enable admin first" errors)

If the deploy script fails with a message like "Run 'az acr update -n umproductionacr --admin-enabled true'", run this once (replace names if needed):

```bash
az acr update --name umproductionacr --resource-group um-prod-rg-appservice --admin-enabled true
```

Then run the deploy script again.

### 5.4 Run the deploy script

```bash
chmod +x scripts/deploy-container.sh
./scripts/deploy-container.sh
```

Wait until the script finishes without errors.

### 5.5 What the deploy script does

The script does the following in order:

1. **Build** the Docker image from the project's `Dockerfile` (Go 1.25, Alpine).
2. **Log in** to your Azure Container Registry (`az acr login`).
3. **Tag** the image for ACR (e.g. `umproductionacr.azurecr.io/b2b-aggregator-api:latest`). If your ACR uses a different login server, the script uses that.
4. **Push** the image to ACR.
5. **Enable ACR admin user** if it is not already enabled (when applicable).
6. **Retrieve** ACR username and password.
7. **Configure** the **production** App Service (`APP_NAME`) to use the pushed image and registry credentials.
8. **Restart** the App Service.

If any step fails, see [Troubleshooting](#troubleshooting).

### 5.6 Manual deploy (without the script)

If you cannot use the script:

```bash
# From project root
docker build -t b2b-aggregator-api:latest .
az acr login --name umproductionacr
docker tag b2b-aggregator-api:latest <YOUR_ACR_LOGIN_SERVER>/b2b-aggregator-api:latest
docker push <YOUR_ACR_LOGIN_SERVER>/b2b-aggregator-api:latest
```

To get your ACR login server:

```bash
az acr show --name umprodacr --resource-group um-prod-rg-appservice --query loginServer -o tsv
```

Then in **Azure Portal** → **um-prod-api** → **Deployment Center** (or **Container settings**): set **Image source** to Azure Container Registry, choose your registry, image `b2b-aggregator-api`, tag `latest` (or your version tag), and use **Admin Credentials** (username and password from ACR → **Access keys**) if that is your setup.

---

## Part 6: Configure Application settings in Azure Portal

The app needs environment variables (database, JWT, etc.). They are **not** read from a file in the container; they must be set in Azure for **production**.

### 6.1 Where to find Application settings

In the Azure Portal, application settings are under **Configuration** (the label may be **Configuration (preview)** or **Environment variables** depending on the portal version).

1. Open [Azure Portal](https://portal.azure.com).
2. Search for your **production** App Service name (e.g. **um-prod-api**) and open it.
3. In the **left menu**, under **Settings**, click **Configuration (preview)** or **Configuration** or **Environment variables**.

### 6.2 Add Application settings

1. Open the **Application settings** tab (if you see tabs).
2. Click **+ New application setting** (or **+ Add** / **Add new application setting**).
3. Add each of the following (Name and Value). Replace placeholders with your **production** values.

| Name | Value (production examples) | Required |
|------|-----------------------------|----------|
| `PORT` | `8080` | Yes (Azure may set this; app reads it) |
| `DB_SERVER` | `um-prod-server-database.database.windows.net:1433` | Yes (your prod SQL endpoint) |
| `DB_DATABASE_NAME` | `um-prod-DB` | Yes |
| `DB_USER` | *Production SQL user* | Yes |
| `DB_PASSWORD` | *Production database password* | **Yes** – never put in code |
| `DB_ENCRYPT` | `true` | Yes |
| `DB_TRUST_SERVER_CERT` | `false` | Yes (or as required by your SQL setup) |
| `JWT_SECRET` | *Strong secret, unique to production* | **Yes** |
| `JWT_EXPIRES_IN` | e.g. `24h` | Optional |
| `JWT_REFRESH_EXPIRES_IN` | e.g. `7d` | Optional |
| `ENVIRONMENT` | `prod` | Recommended – shown in `/ping` |
| `LAST_BUILD_PUSHED` | e.g. `07-04-2026 14:45:00` | Optional – set at deploy; shown in `/ping` |
| `LATEST_COMMIT` | e.g. `release: v1.2.3` | Optional – set at deploy; shown in `/ping` |
| `LOGIN_ENC_KEY` | *Production key* | **Yes** for login – must match keys used when prod passwords were stored |
| `LOGIN_ENC_SALT` | *Production salt* | **Yes** for login |
| `DOCKER_REGISTRY_SERVER_URL` | `https://<login-server>` | As needed for container pull (see below) |
| `DOCKER_REGISTRY_SERVER_PASSWORD` | *ACR admin password* | As needed for container pull (see below) |

4. Click **OK** for each setting.
5. Click **Save** at the top of the Configuration blade, then **Continue** when prompted to restart the app.

### 6.3 Docker registry application settings (`DOCKER_REGISTRY_*`)

Some **Linux container** App Service setups expect registry credentials as application settings. Use your **production** ACR name and resource group in the commands below (examples use `umproductionacr` and `um-prod-rg-appservice`).

**`DOCKER_REGISTRY_SERVER_URL`**

Get the login server, then set the app setting to `https://` plus that host (App Service expects a URL, not a bare hostname):

```bash
az acr show --name umproductionacr --resource-group um-prod-rg-appservice --query loginServer -o tsv
```

Example: if the command prints `umproductionacr.azurecr.io`, set `DOCKER_REGISTRY_SERVER_URL` to `https://umproductionacr.azurecr.io`.

**`DOCKER_REGISTRY_SERVER_PASSWORD`**

Enable the ACR admin account (required for admin-based pull credentials), then print the password to copy into the app setting:

```bash
az acr update -n umproductionacr -g um-prod-rg-appservice --admin-enabled true
az acr credential show -n umproductionacr -g um-prod-rg-appservice --query "passwords[0].value" -o tsv
```

Treat the password as a secret: paste it only into Azure **Application settings**, not into source control. If your app settings include `DOCKER_REGISTRY_SERVER_USERNAME`, use the **Username** from the same ACR **Access keys** blade (or `az acr credential show ... --query username -o tsv`).

### 6.4 If the App Service cannot pull the image

- In **ACR** → **Access keys**: ensure **Admin user** is **Enabled** (if you use admin pull).
- In **App Service** → **Deployment Center** or **Container settings**: set **Image source** to Azure Container Registry, select your registry, and use the correct credentials per your setup.

---

## Part 7: Verify deployment

1. **Health endpoint**  
   Open in a browser or run:
   ```bash
   curl https://um-prod-api.azurewebsites.net/ping
   ```
   Expected response includes `pong` and, if configured, production-oriented metadata.

2. **API base URL**  
   Example: `https://um-prod-api.azurewebsites.net/api/v1/...`

3. **Log stream**  
   In Azure Portal: **um-prod-api** → **Log stream**. Check for startup messages and any errors (e.g. database connection).

---

## Part 8: Deploying updates later

After you change code, rebuild and push a new image, then restart. **For production**, set a **version tag** when you cut a release:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=umprodacr
export APP_NAME=um-prod-api
export RESOURCE_GROUP=um-prod-rg-appservice
export IMAGE_TAG=v1.2.3
./scripts/deploy-container.sh
```

Then in Azure Portal → **um-prod-api** → **Container settings**, confirm **Tag** is `v1.2.3` (or the tag you used) and save.

To roll back: change the container **Tag** in Portal back to the previous working image tag (no rebuild required if the image still exists in ACR).

---

## Part 9: Binary / Zip deployment (non-container)

Use this when the **production** App Service is **not** set to **Publishing model: Container** (e.g. Code or Zip deploy).

### 9.1 Build the Linux binary

From the repo root:

```bash
cd src
GOOS=linux GOARCH=amd64 go build -o b2b-aggregator ./cmd/api
```

If your App Service plan is ARM64:

```bash
GOOS=linux GOARCH=arm64 go build -o b2b-aggregator ./cmd/api
```

### 9.2 Prepare the deployment package

From `src/`:

```bash
mkdir -p deploy
cp b2b-aggregator deploy/
cp .env deploy/
```

Use a **production** `.env` or set variables only in Portal (preferred for secrets). Do not deploy staging `.env` to production.

Optional start script:

```bash
cat > deploy/start.sh <<'EOF'
#!/usr/bin/env sh
set -e
./b2b-aggregator
EOF
chmod +x deploy/start.sh
```

### 9.3 Create the Azure App Service (Linux)

In Azure Portal: Create **App Service** (Linux), Runtime stack **Custom** (or **Go**), set region and SKU per production standards.

### 9.4 Configure environment variables

- **App Service** → **Configuration** → **Application settings** – add **production** values (mirror Part 6).

### 9.5 Deploy the binary

**Option A – Zip deploy**

```bash
cd src/deploy
zip -r ../deploy.zip .
cd ..
az webapp deployment source config-zip --resource-group um-prod-rg-appservice --name um-prod-api --src deploy.zip
```

**Option B – SSH / SFTP**

- App Service → Development Tools → SSH.
- Upload `b2b-aggregator` and configuration into `/home/site/wwwroot`.
- Run: `chmod +x /home/site/wwwroot/b2b-aggregator`

### 9.6 Configure startup command

App Service → **Configuration** → **General settings** → **Startup Command**: set `./b2b-aggregator` (or `./start.sh` if you created it).

### 9.7 Verify

- App Service → **Log stream**
- Browse to `https://um-prod-api.azurewebsites.net/ping`

If DB is down, the API may return `503 Database unavailable`.

---

## Troubleshooting

### 503 – Database unavailable

Same diagnosis as staging: the app is running but **cannot connect to the database** from Azure. Use **production** `DB_*` settings and ensure the **production** Azure SQL firewall allows the **production** App Service (outbound IPs or “Allow Azure services” as appropriate).

**Checklist:**

1. **Application settings** on **um-prod-api** – correct `DB_SERVER`, `DB_DATABASE_NAME`, `DB_USER`, `DB_PASSWORD`, `DB_ENCRYPT`, `DB_TRUST_SERVER_CERT`.
2. **Azure SQL server (production)** → **Networking** – firewall rules for the prod App Service.
3. **Log stream** – actual connection error after restart.

---

| Issue | What to do |
|-------|------------|
| `az: command not found` after install | Log out and back into WSL, or run `source ~/.bashrc`. Try `/usr/bin/az --version`. |
| `az login` doesn't open browser | Use device code: `az login --use-device-code` and follow the URL and code in the terminal. |
| Docker permission denied | Run `sudo usermod -aG docker $USER`, then log out and back into WSL. |
| Script: "ACR_NAME is required" | Run `export ACR_NAME=umprodacr` (and `APP_NAME` / `RESOURCE_GROUP` for prod) in the **same** terminal before the script. |
| Wrong App Service updated | You forgot `export APP_NAME=um-prod-api` and `export RESOURCE_GROUP=um-prod-rg-appservice`; the script defaults to staging. |
| "enable admin first" or ACR credential error | Run `az acr update -n umprodacr --resource-group um-prod-rg-appservice --admin-enabled true`, then run the script again. |
| Wrong subscription | Run `az account set --subscription "Your production subscription"`. |
| Image pull fails (App Service) | Check ACR → Access keys and App Service → Container settings (registry URL, image name, tag, credentials). |
| App keeps restarting | Check **Log stream**. Often missing `DB_PASSWORD` or wrong `DB_*` settings. |
| 500 "Error validating credentials" | Ensure **LOGIN_ENC_KEY** and **LOGIN_ENC_SALT** match production password encryption; check **Log stream**. |
| 503 / "Database unavailable" | Fix prod `DB_*` and SQL firewall for prod App Service; restart. |
| Docker build fails (Go version) | Ensure `src/go.mod` matches the Dockerfile Go version. Build from repo root. |
| Binary: wrong architecture | Rebuild with correct GOOS/GOARCH. |
| Binary: Permission denied | Run `chmod +x b2b-aggregator`. |

---

## Quick reference

Copy-paste in WSL (adjust paths and **production** names):

```bash
# Per session (or when token expires): Log in to Azure
az login
az account set --subscription "Your production subscription"

# Deploy (from project root) — always set prod App Service and RG
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=umprodacr
export APP_NAME=um-prod-api
export RESOURCE_GROUP=um-prod-rg-appservice
export IMAGE_TAG=v1.0.0
chmod +x scripts/deploy-container.sh
./scripts/deploy-container.sh
```

After deploy, confirm **Application settings** on **um-prod-api** include production `DB_*`, `JWT_SECRET`, `LOGIN_ENC_KEY`, `LOGIN_ENC_SALT`, and `ENVIRONMENT=prod`.

---

## Deployment checklist (quick)

For operators who have already run through the full guide once:

1. **Prerequisites:** Docker and Azure CLI in WSL; production ACR; correct Azure subscription selected.
2. **From project root:**
   ```bash
   export ACR_NAME=umprodacr
   export APP_NAME=um-prod-api
   export RESOURCE_GROUP=um-prod-rg-appservice
   export IMAGE_TAG=<release-tag>
   ./scripts/deploy-container.sh
   ```
3. **If "enable admin first":** `az acr update -n umprodacr --resource-group um-prod-rg-appservice --admin-enabled true` then re-run the script.
4. **Portal:** **um-prod-api** → **Configuration** → **Application settings** – production `DB_*`, `JWT_*`, `LOGIN_ENC_*`, `ENVIRONMENT=prod`, `PORT`.
5. **Verify:** `https://um-prod-api.azurewebsites.net/ping` → healthy response with expected environment metadata.

For staging deployments, use [DEPLOYMENT.md](./DEPLOYMENT.md). For full steps and troubleshooting, use the sections above.
