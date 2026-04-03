# Deploying `fitness-worker` to Windows App Service (Publish: Code)

On **Free F1**, **Always On** is off, so the worker is best run as a **scheduled WebJob**: each run uses **run-once** mode (process one batch and exit) until the next schedule.

## You already have the App Service (Code, Windows, Free F1)

Do the following **in this order** in [Azure Portal](https://portal.azure.com/) → open your **Web App**.

| Order | What to do | Where in this doc |
|------|------------|-------------------|
| 1 | Add **Application settings** (DB, Azure Blob, fitness worker vars, optional `CHROMIUM_PATH`). **Save** and let the app restart. | [Step 1](#step-1-application-settings) |
| 2 | On your dev machine: build `fitness-worker.exe` and create `worker.zip` (`exe` + `templates/` at ZIP root). | [Step 2](#step-2-build-the-windows-exe-and-create-workerzip) |
| 3 | **WebJobs** → **+ Add** → upload `worker.zip` as a **Triggered** + **Scheduled** job. | [Step 3](#step-3-add-the-webjob) |
| 4 | Open **WebJobs** → your job → **Logs** (or Log stream / Kudu) and fix any connection or browser errors. | [Step 4](#step-4-logs-and-verification) |

You do **not** need to redeploy the main site or change **Deployment Center** for this pattern: the WebJob is added separately under **WebJobs**.

---

## Step-by-step walkthrough (WebJob + Chrome bundled in ZIP — Option A)

Follow these in order. Values like status IDs (`9` / `10`) must match **your** database.

### 1. Open your App Service in Azure Portal

[Azure Portal](https://portal.azure.com/) → **App Services** → select the **Windows** Web App that will run the job (Code publish is fine; runtime stack e.g. .NET only picks the OS image — your worker is a standalone `.exe`).

### 2. Add application settings (Configuration)

**Settings** → **Configuration** → **Application settings** (or **Environment variables** in the newer UI) → **+ New application setting** for each row below. Use your real secrets and endpoints.

**Database**

| Name | Example / notes |
|------|------------------|
| `DB_SERVER` | `yourserver.database.windows.net` |
| `DB_DATABASE_NAME` | Your database name |
| `DB_USER` | SQL login |
| `DB_PASSWORD` | SQL password |
| `DB_ENCRYPT` | `true` for Azure SQL |
| `DB_TRUST_SERVER_CERT` | Often `false` in production |

**Azure Blob (reports / certificates)**

| Name | Example / notes |
|------|------------------|
| `AZURE_STORAGE_ACCOUNT` | Storage account name |
| `AZURE_STORAGE_KEY` | Or use connection string instead |
| `AZURE_STORAGE_CONNECTION_STRING` | Optional if you prefer one string |
| `AZURE_STORAGE_ENDPOINT_SUFFIX` | Usually `core.windows.net` |
| `AZURE_DIAGNOSTIC_REPORTS_CONTAINER` | e.g. `diagnostic-reports` |

**Fitness worker**

| Name | Value / notes |
|------|----------------|
| `FITNESS_CERT_TEMPLATE_DIR` | `./templates` |
| `FITNESS_CERT_PENDING_LEAD_STATUS_ID` | e.g. `9` |
| `FITNESS_CERT_DONE_LEAD_STATUS_ID` | e.g. `10` |
| `FITNESS_CERT_POLL_INTERVAL_SECONDS` | e.g. `120` (mostly for non–run-once) |
| `FITNESS_CERT_BATCH_SIZE` | e.g. `10` |
| `FITNESS_CERT_WORKER_RUN_ONCE` | `true` — **required** for scheduled WebJob on Free F1 |
| `FITNESS_CERT_WORKER_USER_ID` | Non-negative integer (audit field), e.g. `0` |
| `CHROMIUM_PATH` | `chrome-win64\chrome.exe` — **only if** your ZIP contains that folder next to the exe (see step 5). If the folder name differs, set the path to **your** `chrome.exe` relative to the exe directory, or use a full path (Option B). |

Click **Save**. The app may restart.

**SQL firewall:** Allow the App Service to reach the database (outbound IPs on the Web App, or “Allow Azure services” on the SQL server/firewall rules).

### 3. Download Chrome for Testing (Windows x64)

1. Open [Chrome for Testing availability](https://googlechromelabs.github.io/chrome-for-testing/).
2. Choose a **Stable** (or known-good) build, platform **win64**, and download the **chrome** ZIP (the full browser), **not** chromedriver-only.
3. Extract the ZIP on your machine. You should get a folder (commonly `chrome-win64`) containing **`chrome.exe`** and many other files. **Do not delete subfolders** — Chrome needs the full tree.
4. If `chrome.exe` is nested (e.g. `chrome-win64\chrome-win64\chrome.exe`), note that path; your `CHROMIUM_PATH` must match (e.g. `chrome-win64\chrome-win64\chrome.exe`).

### 4. Build `fitness-worker.exe` (Windows amd64)

**From WSL/Linux/macOS** (Go module root is `src/` where `go.mod` lives):

```bash
cd /path/to/repo/src
GOOS=windows GOARCH=amd64 go build -o fitness-worker.exe ./cmd/fitness-worker
```

**From Windows** (run `build-windows.bat` in `src\cmd\fitness-worker\` — it writes `fitness-worker.exe` next to the script):

```bat
cd <repo>\src\cmd\fitness-worker
build-windows.bat
```

If you use the batch file, copy the built `fitness-worker.exe` into a **staging folder** where you will assemble the ZIP (next step). If you built in WSL, the `fitness-worker.exe` is already in `src/`.

### 5. Assemble the WebJob folder (exact layout)

In one **empty staging folder** (this will become the root inside `worker.zip`):

1. Place **`fitness-worker.exe`** at the **root** of that folder (not inside `bin\` or another wrapper folder).
2. Copy the **entire** repo folder `src/cmd/fitness-worker/templates` → staging folder as **`templates`** (so you have `templates\certificate_1.html`, images, etc.).
3. Copy the **entire** extracted Chrome folder (e.g. **`chrome-win64`**) next to the exe — same level as `templates`.

You should have:

```text
staging\
  fitness-worker.exe
  templates\
    certificate_1.html
    ...
  chrome-win64\
    chrome.exe
    ... (all other Chrome files)
```

### 6. Create `worker.zip` (root = contents of staging, not `staging` itself)

**Important:** The ZIP must expand so **`fitness-worker.exe`** is at the **root** of the archive.

- **Windows:** Select **only** the **contents** of `staging` (`fitness-worker.exe`, `templates`, `chrome-win64`) → **Send to** → **Compressed (zipped) folder** → rename to `worker.zip`.  
  **Wrong:** zipping the `staging` folder itself so the exe ends up at `staging\fitness-worker.exe` inside the zip.

- **WSL/Linux** (from inside `staging`):

```bash
cd /path/to/staging
zip -r ../worker.zip fitness-worker.exe templates/ chrome-win64/
```

Adjust `chrome-win64/` if your folder name differs.

### 7. Add the WebJob in Azure

1. Portal → your Web App → **WebJobs** → **+ Add**.
2. **Name:** e.g. `fitness-worker-job`.
3. **File upload:** `worker.zip`.
4. **Type:** **Triggered**.
5. **Triggers:** **Scheduled**.
6. **CRON** (NCRONTAB, **six** fields: `{second} {minute} {hour} {day} {month} {day-of-week}`): e.g. `0 */5 * * * *` = every 5 minutes at second 0. See [NCRONTAB expressions](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create#ncrontab-expressions). (`*/5` in the **first** field would mean every five **seconds**, which is usually not what you want.)
7. Save.

### 8. Run once and read logs

**WebJobs** → your job → **Run** (if available) or wait for the schedule → **Logs**.

Look for:

- Successful DB and blob access.
- `fitness cert: HTML to PDF via Chromium` with a path that resolves to your `chrome.exe`.
- No `cannot find the path` / missing `chrome.exe` / VC++ errors.

### 9. If something fails

| Problem | What to check |
|--------|----------------|
| `chrome.exe` not found | Open Kudu → find the WebJob extract path → confirm `chrome-win64\chrome.exe` (or your path) exists next to the exe. Fix ZIP layout or `CHROMIUM_PATH`. |
| Wrong CRON | Six fields; job never runs → fix expression in portal. |
| SQL / blob errors | Application settings, firewall, container name, keys. |

### 10. Updates later

- **Code change:** Rebuild `fitness-worker.exe`, rebuild `worker.zip`, **re-upload** the WebJob (replace).
- **Chrome security update:** Re-download Chrome for Testing, replace `chrome-win64` in staging, re-zip, re-upload.

---

## What you upload vs what you configure

| Item | Action |
|------|--------|
| `fitness-worker.exe` | Include in the WebJob ZIP at the **root** (not inside another folder). |
| `templates/` folder | Include **in full** next to the `.exe` — all HTML, images (e.g. seals, doctor details), and any assets referenced by those templates. The code resolves `./templates` relative to the executable directory (see `resolveFitnessWorkerPath` in `src/internal/config/fitness_worker.go`). |
| Database, Azure Blob, secrets | **Do not** put secrets in the ZIP. Add them as **Application settings** on the App Service (portal: **Configuration** or **Environment variables**). |
| Chrome / Chromium | The platform image does **not** include a browser. You cannot rely on a normal per-machine “install” like on a VM. Use one of the approaches in [Chrome / Chromium on Windows App Service](#chrome--chromium-on-windows-app-service) (bundle in the WebJob ZIP, or upload under `site/wwwroot` via Kudu). If that is not workable, prefer the **Linux container** worker in [FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md](./FITNESS_WORKER_APP_SERVICE_DEPLOYMENT.md). |

### ZIP layout (required)

After extraction, Azure must see this structure at the WebJob root:

```text
fitness-worker.exe
templates/
  certificate_1.html
  certificate_2.html
  ... (every file under src/cmd/fitness-worker/templates in your repo)
```

If you zip a parent directory by mistake (so the exe ends up at `myjob/fitness-worker.exe`), the WebJob may not start the binary correctly.

---

## Chrome / Chromium on Windows App Service

Azure App Service (Windows) is a **sandboxed** environment. You **do not** get Administrator rights to run the Google Chrome **installer (MSI/EXE)** the way you would on a plain Windows Server VM. Microsoft also does **not** ship Chrome in the default image.

What works in practice:

1. **Ship a browser as files** (portable layout): either inside the **WebJob ZIP** next to `fitness-worker.exe`, or under **`D:\home\site\wwwroot\...`** uploaded via **Kudu** / FTPS / ZIP deploy.
2. Point the worker at the browser binary with **`CHROMIUM_PATH`**.

For **triggered WebJobs**, the process **working directory** is not always the extracted job folder. The worker resolves a **relative** `CHROMIUM_PATH` (e.g. `chrome-win64\chrome.exe`) from the **same directory as `fitness-worker.exe`** (same rule as `FITNESS_CERT_TEMPLATE_DIR`), so Chrome bundled next to the exe in the ZIP works without hard-coding `D:\home\...`. Use an **absolute** path (Option B) when Chrome lives elsewhere, e.g. under `site\wwwroot`.

### Option A (recommended): Bundle Chrome in `worker.zip`

Use a **Windows x64** build intended for automation, for example **Chrome for Testing** ([Chrome for Testing availability](https://googlechromelabs.github.io/chrome-for-testing/)): download the **win64** **chrome** zip, unpack it locally, and place the folder **next to** `fitness-worker.exe` before zipping.

Example layout inside `worker.zip` (folder names match whatever the zip contains; adjust paths accordingly):

```text
fitness-worker.exe
templates/
  ...
chrome-win64/
  chrome.exe
  ... (other Chrome files — keep the full folder)
```

Then set an application setting:

```text
CHROMIUM_PATH=chrome-win64\chrome.exe
```

Use backslashes or forward slashes; Go accepts both on Windows. If your unpacked layout differs (e.g. nested `chrome-win64\chrome-win64\chrome.exe`), set `CHROMIUM_PATH` to the **actual** path to `chrome.exe` **relative to the WebJob root**.

**Size:** A full Chrome/Chromium folder is large (on the order of hundreds of MB). WebJob/deployment ZIPs are subject to an App Service limit (order of **2 GB** for zip-based uploads); bundling Chrome is usually still within that limit.

**Updates:** When you refresh Chrome for security, re-download, replace the folder in your zip, and re-upload the WebJob.

### Option B: Upload browser once under `site/wwwroot` (Kudu)

Use this if you want a **single** copy of Chrome shared by the site and not embedded in every WebJob zip.

1. Open the App Service → **Development Tools** → **Advanced Tools (Kudu)** → **Go**.
2. **Debug console** → **CMD** → navigate to `site\wwwroot`.
3. Create a folder such as `tools\chrome-for-testing` and upload the **entire unpacked** Chrome directory (drag-and-drop or zip upload then unzip in Kudu).
4. Note the full path to `chrome.exe`. Under Azure it is typically under **`D:\home\site\wwwroot\...`** (Kudu shows `D:\home` as the home drive).

Set **`CHROMIUM_PATH`** to that **full path**, for example:

```text
D:\home\site\wwwroot\tools\chrome-for-testing\chrome-win64\chrome.exe
```

(Your path must match where `chrome.exe` actually lives.)

`D:\home` is **persistent** across restarts; `D:\local` is **not** — do not install Chrome only under `D:\local` if you need it to survive recycling.

### Option C: Site extensions (not recommended)

Third-party **site extensions** that bundle Chrome have appeared in the past in the extension gallery. They are **not** a first-party Microsoft guarantee, may lag behind security updates, and can break with platform changes. Prefer **Option A** or **B**, or the **Linux container** worker doc.

### After setup: verify

1. In Kudu **CMD**, `cd` into the WebJob run folder or `site\wwwroot` and run `dir` to confirm `chrome.exe` exists at the path you configured.
2. Trigger the WebJob once (or wait for the schedule) and check **WebJobs** → **Logs** for errors such as executable not found or missing DLLs.

### Troubleshooting

| Symptom | What to try |
|--------|-------------|
| `chrome.exe` not found | Fix relative vs absolute `CHROMIUM_PATH`; confirm folder is inside the WebJob zip at the expected path, or under `site\wwwroot` for Option B. |
| Missing **VC++** / side-by-side / `VCRUNTIME` errors | The Windows App Service image usually includes the Visual C++ runtime; if a **very new** Chrome build still fails, try an older stable **Chrome for Testing** build or Chromium zip built for generic Windows x64. |
| Sandbox / launch failures | The worker already passes `NoSandbox` to `chromedp`; if failures persist, capture the exact log line and consider the Linux container deployment where Chromium is fully supported. |

---

## Step 1: Application settings

In the portal: your App Service → **Environment variables** (new UI) or **Settings** → **Configuration** → **Application settings** → **+ New application setting**.

Add everything the worker needs (see tables below). Use **`FITNESS_CERT_WORKER_RUN_ONCE` = `true`** so each scheduled WebJob run does one batch and exits (right for F1).

### Database (required)

```text
DB_SERVER
DB_DATABASE_NAME
DB_USER
DB_PASSWORD
DB_ENCRYPT                 # e.g. true
DB_TRUST_SERVER_CERT       # e.g. false
```

### Azure Blob (required)

```text
AZURE_STORAGE_ACCOUNT
AZURE_STORAGE_KEY                    # or rely on AZURE_STORAGE_CONNECTION_STRING
AZURE_STORAGE_CONNECTION_STRING      # optional alternative to account + key
AZURE_STORAGE_ENDPOINT_SUFFIX        # usually core.windows.net
AZURE_DIAGNOSTIC_REPORTS_CONTAINER   # e.g. diagnostic-reports
```

### Fitness worker (required)

```text
FITNESS_CERT_TEMPLATE_DIR=./templates
FITNESS_CERT_PENDING_LEAD_STATUS_ID   # e.g. 9
FITNESS_CERT_DONE_LEAD_STATUS_ID      # e.g. 10
FITNESS_CERT_POLL_INTERVAL_SECONDS    # e.g. 120 (mainly for non–run-once mode)
FITNESS_CERT_BATCH_SIZE               # e.g. 10
FITNESS_CERT_WORKER_RUN_ONCE=true     # required for scheduled WebJob on F1
FITNESS_CERT_WORKER_USER_ID           # integer ≥ 0 (audit user for DB updates)
CHROMIUM_PATH                         # full path to chrome.exe if PDF step fails without it
```

### Logging (optional)

```text
LOG_DIR
LOG_RETENTION_HOURS
```

**Save** / **Apply**. More detail and comments: see the expanded reference in [Application settings reference](#application-settings-reference-expanded) below.

**Firewall:** If SQL is Azure SQL, add the App Service **outbound IPs** (or “Allow Azure services”) so the worker can connect.

---

## Step 2: Build the Windows `.exe` and create `worker.zip`

From WSL or any machine with Go, build from the **Go module root** (`src/`, where `go.mod` lives):

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
GOOS=windows GOARCH=amd64 go build -o fitness-worker.exe ./cmd/fitness-worker
```

Package the **executable** and the **templates** directory from `cmd/fitness-worker/templates` (so the archive matches the layout above):

```bash
# From the same src/ directory — copies templates next to the exe for zipping
rm -rf ./templates
cp -r cmd/fitness-worker/templates ./templates
zip -r worker.zip fitness-worker.exe templates/
```

If you use [Option A](#option-a-recommended-bundle-chrome-in-workerzip) (Chrome inside the WebJob), unpack **Chrome for Testing** (win64) next to those files and include it in the zip, for example:

```bash
# Example only — path must match your unpacked Chrome folder name
zip -r worker.zip fitness-worker.exe templates/ chrome-win64/
```

On Windows, you can use `build-windows.bat` under `src/cmd/fitness-worker/` to build the exe into that folder, then zip `fitness-worker.exe`, the `templates` folder, and optionally the Chrome folder together.

**Comment — what this ZIP is:** The compiled binary and template assets (and optionally the full Chrome directory). No `go.mod`, no source code, no `.env` file.

---

## Step 3: Add the WebJob

1. App Service → **WebJobs** → **+ Add**.
2. **Name:** e.g. `fitness-worker-job`.
3. **File upload:** `worker.zip` from Step 2.
4. **Type:** **Triggered**.
5. **Triggers:** **Scheduled**.
6. **CRON expression (NCRONTAB, six fields):** e.g. `0 */5 * * * *` runs every 5 minutes at second 0. Adjust minute/hour fields as needed; the first field is **seconds**.
7. Confirm **OK** / **Create**.

Azure extracts the ZIP under the WebJob folder and can run `fitness-worker.exe` at the root when the schedule fires.

---

## Step 4: Logs and verification

1. **WebJobs** → select the job → **Logs** (or **Kudu** → `Debug console` → `LogFiles` for WebJob output).
2. Confirm no errors for: database connection, blob auth, **template directory not found**, or **Chromium / Chrome not found**.
3. If PDF generation fails with browser errors, fix `CHROMIUM_PATH` or install/bundle a browser compatible with your plan’s constraints.

---

## Application settings reference (expanded)

Same variables as **Step 1**, with comments. The API’s `JWT_SECRET`, `LOGIN_ENC_KEY`, etc. are **not** required for this worker.

### Database (required — worker talks to SQL Server)

```text
DB_SERVER                 # SQL host, e.g. yourserver.database.windows.net
DB_DATABASE_NAME          # Database name
DB_USER                   # SQL login user
DB_PASSWORD               # SQL login password (prefer Key Vault reference in production)
DB_ENCRYPT                # true/false — TLS to SQL
DB_TRUST_SERVER_CERT      # true/false — trust server certificate (often false in production)
```

Optional pool tuning (if you use the same keys as the API): `DB_POOL_MAX`, `DB_POOL_MIN`, `DB_IDLE_TIMEOUT`, `DB_CONN_MAX_LIFETIME_MS`.

### Azure Blob (required — fitness certificates upload to blob storage)

```text
AZURE_STORAGE_ACCOUNT              # Storage account name
AZURE_STORAGE_KEY                  # Account key (or use connection string below)
AZURE_STORAGE_CONNECTION_STRING    # Optional if you prefer connection string over account+key
AZURE_STORAGE_ENDPOINT_SUFFIX      # Usually core.windows.net
AZURE_DIAGNOSTIC_REPORTS_CONTAINER # Container name, e.g. diagnostic-reports
```

Optional (if your config expects them): `MOU_*`, `DIAGNOSTIC_REPORTS_*`, `AZURE_LEGAL_CONTAINER_NAME` — see `src/.env.example`.

### Fitness worker behaviour (required / important)

```text
FITNESS_CERT_TEMPLATE_DIR            # ./templates — resolved next to fitness-worker.exe
FITNESS_CERT_PENDING_LEAD_STATUS_ID  # e.g. 9
FITNESS_CERT_DONE_LEAD_STATUS_ID     # e.g. 10
FITNESS_CERT_POLL_INTERVAL_SECONDS   # Used in loop mode; run-once ignores between-schedule timing
FITNESS_CERT_BATCH_SIZE              # Rows per batch
FITNESS_CERT_WORKER_RUN_ONCE         # true = scheduled WebJob pattern on F1
FITNESS_CERT_WORKER_USER_ID          # Non-negative integer; audit field for DB updates
CHROMIUM_PATH                        # Full path to chrome.exe / chromium.exe if not on PATH
```

### Logging (optional)

```text
LOG_DIR                # Hourly log files directory; default logs
LOG_RETENTION_HOURS    # Retention for rotated logs
```

---

## Optional: Creating a new Web App (if you do not have one yet)

1. **Create a resource** → **Web App**.
2. **Publish:** **Code**. **OS:** **Windows**. **Runtime stack:** **.NET 8 (LTS)** (or another .NET Windows stack — only selects the worker image; your Go `.exe` is standalone).
3. **Pricing plan:** **Free F1** for the WebJob + run-once pattern.
4. **Review + create** → **Create**.

---

## Optional: paid plan — continuous worker instead of WebJob

If you move off **Free F1** and enable **Always On**, you can run the worker as a **Continuous** WebJob (or another process model) with `FITNESS_CERT_WORKER_RUN_ONCE=false` so it loops with `FITNESS_CERT_POLL_INTERVAL_SECONDS`. You still need the same ZIP layout, application settings, and a working Chrome/Chromium on the instance.

---

## Quick checklist

- [ ] `worker.zip` root contains `fitness-worker.exe` and `templates/` with **all** files from `src/cmd/fitness-worker/templates`.
- [ ] All DB and Azure Blob application settings are set.
- [ ] `FITNESS_CERT_WORKER_USER_ID` set (integer ≥ 0).
- [ ] `FITNESS_CERT_WORKER_RUN_ONCE=true` for scheduled triggered WebJobs.
- [ ] `CHROMIUM_PATH` points at `chrome.exe` (relative to WebJob root if bundled, or full path under `D:\home\site\wwwroot\...` if uploaded via Kudu) — see [Chrome / Chromium on Windows App Service](#chrome--chromium-on-windows-app-service).
