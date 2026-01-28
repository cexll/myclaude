#!/usr/bin/env bash

set -euo pipefail

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

json_escape() {
	local s="${1:-}"
	s=${s//\\/\\\\}
	s=${s//\"/\\\"}
	s=${s//$'\n'/\\n}
	s=${s//$'\r'/\\r}
	s=${s//$'\t'/\\t}
	printf "%s" "$s"
}

project_dir="${CLAUDE_PROJECT_DIR:-$PWD}"
state_dir="${project_dir}/.claude"

shopt -s nullglob
if [ -n "${DO_TASK_ID:-}" ]; then
	candidate="${state_dir}/do.${DO_TASK_ID}.local.md"
	if [ -f "$candidate" ]; then
		state_files=("$candidate")
	else
		state_files=()
	fi
else
	state_files=("${state_dir}"/do.*.local.md)
fi
shopt -u nullglob

if [ ${#state_files[@]} -eq 0 ]; then
	exit 0
fi

stdin_payload=""
if [ ! -t 0 ]; then
	stdin_payload="$(cat || true)"
fi

frontmatter_get() {
	local file="$1" key="$2"
	awk -v k="$key" '
		BEGIN { in_fm=0 }
		NR==1 && $0=="---" { in_fm=1; next }
		in_fm==1 && $0=="---" { exit }
		in_fm==1 {
			if ($0 ~ "^"k":[[:space:]]*") {
				sub("^"k":[[:space:]]*", "", $0)
				gsub(/^[[:space:]]+|[[:space:]]+$/, "", $0)
				if ($0 ~ /^".*"$/) { sub(/^"/, "", $0); sub(/"$/, "", $0) }
				print $0
				exit
			}
		}
	' "$file"
}

check_state_file() {
	local state_file="$1"

	local active_raw active_lc
	active_raw="$(frontmatter_get "$state_file" active || true)"
	active_lc="$(printf "%s" "$active_raw" | tr '[:upper:]' '[:lower:]')"
	case "$active_lc" in
	true|1|yes|on) ;;
	*) return 0 ;;
	esac

	local current_phase_raw max_phases_raw phase_name completion_promise
	current_phase_raw="$(frontmatter_get "$state_file" current_phase || true)"
	max_phases_raw="$(frontmatter_get "$state_file" max_phases || true)"
	phase_name="$(frontmatter_get "$state_file" phase_name || true)"
	completion_promise="$(frontmatter_get "$state_file" completion_promise || true)"

	local current_phase=1
	if [[ "${current_phase_raw:-}" =~ ^[0-9]+$ ]]; then
		current_phase="$current_phase_raw"
	fi

	local max_phases=7
	if [[ "${max_phases_raw:-}" =~ ^[0-9]+$ ]]; then
		max_phases="$max_phases_raw"
	fi

	if [ -z "${phase_name:-}" ]; then
		phase_name="$(phase_name_for "$current_phase")"
	fi

	if [ -z "${completion_promise:-}" ]; then
		completion_promise="<promise>DO_COMPLETE</promise>"
	fi

	local phases_done=0
	if [ "$current_phase" -ge "$max_phases" ]; then
		phases_done=1
	fi

	local promise_met=0
	if [ -n "$completion_promise" ]; then
		if [ -n "$stdin_payload" ] && printf "%s" "$stdin_payload" | grep -Fq -- "$completion_promise"; then
			promise_met=1
		else
			local body
			body="$(
				awk '
					BEGIN { in_fm=0; body=0 }
					NR==1 && $0=="---" { in_fm=1; next }
					in_fm==1 && $0=="---" { body=1; in_fm=0; next }
					body==1 { print }
				' "$state_file"
			)"
			if [ -n "$body" ] && printf "%s" "$body" | grep -Fq -- "$completion_promise"; then
				promise_met=1
			fi
		fi
	fi

	if [ "$phases_done" -eq 1 ] && [ "$promise_met" -eq 1 ]; then
		rm -f "$state_file"
		return 0
	fi

	local reason
	if [ "$phases_done" -eq 0 ]; then
		reason="do loop incomplete: current phase ${current_phase}/${max_phases} (${phase_name}). Continue with remaining phases; update ${state_file} current_phase/phase_name after each phase. Include completion_promise in final output when done: ${completion_promise}. To exit early, set active to false."
	else
		reason="do reached final phase (current_phase=${current_phase} / max_phases=${max_phases}, phase_name=${phase_name}), but completion_promise not detected: ${completion_promise}. Please include this marker in your final output (or write it to ${state_file} body), then finish; to force exit, set active to false."
	fi

	printf "%s" "$reason"
}

blocking_reasons=()
for state_file in "${state_files[@]}"; do
	reason="$(check_state_file "$state_file")"
	if [ -n "$reason" ]; then
		blocking_reasons+=("$reason")
	fi
done

if [ ${#blocking_reasons[@]} -eq 0 ]; then
	exit 0
fi

combined_reason="${blocking_reasons[*]}"
printf '{"decision":"block","reason":"%s"}\n' "$(json_escape "$combined_reason")"
exit 0
