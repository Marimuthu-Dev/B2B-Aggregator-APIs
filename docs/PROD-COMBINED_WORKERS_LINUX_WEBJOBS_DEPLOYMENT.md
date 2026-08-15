# PROD — Combined Linux WebJobs: `fitness-worker` + `email-worker` (single deploy)

Deploy **both** triggered WebJobs in **one** package under `App_Data/jobs/triggered/`, so every release replaces **all** worker jobs together and nothing is left behind from an older single-job zip.

**Target App Service (example):** `um-prod-worker-process`  
**Single-job references:** [PROD — fitness-worker](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md), [PROD — email-worker](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) (application settings, troubleshooting).

**Reusable build + zip:** `scripts/deploy-all-workers.sh` in this repository (from repo root).

---

## Why one zip under `App_Data/jobs/triggered/`

| Approach | Behavior |
|----------|----------|
| **Portal → WebJobs → Add** (per job) | Each upload is its own zip; fine for small teams. |
| **Combined layout** (this doc) | One tree: `App_Data/jobs/triggered/<job-name>/run.sh` + binaries (+ `templates/` for fitness). **Redeploy always includes every job** so you never overwrite only one half of production. |

Official layout: [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create).

---

## Important: ZIP deploy replaces `wwwroot`

`az webapp deployment source config-zip` (and similar “deploy this zip to the app”) extracts the zip into **`/home/site/wwwroot`** and **removes existing files under `wwwroot` that are not in the new zip** (see [ZIP deploy](https://learn.microsoft.com/en-us/azure/app-service/deploy-zip)).

- **Dedicated worker app** (no real website under `wwwroot`): zipping **only** `App_Data/jobs/...` is normal; the app exists mainly to host WebJobs and settings.
- **If `wwwroot` must keep other files** (custom site, `robots.txt`, etc.): either include those paths in the same zip or use **Portal / Kudu** to upload the WebJob zips without wiping the rest of the site.

---

## Final folder layout (inside the zip)

```text
App_Data/
  jobs/
    triggered/
      fitness-worker-job/
        run.sh                 # from src/run-fitness-worker.sh
        fitness-worker         # Linux amd64 binary
        templates/             # required for fitness-worker (HTML templates)
        chrome-linux64/        # required for HTML→PDF (Chrome for Testing linux64)
        chrome-linux-deps/     # required on App Service Code (bundled .so libs; no root apt)
      email-worker-job/
        run.sh                 # from src/run-email-worker.sh
        email-worker           # Linux amd64 binary
```

Each job directory name (`fitness-worker-job`, `email-worker-job`) becomes the WebJob name in the portal and in the SCM API.

**Do not** copy both the binary and the script onto the same destination path (a common mistake). Copy **`run-fitness-worker.sh` → `run.sh`** and **`fitness-worker`** as **two separate files** inside the same job folder (same for email).

---

## Step 1 — Application settings (unchanged)

Configure **`um-prod-worker-process`** exactly as the two single-job guides describe (DB, blob, fitness vars, email / ACS vars, `WEBSITE_SKIP_RUNNING_KUDUAGENT=false`, **Always On** where supported):

- [PROD — fitness-worker — Step 1](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-1-application-settings)
- [PROD — email-worker — Step 1](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md#step-1-application-settings-production)

`ACS_CONNECTION_STRING` must be the full ACS string: `endpoint=https://...;accesskey=...` (see [EMAIL_WORKER.md](./EMAIL_WORKER.md)).

---

## Step 2 — Build, assemble, and zip

Go module root is **`src/`** (where `go.mod` lives).




### Option B — Manual commands

Run in **WSL/bash on your laptop** (repo on `/mnt/d/...`). Do **not** run this in Azure Kudu SSH or plain Windows PowerShell — there is no `src/templates` or local `chrome-linux64` there.

**Easier:** from repo root run `./scripts/deploy-all-workers.sh` (same layout).

Option A (recommended)
```
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
./scripts/deploy-all-workers.sh
```

Option B (manual) — preflight first

```
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs/src
pwd
ls templates chrome-linux64/chrome chrome-linux-deps

```

```bash
# WSL/bash — repo root
REPO_ROOT="/mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs"
SRC="$REPO_ROOT/src"
cd "$SRC"

# Preflight (must all succeed)
ls -la templates/certificate_1.html chrome-linux64/chrome run-fitness-worker.sh
ls -d chrome-linux-deps/usr chrome-linux-deps/lib 2>/dev/null || ls -d chrome-linux-deps/usr
# If chrome-linux-deps missing: (cd "$REPO_ROOT" && ./scripts/bundle-chrome-linux-deps.sh)

# Binaries
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker ./cmd/fitness-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o email-worker   ./cmd/email-worker

BUILD="$REPO_ROOT/build/webjobs-combined"
rm -rf "$BUILD"
FITNESS_JOB="$BUILD/App_Data/jobs/triggered/fitness-worker-job"
EMAIL_JOB="$BUILD/App_Data/jobs/triggered/email-worker-job"
mkdir -p "$FITNESS_JOB" "$EMAIL_JOB"

# Fitness job: binary + run.sh + templates + Chromium + deps
cp "$SRC/fitness-worker" "$SRC/run-fitness-worker.sh" "$FITNESS_JOB/"
mv "$FITNESS_JOB/run-fitness-worker.sh" "$FITNESS_JOB/run.sh"
cp -r "$SRC/templates" "$FITNESS_JOB/"
if [ ! -f "$SRC/chrome-linux64/chrome" ]; then
  echo "ERROR: $SRC/chrome-linux64/chrome missing."
  exit 1
fi
cp -a "$SRC/chrome-linux64" "$FITNESS_JOB/"
if [ ! -d "$SRC/chrome-linux-deps/usr" ] && [ ! -d "$SRC/chrome-linux-deps/lib" ]; then
  echo "ERROR: $SRC/chrome-linux-deps missing. Run: $REPO_ROOT/scripts/bundle-chrome-linux-deps.sh"
  exit 1
fi
cp -a "$SRC/chrome-linux-deps" "$FITNESS_JOB/"

# Email job
cp "$SRC/email-worker" "$SRC/run-email-worker.sh" "$EMAIL_JOB/"
mv "$EMAIL_JOB/run-email-worker.sh" "$EMAIL_JOB/run.sh"

chmod +x "$FITNESS_JOB/run.sh" "$FITNESS_JOB/fitness-worker" "$FITNESS_JOB/chrome-linux64/chrome" \
  "$EMAIL_JOB/run.sh" "$EMAIL_JOB/email-worker"

mkdir -p "$REPO_ROOT/build"
cd "$BUILD"
zip -r "$REPO_ROOT/build/worker-combined.zip" App_Data
```

**Before zip deploy:** confirm the fitness job folder (or zip) contains `chrome-linux64/chrome` **and** `chrome-linux-deps/` (shared libraries). App Service Kudu users are **not root** — `apt install chromium` will always fail with Permission denied; do not rely on apt on the server.

App setting on the worker app:

| Setting | Value |
|--------|--------|
| `CHROMIUM_PATH` | `./chrome-linux64/chrome` |

`scripts/deploy-all-workers.sh` copies `src/chrome-linux64` and `src/chrome-linux-deps` when present. Build deps with `./scripts/bundle-chrome-linux-deps.sh` (once, or when Chrome libs change).

---

## Step 3 — Deploy the zip

`--src` / `--src-path` is resolved **relative to your shell’s current directory**, not relative to the repo. If you run the command from `build/webjobs-combined`, then `build/worker-combined.zip` incorrectly becomes `.../webjobs-combined/build/worker-combined.zip` and fails with **No such file or directory**.

**Safe options:**

1. **`cd` to the repository root** (parent of `build/`), then use a relative zip path:

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
az webapp deployment source config-zip \
  --resource-group um-prod-rg-appservice \
  --name um-prod-worker-process \
  --src build/worker-combined.zip
```

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
az webapp deployment source config-zip \
  --resource-group med-prod-rg-appservice \
  --name med-prod-worker-process \
  --src build/worker-combined.zip
```

3. **Use the deploy script** (`./scripts/deploy-all-workers.sh --deploy`), which passes an **absolute** path to Azure CLI.

Microsoft recommends **`az webapp deploy`** instead of the deprecated `config-zip` (same `wwwroot` replace behavior for `type zip`):

```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
az webapp deploy \
  --resource-group um-prod-rg-appservice \
  --name um-prod-worker-process \
  --src-path build/worker-combined.zip \
  --type zip
```


```bash
cd /mnt/d/Code/MMK_Projects/B2B-Diagnostic-Aggregator/GitHub/B2B-Aggregator-APIs
az webapp deploy \
  --resource-group med-prod-rg-appservice \
  --name med-prod-worker-process \
  --src-path build/worker-combined.zip \
  --type zip
```

Use your real resource group and app name if they differ.

### Portal

**Advanced Tools (Kudu)** → **Debug console** → upload/extract, or use **Deployment Center** if your team standardizes on that — as long as the result under **`/home/site/wwwroot/App_Data/jobs/triggered/`** matches the layout above.

---

## Step 4 — Schedules (`settings.job`) and first run

After the zip is deployed, each triggered job needs a **schedule** so Azure runs it on a timer (not only on manual trigger). On Linux App Service this is normally done with a file named **`settings.job`** sitting **next to** `run.sh` inside each job folder.

Microsoft describes this pattern here: [Develop and deploy WebJobs](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create) (see **NCRONTAB** and triggered WebJobs).

### NCRONTAB format (six fields, UTC by default)

Order: `{second} {minute} {hour} {day} {month} {day-of-week}`.

| Example | Meaning |
|--------|---------|
| `0 */10 * * * *` | At second **0**, every **10** minutes |
| `0 0 */2 * * *` | Every **2** hours at minute 0 |
| `0 0 9 * * *` | Every day at **09:00** UTC |

Tune for production load; see [NCRONTAB expressions](https://learn.microsoft.com/en-us/azure/app-service/webjobs-create#ncrontab-expressions).

### Option A — Include `settings.job` in the zip (recommended)

1. **File name** must be exactly **`settings.job`** (all lowercase).
2. **Location** in the deployed app (and therefore inside your zip):

   - `App_Data/jobs/triggered/fitness-worker-job/settings.job`
   - `App_Data/jobs/triggered/email-worker-job/settings.job`

3. **Minimal content** (same or different cron per job):

```json
{
  "schedule": "0 */10 * * * *"
}
```

4. **In this repository**, example files live under **`deploy/linux-webjobs/`**:

   - `deploy/linux-webjobs/fitness-worker-job/settings.job`
   - `deploy/linux-webjobs/email-worker-job/settings.job`

   **`scripts/deploy-all-workers.sh`** copies those into the zip automatically when they exist. Edit the JSON there (or add the files) to match your production cadence, rebuild `build/worker-combined.zip`, and redeploy.

5. **Manual zip build:** after assembling `build/webjobs-combined/App_Data/jobs/triggered/...`, copy `settings.job` next to each `run.sh`, then zip again.

### Option B — Add or fix schedule in Azure Portal

1. Open [Azure Portal](https://portal.azure.com/) → your App Service (**`um-prod-worker-process`**).
2. In the left menu, open **WebJobs** (under **Settings** or **Development Tools**, depending on portal version).
3. Open **`fitness-worker-job`** (or **`email-worker-job`**).
4. If the UI offers **Schedule** / **NCRONTAB**, set the six-field expression and **Save**.  
   If the job was created only by dropping files and no schedule appears, use **Option A** or **Option C**.

### Option C — Edit on the server (Kudu)

1. Portal → your app → **Development Tools** → **Advanced Tools (Go)** → **Debug console** → **Bash**.
2. Go to `site/wwwroot/App_Data/jobs/triggered/fitness-worker-job/` (and the email job folder).
3. Create or edit **`settings.job`** with the JSON above, **Save**.
4. Restart the WebJob or the app if the schedule does not pick up immediately.

**Why prefer Option A:** the next **`config-zip` / `az webapp deploy`** replaces `wwwroot`. If `settings.job` exists only in the portal or only in Kudu, it can be **removed** on the next deploy unless it is **inside the zip** again.

### First run sanity check

- Trigger once manually: [Run a job on demand (SCM API)](#run-a-job-on-demand-scm-api) or Portal → WebJobs → **Run**.
- Confirm application settings: **`FITNESS_CERT_WORKER_RUN_ONCE=true`**, **`EMAIL_WORKER_SINGLE_BATCH=true`** for batch WebJob runs (see [fitness](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) and [email](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md) docs).

---

## Run a job on demand (SCM API)

Replace `{user}` and `{pass}` with deployment credentials (**Portal → App Service → Deployment Center / Deployment credentials**), or use a token your org allows.

```bash
curl -X POST -u '{user}:{pass}' \
  "https://um-prod-worker-process.scm.azurewebsites.net/api/triggeredwebjobs/fitness-worker-job/run"

curl -X POST -u '{user}:{pass}' \
  "https://um-prod-worker-process.scm.azurewebsites.net/api/triggeredwebjobs/email-worker-job/run"
```

---

## Checklist

- [ ] **`WEBSITE_SKIP_RUNNING_KUDUAGENT=false`**, **Always On** (supported SKU) on the worker app.
- [ ] All **fitness** and **email** application settings present ([fitness](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md), [email](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)).
- [ ] Combined zip contains **both** job folders; fitness folder includes **`templates/`** and **`chrome-linux64/chrome`**.
- [ ] App setting **`CHROMIUM_PATH=./chrome-linux64/chrome`** on the worker app.
- [ ] **`chmod +x`** on both `run.sh` files and both binaries before zipping (the script does this).
- [ ] **`FITNESS_CERT_WORKER_RUN_ONCE=true`** and **`EMAIL_WORKER_SINGLE_BATCH=true`** for scheduled batch runs.
- [ ] **`ACS_CONNECTION_STRING`** uses `endpoint=...;accesskey=...`.
- [ ] Understand **`config-zip`** impact on **`wwwroot`** (see above).
- [ ] Each job has **`settings.job`** in the zip (or you accept re-adding schedule after each deploy — see [Step 4](#step-4--schedules-settingsjob-and-first-run)).

---

## Related

- [PROD — fitness-worker Linux WebJobs](./PROD-FITNESS_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [PROD — email-worker Linux WebJobs](./PROD-EMAIL_WORKER_LINUX_WEBJOBS_DEPLOYMENT.md)
- [DEPLOYMENT-PROD.md](./DEPLOYMENT-PROD.md)
