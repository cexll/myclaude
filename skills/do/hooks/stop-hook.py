#!/usr/bin/env python3
import glob
import json
import os
import re
import sys

PHASE_NAMES = {
    1: "Understand",
    2: "Clarify",
    3: "Design",
    4: "Implement",
    5: "Complete",
}

def phase_name_for(n: int) -> str:
    return PHASE_NAMES.get(n, f"Phase {n}")

def frontmatter_get(file_path: str, key: str) -> str:
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            lines = f.readlines()
    except Exception:
        return ""

    if not lines or lines[0].strip() != "---":
        return ""

    for i, line in enumerate(lines[1:], start=1):
        if line.strip() == "---":
            break
        match = re.match(rf"^{re.escape(key)}:\s*(.*)$", line)
        if match:
            value = match.group(1).strip()
            if value.startswith('"') and value.endswith('"'):
                value = value[1:-1]
            return value
    return ""

def get_body(file_path: str) -> str:
    try:
        with open(file_path, "r", encoding="utf-8") as f:
            content = f.read()
    except Exception:
        return ""

    parts = content.split("---", 2)
    if len(parts) >= 3:
        return parts[2]
    return ""

def check_state_file(state_file: str, stdin_payload: str) -> str:
    active_raw = frontmatter_get(state_file, "active")
    active_lc = active_raw.lower()
    if active_lc not in ("true", "1", "yes", "on"):
        return ""

    current_phase_raw = frontmatter_get(state_file, "current_phase")
    max_phases_raw = frontmatter_get(state_file, "max_phases")
    phase_name = frontmatter_get(state_file, "phase_name")
    completion_promise = frontmatter_get(state_file, "completion_promise")

    try:
        current_phase = int(current_phase_raw)
    except (ValueError, TypeError):
        current_phase = 1

    try:
        max_phases = int(max_phases_raw)
    except (ValueError, TypeError):
        max_phases = 5

    if not phase_name:
        phase_name = phase_name_for(current_phase)

    if not completion_promise:
        completion_promise = "<promise>DO_COMPLETE</promise>"

    phases_done = current_phase >= max_phases

    if phases_done:
        # 阶段已完成，清理状态文件并允许退出
        # promise 检测作为可选确认，不阻止退出
        try:
            os.remove(state_file)
        except Exception:
            pass
        return ""

    return (f"do loop incomplete: current phase {current_phase}/{max_phases} ({phase_name}). "
            f"Continue with remaining phases; update {state_file} current_phase/phase_name after each phase. "
            f"Include completion_promise in final output when done: {completion_promise}. "
            f"To exit early, set active to false.")

def main():
    project_dir = os.environ.get("CLAUDE_PROJECT_DIR", os.getcwd())
    state_dir = os.path.join(project_dir, ".claude")

    do_task_id = os.environ.get("DO_TASK_ID", "")

    if do_task_id:
        candidate = os.path.join(state_dir, f"do.{do_task_id}.local.md")
        state_files = [candidate] if os.path.isfile(candidate) else []
    else:
        state_files = glob.glob(os.path.join(state_dir, "do.*.local.md"))

    if not state_files:
        sys.exit(0)

    stdin_payload = ""
    if not sys.stdin.isatty():
        try:
            stdin_payload = sys.stdin.read()
        except Exception:
            pass

    blocking_reasons = []
    for state_file in state_files:
        reason = check_state_file(state_file, stdin_payload)
        if reason:
            blocking_reasons.append(reason)

    if not blocking_reasons:
        sys.exit(0)

    combined_reason = " ".join(blocking_reasons)
    print(json.dumps({"decision": "block", "reason": combined_reason}))
    sys.exit(0)

if __name__ == "__main__":
    main()
