#!/usr/bin/env bash
set -euo pipefail

VERSION="${1:-latest}"
REPO_OWNER="${REPO_OWNER:-Tresor-Kasenda}"
REPO_NAME="${REPO_NAME:-apix}"
BINARY_NAME="${BINARY_NAME:-apix}"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
RELEASE_BASE_URL="${RELEASE_BASE_URL:-}"

OS_RAW="$(uname -s)"
ARCH_RAW="$(uname -m)"

case "$OS_RAW" in
  Darwin) OS="darwin" ;;
  Linux) OS="linux" ;;
  *)
    echo "Unsupported OS: $OS_RAW" >&2
    echo "Use release archives manually for Windows." >&2
    exit 1
    ;;
esac

case "$ARCH_RAW" in
  x86_64|amd64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH_RAW" >&2
    exit 1
    ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO_OWNER}/${REPO_NAME}/releases/latest" \
    | awk -F '"' '/tag_name/ {print $4; exit}')"
  if [[ -z "$VERSION" ]]; then
    echo "Unable to resolve latest version from GitHub API." >&2
    exit 1
  fi
fi

VERSION_CLEAN="${VERSION#v}"
ARCHIVE_CANDIDATES=(
  "${BINARY_NAME}_${VERSION_CLEAN}_${OS}_${ARCH}.tar.gz"
  "${BINARY_NAME}-${VERSION_CLEAN}-${OS}-${ARCH}.tar.gz"
)
BASE_URL="https://github.com/${REPO_OWNER}/${REPO_NAME}/releases/download/${VERSION}"
if [[ -n "$RELEASE_BASE_URL" ]]; then
  BASE_URL="${RELEASE_BASE_URL%/}"
fi
CHECKSUMS_URL="${BASE_URL}/checksums.txt"

TMP_DIR="$(mktemp -d)"
cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

echo "Installing ${BINARY_NAME} ${VERSION} for ${OS}/${ARCH}..."

ARCHIVE=""
for candidate in "${ARCHIVE_CANDIDATES[@]}"; do
  if curl -fsSL "${BASE_URL}/${candidate}" -o "$TMP_DIR/$candidate" 2>/dev/null; then
    ARCHIVE="$candidate"
    break
  fi
done

if [[ -z "$ARCHIVE" ]]; then
  echo "No matching release archive found for ${OS}/${ARCH}." >&2
  exit 1
fi

if curl -fsSL "$CHECKSUMS_URL" -o "$TMP_DIR/checksums.txt"; then
  expected="$(awk -v file="$ARCHIVE" '$2 == file {print $1}' "$TMP_DIR/checksums.txt")"
  if [[ -n "$expected" ]]; then
    if command -v sha256sum >/dev/null 2>&1; then
      actual="$(sha256sum "$TMP_DIR/$ARCHIVE" | awk '{print $1}')"
    else
      actual="$(shasum -a 256 "$TMP_DIR/$ARCHIVE" | awk '{print $1}')"
    fi

    if [[ "$expected" != "$actual" ]]; then
      echo "Checksum verification failed for $ARCHIVE" >&2
      exit 1
    fi
  fi
fi

tar -xzf "$TMP_DIR/$ARCHIVE" -C "$TMP_DIR"
SOURCE_BIN=""
for extracted in \
  "$TMP_DIR/$BINARY_NAME" \
  "$TMP_DIR/${BINARY_NAME}_${VERSION_CLEAN}_${OS}_${ARCH}/$BINARY_NAME" \
  "$TMP_DIR/${BINARY_NAME}-${VERSION_CLEAN}-${OS}-${ARCH}/$BINARY_NAME"; do
  if [[ -f "$extracted" ]]; then
    SOURCE_BIN="$extracted"
    break
  fi
done

if [[ -z "$SOURCE_BIN" ]]; then
  echo "Binary not found in extracted archive: $ARCHIVE" >&2
  exit 1
fi

TARGET_BIN="$INSTALL_DIR/$BINARY_NAME"
if [[ -w "$INSTALL_DIR" ]]; then
  install -m 0755 "$SOURCE_BIN" "$TARGET_BIN"
else
  sudo install -m 0755 "$SOURCE_BIN" "$TARGET_BIN"
fi

echo "Installed: $TARGET_BIN"
"$TARGET_BIN" --version
