#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./build-macos-arm64.sh v0.0.2
#   ./build-macos-arm64.sh            # defaults to v0.0.2
#
# It rebuilds and packages the Flutter macOS app bundle:
#   ../pokerui/flutterui/pokerui/build/macos/Build/Products/Release/dcrpoker.app
#
# Output:
#   ../releases/dcrpoker-macos-arm64-<version>.dmg

APP="dcrpoker"
VER="${1:-v0.0.2}"
PLAT="macos-arm64"

# Repo root (2 level up from scripts/)
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
POKERUI_DIR="$ROOT/pokerui"
FLUTTER_APP_DIR="$POKERUI_DIR/flutterui/pokerui"

# Flutter app bundle dir
BUILD_APP="$FLUTTER_APP_DIR/build/macos/Build/Products/Release/${APP}.app"

# Output dir for release artifacts
OUT_DIR="$ROOT/releases"
mkdir -p "$OUT_DIR"

NAME="$APP-$PLAT-$VER"
DMG_NAME="$OUT_DIR/$NAME.dmg"
VOLUME_NAME="${APP} Installer"

# Temporary dist directory
DIST_DIR=$(mktemp -d)
trap "rm -rf '$DIST_DIR'" EXIT

echo "Generating golib.dylib for macOS"
(
  cd "$POKERUI_DIR"
  go generate ./golibbuilder
)

echo "Building macOS release app"
(
  cd "$FLUTTER_APP_DIR"
  flutter build macos --release
)

[[ -d "$BUILD_APP" ]] || { echo "App bundle not found after build: $BUILD_APP" >&2; exit 1; }

echo "Packaging $NAME from: $BUILD_APP"

# Copy .app
cp -R "$BUILD_APP" "$DIST_DIR/"

# Create alias to /Applications
cd "$DIST_DIR"
ln -s /Applications Applications

# Create DMG
cd "$ROOT"
hdiutil create \
  -volname "$VOLUME_NAME" \
  -srcfolder "$DIST_DIR" \
  -ov \
  -format UDZO \
  "$DMG_NAME"

echo "Created: $DMG_NAME"
