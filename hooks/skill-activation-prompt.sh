#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT="$SCRIPT_DIR/skill-activation-prompt.js"

if command -v node >/dev/null 2>&1; then
  node "$SCRIPT" "$@" || true
else
  echo '{"suggestedSkills":[],"meta":{"warning":"node not found"}}'
fi

exit 0
