#!/bin/bash
# EHRB Risk Detection Script
# Heuristically detects high-risk changes/specs and writes flags to .sparv/state.yaml:ehrb_flags.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/state-lock.sh"

usage() {
	cat <<'EOF'
Usage: check-ehrb.sh [options] [FILE...]

Options:
  --diff            Scan current git diff (staged + unstaged) and changed file names
  --clear           Clear ehrb_flags in .sparv/state.yaml (no scan needed)
  --dry-run         Do not write .sparv/state.yaml (print detected flags only)
  --fail-on-flags   Exit with code 2 if any flags are detected
  -h, --help        Show this help

Input:
  - --diff
  - positional FILE...
  - stdin (if piped)

Examples:
  check-ehrb.sh --diff --fail-on-flags
  check-ehrb.sh docs/feature-prd.md
  echo "touching production db" | check-ehrb.sh --fail-on-flags
EOF
}

die() {
	echo "❌ $*" >&2
	exit 1
}

is_piped_stdin() {
	[ ! -t 0 ]
}

git_text() {
	git diff --cached 2>/dev/null || true
	git diff 2>/dev/null || true
	(git diff --name-only --cached 2>/dev/null; git diff --name-only 2>/dev/null) | sort -u || true
}

render_inline_list() {
	if [ "$#" -eq 0 ]; then
		printf "[]"
		return 0
	fi
	printf "["
	local first=1 item
	for item in "$@"; do
		if [ "$first" -eq 1 ]; then
			first=0
		else
			printf ", "
		fi
		printf "\"%s\"" "$item"
	done
	printf "]"
}

write_ehrb_flags() {
	local list_value="$1"
	sparv_require_state_file
	sparv_state_validate_or_die
	sparv_yaml_set_raw ehrb_flags "$list_value"
}

scan_diff=0
dry_run=0
clear=0
fail_on_flags=0
declare -a files=()

while [ $# -gt 0 ]; do
	case "$1" in
	-h|--help)
		usage
		exit 0
		;;
	--diff)
		scan_diff=1
		shift
		;;
	--clear)
		clear=1
		shift
		;;
	--dry-run)
		dry_run=1
		shift
		;;
	--fail-on-flags)
		fail_on_flags=1
		shift
		;;
	--)
		shift
		break
		;;
	-*)
		die "Unknown argument: $1 (use --help for usage)"
		;;
	*)
		files+=("$1")
		shift
		;;
	esac
done

for path in "$@"; do
	files+=("$path")
done

	scan_text=""

if [ "$scan_diff" -eq 1 ]; then
	if git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
		scan_text+=$'\n'"$(git_text)"
	else
		die "--diff requires running inside a git repository"
	fi
fi

if [ "${#files[@]}" -gt 0 ]; then
	for path in "${files[@]}"; do
		[ -f "$path" ] || die "File not found: $path"
		scan_text+=$'\n'"$(cat "$path")"
	done
fi

	if is_piped_stdin; then
		scan_text+=$'\n'"$(cat)"
	fi

	declare -a flags=()
	if [ "$clear" -eq 1 ]; then
		flags=()
	else
		[ -n "$scan_text" ] || die "No scannable input (use --help to see input methods)"

		if printf "%s" "$scan_text" | grep -Eiq '(^|[^a-z])(prod(uction)?|live)([^a-z]|$)|kubeconfig|kubectl|terraform|helm|eks|gke|aks'; then
			flags+=("production-access")
		fi
		if printf "%s" "$scan_text" | grep -Eiq 'pii|phi|hipaa|ssn|password|passwd|secret|token|api[ _-]?key|private key|credit card|身份证|银行卡|医疗|患者'; then
			flags+=("sensitive-data")
		fi
		if printf "%s" "$scan_text" | grep -Eiq 'rm[[:space:]]+-rf|drop[[:space:]]+table|delete[[:space:]]+from|truncate|terraform[[:space:]]+destroy|kubectl[[:space:]]+delete|drop[[:space:]]+database|wipe|purge'; then
			flags+=("destructive-ops")
		fi
		if printf "%s" "$scan_text" | grep -Eiq 'stripe|paypal|billing|charge|invoice|subscription|metering|twilio|sendgrid|openai|anthropic|cost|usage'; then
			flags+=("billing-external-api")
		fi
		if printf "%s" "$scan_text" | grep -Eiq 'auth|authentication|authorization|oauth|jwt|sso|encryption|crypto|tls|ssl|mfa|rbac|permission|权限|登录|认证'; then
			flags+=("security-critical")
		fi
	fi

if [ "${#flags[@]}" -eq 0 ]; then
	echo "EHRB: No risk flags detected"
else
	echo "EHRB: Risk flags detected (require explicit user confirmation):"
	for f in ${flags[@]+"${flags[@]}"}; do
		echo "  - $f"
	done
fi

if [ "$dry_run" -eq 0 ]; then
	list_value="$(render_inline_list ${flags[@]+"${flags[@]}"})"
	write_ehrb_flags "$list_value"
	echo "Written to: $STATE_FILE (ehrb_flags: $list_value)"
fi

if [ "$fail_on_flags" -eq 1 ] && [ "${#flags[@]}" -gt 0 ]; then
	exit 2
fi

exit 0
