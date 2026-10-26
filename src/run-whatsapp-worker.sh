#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

# If running as an Azure triggered WebJob, default to single-batch mode so the job completes each scheduled run
if [ "${WEBJOBS_TYPE:-}" = "triggered" ] && [ -z "${WHATSAPP_WORKER_SINGLE_BATCH:-}" ]; then
  export WHATSAPP_WORKER_SINGLE_BATCH=true
fi

# If running as an Azure continuous WebJob, always use loop mode (RunLoop) so the worker keeps polling
if [ "${WEBJOBS_TYPE:-}" = "continuous" ]; then
  export WHATSAPP_WORKER_SINGLE_BATCH=false
  export WHATSAPP_WORKER_RUN_ONCE=false
fi

echo "Starting whatsapp-worker..."
exec ./whatsapp-worker
