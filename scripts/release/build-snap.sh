#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "snap packaging is only supported on Linux hosts" >&2
  exit 1
fi

if ! command -v snapcraft >/dev/null 2>&1; then
  echo "snapcraft is required (https://snapcraft.io/docs/installing-snapcraft)" >&2
  exit 1
fi

APP_NAME="${APP_NAME:-apix}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION_CLEAN="${VERSION#v}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
SNAP_ARCH="${SNAP_ARCH:-amd64}"

BINARY_ARCH="${SNAP_BINARY_ARCH:-$SNAP_ARCH}"
BINARY_ARCHIVE="$DIST_DIR/${APP_NAME}_${VERSION_CLEAN}_linux_${BINARY_ARCH}.tar.gz"
if [[ ! -f "$BINARY_ARCHIVE" ]]; then
  echo "missing binary archive: $BINARY_ARCHIVE" >&2
  echo "run scripts/release/build-release.sh first" >&2
  exit 1
fi

WORK_DIR="$DIST_DIR/snap-work"
rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR/stage"

tar -xzf "$BINARY_ARCHIVE" -C "$WORK_DIR/stage"
EXTRACT_ROOT="$WORK_DIR/stage/${APP_NAME}_${VERSION_CLEAN}_linux_${BINARY_ARCH}"
if [[ ! -f "$EXTRACT_ROOT/$APP_NAME" ]]; then
  echo "expected binary not found in archive: $EXTRACT_ROOT/$APP_NAME" >&2
  exit 1
fi

SNAP_SRC="$WORK_DIR/src"
mkdir -p "$SNAP_SRC"
cp "$EXTRACT_ROOT/$APP_NAME" "$SNAP_SRC/$APP_NAME"

cat > "$SNAP_SRC/snapcraft.yaml" <<SNAP
name: ${APP_NAME}
base: core22
version: "${VERSION_CLEAN}"
summary: Terminal-first API tester
description: |
  apix is a modern, framework-agnostic CLI API tester for terminal-first developers.

grade: stable
confinement: strict

apps:
  ${APP_NAME}:
    command: bin/${APP_NAME}
    plugs:
      - network
      - home

parts:
  ${APP_NAME}:
    plugin: dump
    source: .
    organize:
      ${APP_NAME}: bin/${APP_NAME}
SNAP

OUTPUT_FILE="$DIST_DIR/${APP_NAME}_${VERSION_CLEAN}_linux_${SNAP_ARCH}.snap"
(
  cd "$SNAP_SRC"
  snapcraft --destructive-mode --target-arch="$SNAP_ARCH" --output "$OUTPUT_FILE"
)

echo "Snap package generated: $OUTPUT_FILE"
