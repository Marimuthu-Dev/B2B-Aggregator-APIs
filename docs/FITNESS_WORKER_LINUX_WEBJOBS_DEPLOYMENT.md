# Deploying `fitness-worker` to Linux App Service with WebJobs (Publish: Code)

This guide mirrors [FITNESS_WORKER_WINDOWS_DEPLOYMENT.md](./FITNESS_WORKER_WINDOWS_DEPLOYMENT.md) for **Linux** App Service: a **triggered**, **scheduled** WebJob runs the Go worker in **run-once** mode each time the schedule fires.

## Linux WebJobs vs Windows (important)

| Topic | Linux |
|--------|--------|
| Entry point | Azure expects a script or runnable file at the **ZIP root**. For a compiled Go binary, include **`run.sh`** at the root that executes `./fitness-worker`. |
| Permissions | After unzip on the app, scripts and binaries must be executable: **`chmod +x run.sh fitness-worker`** before you zip (see [Assemble the WebJob folder](#assemble-the-webjob-folder-exact-layout)). |
| Scheduled reliability | Microsoft documents that **[Always On](https://learn.microsoft.com/en-us/azure/app-service/configure-common?tabs=portal#configure-general-settings)** should be enabled for **scheduled** WebJobs to run reliably, and that **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** must be **`false`** on Linux. **Free F1** does **not** support Always On — schedules may be **unreliable** when the site is cold; use **Basic (B1)** or higher for production cron behavior, or use **Manual** triggers while testing. |
| Chromium | Built-in Linux stacks do **not** include a full browser. You can try bundling **[Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/)** (`linux64`), but missing system libraries on the App Service image is common. If PDF generation fails, prefer **[FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md)** (Linux container with `apt`-installed Chromium). |

Official references: [Tutorial: scheduled WebJob](https://learn.microsoft.com/en-us/azure/app-service/tutorial-webjobs), [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create).

**Custom Docker** Linux apps: WebJobs are **not** supported on **Alpine**-based container images (and some older Java stacks). This doc targets **Code**-published Linux Web Apps.

---

## You already have the App Service (Code, Linux)

Do the following **in this order** in [Azure Portal](https://portal.azure.com/) → open your **Linux** Web App.

| Order | What to do | Where in this doc |
|------|------------|-------------------|
| 1 | Set **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`** (if not already). For reliable schedules, enable **Always On** (requires a tier that supports it, e.g. Basic+). | [Prerequisites](#prerequisites-on-the-app-service) |
| 2 | Add **Application settings** (DB, Azure Blob, fitness worker vars, optional `CHROMIUM_PATH`). **Save**. | [Step 1](#step-1-application-settings) |
| 3 | On your dev machine: build **`fitness-worker`** (Linux `amd64`), add **`run.sh`**, **`templates/`**, optional Chromium folder → **`worker.zip`** (correct root layout). | [Step 2](#step-2-build-the-linux-binary-and-create-workerzip) |
| 4 | **WebJobs** → **+ Add** → upload `worker.zip` as **Triggered** + **Scheduled**. | [Step 3](#step-3-add-the-webjob) |
| 5 | **WebJobs** → your job → **Logs** / **Log stream** / **SSH** or Kudu → fix errors. | [Step 4](#step-4-logs-and-verification) |

You do **not** need to change **Deployment Center** for the main site unless that is how you deploy the WebJob; portal upload under **WebJobs** is enough.

---

## Prerequisites on the App Service

1. **Application settings** → add **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`** (required for Linux WebJobs per Microsoft).
2. **Configuration** → **General settings** → enable **Always On** if your **pricing tier** allows it. Recommended for **scheduled** jobs.
3. **Runtime stack** (e.g. .NET 8, Node, Python) only selects the **host image**; your WebJob runs **`run.sh`**, which starts the **Go** binary. You do **not** need the .NET runtime for the worker itself.

---

## Step 1: Application settings

**Settings** → **Configuration** → **Application settings** (or **Environment variables**) → add the same variables as in the Windows guide. Use your real secrets.

**Database**

| Name | Example / notes |
|------|------------------|
| `DB_SERVER` | `yourserver.database.windows.net` |
| `DB_DATABASE_NAME` | Your database name |
| `DB_USER` | SQL login |
| `DB_PASSWORD` | SQL password |
| `DB_ENCRYPT` | `true` for Azure SQL |
| `DB_TRUST_SERVER_CERT` | Often `false` in production |

**Azure Blob**

| Name | Example / notes |
|------|------------------|
| `AZURE_STORAGE_ACCOUNT` | Storage account name |
| `AZURE_STORAGE_KEY` | Or connection string |
| `AZURE_STORAGE_CONNECTION_STRING` | Optional |
| `AZURE_STORAGE_ENDPOINT_SUFFIX` | Usually `core.windows.net` |
| `AZURE_DIAGNOSTIC_REPORTS_CONTAINER` | e.g. `diagnostic-reports` |

**Fitness worker**

| Name | Value / notes |
|------|----------------|
| `FITNESS_CERT_TEMPLATE_DIR` | `./templates` |
| `FITNESS_CERT_PENDING_LEAD_STATUS_ID` | e.g. `9` |
| `FITNESS_CERT_DONE_LEAD_STATUS_ID` | e.g. `10` |
| `FITNESS_CERT_POLL_INTERVAL_SECONDS` | e.g. `120` (mainly for non–run-once) |
| `FITNESS_CERT_BATCH_SIZE` | e.g. `10` |
| `FITNESS_CERT_WORKER_RUN_ONCE` | `true` — **required** for scheduled WebJob (each run processes a batch and exits) |
| `FITNESS_CERT_WORKER_USER_ID` | Non-negative integer, e.g. `0` |
| `CHROMIUM_PATH` | e.g. `./chrome-linux64/chrome` if you bundle [Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/) **linux64** next to the binary; or an **absolute** path under `/home/site/wwwroot/...` if you upload Chromium separately |

**Save**. Allow the app to restart if prompted.

**SQL firewall:** Allow the App Service outbound IPs (or “Allow Azure services”) so the worker can reach the database.

---

## Step 2: Build the Linux binary and create `worker.zip`

Go module root is **`src/`** (where `go.mod` lives).

### Build static Linux `amd64` binary

A **static** binary (`CGO_ENABLED=0`) avoids libc mismatches for the **worker executable** itself:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker ./cmd/fitness-worker
```

### Create `run.sh` (must be at ZIP root)

Create **`run.sh`** next to `fitness-worker`:

```bash
#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Attempting to install Chromium and necessary shared libraries..."
# We use || true so that if it's already installed or if updating repos fails,
# it won't completely crash the script because of the 'set -e' requirement above.
apt-get update -qq -y || true
apt-get install -qq -y chromium ca-certificates fonts-liberation libgbm1 libasound2 libatk-bridge2.0-0 libatk1.0-0 libnss3 lsb-release xdg-utils || true

echo "Starting fitness-worker..."
exec ./fitness-worker
```

`cd` to the script directory matches how the worker resolves **`./templates`** and relative **`CHROMIUM_PATH`** via the executable directory (see `resolveAgainstExecutableDir` in `src/internal/config/fitness_worker.go`).

### Assemble the WebJob folder (exact layout)

In one **empty staging folder** (this becomes the **root inside** `worker.zip`):

1. Copy **`fitness-worker`** to the **root** of staging.
2. Copy **`run.sh`** to the **root** of staging.
3. Copy **`src/cmd/fitness-worker/templates`** → staging as **`templates/`** (full tree).
4. *(Optional)* Add unpacked **Chrome for Testing** **`linux64`** tree, e.g. folder **`chrome-linux64/`** with the `chrome` launcher inside (see [Chromium on Linux App Service (WebJob)](#chromium-on-linux-app-service-webjob)).

Set execute bits **before** zipping:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
chmod +x run.sh fitness-worker
```

Expected layout:

```text
staging/
  run.sh
  fitness-worker
  templates/
    certificate_1.html
    ...
  chrome-linux64/          # optional
    chrome
    ...
```

### Create `worker.zip` (root = contents of `staging`, not `staging` itself)

Zip the **contents** of `staging` so **`run.sh`** is at the archive root (not `staging/run.sh`).

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs

# bundled Chromium:
zip -r ../worker.zip run.sh fitness-worker templates/ chrome-linux64/
```

**Wrong:** zipping the `staging` folder so files end up as `staging/run.sh` inside the zip.

---

## Step 3: Add the WebJob

1. Portal → your **Linux** Web App → **WebJobs** → **+ Add**.
2. **Name:** e.g. `fitness-worker-job`.
3. **File upload:** `worker.zip`.
4. **Type:** **Triggered**.
5. **Triggers:** **Scheduled** (or **Manual** for smoke tests).
6. **CRON** — **NCRONTAB**, **six** fields: `{second} {minute} {hour} {day} {month} {day-of-week}`. Example: `0 */5 * * * *` = every 5 minutes at second 0. See [NCRONTAB expressions](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create#ncrontab-expressions).
7. Save.

Default schedule timezone is **UTC** unless you configure time zone settings documented for App Service / Functions timer bindings.

---

## Step 4: Logs and verification

1. **WebJobs** → select the job → **Logs** (or **Log stream**).
2. Use **SSH** (Development Tools → **SSH** / **Advanced Tools**) if you need to inspect **`/home/site/wwwroot/App_Data/jobs/triggered/<jobname>/`** and confirm `run.sh`, `fitness-worker`, and `templates/` exist.
3. Confirm logs show DB and blob access, template resolution, and (if used) Chromium PDF steps without “executable not found” or missing shared library errors.

---

## Chromium on Linux App Service (WebJob)

The platform image does **not** include a full browser. Options:

### Option A: Bundle Chrome for Testing (`linux64`)

1. Open **[Chrome for Testing availability](https://googlechromelabs.github.io/chrome-for-testing/)**.
2. Under **Stable** (or your chosen channel), find **Binary** = **`chrome`**, **Platform** = **`linux64`**. Download and unpack.
3. Place the folder (commonly **`chrome-linux64`**) at the **same level** as `fitness-worker` in the WebJob zip.
4. Set **`CHROMIUM_PATH`** to a path **relative to the executable directory**, e.g. `./chrome-linux64/chrome` (or whatever path contains the `chrome` binary in your unpacked layout).

**Caveat:** The `chrome` binary still needs compatible **system libraries** on the App Service host. If you see errors about missing `.so` files, bundling the browser alone may be insufficient — use the **container** approach in [FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md).

### Option B: Upload once under `/home/site/wwwroot`

Upload the unpacked **`linux64`** tree via **SSH**, **FTPS**, or deployment, then set **`CHROMIUM_PATH`** to the **absolute** path to the `chrome` binary (e.g. under `/home/site/wwwroot/tools/...`).

### Option C: Prefer Docker for Chromium-heavy workloads

If headless PDF generation is flaky on Code + WebJob, use **[FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md)** (`apt`-installed Chromium in your own image).

---

## Troubleshooting

| Symptom | What to try |
|--------|-------------|
| WebJob does not run on schedule | Enable **Always On** (Basic+). Confirm **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**. On **Free F1**, cold start may delay or skip runs — upgrade tier or use **Manual** run to verify the package. |
| `Permission denied` on `run.sh` or `fitness-worker` | Re-create the zip after **`chmod +x run.sh fitness-worker`**. |
| `templates` not found | Confirm `templates/` is inside the zip at the **root** next to `fitness-worker`; confirm `FITNESS_CERT_TEMPLATE_DIR=./templates`. |
| Chromium fails / missing `.so` | Full `chrome-linux64` tree present; try **container** deployment with distro packages. |
| SQL / blob errors | Application settings, firewall, container name, keys — same as Windows doc. |

---

## Optional: Create a new Linux Web App

1. **Create a resource** → **Web App**.
2. **Publish:** **Code**. **OS:** **Linux**. **Runtime stack:** any supported stack (e.g. **.NET 8 (LTS)**) — it only defines the host; the WebJob runs **`run.sh`** and your Go binary.
3. **Pricing plan:** **Free F1** for experiments (no Always On); **Basic B1** or higher for **Always On** and more dependable **scheduled** WebJobs.
4. **Review + create** → **Create**.

---

## Optional: Continuous WebJob (paid tier)

With **Always On** and a tier that allows it, you can use a **Continuous** WebJob and set **`FITNESS_CERT_WORKER_RUN_ONCE=false`** so the worker loops with **`FITNESS_CERT_POLL_INTERVAL_SECONDS`**. You still need a working Chromium setup and the same application settings.

---

## Quick checklist

- [ ] **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`** on the Linux app.
- [ ] **Always On** enabled if you rely on a **schedule** (requires a supporting SKU).
- [ ] `worker.zip` root contains **`run.sh`**, **`fitness-worker`**, and **`templates/`** (and optional `chrome-linux64/`).
- [ ] **`chmod +x`** applied to **`run.sh`** and **`fitness-worker`** before zipping.
- [ ] **`FITNESS_CERT_WORKER_RUN_ONCE=true`** for triggered scheduled batches.
- [ ] **`CHROMIUM_PATH`** set if the image has no usable browser on `PATH`.
- [ ] DB and Azure Blob application settings configured; SQL firewall allows the app.

---

## Related docs

- [FITNESS_WORKER_WINDOWS_DEPLOYMENT.md](./FITNESS_WORKER_WINDOWS_DEPLOYMENT.md) — Windows WebJob + Chrome for Testing **win64**.
- [FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md) — Linux **container** worker with Chromium via `apt`.
