#!/usr/bin/env python3
# /// script
# requires-python = ">=3.8"
# dependencies = []
# ///
"""
Codex CLI wrapper with cross-platform support and session management.
**FIXED**: Auto-detect long inputs and use stdin mode to avoid shell argument issues.

Usage:
    New session:  uv run codex.py "task" [model] [workdir]
    Resume:       uv run codex.py resume <session_id> "task" [model] [workdir]
    Alternative:  python3 codex.py "task"
    Direct exec:  ./codex.py "task"
"""
import subprocess
import json
import sys
import os
from typing import Optional

DEFAULT_MODEL = 'gpt-5.1-codex'
DEFAULT_WORKDIR = '.'
DEFAULT_TIMEOUT = 7200  # 2 hours in seconds
FORCE_KILL_DELAY = 5
STDIN_THRESHOLD = 800  # Auto-switch to stdin for prompts longer than 800 chars


def log_error(message: str):
    """输出错误信息到 stderr"""
    sys.stderr.write(f"ERROR: {message}\n")


def log_warn(message: str):
    """输出警告信息到 stderr"""
    sys.stderr.write(f"WARN: {message}\n")


def resolve_timeout() -> int:
    """解析超时配置（秒）"""
    raw = os.environ.get('CODEX_TIMEOUT', '')
    if not raw:
        return DEFAULT_TIMEOUT

    try:
        parsed = int(raw)
        if parsed <= 0:
            log_warn(f"Invalid CODEX_TIMEOUT '{raw}', falling back to {DEFAULT_TIMEOUT}s")
            return DEFAULT_TIMEOUT
        # 环境变量是毫秒，转换为秒
        return parsed // 1000 if parsed > 10000 else parsed
    except ValueError:
        log_warn(f"Invalid CODEX_TIMEOUT '{raw}', falling back to {DEFAULT_TIMEOUT}s")
        return DEFAULT_TIMEOUT


def normalize_text(text) -> Optional[str]:
    """规范化文本：字符串或字符串数组"""
    if isinstance(text, str):
        return text
    if isinstance(text, list):
        return ''.join(text)
    return None


def parse_args():
    """解析命令行参数"""
    if len(sys.argv) < 2:
        log_error('Task required')
        sys.exit(1)

    # 检测是否为 resume 模式
    if sys.argv[1] == 'resume':
        if len(sys.argv) < 4:
            log_error('Resume mode requires: resume <session_id> <task>')
            sys.exit(1)
        return {
            'mode': 'resume',
            'session_id': sys.argv[2],
            'task': sys.argv[3],
            'model': sys.argv[4] if len(sys.argv) > 4 else DEFAULT_MODEL,
            'workdir': sys.argv[5] if len(sys.argv) > 5 else DEFAULT_WORKDIR
        }
    else:
        return {
            'mode': 'new',
            'task': sys.argv[1],
            'model': sys.argv[2] if len(sys.argv) > 2 else DEFAULT_MODEL,
            'workdir': sys.argv[3] if len(sys.argv) > 3 else DEFAULT_WORKDIR
        }


def build_codex_args(params: dict, use_stdin: bool) -> list:
    """
    构建 codex CLI 参数

    Args:
        params: 参数字典
        use_stdin: 是否使用 stdin 模式（不在命令行参数中传递 task）
    """
    if params['mode'] == 'resume':
        if use_stdin:
            return [
                'codex', 'e',
                '--skip-git-repo-check',
                '--json',
                'resume',
                params['session_id'],
                '-'  # 从 stdin 读取
            ]
        else:
            return [
                'codex', 'e',
                '--skip-git-repo-check',
                '--json',
                'resume',
                params['session_id'],
                params['task']
            ]
    else:
        base_args = [
            'codex', 'e',
            '-m', params['model'],
            '--dangerously-bypass-approvals-and-sandbox',
            '--skip-git-repo-check',
            '-C', params['workdir'],
            '--json'
        ]

        if use_stdin:
            base_args.append('-')  # 从 stdin 读取
        else:
            base_args.append(params['task'])

        return base_args


def main():
    params = parse_args()
    timeout_sec = resolve_timeout()

    # **FIX: Auto-detect long inputs and enable stdin mode**
    task_length = len(params['task'])
    use_stdin = task_length > STDIN_THRESHOLD

    if use_stdin:
        log_warn(f"Task length ({task_length} chars) exceeds threshold, using stdin mode to avoid shell escaping issues")

    codex_args = build_codex_args(params, use_stdin)

    thread_id: Optional[str] = None
    last_agent_message: Optional[str] = None

    try:
        # 启动 codex 子进程
        process = subprocess.Popen(
            codex_args,
            stdin=subprocess.PIPE if use_stdin else None,  # **FIX: Enable stdin**
            stdout=subprocess.PIPE,
            stderr=sys.stderr,  # 错误直接透传到 stderr
            text=True,
            bufsize=1  # 行缓冲
        )

        # **FIX: 如果使用 stdin 模式，写入任务到 stdin**
        if use_stdin:
            process.stdin.write(params['task'])
            process.stdin.close()

        # 逐行解析 JSON 输出
        for line in process.stdout:
            line = line.strip()
            if not line:
                continue

            try:
                event = json.loads(line)

                # 捕获 thread_id
                if event.get('type') == 'thread.started':
                    thread_id = event.get('thread_id')

                # 捕获 agent_message
                if (event.get('type') == 'item.completed' and
                    event.get('item', {}).get('type') == 'agent_message'):
                    text = normalize_text(event['item'].get('text'))
                    if text:
                        last_agent_message = text

            except json.JSONDecodeError:
                log_warn(f"Failed to parse line: {line}")

        # 等待进程结束
        returncode = process.wait(timeout=timeout_sec)

        if returncode == 0:
            if last_agent_message:
                # 输出 agent_message
                sys.stdout.write(f"{last_agent_message}\n")

                # 输出 session_id（如果存在）
                if thread_id:
                    sys.stdout.write(f"\n---\nSESSION_ID: {thread_id}\n")

                sys.exit(0)
            else:
                log_error('Codex completed without agent_message output')
                sys.exit(1)
        else:
            log_error(f'Codex exited with status {returncode}')
            sys.exit(returncode)

    except subprocess.TimeoutExpired:
        log_error('Codex execution timeout')
        process.kill()
        try:
            process.wait(timeout=FORCE_KILL_DELAY)
        except subprocess.TimeoutExpired:
            pass
        sys.exit(124)

    except FileNotFoundError:
        log_error("codex command not found in PATH")
        sys.exit(127)

    except KeyboardInterrupt:
        process.terminate()
        try:
            process.wait(timeout=FORCE_KILL_DELAY)
        except subprocess.TimeoutExpired:
            process.kill()
        sys.exit(130)


if __name__ == '__main__':
    main()
