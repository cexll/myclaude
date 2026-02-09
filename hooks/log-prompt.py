#!/usr/bin/env python3
"""
Log Prompt Hook - Record user prompts to session-specific log files.
Used for review on Stop.
Uses session-isolated logs to handle concurrency.
"""

import json
import os
import sys
from datetime import datetime
from pathlib import Path


def get_session_id() -> str:
    """Get unique session identifier."""
    return os.environ.get("CLAUDE_CODE_SSE_PORT", "default")


def write_log(prompt: str) -> None:
    """Write prompt to session log file."""
    log_dir = Path(".claude/state")
    session_id = get_session_id()
    log_file = log_dir / f"session-{session_id}.log"

    log_dir.mkdir(parents=True, exist_ok=True)

    timestamp = datetime.now().isoformat()
    entry = f"[{timestamp}] {prompt[:500]}\n"

    with open(log_file, "a", encoding="utf-8") as f:
        f.write(entry)


def main():
    input_data = ""
    if not sys.stdin.isatty():
        try:
            input_data = sys.stdin.read()
        except Exception:
            pass

    prompt = ""
    try:
        data = json.loads(input_data)
        prompt = data.get("prompt", "")
    except json.JSONDecodeError:
        prompt = input_data.strip()

    if prompt:
        write_log(prompt)


if __name__ == "__main__":
    main()
