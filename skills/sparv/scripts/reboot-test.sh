#!/bin/bash
# SPARV 3-Question Reboot Test Script
# Prints (and optionally validates) the "3 questions" using the current session state.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

usage() {
	cat <<'EOF'
Usage: reboot-test.sh [options]

Options:
  --strict            Exit non-zero if critical answers are missing or unsafe
  -h, --help          Show this help

Auto-detects active session in .sparv/plan/<session_id>/
EOF
}

die() {
	echo "❌ $*" >&2
	exit 1
}

tail_file() {
	local path="$1"
	local lines="${2:-20}"
	if [ -f "$path" ]; then
		tail -n "$lines" "$path"
	else
		echo "(missing: $path)"
	fi
}

strict=0

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help) usage; exit 0 ;;
	--strict) strict=1; shift ;;
	*) die "Unknown argument: $1 (use --help for usage)" ;;
	esac
done

# Auto-detect session (sets SPARV_DIR, STATE_FILE, JOURNAL_FILE)
sparv_require_state_file
sparv_state_validate_or_die

session_id="$(sparv_yaml_get session_id "")"
feature_name="$(sparv_yaml_get feature_name "")"
current_phase="$(sparv_yaml_get current_phase "")"
completion_promise="$(sparv_yaml_get completion_promise "")"
iteration_count="$(sparv_yaml_get_int iteration_count 0)"
max_iterations="$(sparv_yaml_get_int max_iterations 0)"
consecutive_failures="$(sparv_yaml_get_int consecutive_failures 0)"
ehrb_flags="$(sparv_yaml_get ehrb_flags "")"

case "$current_phase" in
specify) next_phase="plan" ;;
plan) next_phase="act" ;;
act) next_phase="review" ;;
review) next_phase="vault" ;;
vault) next_phase="done" ;;
*) next_phase="unknown" ;;
esac

echo "== 3-Question Reboot Test =="
echo "session_id: ${session_id:-"(unknown)"}"
if [ -n "$feature_name" ]; then
	echo "feature_name: $feature_name"
fi
echo
echo "1) Where am I?"
echo "   current_phase: ${current_phase:-"(empty)"}"
echo
echo "2) Where am I going?"
echo "   next_phase: $next_phase"
echo
echo "3) How do I prove completion?"
if [ -n "$completion_promise" ]; then
	echo "   completion_promise: $completion_promise"
else
	echo "   completion_promise: (empty)"
fi
echo
echo "journal tail (20 lines):"
tail_file "$JOURNAL_FILE" 20
echo
echo "Counters: failures=$consecutive_failures, iteration=$iteration_count/$max_iterations"
if [ -n "$ehrb_flags" ] && [ "$ehrb_flags" != "[]" ]; then
	echo "EHRB: $ehrb_flags"
fi

if [ "$strict" -eq 1 ]; then
	exit_code=0

	case "$current_phase" in
	specify|plan|act|review|vault) ;;
	*) echo "❌ strict: current_phase invalid/empty: $current_phase" >&2; exit_code=1 ;;
	esac

	if [ -z "$completion_promise" ]; then
		echo "❌ strict: completion_promise is empty; fill in a verifiable completion commitment in $STATE_FILE first." >&2
		exit_code=1
	fi

	if [ "$max_iterations" -gt 0 ] && [ "$iteration_count" -ge "$max_iterations" ]; then
		echo "❌ strict: iteration_count >= max_iterations; stop hook triggered, should pause and escalate to user." >&2
		exit_code=1
	fi

	if [ "$consecutive_failures" -ge 3 ]; then
		echo "❌ strict: consecutive_failures >= 3; 3-Failure Protocol triggered, should pause and escalate to user." >&2
		exit_code=1
	fi

	if [ -n "$ehrb_flags" ] && [ "$ehrb_flags" != "[]" ]; then
		echo "❌ strict: ehrb_flags not empty; EHRB risk exists, requires explicit user confirmation before continuing." >&2
		exit_code=1
	fi

	exit "$exit_code"
fi

exit 0
