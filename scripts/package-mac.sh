#!/usr/bin/env bash
set -euo pipefail

# Lumina Photo Sync - macOS Bundle Packager
# Author: eddy

APP_NAME="LuminaPhotoSync.app"
DIST_DIR="bin/macos"
CONTENTS_DIR="${DIST_DIR}/${APP_NAME}/Contents"
MACOS_DIR="${CONTENTS_DIR}/MacOS"
RESOURCES_DIR="${CONTENTS_DIR}/Resources"

mkdir -p "${MACOS_DIR}" "${RESOURCES_DIR}"

cp dist/Info.plist "${CONTENTS_DIR}/Info.plist"

if [ -f "${1:-}" ]; then
    echo "Installing binary: $1 -> ${MACOS_DIR}/lumina-photo-sync"
    cp "$1" "${MACOS_DIR}/lumina-photo-sync"
    chmod +x "${MACOS_DIR}/lumina-photo-sync"
else
    echo "Notice: No binary provided. Created bundle skeleton at ${DIST_DIR}/${APP_NAME}"
fi

cd "${DIST_DIR}"
rm -f "LuminaPhotoSync-macOS.zip"
zip -r -q "LuminaPhotoSync-macOS.zip" "${APP_NAME}"
cd - > /dev/null

echo "macOS App bundle created at: ${DIST_DIR}/${APP_NAME}"
echo "macOS Distribution archive: ${DIST_DIR}/LuminaPhotoSync-macOS.zip"
