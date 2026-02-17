#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

APP_NAME="${APP_NAME:-apix}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
VERSION_CLEAN="${VERSION#v}"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist}"
FORMULA_DIR="${FORMULA_DIR:-$DIST_DIR/homebrew}"
FORMULA_CLASS="${FORMULA_CLASS:-Apix}"
OWNER="${REPO_OWNER:-Tresor-Kasend}"
REPO="${REPO_NAME:-apix}"

CHECKSUMS_FILE="$DIST_DIR/checksums.txt"
if [[ ! -f "$CHECKSUMS_FILE" ]]; then
  echo "checksums file not found: $CHECKSUMS_FILE" >&2
  echo "run scripts/release/build-release.sh first" >&2
  exit 1
fi

get_sha() {
  local file="$1"
  awk -v f="$file" '$2 == f {print $1}' "$CHECKSUMS_FILE"
}

DARWIN_AMD64_FILE="${APP_NAME}_${VERSION_CLEAN}_darwin_amd64.tar.gz"
DARWIN_ARM64_FILE="${APP_NAME}_${VERSION_CLEAN}_darwin_arm64.tar.gz"
LINUX_AMD64_FILE="${APP_NAME}_${VERSION_CLEAN}_linux_amd64.tar.gz"
LINUX_ARM64_FILE="${APP_NAME}_${VERSION_CLEAN}_linux_arm64.tar.gz"

DARWIN_AMD64_SHA="$(get_sha "$DARWIN_AMD64_FILE")"
DARWIN_ARM64_SHA="$(get_sha "$DARWIN_ARM64_FILE")"
LINUX_AMD64_SHA="$(get_sha "$LINUX_AMD64_FILE")"
LINUX_ARM64_SHA="$(get_sha "$LINUX_ARM64_FILE")"

for value in "$DARWIN_AMD64_SHA" "$DARWIN_ARM64_SHA" "$LINUX_AMD64_SHA" "$LINUX_ARM64_SHA"; do
  if [[ -z "$value" ]]; then
    echo "missing required checksum entry in $CHECKSUMS_FILE" >&2
    exit 1
  fi
done

mkdir -p "$FORMULA_DIR"
FORMULA_PATH="$FORMULA_DIR/${APP_NAME}.rb"
BASE_URL="https://github.com/${OWNER}/${REPO}/releases/download/${VERSION}"

cat > "$FORMULA_PATH" <<FORMULA
class ${FORMULA_CLASS} < Formula
  desc "Terminal-first API tester"
  homepage "https://github.com/${OWNER}/${REPO}"
  version "${VERSION_CLEAN}"

  on_macos do
    if Hardware::CPU.arm?
      url "${BASE_URL}/${DARWIN_ARM64_FILE}"
      sha256 "${DARWIN_ARM64_SHA}"
    else
      url "${BASE_URL}/${DARWIN_AMD64_FILE}"
      sha256 "${DARWIN_AMD64_SHA}"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "${BASE_URL}/${LINUX_ARM64_FILE}"
      sha256 "${LINUX_ARM64_SHA}"
    else
      url "${BASE_URL}/${LINUX_AMD64_FILE}"
      sha256 "${LINUX_AMD64_SHA}"
    end
  end

  def install
    bin.install "${APP_NAME}"
  end

  test do
    assert_match "${VERSION}", shell_output("#{bin}/${APP_NAME} --version")
  end
end
FORMULA

echo "Homebrew formula generated: $FORMULA_PATH"
