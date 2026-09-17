#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

# Do NOT export LD_LIBRARY_PATH for the whole script — bundled Chrome deps (e.g. libselinux)
# can break host tools like mkdir with GLIBC mismatches. The Go worker sets LD_LIBRARY_PATH
# only on the Chrome child via chromedp (see internal/fitnesscert/html_pdf.go).

# Writable home + headless Chrome env (App Service has no D-Bus / desktop schemas).
mkdir -p /tmp/fitness-chrome-home
export HOME=/tmp/fitness-chrome-home
export DBUS_SESSION_BUS_ADDRESS="${DBUS_SESSION_BUS_ADDRESS:-/dev/null}"
export GSETTINGS_BACKEND="${GSETTINGS_BACKEND:-memory}"

if [ -d "./chrome-linux-deps/usr/share" ]; then
  export XDG_DATA_DIRS="$(pwd)/chrome-linux-deps/usr/share${XDG_DATA_DIRS:+:$XDG_DATA_DIRS}"
fi

echo "Starting fitness-worker..."
exec ./fitness-worker
