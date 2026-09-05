#!/bin/bash
#
# Shared helpers for .sparv state operations.
# Supports new directory structure: .sparv/plan/<session_id>/

sparv_die() {
	echo "❌ $*" >&2
	exit 1
}

# Find active session directory
sparv_find_active_session() {
	local plan_dir=".sparv/plan"
	if [ -d "$plan_dir" ]; then
		local session
		session="$(ls -1 "$plan_dir" 2>/dev/null | head -1)"
		if [ -n "$session" ] && [ -f "$plan_dir/$session/state.yaml" ]; then
			echo "$plan_dir/$session"
		fi
	fi
}

# Auto-detect SPARV_DIR and STATE_FILE
sparv_auto_detect() {
	local session_dir
	session_dir="$(sparv_find_active_session)"
	if [ -n "$session_dir" ]; then
		SPARV_DIR="$session_dir"
		STATE_FILE="$session_dir/state.yaml"
		JOURNAL_FILE="$session_dir/journal.md"
		export SPARV_DIR STATE_FILE JOURNAL_FILE
		return 0
	fi
	return 1
}

sparv_require_state_env() {
	if [ -z "${SPARV_DIR:-}" ] || [ -z "${STATE_FILE:-}" ]; then
		if ! sparv_auto_detect; then
			sparv_die "No active session found; run init-session.sh first"
		fi
	fi
}

sparv_require_state_file() {
	sparv_require_state_env
	[ -f "$STATE_FILE" ] || sparv_die "File not found: $STATE_FILE; run init-session.sh first"
}

# Read a YAML value (simple key: value format)
sparv_yaml_get() {
	local key="$1"
	local default="${2:-}"
	sparv_require_state_file

	local line value
	line="$(grep -E "^${key}:" "$STATE_FILE" | head -n 1 || true)"
	if [ -z "$line" ]; then
		printf "%s" "$default"
		return 0
	fi
	value="${line#${key}:}"
	value="$(printf "%s" "$value" | sed -E 's/^[[:space:]]+//; s/^"//; s/"$//')"
	printf "%s" "$value"
}

sparv_yaml_get_int() {
	local key="$1"
	local default="${2:-0}"
	local value
	value="$(sparv_yaml_get "$key" "$default")"
	if printf "%s" "$value" | grep -Eq '^[0-9]+$'; then
		printf "%s" "$value"
	else
		printf "%s" "$default"
	fi
}

# Write a YAML value (in-place update)
sparv_yaml_set_raw() {
	local key="$1"
	local raw_value="$2"
	sparv_require_state_file

	local tmp
	tmp="$(mktemp)"

	awk -v key="$key" -v repl="${key}: ${raw_value}" '
	BEGIN { in_block = 0; replaced = 0 }
	{
		if (in_block) {
			if ($0 ~ /^[[:space:]]*-/) next
			in_block = 0
		}
		if ($0 ~ ("^" key ":")) {
			print repl
			in_block = 1
			replaced = 1
			next
		}
		print
	}
	END {
		if (!replaced) print repl
	}
	' "$STATE_FILE" >"$tmp"

	mv -f "$tmp" "$STATE_FILE"
}

sparv_yaml_set_int() {
	local key="$1"
	local value="$2"
	[ "$value" -ge 0 ] 2>/dev/null || sparv_die "$key must be a non-negative integer"
	sparv_yaml_set_raw "$key" "$value"
}

# Validate state.yaml has required fields (4 core fields only)
sparv_state_validate() {
	sparv_require_state_file

	local missing=0
	local key

	for key in session_id current_phase action_count consecutive_failures; do
		grep -Eq "^${key}:" "$STATE_FILE" || missing=1
	done

	local phase
	phase="$(sparv_yaml_get current_phase "")"
	case "$phase" in
	specify|plan|act|review|vault) ;;
	*) missing=1 ;;
	esac

	[ "$missing" -eq 0 ]
}

sparv_state_validate_or_die() {
	if ! sparv_state_validate; then
		sparv_die "Corrupted state.yaml: $STATE_FILE. Run init-session.sh --force to rebuild."
	fi
}
