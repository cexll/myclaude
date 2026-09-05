#!/bin/bash
# SPARV 3-Failure Protocol Tracker
# Maintains consecutive_failures and escalates when reaching 3.
# Notes are appended to journal.md (unified log).

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

THRESHOLD=3

usage() {
	cat <<'EOF'
Usage: failure-tracker.sh <command> [options]

Commands:
  status                 Show current consecutive_failures and protocol level
  fail [--note TEXT]     Increment consecutive_failures (exit 3 when reaching threshold)
  reset                  Set consecutive_failures to 0

Auto-detects active session in .sparv/plan/<session_id>/
EOF
}

die() {
	echo "❌ $*" >&2
	exit 1
}

require_state() {
	# Auto-detect session (sets SPARV_DIR, STATE_FILE, JOURNAL_FILE)
	sparv_require_state_file
	sparv_state_validate_or_die
}

append_journal() {
	local level="$1"
	local note="${2:-}"
	local ts
	ts="$(date '+%Y-%m-%d %H:%M')"

	[ -f "$JOURNAL_FILE" ] || sparv_die "Cannot find $JOURNAL_FILE; run init-session.sh first"

	{
		echo
		echo "## Failure Protocol - $ts"
		echo "- level: $level"
		if [ -n "$note" ]; then
			echo "- note: $note"
		fi
	} >>"$JOURNAL_FILE"
}

protocol_level() {
	local count="$1"
	if [ "$count" -le 0 ]; then
		echo "0"
	elif [ "$count" -eq 1 ]; then
		echo "1"
	elif [ "$count" -eq 2 ]; then
		echo "2"
	else
		echo "3"
	fi
}

cmd="${1:-status}"
shift || true

note=""
case "$cmd" in
	-h|--help)
		usage
		exit 0
		;;
	status)
		require_state
		current="$(sparv_yaml_get_int consecutive_failures 0)"
		level="$(protocol_level "$current")"
		echo "consecutive_failures: $current"
		case "$level" in
			0) echo "protocol: clean (no failures)" ;;
			1) echo "protocol: Attempt 1 - Diagnose and fix" ;;
			2) echo "protocol: Attempt 2 - Alternative approach" ;;
			3) echo "protocol: Attempt 3 - Escalate (pause, document, ask user)" ;;
		esac
		exit 0
		;;
	fail)
		require_state
		if [ "${1:-}" = "--note" ]; then
			[ $# -ge 2 ] || die "--note requires an argument"
			note="$2"
			shift 2
		else
			note="$*"
			shift $#
		fi
		[ "$#" -eq 0 ] || die "Unknown argument: $1 (use --help for usage)"

		current="$(sparv_yaml_get_int consecutive_failures 0)"
		new_count=$((current + 1))
		sparv_yaml_set_int consecutive_failures "$new_count"

		level="$(protocol_level "$new_count")"
		case "$level" in
			1)
				echo "Attempt 1/3: Diagnose and fix"
				[ -n "$note" ] && append_journal "1" "$note"
				exit 0
				;;
			2)
				echo "Attempt 2/3: Alternative approach"
				[ -n "$note" ] && append_journal "2" "$note"
				exit 0
				;;
			3)
				echo "Attempt 3/3: Escalate"
				echo "3-Failure Protocol triggered: pause, document blocker and attempted solutions, request user decision."
				append_journal "3" "${note:-"(no note)"}"
				exit "$THRESHOLD"
				;;
		esac
		;;
	reset)
		require_state
		sparv_yaml_set_int consecutive_failures 0
		echo "consecutive_failures reset to 0"
		exit 0
		;;
	*)
		die "Unknown command: $cmd (use --help for usage)"
		;;
esac
