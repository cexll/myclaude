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
state_file="${project_dir}/.claude/feature-dev.local.md"

if [ ! -f "$state_file" ]; then
	exit 0
fi

stdin_payload=""
if [ ! -t 0 ]; then
	stdin_payload="$(cat || true)"
fi

frontmatter_get() {
	local key="$1"
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
	' "$state_file"
}

active_raw="$(frontmatter_get active || true)"
active_lc="$(printf "%s" "$active_raw" | tr '[:upper:]' '[:lower:]')"
case "$active_lc" in
true|1|yes|on) ;;
*) exit 0 ;;
esac

current_phase_raw="$(frontmatter_get current_phase || true)"
max_phases_raw="$(frontmatter_get max_phases || true)"
phase_name="$(frontmatter_get phase_name || true)"
completion_promise="$(frontmatter_get completion_promise || true)"

current_phase=1
if [[ "${current_phase_raw:-}" =~ ^[0-9]+$ ]]; then
	current_phase="$current_phase_raw"
fi

max_phases=7
if [[ "${max_phases_raw:-}" =~ ^[0-9]+$ ]]; then
	max_phases="$max_phases_raw"
fi

if [ -z "${phase_name:-}" ]; then
	phase_name="$(phase_name_for "$current_phase")"
fi

if [ -z "${completion_promise:-}" ]; then
	completion_promise="<promise>FEATURE_COMPLETE</promise>"
fi

phases_done=0
if [ "$current_phase" -ge "$max_phases" ]; then
	phases_done=1
fi

promise_met=0
if [ -n "$completion_promise" ]; then
	if [ -n "$stdin_payload" ] && printf "%s" "$stdin_payload" | grep -Fq -- "$completion_promise"; then
		promise_met=1
	else
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
	exit 0
fi

if [ "$phases_done" -eq 0 ]; then
	reason="feature-dev 循环未完成：当前阶段 ${current_phase}/${max_phases}（${phase_name}）。继续执行剩余阶段；完成每个阶段后更新 ${state_file} 的 current_phase/phase_name。全部完成后在最终输出中包含 completion_promise：${completion_promise}。如需退出，将 active 设为 false。"
else
	reason="feature-dev 已到最终阶段（current_phase=${current_phase} / max_phases=${max_phases}，phase_name=${phase_name}），但未检测到 completion_promise：${completion_promise}。请在最终输出中包含该标记（或写入 ${state_file} 正文），然后再结束；如需强制退出，将 active 设为 false。"
fi

printf '{"decision":"block","reason":"%s"}\n' "$(json_escape "$reason")"
exit 0
