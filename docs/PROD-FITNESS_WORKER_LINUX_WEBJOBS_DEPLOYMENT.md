# PROD — Deploying `fitness-worker` to Linux App Service with WebJobs (Publish: Code)

**Guide:** **PROD** — App Service **`um-prod-worker-process`**. For **Staging**, see [Staging — fitness-worker Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md).

This guide mirrors [Staging — fitness-worker Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) (**Staging**) for **production**: a **triggered**, **scheduled** WebJob runs the Go worker in **run-once** mode each time the schedule fires.

**Target App Service (PROD worker):** **`um-prod-worker-process`**  
**Staging counterpart:** Same technical steps on your staging Linux Web App — [Staging — fitness-worker Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md).

**Related:** Production API and container naming patterns are described in [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md) (e.g. `um-prod-api`, resource groups). The worker is a **separate** App Service (`um-prod-worker-process`).

---

## PROD vs Staging (before you deploy)

| Topic | PROD (`um-prod-worker-process`) |
|--------|----------------------------------------|
| **Secrets** | Use **production** database, storage, and keys. Do **not** copy staging application settings verbatim. |
| **SQL firewall** | Allow **this** App Service’s **outbound IPs** (or your org’s approved pattern) against the **production** Azure SQL server. |
| **Scheduling** | Use an **NCRONTAB** schedule agreed for production (often less aggressive than staging). Default WebJob timezone is **UTC** unless your team configures otherwise. |
| **Tier** | Prefer **Basic (B1)** or higher with **Always On** for reliable scheduled WebJobs; **Free F1** does not support Always On and is unsuitable for dependable production cron. |
| **Change control** | Follow your release process (approvals, maintenance windows) before replacing `worker.zip` or changing app settings. |

---

## Linux WebJobs vs Windows (important)

| Topic | Linux |
|--------|--------|
| Entry point | Azure expects **`run.sh`** at the **ZIP root**. Source in repo: **`src/run-fitness-worker.sh`** — copy to **`run.sh`** when building the zip ([Step 2](#step-2-build-the-linux-binary-and-create-workerzip)). |
| Permissions | **`chmod +x run.sh fitness-worker`** before you zip (see [Assemble the WebJob folder](#assemble-the-webjob-folder-exact-layout)). |
| Scheduled reliability | Microsoft documents that **[Always On](https://learn.microsoft.com/en-us/azure/app-service/configure-common?tabs=portal#configure-general-settings)** should be enabled for **scheduled** WebJobs to run reliably, and that **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** must be **`false`** on Linux. **Free F1** does **not** support Always On — schedules may be **unreliable** when the site is cold; use **Basic (B1)** or higher for production cron behavior, or use **Manual** triggers while testing. |
| Chromium | Built-in Linux stacks do **not** include a full browser. You can try bundling **[Chrome for Testing](https://googlechromelabs.github.io/chrome-for-testing/)** (`linux64`), but missing system libraries on the App Service image is common. If PDF generation fails, prefer **[FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md)** (Linux container with `apt`-installed Chromium). |

Official references: [Tutorial: scheduled WebJob](https://learn.microsoft.com/en-us/azure/app-service/tutorial-webjobs), [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create).

**Custom Docker** Linux apps: WebJobs are **not** supported on **Alpine**-based container images (and some older Java stacks). This doc targets **Code**-published Linux Web Apps.

---

## PROD — You already have the App Service (Code, Linux)

Do the following **in this order** in [Azure Portal](https://portal.azure.com/) → search for and open **`um-prod-worker-process`** (**PROD** Linux Web App).

| Order | What to do | Where in this doc |
|------|------------|-------------------|
| 1 | Set **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`** (if not already). For reliable schedules, enable **Always On** (requires a tier that supports it, e.g. Basic+). | [Prerequisites](#prerequisites-on-the-app-service) |
| 2 | Add **Application settings** for **production** (DB, Azure Blob, fitness worker vars, optional `CHROMIUM_PATH`). **Save**. | [Step 1](#step-1-application-settings) |
| 3 | On your dev machine: build **`fitness-worker`** (Linux `amd64`), copy **`run-fitness-worker.sh`** → **`run.sh`**, **`templates/`**, optional Chromium folder → **`worker.zip`** (correct root layout). | [Step 2](#step-2-build-the-linux-binary-and-create-workerzip) |
| 4 | **WebJobs** → **+ Add** → upload `worker.zip` as **Triggered** + **Scheduled**. | [Step 3](#step-3-add-the-webjob-on-um-prod-worker-process) |
| 5 | **WebJobs** → your job → **Logs** / **Log stream** / **SSH** or Kudu → fix errors. | [Step 4](#step-4-logs-and-verification) |

You do **not** need to change **Deployment Center** for the main site unless that is how you deploy the WebJob; portal upload under **WebJobs** is enough.

---

## Prerequisites on the App Service

On **`um-prod-worker-process`**:

1. **Application settings** → add **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`** (required for Linux WebJobs per Microsoft).
2. **Configuration** → **General settings** → enable **Always On** if your **pricing tier** allows it. Recommended for **scheduled** jobs in production.
3. **Runtime stack** (e.g. .NET 8, Node, Python) only selects the **host image**; your WebJob runs **`run.sh`**, which starts the **Go** binary. You do **not** need the .NET runtime for the worker itself.

---

## Step 1: Application settings

**Settings** → **Configuration** → **Application settings** (or **Environment variables**) on **`um-prod-worker-process`** → add the same variables as in the [staging Linux guide](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-1-application-settings) and [Windows guide](./FITNESS_WORKER_WINDOWS_DEPLOYMENT.md), using **production** values only.

**Database (production)**

| Name | Example / notes |
|------|------------------|
| `DB_SERVER` | Production Azure SQL host, e.g. `your-prod-server.database.windows.net` |
| `DB_DATABASE_NAME` | Production database name |
| `DB_USER` | SQL login for production |
| `DB_PASSWORD` | Production SQL password (managed via your secret process) |
| `DB_ENCRYPT` | `true` for Azure SQL |
| `DB_TRUST_SERVER_CERT` | Set per your security policy (often `false` in production) |

**Azure Blob (production)**

| Name | Example / notes |
|------|------------------|
| `AZURE_STORAGE_ACCOUNT` | Production storage account |
| `AZURE_STORAGE_KEY` | Or connection string — production only |
| `AZURE_STORAGE_CONNECTION_STRING` | Optional |
| `AZURE_STORAGE_ENDPOINT_SUFFIX` | Usually `core.windows.net` |
| `AZURE_DIAGNOSTIC_REPORTS_CONTAINER` | Production container name (e.g. `diagnostic-reports`) |

**Fitness worker**

| Name | Value / notes |
|------|----------------|
| `FITNESS_CERT_TEMPLATE_DIR` | `./templates` |
| `FITNESS_CERT_PENDING_LEAD_STATUS_ID` | Production IDs (may match or differ from staging — confirm with your data model) |
| `FITNESS_CERT_DONE_LEAD_STATUS_ID` | Same |
| `FITNESS_CERT_POLL_INTERVAL_SECONDS` | e.g. `120` (mainly for non–run-once) |
| `FITNESS_CERT_BATCH_SIZE` | e.g. `10` — tune for production load |
| `FITNESS_CERT_WORKER_RUN_ONCE` | `true` — **required** for scheduled WebJob (each run processes a batch and exits) |
| `FITNESS_CERT_WORKER_USER_ID` | Non-negative integer, e.g. `0` |
| `CHROMIUM_PATH` | e.g. `./chrome-linux64/chrome` if you bundle **Chrome for Testing** **linux64** next to the binary; or an **absolute** path under `/home/site/wwwroot/...` if you upload Chromium separately |

**Save**. Allow the app to restart if prompted.

**SQL firewall:** Allow **`um-prod-worker-process`** outbound IPs (or “Allow Azure services” if that matches your policy) so the worker can reach the **production** database.

---

## Step 2: Build the Linux binary and create `worker.zip`

Go module root is **`src/`** (where `go.mod` lives). Steps are identical to the [staging document](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-2-build-the-linux-binary-and-create-workerzip); summary below.

### Build static Linux `amd64` binary

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker ./cmd/fitness-worker
```

### Entry script: `run-fitness-worker.sh` → **`run.sh`** at zip root

Same as [staging](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#entry-script-run-fitness-worker-sh--runsh-at-zip-root): use **`src/run-fitness-worker.sh`**, copy to **`run.sh`** next to **`fitness-worker`** before zipping:

```bash
cp run-fitness-worker.sh run.sh
chmod +x run.sh
```

### Assemble the WebJob folder (exact layout)

In one **empty folder** (this becomes the **root inside** `worker.zip`):

1. Copy **`fitness-worker`** to the **root**.
2. Copy **`src/run-fitness-worker.sh`** → **`run.sh`** at the **root**.
3. Copy **`src/cmd/fitness-worker/templates`** → **`templates/`** (full tree).
4. *(Optional)* Add unpacked **Chrome for Testing** **`linux64`** tree, e.g. **`chrome-linux64/`**.

```bash
cd /path/to/your/staging-folder
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

### Create `worker.zip` (root = contents of the folder, not the folder itself)

```bash
zip -r ../worker.zip run.sh fitness-worker templates/ chrome-linux64/
```

**Wrong:** zipping the parent folder so files end up as `folder/run.sh` inside the zip.

---

## Step 3: Add the WebJob on `um-prod-worker-process`

1. Portal → **`um-prod-worker-process`** → **WebJobs** → **+ Add**.
2. **Name:** e.g. `fitness-worker-job` (or your production naming standard).
3. **File upload:** `worker.zip`.
4. **Type:** **Triggered**.
5. **Triggers:** **Scheduled** (or **Manual** for a controlled production smoke test).
6. **CRON** — **NCRONTAB**, **six** fields: `{second} {minute} {hour} {day} {month} {day-of-week}`. Example: `0 */5 * * * *` = every 5 minutes at second 0 — use the schedule your team approved for production. See [NCRONTAB expressions](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create#ncrontab-expressions).
7. Save.

Default schedule timezone is **UTC** unless you configure time zone settings documented for App Service / Functions timer bindings.

---

## Step 4: Logs and verification

1. **`um-prod-worker-process`** → **WebJobs** → select the job → **Logs** (or **Log stream**).
2. Use **SSH** (Development Tools → **SSH** / **Advanced Tools**) if you need to inspect **`/home/site/wwwroot/App_Data/jobs/triggered/<jobname>/`** and confirm `run.sh`, `fitness-worker`, and `templates/` exist.
3. Confirm logs show DB and blob access, template resolution, and (if used) Chromium PDF steps without “executable not found” or missing shared library errors.

---

## Chromium on Linux App Service (WebJob)

Same options as the [staging Linux WebJob doc](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#chromium-on-linux-app-service-webjob): bundle **Chrome for Testing** (`linux64`), upload under `/home/site/wwwroot`, or move to a **container** deployment per [FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md) if headless PDF is unreliable.

---

## Troubleshooting

| Symptom | What to try |
|--------|-------------|
| WebJob does not run on schedule | Enable **Always On** (Basic+). Confirm **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**. On **Free F1**, cold start may delay or skip runs. |
| `Permission denied` on `run.sh` or `fitness-worker` | Re-create the zip after **`chmod +x run.sh fitness-worker`**. |
| `templates` not found | Confirm `templates/` is inside the zip at the **root** next to `fitness-worker`; confirm `FITNESS_CERT_TEMPLATE_DIR=./templates`. |
| Chromium fails / missing `.so` | Full `chrome-linux64` tree present; try **container** deployment with distro packages. |
| SQL / blob errors | **Production** application settings, firewall for **`um-prod-worker-process`**, container name, keys — align with [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md) patterns. |

---

## Optional: Continuous WebJob (paid tier)

With **Always On** and a tier that allows it, you can use a **Continuous** WebJob and set **`FITNESS_CERT_WORKER_RUN_ONCE=false`** so the worker loops with **`FITNESS_CERT_POLL_INTERVAL_SECONDS`**. You still need a working Chromium setup and the same application settings.

---

## PROD — Quick checklist

- [ ] Target app is **`um-prod-worker-process`** (**PROD** Linux).
- [ ] **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`** on the app.
- [ ] **Always On** enabled if you rely on a **schedule** (requires a supporting SKU).
- [ ] `worker.zip` root contains **`run.sh`** (from **`run-fitness-worker.sh`**), **`fitness-worker`**, and **`templates/`** (and optional `chrome-linux64/`).
- [ ] **`chmod +x`** applied to **`run.sh`** and **`fitness-worker`** before zipping.
- [ ] **`FITNESS_CERT_WORKER_RUN_ONCE=true`** for triggered scheduled batches.
- [ ] **`CHROMIUM_PATH`** set if the image has no usable browser on `PATH`.
- [ ] **Production** DB and Azure Blob application settings; SQL firewall allows this App Service.

---

## Related docs

- [Staging — Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) — same WebJob flow for **staging** (generic Linux Web App).
- [PROD — email-worker Linux WebJobs](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) — add **`email-worker`** as a **second** WebJob on **`um-prod-worker-process`** (or your PROD worker app).
- [FITNESS_WORKER_WINDOWS_DEPLOYMENT.md](./FITNESS_WORKER_WINDOWS_DEPLOYMENT.md) — Windows WebJob + Chrome for Testing **win64**.
- [FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md) — Linux **container** worker with Chromium via `apt`.
- [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md) — production API container deploy and Azure naming examples.
