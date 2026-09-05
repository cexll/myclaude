#!/bin/bash
# SPARV Session Initialization
# Creates .sparv/plan/<session_id>/ with state.yaml and journal.md

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

usage() {
	cat <<'EOF'
Usage: init-session.sh [--force] [feature_name]

Creates .sparv/plan/<session_id>/ directory:
  - state.yaml  (session state)
  - journal.md  (unified log)

Also initializes:
  - .sparv/history/index.md (if not exists)
  - .sparv/CHANGELOG.md (if not exists)

Options:
  --force       Archive current session and start new one
  feature_name  Optional feature name for the session
EOF
}

SPARV_ROOT=".sparv"
PLAN_DIR="$SPARV_ROOT/plan"
HISTORY_DIR="$SPARV_ROOT/history"

force=0
feature_name=""

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help) usage; exit 0 ;;
	--force) force=1; shift ;;
	-*) usage >&2; exit 1 ;;
	*) feature_name="$1"; shift ;;
	esac
done

# Find current active session
find_active_session() {
	if [ -d "$PLAN_DIR" ]; then
		local session
		session="$(ls -1 "$PLAN_DIR" 2>/dev/null | head -1)"
		if [ -n "$session" ] && [ -f "$PLAN_DIR/$session/state.yaml" ]; then
			echo "$session"
		fi
	fi
}

# Archive a session to history
archive_session() {
	local session_id="$1"
	local src_dir="$PLAN_DIR/$session_id"
	local dst_dir="$HISTORY_DIR/$session_id"

	[ -d "$src_dir" ] || return 0

	mkdir -p "$HISTORY_DIR"
	mv "$src_dir" "$dst_dir"

	# Update index.md
	update_history_index "$session_id"

	echo "📦 Archived: $dst_dir"
}

# Update history/index.md
update_history_index() {
	local session_id="$1"
	local index_file="$HISTORY_DIR/index.md"
	local state_file="$HISTORY_DIR/$session_id/state.yaml"

	# Get feature name from state.yaml
	local fname=""
	if [ -f "$state_file" ]; then
		fname="$(grep -E '^feature_name:' "$state_file" | sed -E 's/^feature_name:[[:space:]]*"?([^"]*)"?$/\1/' || true)"
	fi
	[ -z "$fname" ] && fname="unnamed"

	local month="${session_id:0:6}"
	local formatted_month="${month:0:4}-${month:4:2}"
	local timestamp="${session_id:0:12}"

	# Append to index
	if [ -f "$index_file" ]; then
		# Add to monthly section if not exists
		if ! grep -q "### $formatted_month" "$index_file"; then
			echo -e "\n### $formatted_month\n" >> "$index_file"
		fi
		echo "- \`${session_id}\` - $fname" >> "$index_file"
	fi
}

# Initialize history/index.md if not exists
init_history_index() {
	local index_file="$HISTORY_DIR/index.md"
	[ -f "$index_file" ] && return 0

	mkdir -p "$HISTORY_DIR"
	cat > "$index_file" << 'EOF'
# History Index

This file records all completed sessions for traceability.

---

## Index

| Timestamp | Feature | Type | Status | Path |
|-----------|---------|------|--------|------|

---

## Monthly Archive

EOF
}

# Initialize CHANGELOG.md if not exists
init_changelog() {
	local changelog="$SPARV_ROOT/CHANGELOG.md"
	[ -f "$changelog" ] && return 0

	cat > "$changelog" << 'EOF'
# Changelog

All notable changes to this project will be documented in this file.
Format based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).

## [Unreleased]

EOF
}

# Initialize kb.md (knowledge base) if not exists
init_kb() {
	local kb_file="$SPARV_ROOT/kb.md"
	[ -f "$kb_file" ] && return 0

	cat > "$kb_file" << 'EOF'
# Knowledge Base

Cross-session knowledge accumulated during SPARV workflows.

---

## Patterns

<!-- Reusable code patterns discovered -->

## Decisions

<!-- Architectural choices + rationale -->
<!-- Format: - [YYYY-MM-DD]: decision | rationale -->

## Gotchas

<!-- Common pitfalls + solutions -->
<!-- Format: - [issue]: cause | solution -->

EOF
}

# Check for active session
active_session="$(find_active_session)"

if [ -n "$active_session" ]; then
	if [ "$force" -eq 0 ]; then
		echo "⚠️  Active session exists: $active_session"
		echo "   Use --force to archive and start new session"
		echo "   Or run: archive-session.sh"
		exit 0
	else
		archive_session "$active_session"
	fi
fi

# Generate new session ID
SESSION_ID=$(date +%Y%m%d%H%M%S)
SESSION_DIR="$PLAN_DIR/$SESSION_ID"

# Create directory structure
mkdir -p "$SESSION_DIR"
mkdir -p "$HISTORY_DIR"

# Initialize global files
init_history_index
init_changelog
init_kb

# Create state.yaml
cat > "$SESSION_DIR/state.yaml" << EOF
session_id: "$SESSION_ID"
feature_name: "$feature_name"
current_phase: "specify"
action_count: 0
consecutive_failures: 0
max_iterations: 12
iteration_count: 0
completion_promise: ""
ehrb_flags: []
EOF

# Create journal.md
cat > "$SESSION_DIR/journal.md" << EOF
# SPARV Journal
Session: $SESSION_ID
Feature: $feature_name
Created: $(date '+%Y-%m-%d %H:%M')

## Plan
<!-- Task breakdown, sub-issues, success criteria -->

## Progress
<!-- Auto-updated every 2 actions -->

## Findings
<!-- Learnings, patterns, discoveries -->
EOF

# Verify files created
if [ ! -f "$SESSION_DIR/state.yaml" ] || [ ! -f "$SESSION_DIR/journal.md" ]; then
	echo "❌ Failed to create files"
	exit 1
fi

echo "✅ SPARV session: $SESSION_ID"
[ -n "$feature_name" ] && echo "📝 Feature: $feature_name"
echo "📁 $SESSION_DIR/state.yaml"
echo "📁 $SESSION_DIR/journal.md"
