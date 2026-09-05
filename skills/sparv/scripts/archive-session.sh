#!/bin/bash
# SPARV Session Archive Script
# Archives completed session from .sparv/plan/<session_id>/ to .sparv/history/<session_id>/

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

usage() {
	cat <<'EOF'
Usage: archive-session.sh [--dry-run]

Moves current session from .sparv/plan/<session_id>/ to .sparv/history/<session_id>/
Updates .sparv/history/index.md with session info.

Options:
  --dry-run    Show what would be archived without doing it
EOF
}

SPARV_ROOT=".sparv"
PLAN_DIR="$SPARV_ROOT/plan"
HISTORY_DIR="$SPARV_ROOT/history"

dry_run=0

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help) usage; exit 0 ;;
	--dry-run) dry_run=1; shift ;;
	*) usage >&2; exit 1 ;;
	esac
done

# Find active session
find_active_session() {
	if [ -d "$PLAN_DIR" ]; then
		local session
		session="$(ls -1 "$PLAN_DIR" 2>/dev/null | head -1)"
		if [ -n "$session" ] && [ -f "$PLAN_DIR/$session/state.yaml" ]; then
			echo "$session"
		fi
	fi
}

# Update history/index.md
update_history_index() {
	local session_id="$1"
	local index_file="$HISTORY_DIR/index.md"
	local state_file="$HISTORY_DIR/$session_id/state.yaml"

	[ -f "$index_file" ] || return 0

	# Get feature name from state.yaml
	local fname=""
	if [ -f "$state_file" ]; then
		fname="$(grep -E '^feature_name:' "$state_file" | sed -E 's/^feature_name:[[:space:]]*"?([^"]*)"?$/\1/' || true)"
	fi
	[ -z "$fname" ] && fname="unnamed"

	local month="${session_id:0:6}"
	local formatted_month="${month:0:4}-${month:4:2}"

	# Add to monthly section if not exists
	if ! grep -q "### $formatted_month" "$index_file"; then
		echo -e "\n### $formatted_month\n" >> "$index_file"
	fi
	echo "- \`${session_id}\` - $fname" >> "$index_file"
}

SESSION_ID="$(find_active_session)"

if [ -z "$SESSION_ID" ]; then
	echo "No active session to archive"
	exit 0
fi

SRC_DIR="$PLAN_DIR/$SESSION_ID"
DST_DIR="$HISTORY_DIR/$SESSION_ID"

if [ "$dry_run" -eq 1 ]; then
	echo "Would archive: $SRC_DIR -> $DST_DIR"
	exit 0
fi

# Create history directory and move session
mkdir -p "$HISTORY_DIR"
mv "$SRC_DIR" "$DST_DIR"

# Update index
update_history_index "$SESSION_ID"

echo "✅ Session archived: $SESSION_ID"
echo "📁 Location: $DST_DIR"
