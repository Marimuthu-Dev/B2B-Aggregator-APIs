# PROD — Deploying `email-worker` to Linux App Service with WebJobs (Publish: Code)

**Guide:** **PROD** — add the **`email-worker`** WebJob on the **same** Linux App Service that runs **`fitness-worker`** in production (e.g. **`um-prod-worker-process`**), following the same packaging model as [Staging — email-worker Linux WebJobs](./STAGING-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md).

**Target App Service (example):** **`um-prod-worker-process`** (align with [PROD — fitness-worker Linux WebJobs](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)).

**Staging counterpart:** [Staging — email-worker Linux WebJobs](./STAGING-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md).

**Conceptual reference:** [EMAIL_WORKER.md](./EMAIL_WORKER.md).

---

## PROD vs Staging (before you deploy)

| Topic | PROD |
|--------|------|
| **Secrets** | **Production** ACS connection string and **production** database credentials. Do **not** copy staging values. |
| **SQL firewall** | Allow **this** App Service’s **outbound IPs** against the **production** Azure SQL server. |
| **Scheduling** | **NCRONTAB** agreed for production (often less aggressive than staging). Default timezone is **UTC** unless configured otherwise. |
| **Tier** | **Basic (B1)** or higher with **Always On** for dependable scheduled WebJobs; **Free F1** is unsuitable for reliable production cron. |
| **Second WebJob** | **WebJobs** → **+ Add** with a **new** name (e.g. **`email-worker-job`**) and **`email-worker.zip`** — does **not** require a second App Service. |

---

## Linux WebJobs (email-worker)

| Topic | Linux |
|--------|--------|
| Entry point | **`run.sh`** at ZIP root; source **`src/run-email-worker.sh`** — copy to **`run.sh`** when building the zip. |
| Permissions | **`chmod +x run.sh email-worker`** before zipping. |
| Reliability | **Always On** + **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**. |
| Templates / Chromium | **Not used** by email-worker. |

Official references: [Tutorial: scheduled WebJob](https://learn.microsoft.com/en-us/azure/app-service/tutorial-webjobs), [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create).

**Custom Docker** Linux apps: WebJobs are **not** supported on **Alpine**-based container images. This doc targets **Code**-published Linux Web Apps.

---

## PROD — Steps on the worker App Service

In [Azure Portal](https://portal.azure.com/) → open **`um-prod-worker-process`** (or your **production** Linux worker Web App).

| Order | What to do | Where in this doc |
|------|------------|-------------------|
| 1 | **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**, **Always On** (supported SKU). | [Prerequisites](#prerequisites-on-the-app-service) |
| 2 | Add **production** **email** application settings (`ACS_*`, `EMAIL_*`). **Save**. DB vars usually exist from fitness. | [Step 1](#step-1-application-settings-production) |
| 3 | Build **`email-worker`** (Linux `amd64`), **`run-email-worker.sh`** → **`run.sh`** → **`email-worker.zip`**. | [Step 2](#step-2-build-the-linux-binary-and-create-email-workerzip) |
| 4 | **WebJobs** → **+ Add** → **`email-worker.zip`**, **Triggered** + **Scheduled**. | [Step 3](#step-3-add-the-email-worker-webjob-on-production) |
| 5 | **Logs** / **Log stream** / SQL verification. | [Step 4](#step-4-logs-and-verification) |

---

## Prerequisites on the App Service

On the **production** worker app:

1. **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`**.
2. **Always On** enabled where the SKU allows it.
3. Runtime stack only defines the host; the WebJob runs **`run.sh`** and the Go binary.

---

## Step 1: Application settings (production)

**Configuration** → **Application settings** → add **production** values. **Database** entries should already match [PROD fitness Step 1](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-1-application-settings) if **`fitness-worker`** is deployed on the same app.

**Email worker (production)**

| Name | Notes |
|------|--------|
| `ACS_CONNECTION_STRING` | **Production** ACS resource — **Keys** blade; store as secret / Key Vault reference if your policy requires. |
| `EMAIL_WORKER_SINGLE_BATCH` | **`true`** for **Triggered** + **Scheduled** WebJob |
| `EMAIL_BATCH_SIZE` | Tune for production queue depth |
| `EMAIL_SEND_TIMEOUT_SECONDS` | e.g. `60` |
| `ACS_SENDER_DISPLAY_NAME` | Optional; align with **Mail From** display name (Portal or `az communication email domain sender-username update`) |
| `ACS_EMAIL_API_VERSION` | Optional |

**Logging**

| Name | Notes |
|------|--------|
| `LOG_DIR` / `LOG_RETENTION_HOURS` | Optional; same pattern as other workers |

**Save**. Allow restart.

**SQL firewall:** Allow **`um-prod-worker-process`** (or your app name) outbound IPs to the **production** database.

---

## Step 2: Build the Linux binary and create `email-worker.zip`

Go module root is **`src/`**.

### Build

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o email-worker ./cmd/email-worker
```

### `run-email-worker.sh` → **`run.sh`** (ZIP root)

Same as [staging email doc](./STAGING-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#entry-script-run-email-worker-sh--runsh-at-zip-root). From **`src/`**:

```bash
cp run-email-worker.sh run.sh
chmod +x run.sh email-worker
zip -r ../email-worker.zip run.sh email-worker
rm -f run.sh   # optional: remove generated run.sh after zipping
```

---

## Step 3: Add the email-worker WebJob on production

1. Portal → **your PROD worker Web App** → **WebJobs** → **+ Add**.
2. **Name:** e.g. **`email-worker-job`** (unique among WebJobs on this app).
3. **File upload:** **`email-worker.zip`**.
4. **Type:** **Triggered** → **Scheduled** (or **Manual** for a controlled smoke test).
5. **CRON:** production-approved **NCRONTAB** (six fields). Example: `0 */5 * * * *` = every 5 minutes — **adjust** per operations.
6. **Save**.

---

## Step 4: Logs and verification

1. **WebJobs** → **`email-worker-job`** → **Logs** / **Log stream**.
2. **SSH** → **`App_Data/jobs/triggered/email-worker-job/`** — verify **`run.sh`** and **`email-worker`**.
3. Monitor **`MediAdmin.tbl_Emails`** for **`IsSent`** transitions and ACS metrics if enabled.

---

## Troubleshooting

| Symptom | What to try |
|--------|-------------|
| Job never runs | **Always On**, **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**, CRON in UTC. |
| ACS / send errors | **Production** connection string; **Mail From** verified; **`FromAddress`** in DB. |
| Wrong DB | **Production** `DB_*` only; confirm slot / app is the worker app, not API. |

---

## Optional: Continuous WebJob

**Continuous** WebJob + **`EMAIL_WORKER_SINGLE_BATCH=false`** runs **`RunLoop`** until the process is recycled — use only if operations accept long-running processes on the worker app.

---

## PROD — Quick checklist

- [ ] **Production** worker App Service (e.g. **`um-prod-worker-process`**).
- [ ] **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**, **Always On** (supported SKU).
- [ ] **Production** `ACS_CONNECTION_STRING`, **`EMAIL_WORKER_SINGLE_BATCH=true`**.
- [ ] **`email-worker.zip`** with **`run.sh`** (from **`run-email-worker.sh`**) + **`email-worker`** at root; **`chmod +x`** before zip.
- [ ] WebJob **name** unique vs fitness job.
- [ ] **Production** SQL firewall for this app.

---

## Related docs

- [Staging — email-worker Linux WebJobs](./STAGING-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [PROD — fitness-worker Linux WebJobs](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [EMAIL_WORKER.md](./EMAIL_WORKER.md)
- [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md) — broader production naming patterns
