## Azure App Service deployment (Go API)

These steps assume:
- The API is built from `src/` (where `go.mod` lives)
- Target is **Azure App Service** on **Linux**
- You will provide environment values in `.env` or App Service settings

### 1) Build the Linux binary

From the repo root:

```
cd src
GOOS=linux GOARCH=amd64 go build -o b2b-aggregator ./cmd/api
```

If your App Service plan is ARM64:

```
GOOS=linux GOARCH=arm64 go build -o b2b-aggregator ./cmd/api
```

### 2) Prepare the deployment package

From `src/`:

```
mkdir -p deploy
cp b2b-aggregator deploy/
cp .env deploy/
```

Optional: add a simple start script:

```
cat > deploy/start.sh <<'EOF'
#!/usr/bin/env sh
set -e
./b2b-aggregator
EOF
chmod +x deploy/start.sh
```

### 3) Create the Azure App Service (Linux)

In Azure Portal:
- Create **App Service** (Linux)
- Runtime stack: **Custom** (or **Go** if you prefer)
- Set region and SKU

### 4) Configure environment variables

Option A (recommended): use **App Service Configuration**
- App Service → Configuration → Application settings
- Add values from `.env` (one key per setting)

Option B: use `.env` on the server
- Upload `.env` to the app directory (see next step)

### 5) Deploy the binary

Option A: **Zip deploy**
1. From `src/`:
   ```
   cd deploy
   zip -r ../deploy.zip .
   ```
2. Use **Deployment Center** → Zip Deploy (or Azure CLI):
   ```
   az webapp deployment source config-zip \
     --resource-group <RG_NAME> \
     --name <APP_NAME> \
     --src ../deploy.zip
   ```

Option B: **SSH / SFTP**
- App Service → Development Tools → SSH
- Upload `b2b-aggregator` and `.env` into `/home/site/wwwroot`
- Ensure binary is executable:
  ```
  chmod +x /home/site/wwwroot/b2b-aggregator
  ```

### 6) Configure startup command

In Azure Portal:
App Service → Configuration → General settings → **Startup Command**

Set:
```
./b2b-aggregator
```

If you used `start.sh`, set:
```
./start.sh
```

### 7) Verify

Check:
- App Service → Log stream
- Browse to: `https://<APP_NAME>.azurewebsites.net/ping`

If DB is down, the API will return `503 Database unavailable` (expected).

### Troubleshooting

- **"This app can't run on your PC"**: binary built for wrong architecture.
- **Permission denied**: `chmod +x b2b-aggregator`.
- **Config missing**: ensure App Settings or `.env` are present.
- **DB error**: verify firewall and connection string settings.
