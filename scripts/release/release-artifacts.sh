#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

./scripts/release/build-release.sh
./scripts/release/generate-brew-formula.sh

if [[ "${WITH_BREW_PUBLISH:-0}" == "1" ]]; then
  ./scripts/release/publish-brew-tap.sh
else
  echo "Skipping brew tap publish (set WITH_BREW_PUBLISH=1 to enable)."
fi

if [[ "${WITH_SNAP:-0}" == "1" ]]; then
  ./scripts/release/build-snap.sh
else
  echo "Skipping snap package (set WITH_SNAP=1 to enable)."
fi

echo "Release artifacts ready in dist/."
