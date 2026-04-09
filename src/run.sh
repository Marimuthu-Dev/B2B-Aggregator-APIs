#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

echo "Attempting to install Chromium and necessary shared libraries..."
# We use || true so that if it's already installed or if updating repos fails,
# it won't completely crash the script because of the 'set -e' requirement above.
apt-get update -qq -y || true
apt-get install -qq -y chromium ca-certificates fonts-liberation libgbm1 libasound2 libatk-bridge2.0-0 libatk1.0-0 libnss3 lsb-release xdg-utils || true

echo "Starting fitness-worker..."
exec ./fitness-worker