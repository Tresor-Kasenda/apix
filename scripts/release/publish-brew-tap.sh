#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

APP_NAME="${APP_NAME:-apix}"
VERSION="${VERSION:-$(git describe --tags --always --dirty 2>/dev/null || echo dev)}"
TAP_REPO="${TAP_REPO:-}"
TAP_BRANCH="${TAP_BRANCH:-main}"
TAP_FORMULA_PATH="${TAP_FORMULA_PATH:-Formula/${APP_NAME}.rb}"
FORMULA_SOURCE="${FORMULA_SOURCE:-$ROOT_DIR/dist/homebrew/${APP_NAME}.rb}"
WORK_DIR="${WORK_DIR:-$ROOT_DIR/dist/tap-work}"
DRY_RUN="${DRY_RUN:-0}"

usage() {
  cat <<USAGE
Usage:
  TAP_REPO=<owner/homebrew-tap> [GITHUB_TOKEN=token] ./scripts/release/publish-brew-tap.sh

Optional environment variables:
  TAP_BRANCH         Target branch in tap repository (default: main)
  TAP_FORMULA_PATH   Path in tap repository for formula (default: Formula/${APP_NAME}.rb)
  FORMULA_SOURCE     Local formula path (default: dist/homebrew/${APP_NAME}.rb)
  TAP_REMOTE_URL     Explicit git remote URL override
  WORK_DIR           Working directory for temporary clone (default: dist/tap-work)
  DRY_RUN            Set to 1 to skip git push
USAGE
}

if [[ -z "$TAP_REPO" && -z "${TAP_REMOTE_URL:-}" ]]; then
  echo "TAP_REPO is required unless TAP_REMOTE_URL is set." >&2
  usage >&2
  exit 1
fi

if [[ ! -f "$FORMULA_SOURCE" ]]; then
  echo "Formula source not found: $FORMULA_SOURCE" >&2
  echo "Run: make dist-brew" >&2
  exit 1
fi

if [[ -n "${TAP_REMOTE_URL:-}" ]]; then
  REMOTE_URL="$TAP_REMOTE_URL"
elif [[ -n "${GITHUB_TOKEN:-}" ]]; then
  REMOTE_URL="https://x-access-token:${GITHUB_TOKEN}@github.com/${TAP_REPO}.git"
else
  REMOTE_URL="https://github.com/${TAP_REPO}.git"
fi

echo "Publishing Homebrew formula for ${APP_NAME} ${VERSION}"
if [[ -n "$TAP_REPO" ]]; then
  echo "Tap repository: ${TAP_REPO} (branch: ${TAP_BRANCH})"
else
  echo "Tap remote: ${REMOTE_URL} (branch: ${TAP_BRANCH})"
fi

rm -rf "$WORK_DIR"
mkdir -p "$WORK_DIR"

REPO_DIR="$WORK_DIR/repo"
git clone --depth 1 --branch "$TAP_BRANCH" "$REMOTE_URL" "$REPO_DIR"

TARGET_FILE="$REPO_DIR/$TAP_FORMULA_PATH"
mkdir -p "$(dirname "$TARGET_FILE")"
cp "$FORMULA_SOURCE" "$TARGET_FILE"

cd "$REPO_DIR"
git add "$TAP_FORMULA_PATH"

if git diff --cached --quiet; then
  echo "No changes detected in ${TAP_FORMULA_PATH}; nothing to publish."
  exit 0
fi

AUTHOR_NAME="${GIT_AUTHOR_NAME:-apix release bot}"
AUTHOR_EMAIL="${GIT_AUTHOR_EMAIL:-actions@users.noreply.github.com}"
COMMIT_MESSAGE="${COMMIT_MESSAGE:-chore(brew): update ${APP_NAME} formula for ${VERSION}}"

git -c user.name="$AUTHOR_NAME" -c user.email="$AUTHOR_EMAIL" commit -m "$COMMIT_MESSAGE"

if [[ "$DRY_RUN" == "1" ]]; then
  echo "DRY_RUN=1; skipping push."
  git show --stat --oneline -1
  exit 0
fi

git push origin "$TAP_BRANCH"
echo "Formula published successfully to ${TAP_BRANCH}."
