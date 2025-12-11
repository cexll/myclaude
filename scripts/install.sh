#!/usr/bin/env bash
set -euo pipefail

# Legacy alias installer: create codex-wrapper -> codeagent-wrapper symlink
# in the configured install directory (defaults to ~/bin).

BIN_DIR="${INSTALL_DIR:-"$HOME/bin"}"
TARGET_NAME="codeagent-wrapper"
LEGACY_NAME="codex-wrapper"

mkdir -p "$BIN_DIR"
cd "$BIN_DIR"

if [[ ! -x "$TARGET_NAME" ]]; then
  echo "ERROR: $BIN_DIR/$TARGET_NAME not found or not executable; install the wrapper first." >&2
  exit 1
fi

if [[ -L "$LEGACY_NAME" ]]; then
  echo "Legacy alias already present: $BIN_DIR/$LEGACY_NAME -> $(readlink "$LEGACY_NAME")"
  exit 0
fi

if [[ -e "$LEGACY_NAME" ]]; then
  echo "INFO: $BIN_DIR/$LEGACY_NAME exists and is not a symlink; leaving user-managed binary untouched." >&2
  exit 0
fi

ln -s "$TARGET_NAME" "$LEGACY_NAME"
echo "Created legacy alias $BIN_DIR/$LEGACY_NAME -> $TARGET_NAME"
