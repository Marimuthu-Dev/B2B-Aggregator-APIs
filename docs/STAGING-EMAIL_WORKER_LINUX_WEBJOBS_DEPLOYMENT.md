# Staging — Deploying `email-worker` to Linux App Service with WebJobs (Publish: Code)

**Guide:** **Staging** — add a **second** WebJob on the **same** Linux Web App that already runs **`fitness-worker`** (or any worker), or use a dedicated staging worker app. For **PROD**, see [PROD — email-worker Linux WebJobs](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md).

This guide mirrors [Staging — fitness-worker Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md), but the **`email-worker`** package is **smaller** (no `templates/`, no Chromium): **`src/run-email-worker.sh`** → **`run.sh`** at zip root + **`email-worker`** binary.

**Conceptual reference:** [EMAIL_WORKER.md](./EMAIL_WORKER.md) (env vars, `MediAdmin.tbl_Emails`, ACS).

---

## Same App Service, second WebJob (important)

| Topic | Notes |
|--------|--------|
| **One App Service, multiple WebJobs** | **WebJobs** → **+ Add** creates a **separate** job with its **own** `*.zip`. The **fitness** job and **email** job run independently (different schedules, different folders under `App_Data/jobs/triggered/`). |
| **Shared configuration** | **Application settings** are **per App Service**. Variables you add for **`email-worker`** (e.g. `ACS_CONNECTION_STRING`) apply to **all** processes on that app. Existing **DB_*** settings from the fitness worker are **reused** — do **not** duplicate keys. |
| **Scheduled vs continuous** | For **Triggered** + **Scheduled** WebJobs (recommended, same pattern as fitness), set **`EMAIL_WORKER_SINGLE_BATCH=true`** so each run executes **one batch** and exits. |

Official references: [Tutorial: scheduled WebJob](https://learn.microsoft.com/en-us/azure/app-service/tutorial-webjobs), [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create).

**Custom Docker** Linux apps: WebJobs are **not** supported on **Alpine**-based container images. This doc targets **Code**-published Linux Web Apps.

---

## Linux WebJobs (email-worker)

| Topic | Linux |
|--------|--------|
| Entry point | Azure expects **`run.sh`** at the **ZIP root**. Source: **`src/run-email-worker.sh`** — **`cp run-email-worker.sh run.sh`** when assembling the zip. |
| Permissions | **`chmod +x run.sh email-worker`** before zipping. |
| Scheduled reliability | **[Always On](https://learn.microsoft.com/en-us/azure/app-service/configure-common?tabs=portal#configure-general-settings)** (Basic+) and **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**. **Free F1** has no Always On — schedules can be unreliable. |
| Chromium / templates | **Not required** for email-worker. |

---

## Staging — Steps (same App Service as fitness-worker)

In [Azure Portal](https://portal.azure.com/) → open your **staging** Linux Web App (the one that already has the fitness WebJob, if applicable).

| Order | What to do | Where in this doc |
|------|------------|-------------------|
| 1 | Confirm **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`** and **Always On** (if using a schedule). | [Prerequisites](#prerequisites-on-the-app-service) |
| 2 | Add **email-specific** application settings (`ACS_*`, `EMAIL_*`). **Save** (DB settings are usually already present from fitness). | [Step 1](#step-1-application-settings) |
| 3 | Build **`email-worker`** (Linux `amd64`), copy **`run-email-worker.sh`** → **`run.sh`** → **`email-worker.zip`**. | [Step 2](#step-2-build-the-linux-binary-and-create-email-workerzip) |
| 4 | **WebJobs** → **+ Add** → upload `email-worker.zip` as **Triggered** + **Scheduled** (different **name** than the fitness job). | [Step 3](#step-3-add-the-email-worker-webjob) |
| 5 | **WebJobs** → email job → **Logs** / **Log stream** → verify sends. | [Step 4](#step-4-logs-and-verification) |

---

## Prerequisites on the App Service

1. **`WEBSITE_SKIP_RUNNING_KUDUAGENT`** = **`false`** (required for Linux WebJobs).
2. **Always On** enabled if the pricing tier supports it (recommended for **scheduled** jobs).
3. Runtime stack (e.g. .NET) only selects the host image; the WebJob runs **`run.sh`** → **Go** binary.

---

## Step 1: Application settings

**Settings** → **Configuration** → **Application settings** → **add** the variables below (merge with existing **DB_*** settings — same as [fitness staging Step 1](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-1-application-settings)).

**Database (usually already set for `fitness-worker`)**

| Name | Notes |
|------|--------|
| `DB_SERVER` | Azure SQL host |
| `DB_DATABASE_NAME` | Database name |
| `DB_USER` / `DB_PASSWORD` | SQL login |
| `DB_ENCRYPT` | Typically `true` |
| `DB_TRUST_SERVER_CERT` | Per your policy |

**Email worker (new)**

| Name | Value / notes |
|------|----------------|
| `ACS_CONNECTION_STRING` | `endpoint=https://...communication.azure.com/;accesskey=...` from ACS **Keys** |
| `EMAIL_WORKER_SINGLE_BATCH` | **`true`** — **required** for **Triggered** + **Scheduled** WebJob (one batch per run, then exit) |
| `EMAIL_BATCH_SIZE` | e.g. `25` |
| `EMAIL_POLL_INTERVAL_SECONDS` | Used only if you switch to **Continuous** WebJob with **`EMAIL_WORKER_SINGLE_BATCH=false`** |
| `EMAIL_IDLE_WAIT_SECONDS` | Same as above |
| `EMAIL_SEND_TIMEOUT_SECONDS` | e.g. `60` |
| `ACS_EMAIL_API_VERSION` | Optional; default in code is `2023-03-31` |
| `ACS_SENDER_DISPLAY_NAME` | Optional; e.g. `UrMediConnect` — must align with Mail From display name in Azure Email / ACS |

**Logging (optional)**

| Name | Notes |
|------|--------|
| `LOG_DIR` | e.g. `logs` |
| `LOG_RETENTION_HOURS` | e.g. `24` |

**Save** and allow restart.

**SQL firewall:** Allow the App Service outbound IPs to reach Azure SQL (same as fitness worker).

**Note:** The email worker does **not** require Azure Blob settings unless you share them for other features on the same app.

---

## Step 2: Build the Linux binary and create `email-worker.zip`

Go module root is **`src/`** (where `go.mod` lives).

### Build static Linux `amd64` binary

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o email-worker ./cmd/email-worker
```

### Entry script: `run-email-worker.sh` → **`run.sh`** at zip root

The checked-in script is **`src/run-email-worker.sh`**. WebJobs expect the extracted entry file to be named **`run.sh`**.

```bash
cp run-email-worker.sh run.sh
chmod +x run.sh email-worker
```

### Assemble the WebJob folder

In one **empty folder** (this becomes the **root inside** `email-worker.zip`):

1. Copy **`email-worker`** to the **root**.
2. Copy **`src/run-email-worker.sh`** → **`run.sh`** at the **root** (or run **`cp run-email-worker.sh run.sh`** from **`src/`**).

Set execute bits **before** zipping:

```bash
chmod +x run.sh email-worker
```

Expected layout:

```text
email-staging/
  run.sh
  email-worker
```

### Create `email-worker.zip` (root = contents of the folder)

Zip the **contents** so **`run.sh`** is at the archive root:

```bash
cd /path/to/email-staging
zip -r ../email-worker.zip run.sh email-worker
```

**Wrong:** zipping the parent folder so paths become `email-staging/run.sh`.

Use a distinct name (**`email-worker.zip`**) so you do not overwrite the fitness **`worker.zip`**.

---

## Step 3: Add the email-worker WebJob

On the **same** Linux Web App where **`fitness-worker-job`** (or your naming) already exists:

1. **WebJobs** → **+ Add**.
2. **Name:** e.g. **`email-worker-job`** (must **differ** from the fitness WebJob name).
3. **File upload:** **`email-worker.zip`**.
4. **Type:** **Triggered**.
5. **Triggers:** **Scheduled** (or **Manual** for smoke tests).
6. **CRON** — **NCRONTAB**, **six** fields: `{second} {minute} {hour} {day} {month} {day-of-week}`. Example: `0 */2 * * * *` = every 2 minutes at second 0. Tune for staging load. See [NCRONTAB expressions](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create#ncrontab-expressions).
7. **Save**.

Default schedule timezone is **UTC** unless your team configures otherwise.

---

## Step 4: Logs and verification

1. **WebJobs** → **`email-worker-job`** → **Logs** or **Log stream**.
2. **SSH** / **Advanced Tools** → inspect **`/home/site/wwwroot/App_Data/jobs/triggered/email-worker-job/`** — confirm **`run.sh`** and **`email-worker`** exist and are executable.
3. Insert a test row into **`MediAdmin.tbl_Emails`** with **`IsSent = 0`**, valid **`FromAddress`** (verified in ACS Email), and **`ToAddress`** — confirm **`email sent`** in logs and **`IsSent = 1`** in SQL after success.

---

## Troubleshooting

| Symptom | What to try |
|--------|-------------|
| WebJob does not run on schedule | **Always On** (Basic+). **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**. |
| `ACS_CONNECTION_STRING is required` | Variable missing or typo in Application settings. |
| `Permission denied` on `run.sh` / `email-worker` | Re-zip after **`chmod +x`**. |
| Send fails / 401 / 403 | Verified **Mail From** domain; **`FromAddress`** in DB matches ACS; rotate keys if needed. |
| SQL errors | Same DB settings as API/fitness; firewall allows this App Service. |

---

## Optional: Continuous WebJob

Use **Continuous** type and set **`EMAIL_WORKER_SINGLE_BATCH=false`** so **`RunLoop`** runs until the host recycles the process — less common than **Triggered** + **Scheduled** with **`EMAIL_WORKER_SINGLE_BATCH=true`**.

---

## Staging — Quick checklist

- [ ] **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**, **Always On** if scheduled.
- [ ] **`ACS_CONNECTION_STRING`** and **`EMAIL_WORKER_SINGLE_BATCH=true`** on the App Service.
- [ ] **`email-worker.zip`** root: **`run.sh`** (from **`run-email-worker.sh`**), **`email-worker`**; **`chmod +x`** before zip.
- [ ] New WebJob **name** distinct from fitness (e.g. **`email-worker-job`**).
- [ ] SQL firewall; pending rows in **`tbl_Emails`**.

---

## Related docs

- [PROD — email-worker Linux WebJobs](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [Staging — fitness-worker Linux WebJobs](./STAGING-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [EMAIL_WORKER.md](./EMAIL_WORKER.md)
