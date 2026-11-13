#!/usr/bin/env bash
set -euo pipefail

# Lumina Amber - macOS Bundle Packager
# Author: eddy

APP_NAME="LuminaAmber.app"
DIST_DIR="bin/macos"
CONTENTS_DIR="${DIST_DIR}/${APP_NAME}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"

cp dist/Info.plist "${CONTENTS_DIR}/Info.plist"

if [ -f "${1:-}" ]; then
    echo "Installing binary: $1 -> ${MACOS_DIR}/lumina-amber"
    cp "$1" "${MACOS_DIR}/lumina-amber"
    chmod +x "${MACOS_DIR}/lumina-amber"
else
    echo "Notice: No binary provided. Created bundle skeleton at ${DIST_DIR}/${APP_NAME}"
fi

cd "${DIST_DIR}"
rm -f "LuminaAmber-macOS.zip"
zip -r -q "LuminaAmber-macOS.zip" "${APP_NAME}"
cd - > /dev/null

echo "macOS App bundle created at: ${DIST_DIR}/${APP_NAME}"
echo "macOS Distribution archive: ${DIST_DIR}/LuminaAmber-macOS.zip"
