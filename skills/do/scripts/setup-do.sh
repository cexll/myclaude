#!/usr/bin/env bash

set -euo pipefail

usage() {
	cat <<'EOF'
Usage: setup-do.sh [options] PROMPT...

Creates (or overwrites) project state file:
  .claude/do.local.md

Options:
  --max-phases N            Default: 7
  --completion-promise STR  Default: <promise>DO_COMPLETE</promise>
  -h, --help                Show this help
EOF
}

die() {
	echo "❌ $*" >&2
	exit 1
}

phase_name_for() {
	case "${1:-}" in
	1) echo "Discovery" ;;
	2) echo "Exploration" ;;
	3) echo "Clarification" ;;
	4) echo "Architecture" ;;
	5) echo "Implementation" ;;
	6) echo "Review" ;;
	7) echo "Summary" ;;
	*) echo "Phase ${1:-unknown}" ;;
	esac
}

max_phases=7
completion_promise="<promise>DO_COMPLETE</promise>"
declare -a prompt_parts=()

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help)
		usage
		exit 0
		;;
	--max-phases)
		[ $# -ge 2 ] || die "--max-phases requires a value"
		max_phases="$2"
		shift 2
		;;
	--completion-promise)
		[ $# -ge 2 ] || die "--completion-promise requires a value"
		completion_promise="$2"
		shift 2
		;;
	--)
		shift
		while [ $# -gt 0 ]; do
			prompt_parts+=("$1")
			shift
		done
		break
		;;
	-*)
		die "Unknown argument: $1 (use --help)"
		;;
	*)
		prompt_parts+=("$1")
		shift
		;;
	esac
done

prompt="${prompt_parts[*]:-}"
[ -n "$prompt" ] || die "PROMPT is required (use --help)"

if ! [[ "$max_phases" =~ ^[0-9]+$ ]] || [ "$max_phases" -lt 1 ]; then
	die "--max-phases must be a positive integer"
fi

project_dir="${CLAUDE_PROJECT_DIR:-$PWD}"
state_dir="${project_dir}/.claude"

task_id="$(date +%s)-$$-$(head -c 4 /dev/urandom | od -An -tx1 | tr -d ' \n')"
state_file="${state_dir}/do.${task_id}.local.md"

mkdir -p "$state_dir"

phase_name="$(phase_name_for 1)"

cat > "$state_file" << EOF
---
active: true
current_phase: 1
phase_name: "$phase_name"
max_phases: $max_phases
completion_promise: "$completion_promise"
---

# do loop state

## Prompt
$prompt

## Notes
- Update frontmatter current_phase/phase_name as you progress
- When complete, include the frontmatter completion_promise in your final output
EOF

echo "Initialized: $state_file"
echo "task_id: $task_id"
echo "phase: 1/$max_phases ($phase_name)"
echo "completion_promise: $completion_promise"
echo "export DO_TASK_ID=$task_id"
