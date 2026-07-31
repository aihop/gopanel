#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "${PROJECT_ROOT}"

if [[ ! -d admin/node_modules ]]; then
  npm --prefix admin ci
fi
npm --prefix admin run build:public
if [[ "$(uname -s)" == "Darwin" ]]; then
  sips -z 1024 1024 admin/public/icons/pwa-512.png --out desktop/build/appicon.png >/dev/null
else
  cp admin/public/icons/pwa-512.png desktop/build/appicon.png
fi

go run github.com/wailsapp/wails/v2/cmd/wails@v2.13.0 build \
  -clean \
  -skipbindings \
  -s \
  -m \
  -nosyncgomod \
  "$@"

if [[ "$(uname -s)" == "Darwin" ]]; then
  APP_PATH="desktop/build/bin/GoPanel.app"
  test -x "${APP_PATH}/Contents/MacOS/GoPanel"
  BUNDLE_ID="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleIdentifier' "${APP_PATH}/Contents/Info.plist")"
  test "${BUNDLE_ID}" = "io.aihop.gopanel"
fi
