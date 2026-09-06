#!/usr/bin/env bash
set -euo pipefail

# Run from the Go module directory: .../B2B-Aggregator-APIs/src
#
# Builds and deploys a COMBINED WebJob zip containing all 3 workers as separate triggered jobs:
#   App_Data/jobs/triggered/fitness-worker-job
#   App_Data/jobs/triggered/email-worker-job
#   App_Data/jobs/triggered/whatsapp-worker-job  <-- 3rd new job
#
# Mode selection (first positional arg, optional — defaults to 'az' the classic mode):
#
#   ./deploy-webjob.sh [az|kudu|async]
#
#     az      → az webapp deploy (--timeout applied). Can still timeout on B1 with huge zip,
#               but with 30m timeout that is rare.
#     kudu    → push via SCM zipdeploy API directly (never times out waiting for status).
#               Uses basic publish profile credentials (user-level or app-level). Best for B1/South India.
#     async   → az webapp deploy with --async. Returns immediately; then runs its own Kudu-based loop
#               checking deployment status for up to 45m.
#
# Env overrides:
#   AZURE_RG            resource group            (default: um-staging-rg-appservice)
#   AZURE_WEBAPP        webapp name               (default: um-staging-worker-process)
#   DEPLOY_MODE         az|kudu|async             (default: az; also set by $1)
#   DEPLOY_TIMEOUT_S    seconds for 'az' mode     (default: 1800 = 30 minutes)
#   KUDU_USERNAME       for 'kudu' / 'async'      (PublishingUser)
#   KUDU_PASSWORD       for 'kudu' / 'async'      (PublishingPassword / userPWD)
#
# To get KUDU_USERNAME / KUDU_PASSWORD for 'kudu' or 'async':
#   az webapp deployment list-publishing-profiles \
#       --resource-group med-prod-rg-appservice \
#       --name med-prod-worker-process \
#       --xml \
#       --query "[?publishMethod=='MSDeploy'].{userName:userName,userPWD:userPWD}" -o json
# (Or download the PublishProfile XML from the Portal and pick the MSDeploy profile.)

REPO_ROOT="$(cd .. && pwd)"
SRC="$(pwd)"
BUILD="$REPO_ROOT/build/webjobs-combined"
ZIP_OUT="$REPO_ROOT/build/worker-combined.zip"

DEPLOY_MODE="${1:-az}"
case "$DEPLOY_MODE" in
  az|kudu|async) : ;;
  *)
    echo "Usage: $0 [az|kudu|async]"
    echo "  az    → az webapp deploy (default, with long --timeout)"
    echo "  kudu  → direct SCM zipdeploy (never wait-times out for very large zip)"
    echo "  async → az webapp deploy --async then poll Kudu status up to 45m"
    exit 2
    ;;
esac

echo "0. Preflight — running from Go module root (src/)"
if [ ! -f go.mod ] || [ ! -f run-fitness-worker.sh ] || [ ! -f run-email-worker.sh ] || [ ! -f run-whatsapp-worker.sh ]; then
  echo "ERROR: run this script from src/ (where go.mod and run-*-worker.sh live)."
  exit 1
fi

AZURE_RG="${AZURE_RG:-um-staging-rg-appservice}"
AZURE_WEBAPP="${AZURE_WEBAPP:-um-staging-worker-process}"
DEPLOY_TIMEOUT_S="${DEPLOY_TIMEOUT_S:-1800}"
SCM_BASE="https://${AZURE_WEBAPP}.scm.azurewebsites.net"

echo "1. Building Linux amd64 binaries for all 3 workers"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker  ./cmd/fitness-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o email-worker    ./cmd/email-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o whatsapp-worker ./cmd/whatsapp-worker

echo "2. Assembling combined WebJob layout"
rm -rf "$BUILD"
FITNESS_JOB="$BUILD/App_Data/jobs/triggered/fitness-worker-job"
EMAIL_JOB="$BUILD/App_Data/jobs/triggered/email-worker-job"
WHATSAPP_JOB="$BUILD/App_Data/jobs/triggered/whatsapp-worker-job"
mkdir -p "$FITNESS_JOB" "$EMAIL_JOB" "$WHATSAPP_JOB"

# Fitness job
cp fitness-worker run-fitness-worker.sh "$FITNESS_JOB/"
mv "$FITNESS_JOB/run-fitness-worker.sh" "$FITNESS_JOB/run.sh"
cp -r templates "$FITNESS_JOB/"
if [ -f chrome-linux64/chrome ]; then
  cp -a chrome-linux64 "$FITNESS_JOB/"
fi
if [ -d chrome-linux-deps/usr ] || [ -d chrome-linux-deps/lib ]; then
  cp -a chrome-linux-deps "$FITNESS_JOB/"
  find "$FITNESS_JOB/chrome-linux-deps" \( \
    -name 'libselinux.so*' -o -name 'libsystemd.so*' -o -name 'libudev.so*' \
  \) -delete 2>/dev/null || true
fi

# Email job
cp email-worker run-email-worker.sh "$EMAIL_JOB/"
mv "$EMAIL_JOB/run-email-worker.sh" "$EMAIL_JOB/run.sh"

# WhatsApp job (3rd worker)
cp whatsapp-worker run-whatsapp-worker.sh "$WHATSAPP_JOB/"
mv "$WHATSAPP_JOB/run-whatsapp-worker.sh" "$WHATSAPP_JOB/run.sh"

chmod +x \
  "$FITNESS_JOB/run.sh"   "$FITNESS_JOB/fitness-worker" \
  "$EMAIL_JOB/run.sh"     "$EMAIL_JOB/email-worker" \
  "$WHATSAPP_JOB/run.sh"  "$WHATSAPP_JOB/whatsapp-worker"
if [ -f "$FITNESS_JOB/chrome-linux64/chrome" ]; then
  chmod +x "$FITNESS_JOB/chrome-linux64/chrome"
fi

# Schedules
SETTINGS="$REPO_ROOT/deploy/linux-webjobs"
for name in fitness-worker-job email-worker-job whatsapp-worker-job; do
  s="$SETTINGS/$name/settings.job"
  if [ -f "$s" ]; then
    cp "$s" "$BUILD/App_Data/jobs/triggered/$name/settings.job"
    echo "   + schedule $name/settings.job <- $s"
  else
    echo "   ! missing $s — $name has no schedule; add one or set via Portal"
  fi
done

echo "3. Zipping -> $ZIP_OUT"
mkdir -p "$(dirname "$ZIP_OUT")"
rm -f "$ZIP_OUT"
( cd "$BUILD" && zip -qr "$ZIP_OUT" App_Data )
ZIP_SIZE_MB=$(du -m "$ZIP_OUT" | awk '{print $1}')
echo "   zip size: ~${ZIP_SIZE_MB} MB"
if [ "$ZIP_SIZE_MB" -gt 200 ]; then
  echo "   WARNING: zip is > 200 MB. If you are on B1/South India and see 'UnknownDeploymentError / Timeout'"
  echo "            reached while tracking deployment status' — use ./deploy-webjob.sh kudu"
  echo "            (or async). That error means the CLI wait-timer expired, but the deployment is still"
  echo "            running in the background (check az webapp log deployment show)."
fi

echo "4. Combined zip preview (first 60 lines):"
unzip -l "$ZIP_OUT" | head -n 60

if [ -f "$ZIP_OUT" ]; then
  if command -v realpath >/dev/null 2>&1; then
    ZIP_ABS="$(realpath "$ZIP_OUT")"
  else
    ZIP_ABS="$(cd "$(dirname "$ZIP_OUT")" && pwd)/$(basename "$ZIP_OUT")"
  fi
else
  echo "ERROR: zip file missing at $ZIP_OUT"
  exit 1
fi

resolve_kudu_creds() {
  if [ -n "${KUDU_USERNAME:-}" ] && [ -n "${KUDU_PASSWORD:-}" ]; then
    return 0
  fi
  if ! command -v az >/dev/null 2>&1; then
    echo "ERROR: kudu/async mode needs KUDU_USERNAME+KUDU_PASSWORD or az CLI installed to fetch publish profile."
    return 1
  fi
  echo "   (kudu mode: fetching publish-profile credentials from Azure for $AZURE_WEBAPP…)"
  local xml out
  xml=$(az webapp deployment list-publishing-profiles \
    --resource-group "$AZURE_RG" --name "$AZURE_WEBAPP" --xml 2>/dev/null)
  out=$(python3 - "$xml" <<'PY' || true
import sys, re, xml.etree.ElementTree as ET
xml = sys.argv[1] if len(sys.argv) > 1 else sys.stdin.read()
root = ET.fromstring(xml)
for p in root.findall("publishProfile"):
    if p.attrib.get("publishMethod") == "MSDeploy":
        u = p.attrib.get("userName") or ""
        w = p.attrib.get("userPWD") or ""
        if "$" in u:
            u = u.split("$",1)[0]
        print(u + "||" + w)
        raise SystemExit(0)
PY
)
  if [ -n "$out" ]; then
    KUDU_USERNAME="${out%%||*}"
    KUDU_PASSWORD="${out##*||}"
    export KUDU_USERNAME KUDU_PASSWORD
    return 0
  fi
  echo "ERROR: unable to auto-fetch MSDeploy creds for $AZURE_WEBAPP; set KUDU_USERNAME + KUDU_PASSWORD."
  return 1
}

wait_for_kudu_deploy() {
  local max_wait_sec="${1:-2700}"    # 45 minutes
  local interval_sec="${2:-15}"
  local poll="$SCM_BASE/api/deployments/latest"
  local started_at=""
  started_at=$(date +%s)
  echo "   polling deployment status up to ${max_wait_sec}s (interval ${interval_sec}s): $poll"
  while true; do
    local now elapsed raw status
    now=$(date +%s)
    elapsed=$(( now - started_at ))
    if [ "$elapsed" -gt "$max_wait_sec" ]; then
      echo "ERROR: giving up after ${max_wait_sec}s. Visit $poll directly via browser (auth) to confirm status."
      return 1
    fi
    raw=$(curl -sSf -u "$KUDU_USERNAME:$KUDU_PASSWORD" "$poll" 2>/dev/null || true)
    if [ -z "$raw" ]; then
      sleep "$interval_sec"
      continue
    fi
    status=$(python3 - "$raw" <<'PY' 2>/dev/null || echo Unknown
import sys, json
o = json.loads(sys.argv[1])
m = o.get("message", "")
s = str(o.get("status", ""))
if o.get("complete") is True and o.get("success") is True:
    print("Success")
elif o.get("success") is False or "Failed" in s or "failed" in str(m).lower():
    print("Failed")
elif o.get("progress") is True or o.get("complete") is False:
    print("Running")
else:
    print(s if s else "Unknown")
PY
)
    echo "   [+${elapsed}s] status=$status"
    case "$status" in
      Success) return 0 ;;
      Failed)  echo "ERROR: deployment FAILED"; return 1 ;;
      *) : ;;
    esac
    sleep "$interval_sec"
  done
}

case "$DEPLOY_MODE" in
  az)
    echo "5. Deploying combined zip via 'az webapp deploy' (timeout ${DEPLOY_TIMEOUT_S}s) — RG=$AZURE_RG APP=$AZURE_WEBAPP"
    if ! az webapp deploy \
      --resource-group "$AZURE_RG" \
      --name "$AZURE_WEBAPP" \
      --src-path "$ZIP_ABS" \
      --target-path /home/site/wwwroot \
      --timeout "$DEPLOY_TIMEOUT_S" \
      --type zip
    then
      echo ""
      echo "Deploy step returned non-zero. This is often a CLI STATUS-TRACKING TIMEOUT (the upload itself completed)."
      echo "Run these to confirm the real state of your deployment:"
      echo "  az webapp log deployment show -n $AZURE_WEBAPP -g $AZURE_RG"
      echo "  az webapp deployment source list -n $AZURE_WEBAPP -g $AZURE_RG -o table"
      echo "  # If status is 'Running', wait 10-20 min and try the commands above again."
      exit 1
    fi
    ;;

  async)
    resolve_kudu_creds
    echo "5. Deploying combined zip via 'az webapp deploy --async' and polling Kudu status for up to 45m"
    az webapp deploy \
      --resource-group "$AZURE_RG" \
      --name "$AZURE_WEBAPP" \
      --src-path "$ZIP_ABS" \
      --target-path /home/site/wwwroot \
      --type zip \
      --async
    wait_for_kudu_deploy 2700 15 || exit $?
    ;;

  kudu)
    resolve_kudu_creds
    ZIPDEPLOY_URL="$SCM_BASE/api/zipdeploy?isAsync=true&trackDeployment=true"
    echo "5. Deploying combined zip via KUDU ZipDeploy (mode=kudu): $ZIPDEPLOY_URL"
    curl -u "$KUDU_USERNAME:$KUDU_PASSWORD" -sSf -X POST \
      --data-binary @"$ZIP_ABS" -H "Content-Type: application/octet-stream" \
      "$ZIPDEPLOY_URL" >/dev/null
    wait_for_kudu_deploy 2700 15 || exit $?
    ;;
esac

echo ""
echo "Deployment complete! You should see all 3 WebJobs in the Azure portal now:"
echo "   - fitness-worker-job"
echo "   - email-worker-job"
echo "   - whatsapp-worker-job  (NEW — 3rd job)"
echo ""
echo "Tip: override target with: AZURE_RG=... AZURE_WEBAPP=... $0 [az|kudu|async]"
echo "Tip (repo-root version): ../scripts/deploy-all-workers.sh --deploy (also supports 3 modes)"
