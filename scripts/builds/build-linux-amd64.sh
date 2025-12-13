#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./build-linux-amd64.sh v0.0.1
#   ./build-linux-amd64.sh            # defaults to v0.0.1
#
# It packages the Flutter bundle:
#   ../pokerui/flutterui/pokerui/build/linux/x64/release/bundle
#
# Output:
#   ../releases/dcrpoker-linux-amd64-<version>.tar.gz

APP="dcrpoker"
VER="${1:-v0.0.1}"
PLAT="linux-amd64"

# Repo root (2 level up from scripts/)
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"

# Flutter bundle dir (adjust if your layout changes)
BUNDLE_DIR="$ROOT/pokerui/flutterui/pokerui/build/linux/x64/release/bundle"

# Output dir for release artifacts
OUT_DIR="$ROOT/releases"
mkdir -p "$OUT_DIR"

NAME="$APP-$PLAT-$VER"
OUT_TAR="$OUT_DIR/$NAME.tar.gz"

# Sanity checks
[[ -d "$BUNDLE_DIR" ]] || { echo "Bundle dir not found: $BUNDLE_DIR" >&2; exit 1; }
[[ -x "$BUNDLE_DIR/$APP" ]] || { echo "Executable not found: $BUNDLE_DIR/$APP" >&2; exit 1; }
[[ -d "$BUNDLE_DIR/lib" ]] || { echo "Missing dir: $BUNDLE_DIR/lib" >&2; exit 1; }
[[ -d "$BUNDLE_DIR/data" ]] || { echo "Missing dir: $BUNDLE_DIR/data" >&2; exit 1; }

echo "Packaging $NAME from: $BUNDLE_DIR"
cd "$BUNDLE_DIR"

tar -czf "$OUT_TAR" \
  --owner=0 --group=0 --mode='u+rwX,go+rX' \
  --transform "s,^\./,$NAME/," \
  ./dcrpoker ./lib ./data

echo "Created: $OUT_TAR"
# Show a peek of contents (should start with $NAME/)
tar -tzf "$OUT_TAR" | head -n 6
