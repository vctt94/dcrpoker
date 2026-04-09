#!/usr/bin/env bash
set -euo pipefail

# Usage:
#   ./build-android-apk.sh v0.0.2
#   ./build-android-apk.sh            # defaults to v0.0.2
#
# It rebuilds and packages the Flutter Android release APK:
#   ../pokerui/flutterui/pokerui/build/app/outputs/flutter-apk/app-release.apk
#
# Output:
#   ../releases/dcrpoker-android-apk-<version>.apk

APP="dcrpoker"
VER="${1:-v0.0.2}"
PLAT="android-apk"

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
POKERUI_DIR="$ROOT/pokerui"
FLUTTER_APP_DIR="$POKERUI_DIR/flutterui/pokerui"

APK_SRC="$FLUTTER_APP_DIR/build/app/outputs/flutter-apk/app-release.apk"
OUT_DIR="$ROOT/releases"
mkdir -p "$OUT_DIR"

NAME="$APP-$PLAT-$VER"
OUT_APK="$OUT_DIR/$NAME.apk"

echo "Generating golib.aar for Android"
(
  cd "$POKERUI_DIR"
  go generate ./golibbuilder
)

echo "Building Android release APK"
(
  cd "$FLUTTER_APP_DIR"
  flutter build apk --release
)

[[ -f "$APK_SRC" ]] || { echo "APK not found after build: $APK_SRC" >&2; exit 1; }

cp -f "$APK_SRC" "$OUT_APK"
echo "Created: $OUT_APK"
