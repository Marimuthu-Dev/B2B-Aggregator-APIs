# 🚀 Fitness Worker Deployment Guide (Azure WebJob)

## 📍 Prerequisites

* Azure CLI logged in (`az login`)
* Inside project root (`src/`)
* Resource Group: `um-prod-rg-appservice`
* App Name: `um-prod-worker-process`

---

## ✅ 1. Build Go Binary

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker ./cmd/fitness-worker
```

---

## ✅ 2. Prepare Run Script

```bash
cp run-fitness-worker.sh run.sh
chmod +x run.sh fitness-worker
```

---

## ✅ 3. Create WebJob Folder Structure

```bash
rm -rf build worker.zip

mkdir -p build/App_Data/jobs/triggered/fitness-worker-job
```

---

## ✅ 4. Copy Required Files

```bash
cp fitness-worker run.sh build/App_Data/jobs/triggered/fitness-worker-job/

cp -r templates build/App_Data/jobs/triggered/fitness-worker-job/

cp -r chrome-linux64 build/App_Data/jobs/triggered/fitness-worker-job/ 2>/dev/null || true
```

---

## ✅ 5. Create ZIP (IMPORTANT)

```bash
cd build
zip -r ../worker.zip App_Data
cd ..
```

---

## ✅ 6. Verify ZIP Structure

```bash
unzip -l worker.zip
```

Expected:

```
App_Data/jobs/triggered/fitness-worker-job/run.sh
App_Data/jobs/triggered/fitness-worker-job/fitness-worker
```

---

## ✅ 7. Deploy to Azure

```bash
az webapp deployment source config-zip \
  --resource-group um-prod-rg-appservice \
  --name um-prod-worker-process \
  --src worker.zip
```

---

## ✅ 8. Run WebJob

```bash
az webapp webjob triggered run \
  --resource-group um-prod-rg-appservice \
  --name um-prod-worker-process \
  --webjob-name fitness-worker-job
```

---

## ✅ 9. (Optional) Restart App Service

```bash
az webapp restart \
  --name um-prod-worker-process \
  --resource-group um-prod-rg-appservice
```

---

## 🔥 Troubleshooting

### ❌ WebJob not visible

→ ZIP structure is wrong (missing `App_Data/...`)

### ❌ Job runs but exits immediately

→ Check `run.sh` permissions

### ❌ Binary not executing

→ Ensure:

```bash
chmod +x fitness-worker
```

---

## 💡 One-Line Deploy (Quick Mode)

```bash
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker ./cmd/fitness-worker && \
cp run-fitness-worker.sh run.sh && \
chmod +x run.sh fitness-worker && \
rm -rf build worker.zip && \
mkdir -p build/App_Data/jobs/triggered/fitness-worker-job && \
cp fitness-worker run.sh build/App_Data/jobs/triggered/fitness-worker-job/ && \
cp -r templates build/App_Data/jobs/triggered/fitness-worker-job/ && \
cp -r chrome-linux64 build/App_Data/jobs/triggered/fitness-worker-job/ 2>/dev/null || true && \
cd build && zip -r ../worker.zip App_Data && cd .. && \
az webapp deployment source config-zip \
  --resource-group um-prod-rg-appservice \
  --name um-prod-worker-process \
  --src worker.zip
```

---

## ✅ Done 🎉
