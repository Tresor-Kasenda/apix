#!/bin/bash
set -e

VERSION="${1:-latest}"
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64) ARCH="arm64" ;;
  arm64)   ARCH="arm64" ;;
esac

if [ "$VERSION" = "latest" ]; then
  VERSION=$(curl -sL https://api.github.com/repos/Tresor-Kasenda/apix/releases/latest \
    | grep '"tag_name"' | cut -d'"' -f4)
fi

URL="https://github.com/Tresor-Kasenda/apix/releases/download/${VERSION}/apix-${VERSION#v}-${OS}-${ARCH}.tar.gz"

echo "Installing apix ${VERSION} for ${OS}/${ARCH}..."

curl -sL "$URL" | tar xz -C /tmp
sudo mv /tmp/apix /usr/local/bin/apix
sudo chmod +x /usr/local/bin/apix

echo "apix ${VERSION} installed successfully"
apix --version