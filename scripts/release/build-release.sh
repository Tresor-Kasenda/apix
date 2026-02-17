#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

APP_NAME="${APP_NAME:-apix}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION_CLEAN="${VERSION#v}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
BUILD_DIR="${BUILD_DIR:-$DIST_DIR/build}"
BIN_DIR="${BIN_DIR:-$BUILD_DIR/bin}"
CGO_ENABLED="${CGO_ENABLED:-0}"

LDFLAGS="-s -w -X main.version=${VERSION}"
if [[ -n "${EXTRA_LDFLAGS:-}" ]]; then
  LDFLAGS="${LDFLAGS} ${EXTRA_LDFLAGS}"
fi

TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

sha256_file() {
  local file="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$file" | awk '{print $1}'
    return
  fi
  shasum -a 256 "$file" | awk '{print $1}'
}

mkdir -p "$DIST_DIR" "$BUILD_DIR" "$BIN_DIR"
rm -f "$DIST_DIR"/checksums.txt

echo "Building ${APP_NAME} ${VERSION}"

for target in "${TARGETS[@]}"; do
  read -r goos goarch <<<"$target"

  ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  bin_name="${APP_NAME}${ext}"
  build_name="${APP_NAME}_${VERSION_CLEAN}_${goos}_${goarch}${ext}"
  build_path="$BIN_DIR/$build_name"

  echo " -> ${goos}/${goarch}"
  CGO_ENABLED="$CGO_ENABLED" GOOS="$goos" GOARCH="$goarch" \
    go build -trimpath -ldflags "$LDFLAGS" -o "$build_path" ./cmd/apix

  pkg_root="${APP_NAME}_${VERSION_CLEAN}_${goos}_${goarch}"
  staging_dir="$BUILD_DIR/$pkg_root"
  rm -rf "$staging_dir"
  mkdir -p "$staging_dir"

  cp "$build_path" "$staging_dir/$bin_name"
  [[ -f README.md ]] && cp README.md "$staging_dir/"
  [[ -f LICENSE ]] && cp LICENSE "$staging_dir/"

  archive="$DIST_DIR/${APP_NAME}_${VERSION_CLEAN}_${goos}_${goarch}.tar.gz"
  tar -C "$BUILD_DIR" -czf "$archive" "$pkg_root"

  checksum=$(sha256_file "$archive")
  printf "%s  %s\n" "$checksum" "$(basename "$archive")" >> "$DIST_DIR/checksums.txt"
done

echo

echo "Artifacts written to: $DIST_DIR"
echo "Checksums file: $DIST_DIR/checksums.txt"
