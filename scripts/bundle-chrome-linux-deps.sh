#!/usr/bin/env bash
# Download Ubuntu amd64 .deb packages Chrome needs and extract .so files into
# src/chrome-linux-deps/ for bundling with the Linux WebJob (LD_LIBRARY_PATH).
#
# App Service Code WebJobs often have older glibc than SSH reports (errors for
# GLIBC_2.29 / 2.30). Default suite is Ubuntu bionic (18.04, glibc 2.27).
#
# Usage (from repo root):
#   ./scripts/bundle-chrome-linux-deps.sh
# Then: ./scripts/deploy-all-workers.sh && redeploy.
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
OUT="$REPO_ROOT/src/chrome-linux-deps"
WORKDIR="$(mktemp -d)"
cleanup() { rm -rf "$WORKDIR"; }
trap cleanup EXIT

# bionic = glibc 2.27 — safer for older App Service hosts than focal/jammy.
SUITE="${CHROME_DEPS_SUITE:-bionic}"
MIRROR="${CHROME_DEPS_MIRROR:-http://archive.ubuntu.com/ubuntu}"
ARCH=amd64

# Seed only the libs Chrome typically misses on App Service. Do NOT seed systemd/udev/selinux.
SEED_PACKAGES=(
  libnss3
  libnspr4
  libatk1.0-0
  libatk-bridge2.0-0
  libatspi2.0-0
  libcups2
  libavahi-common3
  libavahi-client3
  libffi6
  libxkbcommon0
  libasound2
  libgbm1
  libxcomposite1
  libxdamage1
  libxfixes3
  libxrandr2
  libx11-6
  libx11-xcb1
  libxext6
  libxcb1
  libxcb-shm0
  libxcb-render0
  libxcb-xfixes0
  libxcb-shape0
  libxcb-randr0
  libxcb-image0
  libxcb-keysyms1
  libxcb-icccm4
  libxcb-sync1
  libxcb-xkb1
  libxcb-util1
  libxau6
  libxdmcp6
  libxshmfence1
  libxss1
  libdrm2
  libpango-1.0-0
  libcairo2
  libgtk-3-0
  libdbus-1-3
  libexpat1
  libglib2.0-0
  libwayland-client0
  libwayland-server0
  libwayland-cursor0
  libwayland-egl1
  libxi6
  libxtst6
  libpangocairo-1.0-0
  libpangoft2-1.0-0
  libharfbuzz0b
  libgraphite2-3
  libfontconfig1
  libfreetype6
  libpng16-16
  libxrender1
  libxcursor1
  libxinerama1
  libgdk-pixbuf2.0-0
  libepoxy0
  libfribidi0
  libthai0
  libdatrie1
  libpixman-1-0
  libcairo-gobject2
  libcolord2
  liblcms2-2
  libjson-glib-1.0-0
  librest-0.7-0
  libpcre3
  gsettings-desktop-schemas
)

# Never bundle — host must supply these (newer copies break with GLIBC_2.29/2.30).
SKIP_PACKAGES_REGEX='^(libc6|libc-bin|libselinux1|libmount1|libblkid1|libuuid1|zlib1g|libgcc-s1|libstdc\+\+6|libsystemd0|libudev1|libpam0g|base-files|bash|coreutils|dpkg)$'

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "Error: required command not found: $1"
    exit 1
  }
}
need_cmd curl
need_cmd dpkg-deb
need_cmd gzip
need_cmd awk

echo "=== Fetching Packages index ($SUITE $ARCH) ==="
mkdir -p "$WORKDIR/lists" "$WORKDIR/debs"
for component in main universe; do
  idx="$WORKDIR/lists/${component}.gz"
  url="$MIRROR/dists/$SUITE/$component/binary-$ARCH/Packages.gz"
  echo "GET $url"
  curl -fsSL "$url" -o "$idx"
done

INDEX_TXT="$WORKDIR/index.txt"
: > "$INDEX_TXT"
for gz in "$WORKDIR/lists"/*.gz; do
  gzip -dc "$gz" >> "$INDEX_TXT"
  printf '\n' >> "$INDEX_TXT"
done

awk '
  /^Package: / { pkg=$2 }
  /^Filename: / { if (pkg != "") print pkg "\t" $2 }
' "$INDEX_TXT" > "$WORKDIR/filenames.tsv"

awk '
  /^Package: / { pkg=$2; deps="" }
  /^Depends: / {
    sub(/^Depends: /, "")
    deps=$0
  }
  /^$/ {
    if (pkg != "" && deps != "") print pkg "\t" deps
    pkg=""; deps=""
  }
' "$INDEX_TXT" > "$WORKDIR/depends.tsv"

lookup_filename() {
  awk -F '\t' -v pkg="$1" '$1 == pkg { print $2; exit }' "$WORKDIR/filenames.tsv"
}

lookup_depends() {
  awk -F '\t' -v pkg="$1" '$1 == pkg { print $2; exit }' "$WORKDIR/depends.tsv"
}

should_skip() {
  echo "$1" | grep -Eq "$SKIP_PACKAGES_REGEX"
}

parse_depend_names() {
  local deps_line="$1"
  local IFS=','
  # shellcheck disable=SC2086
  set -- $deps_line
  for token in "$@"; do
    token="$(echo "$token" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    token="${token%%|*}"
    token="$(echo "$token" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    token="${token%% (*}"
    token="$(echo "$token" | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')"
    [ -n "$token" ] || continue
    case "$token" in
      *:* ) continue ;;
    esac
    echo "$token"
  done
}

echo "=== Resolving package set (seed + Depends, depth 1 only) ==="
declare -A WANTED=()
for pkg in "${SEED_PACKAGES[@]}"; do
  WANTED["$pkg"]=1
done

# Depth 1 only — deeper pulls systemd/udev and breaks App Service glibc.
current=("${!WANTED[@]}")
for pkg in "${current[@]}"; do
  deps="$(lookup_depends "$pkg" || true)"
  [ -n "$deps" ] || continue
  while IFS= read -r dep; do
    [ -n "$dep" ] || continue
    should_skip "$dep" && continue
    # Only pull other shared libs, not themes/tools.
    case "$dep" in
      lib*) WANTED["$dep"]=1 ;;
    esac
  done < <(parse_depend_names "$deps")
done

mapfile -t PACKAGES < <(printf '%s\n' "${!WANTED[@]}" | sort -u)
echo "Will download ${#PACKAGES[@]} packages from $SUITE"

echo "=== Downloading .debs ==="
downloaded=0
for pkg in "${PACKAGES[@]}"; do
  should_skip "$pkg" && continue
  file="$(lookup_filename "$pkg" || true)"
  if [ -z "$file" ]; then
    echo "WARN: package not in index: $pkg (skipping)"
    continue
  fi
  url="$MIRROR/$file"
  base="$(basename "$file")"
  echo "GET $pkg <- $base"
  curl -fsSL "$url" -o "$WORKDIR/debs/$base"
  downloaded=$((downloaded + 1))
done

if [ "$downloaded" -eq 0 ]; then
  echo "Error: downloaded 0 packages."
  exit 1
fi

echo "=== Extracting into $OUT ==="
rm -rf "$OUT"
mkdir -p "$OUT"
for deb in "$WORKDIR/debs"/*.deb; do
  env -u LD_LIBRARY_PATH dpkg-deb -x "$deb" "$OUT"
done

# Strip core libs that must come from the App Service host.
find "$OUT" \( \
  -name 'libselinux.so*' -o \
  -name 'libsystemd.so*' -o \
  -name 'libudev.so*' -o \
  -name 'libmount.so*' -o \
  -name 'libblkid.so*' -o \
  -name 'libuuid.so*' -o \
  -name 'libz.so*' \
\) -type f -delete 2>/dev/null || true
find "$OUT" \( \
  -name 'libselinux.so*' -o \
  -name 'libsystemd.so*' -o \
  -name 'libudev.so*' \
\) -type l -delete 2>/dev/null || true

{
  echo "# Generated by scripts/bundle-chrome-linux-deps.sh (suite=$SUITE) — do not edit by hand"
  find "$OUT" -type d \( -path '*/lib/x86_64-linux-gnu' -o -path '*/usr/lib/x86_64-linux-gnu' -o -path '*/lib64' \) \
    | sed "s|^$OUT/||" | sort -u
} > "$OUT/ld_library_path_dirs.txt"

count="$(find "$OUT" -name '*.so*' | wc -l | tr -d ' ')"
echo "Extracted $count shared objects from $SUITE."
ls "$OUT"/usr/lib/x86_64-linux-gnu/libgraphite2.so* 2>/dev/null || echo "WARN: libgraphite2 missing"
test ! -e "$OUT/lib/x86_64-linux-gnu/libselinux.so.1" && echo "OK: libselinux not bundled" || echo "WARN: libselinux still present"
echo "Wrote $OUT/ld_library_path_dirs.txt"
echo "Done. Next: ./scripts/deploy-all-workers.sh && redeploy + restart app."
