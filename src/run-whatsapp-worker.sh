#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

# If running as an Azure triggered WebJob, default to single-batch mode so the job completes each scheduled run
if [ "${WEBJOBS_TYPE:-}" = "triggered" ] && [ -z "${WHATSAPP_WORKER_SINGLE_BATCH:-}" ]; then
  export WHATSAPP_WORKER_SINGLE_BATCH=true
fi

echo "Starting whatsapp-worker..."
exec ./whatsapp-worker
