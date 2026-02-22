# B2B Aggregator API – Setup and Deployment to Azure

**Audience:** Developers (including juniors) deploying the Go API to Azure App Service.

This guide is **reusable**: replace placeholder values (e.g. your ACR name, resource group) with your own when your environment differs. Example values: **um-staging-api** (App Service), **umstagingacr** (ACR), **um-staging-rg-appservice** (resource group).

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
- **Azure subscription** with access to create or use an App Service and a Container Registry.
- **Repo root** = folder that contains `Dockerfile` and the `src/` directory.

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

- A browser window will open (or you get a URL and code). Sign in with your **Azure account**.

### 3.2 Set the correct subscription (if you have more than one)

```bash
az account list --output table
az account set --subscription "Azure subscription 1"
```

Use the exact subscription name or ID from the list.

### 3.3 Confirm you can see your resources

```bash
az acr list --resource-group um-staging-rg-appservice --output table
```

You should see your ACR (e.g. `umstagingacr`). If your resource group name is different, replace it in the command.

---

## Part 4: Create Azure Container Registry (ACR)

Do this only if you **do not yet have** a container registry.

### 4.1 Create ACR (Azure CLI)

Replace `umstagingacr` with a **unique** name (lowercase letters and numbers, 5–50 characters). It must be globally unique across Azure.

```bash
az acr create --name umstagingacr --resource-group um-staging-rg-appservice --sku Basic --admin-enabled true --location centralindia
```

- `--admin-enabled true` is required so the App Service can pull images using username/password.

### 4.2 Create ACR (Azure Portal)

1. Go to [Azure Portal](https://portal.azure.com) → **Create a resource** → **Container Registry**.
2. **Basics:** Subscription, resource group (e.g. `um-staging-rg-appservice`), registry name (e.g. `umstagingacr`), location (e.g. Central India), SKU **Basic**.
3. **Domain name label scope:** Choose **Subscription Reuse** (or **Resource Group Reuse**). Do **not** use **Unsecure**.
4. **Role assignment permissions mode:** **RBAC Registry Permissions**.
5. Create the resource. After deployment, go to the registry → **Settings** → **Access keys** → set **Admin user** to **Enable**.

---

## Part 5: Deploy the container

All commands in this part are run in **WSL**, from the **project root** (the folder that contains `Dockerfile` and `src/`).

### 5.1 Go to project root

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
```

If your repo is elsewhere (e.g. `~/projects/B2B-Aggregator-APIs`), use that path.

### 5.2 Set your ACR name

Use the **exact** name of your registry (e.g. `umstagingacr`):

```bash
export ACR_NAME=umstagingacr
```

Optional overrides (only if your names differ):

```bash
# export RESOURCE_GROUP=um-staging-rg-appservice
# export APP_NAME=um-staging-api
# export IMAGE_TAG=latest
```

### 5.3 Enable ACR admin user (if you see "enable admin first" errors)

If the deploy script later fails with a message like "Run 'az acr update -n umstagingacr --admin-enabled true'", run this once (replace names if needed):

```bash
az acr update --name umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true
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
3. **Tag** the image for ACR (e.g. `umstagingacr.azurecr.io/b2b-aggregator-api:latest`). If your ACR uses a different login server (e.g. with a suffix like `umstagingacr-xxxxx.azurecr.io`), the script uses that.
4. **Push** the image to ACR.
5. **Enable ACR admin user** if it is not already enabled (so credentials can be retrieved).
6. **Retrieve** ACR username and password.
7. **Configure** the App Service to use the pushed image and ACR credentials (using current Azure CLI parameters: `--container-image-name`, `--container-registry-url`, `--container-registry-user`, `--container-registry-password`).
8. **Restart** the App Service.

If any step fails, see [Troubleshooting](#troubleshooting).

### 5.6 Manual deploy (without the script)

If you cannot use the script (e.g. different OS or you prefer manual steps):

```bash
# From project root
docker build -t b2b-aggregator-api:latest .
az acr login --name umstagingacr
docker tag b2b-aggregator-api:latest <YOUR_ACR_LOGIN_SERVER>/b2b-aggregator-api:latest
docker push <YOUR_ACR_LOGIN_SERVER>/b2b-aggregator-api:latest
```

To get your ACR login server:

```bash
az acr show --name umstagingacr --resource-group um-staging-rg-appservice --query loginServer -o tsv
```

Then in **Azure Portal** → **um-staging-api** → **Deployment Center** (or **Container settings**): set **Image source** to Azure Container Registry, choose your registry, image `b2b-aggregator-api`, tag `latest`, and use **Admin Credentials** (username and password from ACR → **Access keys**).

---

## Part 6: Configure Application settings in Azure Portal

The app needs environment variables (database, JWT, etc.). They are **not** read from a file in the container; they must be set in Azure.

### 6.1 Where to find Application settings

In the Azure Portal, application settings are under **Configuration** (the label may be **Configuration (preview)** or **Environment variables** depending on the portal version).

1. Open [Azure Portal](https://portal.azure.com).
2. Search for your App Service name (e.g. **um-staging-api**) and open it.
3. In the **left menu**, under **Settings**, click **Configuration (preview)** or **Configuration** or **Environment variables**.

### 6.2 Add Application settings

1. Open the **Application settings** tab (if you see tabs).
2. Click **+ New application setting** (or **+ Add** / **Add new application setting**).
3. Add each of the following (Name and Value). Replace placeholders with your real values.

| Name | Value | Required |
|------|--------|----------|
| `PORT` | `8080` | Yes (Azure may set this; app reads it) |
| `DB_SERVER` | `um-staging-server-database.database.windows.net:1433` | Yes (or your DB server) |
| `DB_DATABASE_NAME` | `um-staging-DB` | Yes (or your DB name) |
| `DB_USER` | `dev_admin` | Yes (or your DB user) |
| `DB_PASSWORD` | *Your database password* | **Yes** – never put in code |
| `DB_ENCRYPT` | `true` | Yes |
| `DB_TRUST_SERVER_CERT` | `false` | Yes |
| `JWT_SECRET` | *A strong secret string* | **Yes** |
| `JWT_EXPIRES_IN` | `24h` | Optional |
| `JWT_REFRESH_EXPIRES_IN` | `7d` | Optional |
| `ENVIRONMENT` | `prod`, `STG`, or `dev` | Optional – shown in `/ping` (default `dev`) |
| `LAST_BUILD_PUSHED` | e.g. `03-12-2025 14:45:00` | Optional – dev sets at build/deploy; shown in `/ping` |
| `LATEST_COMMIT` | e.g. `feat: updated ping controller` | Optional – dev sets at build/deploy; shown in `/ping` |
| `LOGIN_ENC_KEY` | *A secret key string* | **Yes** for login – used to encrypt passwords (must match the key used when passwords were stored). |
| `LOGIN_ENC_SALT` | *A secret salt string* | **Yes** for login – used with LOGIN_ENC_KEY for password encryption. |

4. Click **OK** for each setting.
5. Click **Save** at the top of the Configuration blade, then **Continue** when prompted to restart the app.

### 6.3 If the App Service cannot pull the image

- In **ACR** → **Access keys**: ensure **Admin user** is **Enabled**.
- In **App Service** → **Deployment Center** or **Container settings**: set **Image source** to Azure Container Registry, select your registry, and use the **admin username** and **password** from ACR Access keys.

---

## Part 7: Verify deployment

1. **Health endpoint**  
   Open in a browser or run:
   ```bash
   curl https://um-staging-api.azurewebsites.net/ping
   ```
   Expected response: `{"message":"pong"}`.

2. **API base URL**  
   Example: `https://um-staging-api.azurewebsites.net/api/v1/...`

3. **Log stream**  
   In Azure Portal: **um-staging-api** → **Log stream**. Check for startup messages and any errors (e.g. database connection).

---

## Part 8: Deploying updates later

After you change code, rebuild and push a new image, then restart:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=umstagingacr
./scripts/deploy-container.sh
```

To use a versioned tag (e.g. for rollbacks):

```bash
export ACR_NAME=umstagingacr
export IMAGE_TAG=v1.0.1
./scripts/deploy-container.sh
```

Then in Azure Portal → **um-staging-api** → **Container settings**, set **Tag** to `v1.0.1` (or the tag you used) and save.

---

## Part 9: Binary / Zip deployment (non-container)

Use this when the App Service is **not** set to **Publishing model: Container** (e.g. Code or Zip deploy).

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

In Azure Portal: Create **App Service** (Linux), Runtime stack **Custom** (or **Go**), set region and SKU.

### 9.4 Configure environment variables

- **App Service** → **Configuration** → **Application settings** – add values from `.env` (one key per setting).

### 9.5 Deploy the binary

**Option A – Zip deploy**

```bash
cd src/deploy
zip -r ../deploy.zip .
cd ..
az webapp deployment source config-zip --resource-group <RG_NAME> --name <APP_NAME> --src deploy.zip
```

**Option B – SSH / SFTP**

- App Service → Development Tools → SSH.
- Upload `b2b-aggregator` and `.env` into `/home/site/wwwroot`.
- Run: `chmod +x /home/site/wwwroot/b2b-aggregator`

### 9.6 Configure startup command

App Service → **Configuration** → **General settings** → **Startup Command**: set `./b2b-aggregator` (or `./start.sh` if you created it).

### 9.7 Verify

- App Service → **Log stream**
- Browse to `https://<APP_NAME>.azurewebsites.net/ping`

If DB is down, the API may return `503 Database unavailable`.

---

## Troubleshooting

### 503 – Database unavailable

If the API returns **503** with `"message": "Database unavailable"` (e.g. on `POST /api/v1/login` or any non-ping request), the app is running but **cannot connect to the database** from Azure. This is a backend/configuration issue, not a frontend bug.

**Checklist for backend/DevOps:**

1. **Application settings**  
   In Azure Portal → **App Service** (e.g. **um-staging-api**) → **Configuration** → **Application settings**, ensure these exist and are correct (no typos, no extra spaces):
   - `DB_SERVER` (e.g. `um-staging-server-database.database.windows.net:1433`)
   - `DB_DATABASE_NAME` (e.g. `um-staging-DB`)
   - `DB_USER` (e.g. `dev_admin`)
   - `DB_PASSWORD` (correct password)
   - `DB_ENCRYPT` = `true`
   - `DB_TRUST_SERVER_CERT` = `false` (or as required)

2. **Azure SQL firewall**  
   In Azure Portal → **Azure SQL server** (e.g. **um-staging-server-database**) → **Networking** (or **Firewalls and virtual networks**):
   - Enable **“Allow Azure services and resources to access this server”**, or  
   - Add the App Service **outbound IP(s)** as a firewall rule.  
   Then **restart** the App Service.

3. **See the actual error**  
   App Service → **Log stream**. Restart the app and watch startup logs for the real connection error (e.g. login failed, firewall, timeout).

**For frontend developers:** No code change needed. Ask backend/DevOps to fix DB configuration and SQL firewall; after that, login and other APIs should work.

---

| Issue | What to do |
|-------|------------|
| `az: command not found` after install | Log out and back into WSL, or run `source ~/.bashrc`. Try `/usr/bin/az --version`. |
| `az login` doesn't open browser | Use device code: `az login --use-device-code` and follow the URL and code in the terminal. |
| Docker permission denied | Run `sudo usermod -aG docker $USER`, then log out and back into WSL. |
| Script: "ACR_NAME is required" | Run `export ACR_NAME=umstagingacr` in the **same** terminal before the script. |
| "enable admin first" or ACR credential error | Run `az acr update -n umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true`, then run the script again. |
| Wrong subscription | Run `az account set --subscription "Your Subscription Name"`. |
| Image pull fails (App Service) | Check ACR → Access keys (Admin user enabled). In App Service → Container settings, check registry URL, image name, tag, and admin username/password. |
| App keeps restarting | Check **Log stream**. Often missing `DB_PASSWORD` or wrong `DB_*` settings. Add or fix in **Configuration** → **Application settings**. |
| 500 "Error validating credentials" | **1)** Add **LOGIN_ENC_KEY** and **LOGIN_ENC_SALT** in App Service → **Configuration** → **Application settings** (required for password encryption; if missing, login returns 500). **2)** Or check **Log stream** for the real error (e.g. DB error during Authenticate). |
| 503 / "Database unavailable" | **1)** In App Service → **Configuration** → **Application settings**, ensure `DB_PASSWORD`, `DB_SERVER`, `DB_USER`, `DB_DATABASE_NAME` are set and correct (no typos). **2)** In **Azure SQL** → **Networking** (or **Firewalls and virtual networks**), allow Azure services and/or add the App Service outbound IPs. Then **restart** the App Service. |
| Docker build fails (Go version) | Ensure `src/go.mod` has a Go version that matches the Dockerfile (e.g. Go 1.25 in both). Build from the folder that contains `Dockerfile` and `src/`. |
| Binary: "This app can't run on your PC" | Binary built for wrong architecture; rebuild with correct GOOS/GOARCH. |
| Binary: Permission denied | Run `chmod +x b2b-aggregator`. |

---

## Quick reference

Copy-paste in WSL (adjust paths and names if needed):

```bash
# One-time: Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Per session (or when token expires): Log in to Azure
az login

# Deploy (from project root)
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
export ACR_NAME=umstagingacr
chmod +x scripts/deploy-container.sh
./scripts/deploy-container.sh
```

After deploy, set **DB_PASSWORD**, **JWT_SECRET**, and other variables in **Azure Portal** → **um-staging-api** → **Configuration (preview)** → **Application settings**.

---

## Deployment checklist (quick)

For those who have already run through the full guide once:

1. **Prerequisites:** Docker and Azure CLI in WSL; ACR created with Admin user enabled.
2. **From project root:** `export ACR_NAME=umstagingacr`
3. **Run:** `./scripts/deploy-container.sh`
4. **If "enable admin first":** `az acr update -n umstagingacr --resource-group um-staging-rg-appservice --admin-enabled true` then re-run script.
5. **Portal:** **um-staging-api** → **Configuration (preview)** → **Application settings** – add `DB_PASSWORD`, `JWT_SECRET`, `PORT`, and other `DB_*` / `JWT_*` as in [Part 6](#part-6-configure-application-settings-in-azure-portal).
6. **Verify:** `https://um-staging-api.azurewebsites.net/ping` → returns JSON with Environment, Current TimeStamp, IST TimeStamp, Last Build Pushed, Latest commit.

For full steps and troubleshooting, use the sections above in this document.
