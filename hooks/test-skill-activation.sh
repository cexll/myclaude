#!/usr/bin/env bash

# Simple test runner for skill-activation-prompt hook.
# Each case feeds JSON to the hook and validates suggested skills.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
HOOK_SCRIPT="$SCRIPT_DIR/skill-activation-prompt.sh"

parse_skills() {
  node -e 'const data = JSON.parse(require("fs").readFileSync(0, "utf8")); const skills = (data.suggestedSkills || []).map(s => s.skill); console.log(skills.join(" "));'
}

run_case() {
  local name="$1"
  local input="$2"
  shift 2
  local expected=("$@")

  local output skills
  output="$("$HOOK_SCRIPT" <<<"$input")"
  skills="$(printf "%s" "$output" | parse_skills)"

  local pass=0
  if [[ ${#expected[@]} -eq 1 && ${expected[0]} == "none" ]]; then
    [[ -z "$skills" ]] && pass=1
  else
    pass=1
    for need in "${expected[@]}"; do
      if [[ " $skills " != *" $need "* ]]; then
        pass=0
        break
      fi
    done
  fi

  if [[ $pass -eq 1 ]]; then
    echo "PASS: $name"
  else
    echo "FAIL: $name"
    echo "  input: $input"
    echo "  expected skills: ${expected[*]}"
    echo "  actual skills: ${skills:-<empty>}"
    return 1
  fi
}

main() {
  local status=0

  run_case "keyword 'issue' => gh-workflow" \
    '{"prompt":"Please open an issue for this bug"}' \
    "gh-workflow" || status=1

  run_case "keyword 'codex' => codex" \
    '{"prompt":"codex please handle this change"}' \
    "codex" || status=1

  run_case "no matching keywords => none" \
    '{"prompt":"Just saying hello"}' \
    "none" || status=1

  run_case "multiple keywords => codex & gh-workflow" \
    '{"prompt":"codex refactor then open an issue"}' \
    "codex" "gh-workflow" || status=1

  if [[ $status -eq 0 ]]; then
    echo "All tests passed."
  else
    echo "Some tests failed."
  fi

  exit "$status"
}

main "$@"
