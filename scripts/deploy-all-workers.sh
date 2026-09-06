#!/usr/bin/env bash
# Build fitness-worker + email-worker + whatsapp-worker (Linux amd64), pack all three WebJobs under
# App_Data/jobs/triggered/, and optionally deploy with Azure CLI.
#
# Usage (from repo root):
#   ./scripts/deploy-all-workers.sh
#   OUTPUT_ZIP=/tmp/out.zip ./scripts/deploy-all-workers.sh
#   AZURE_RESOURCE_GROUP=um-prod-rg-appservice AZURE_WEBAPP_NAME=um-prod-worker-process \
#     ./scripts/deploy-all-workers.sh --deploy
#
# Prerequisites:
#   - Go toolchain
#   - zip(1)
#   - For --deploy: Azure CLI (az), logged in; RESOURCE_GROUP and WEBAPP_NAME set or see env vars above

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SRC="$REPO_ROOT/src"
BUILD="$REPO_ROOT/build/webjobs-combined"
OUTPUT_ZIP="${OUTPUT_ZIP:-$REPO_ROOT/build/worker-combined.zip}"

DO_DEPLOY=0
if [ "${1:-}" = "--deploy" ]; then
  DO_DEPLOY=1
fi

AZURE_RESOURCE_GROUP="${AZURE_RESOURCE_GROUP:-${AZURE_RG:-}}"
AZURE_WEBAPP_NAME="${AZURE_WEBAPP_NAME:-${AZURE_WEBAPP:-}}"

echo "=== Building Linux amd64 binaries (module: $SRC) ==="
cd "$SRC"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o fitness-worker  ./cmd/fitness-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o email-worker    ./cmd/email-worker
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o whatsapp-worker ./cmd/whatsapp-worker

FITNESS_JOB="$BUILD/App_Data/jobs/triggered/fitness-worker-job"
EMAIL_JOB="$BUILD/App_Data/jobs/triggered/email-worker-job"
WHATSAPP_JOB="$BUILD/App_Data/jobs/triggered/whatsapp-worker-job"

echo "=== Assembling WebJob folders (3 jobs: fitness, email, whatsapp) ==="
rm -rf "$BUILD"
mkdir -p "$FITNESS_JOB" "$EMAIL_JOB" "$WHATSAPP_JOB"

cp "$SRC/fitness-worker" "$SRC/run-fitness-worker.sh" "$FITNESS_JOB/"
mv "$FITNESS_JOB/run-fitness-worker.sh" "$FITNESS_JOB/run.sh"
cp -r "$SRC/templates" "$FITNESS_JOB/"

# Bundle Chrome for Testing when present (required for HTML→PDF on App Service).
# Place unpacked linux64 tree at src/chrome-linux64/ with a `chrome` binary inside.
# Set App Setting CHROMIUM_PATH=./chrome-linux64/chrome on the worker app.
CHROME_SRC="${CHROME_LINUX64_DIR:-$SRC/chrome-linux64}"
if [ -x "$CHROME_SRC/chrome" ] || [ -f "$CHROME_SRC/chrome" ]; then
  echo "Including Chromium from $CHROME_SRC"
  cp -a "$CHROME_SRC" "$FITNESS_JOB/chrome-linux64"
  chmod +x "$FITNESS_JOB/chrome-linux64/chrome" || true
else
  echo "WARNING: $CHROME_SRC/chrome not found — fitness PDF generation will fail on App Service."
  echo "         Download Chrome for Testing linux64 and unpack to src/chrome-linux64/"
fi

# Shared libraries for Chrome (App Service has no apt root). Build with:
#   ./scripts/bundle-chrome-linux-deps.sh
CHROME_DEPS="${CHROME_LINUX_DEPS_DIR:-$SRC/chrome-linux-deps}"
if [ -d "$CHROME_DEPS/usr" ] || [ -d "$CHROME_DEPS/lib" ]; then
  echo "Including Chrome deps from $CHROME_DEPS"
  cp -a "$CHROME_DEPS" "$FITNESS_JOB/chrome-linux-deps"
  # Never ship these — they require newer GLIBC than App Service WebJob hosts provide.
  find "$FITNESS_JOB/chrome-linux-deps" \( \
    -name 'libselinux.so*' -o -name 'libsystemd.so*' -o -name 'libudev.so*' \
  \) -delete 2>/dev/null || true
  if find "$FITNESS_JOB/chrome-linux-deps" \( -name 'libselinux.so*' -o -name 'libsystemd.so*' -o -name 'libudev.so*' \) | grep -q .; then
    echo "ERROR: dangerous libs still present under chrome-linux-deps"
    exit 1
  fi
  echo "chrome-linux-deps sanitized (no libselinux/libsystemd/libudev)"
else
  echo "WARNING: $CHROME_DEPS not found — Chrome may fail with missing .so on App Service."
  echo "         Run: ./scripts/bundle-chrome-linux-deps.sh"
fi

cp "$SRC/email-worker" "$SRC/run-email-worker.sh" "$EMAIL_JOB/"
mv "$EMAIL_JOB/run-email-worker.sh" "$EMAIL_JOB/run.sh"

cp "$SRC/whatsapp-worker" "$SRC/run-whatsapp-worker.sh" "$WHATSAPP_JOB/"
mv "$WHATSAPP_JOB/run-whatsapp-worker.sh" "$WHATSAPP_JOB/run.sh"

chmod +x \
  "$FITNESS_JOB/run.sh"   "$FITNESS_JOB/fitness-worker" \
  "$EMAIL_JOB/run.sh"     "$EMAIL_JOB/email-worker" \
  "$WHATSAPP_JOB/run.sh"  "$WHATSAPP_JOB/whatsapp-worker"

SETTINGS="$REPO_ROOT/deploy/linux-webjobs"
if [ -f "$SETTINGS/fitness-worker-job/settings.job" ]; then
  cp "$SETTINGS/fitness-worker-job/settings.job" "$FITNESS_JOB/settings.job"
  echo "Included $SETTINGS/fitness-worker-job/settings.job"
fi
if [ -f "$SETTINGS/email-worker-job/settings.job" ]; then
  cp "$SETTINGS/email-worker-job/settings.job" "$EMAIL_JOB/settings.job"
  echo "Included $SETTINGS/email-worker-job/settings.job"
fi
if [ -f "$SETTINGS/whatsapp-worker-job/settings.job" ]; then
  cp "$SETTINGS/whatsapp-worker-job/settings.job" "$WHATSAPP_JOB/settings.job"
  echo "Included $SETTINGS/whatsapp-worker-job/settings.job"
fi

echo "=== Creating zip: $OUTPUT_ZIP ==="
mkdir -p "$(dirname "$OUTPUT_ZIP")"
rm -f "$OUTPUT_ZIP"
cd "$BUILD"
zip -qr "$OUTPUT_ZIP" App_Data

echo "Done. Zip layout (first 60 lines):"
unzip -l "$OUTPUT_ZIP" | head -n 60

if [ "$DO_DEPLOY" -eq 1 ]; then
  if [ -z "$AZURE_RESOURCE_GROUP" ] || [ -z "$AZURE_WEBAPP_NAME" ]; then
    echo "Error: --deploy requires AZURE_RESOURCE_GROUP and AZURE_WEBAPP_NAME (or AZURE_RG / AZURE_WEBAPP)."
    exit 1
  fi
  echo "=== Deploying to $AZURE_WEBAPP_NAME (resource group: $AZURE_RESOURCE_GROUP) ==="
  cd "$REPO_ROOT"
  # Normalize to absolute path so Azure CLI finds the zip regardless of cwd.
  if command -v realpath >/dev/null 2>&1; then
    ZIP_ABS="$(realpath "$OUTPUT_ZIP")"
  else
    ZIP_ABS="$(cd "$(dirname "$OUTPUT_ZIP")" && pwd)/$(basename "$OUTPUT_ZIP")"
  fi
  az webapp deployment source config-zip \
    --resource-group "$AZURE_RESOURCE_GROUP" \
    --name "$AZURE_WEBAPP_NAME" \
    --src "$ZIP_ABS"
fi
