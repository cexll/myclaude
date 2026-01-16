#!/bin/bash
# SPARV Progress Save Script
# Implements the 2-Action rule (called after each tool call; writes every 2 actions).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

usage() {
	cat <<'EOF'
Usage: save-progress.sh [TOOL_NAME] [RESULT]

Increments action_count and appends to journal.md every 2 actions.
Auto-detects active session in .sparv/plan/<session_id>/
EOF
}

if [ "${1:-}" = "-h" ] || [ "${1:-}" = "--help" ]; then
	usage
	exit 0
fi

# Auto-detect session (sets SPARV_DIR, STATE_FILE, JOURNAL_FILE)
sparv_require_state_file
sparv_state_validate_or_die
[ -f "$JOURNAL_FILE" ] || sparv_die "Cannot find $JOURNAL_FILE; run init-session.sh first"

# Arguments
TOOL_NAME="${1:-unknown}"
RESULT="${2:-no result}"

ACTION_COUNT="$(sparv_yaml_get_int action_count 0)"

# Increment action count
NEW_COUNT=$((ACTION_COUNT + 1))

# Update state file
sparv_yaml_set_int action_count "$NEW_COUNT"

# Only write every 2 actions
if [ $((NEW_COUNT % 2)) -ne 0 ]; then
	exit 0
fi

# Append to journal
TIMESTAMP=$(date '+%H:%M')
cat >> "$JOURNAL_FILE" << EOF

## $TIMESTAMP - Action #$NEW_COUNT
- Tool: $TOOL_NAME
- Result: $RESULT
EOF

echo "📝 journal.md saved: Action #$NEW_COUNT"
