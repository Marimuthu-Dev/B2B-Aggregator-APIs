#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Starting email-worker..."
exec ./email-worker
