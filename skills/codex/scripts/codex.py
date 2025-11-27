#!/usr/bin/env python3
# /// script
# requires-python = ">=3.8"
# dependencies = []
# ///
"""
Codex CLI wrapper with cross-platform support and session management.
**FIXED**: Auto-detect long inputs and use stdin mode to avoid shell argument issues.

Usage:
    New session:  uv run codex.py "task" [workdir]
    Stdin mode:   uv run codex.py - [workdir]
    Resume:       uv run codex.py resume <session_id> "task" [workdir]
    Resume stdin: uv run codex.py resume <session_id> - [workdir]
    Alternative:  python3 codex.py "task"
    Direct exec:  ./codex.py "task"

    Model configuration: Set CODEX_MODEL environment variable (default: gpt-5.1-codex)
"""
import subprocess
import json
import sys
import os
from typing import Optional

DEFAULT_MODEL = os.environ.get('CODEX_MODEL', 'gpt-5.1-codex')
DEFAULT_WORKDIR = '.'
DEFAULT_TIMEOUT = 7200  # 2 hours in seconds
FORCE_KILL_DELAY = 5


def log_error(message: str):
    """输出错误信息到 stderr"""
    sys.stderr.write(f"ERROR: {message}\n")


def log_warn(message: str):
    """输出警告信息到 stderr"""
    sys.stderr.write(f"WARN: {message}\n")


def log_info(message: str):
    """输出信息到 stderr"""
    sys.stderr.write(f"INFO: {message}\n")


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
        task_arg = sys.argv[3]
        return {
            'mode': 'resume',
            'session_id': sys.argv[2],
            'task': task_arg,
            'explicit_stdin': task_arg == '-',
            'workdir': sys.argv[4] if len(sys.argv) > 4 else DEFAULT_WORKDIR,
        }

    task_arg = sys.argv[1]
    return {
        'mode': 'new',
        'task': task_arg,
        'explicit_stdin': task_arg == '-',
        'workdir': sys.argv[2] if len(sys.argv) > 2 else DEFAULT_WORKDIR,
    }


def read_piped_task() -> Optional[str]:
    """
    从 stdin 读取任务文本：
    - 如果 stdin 是管道（非 tty）且存在内容，返回读取到的字符串
    - 否则返回 None
    """
    stdin = sys.stdin
    if stdin is None or stdin.isatty():
        log_info("Stdin is tty or None, skipping pipe read")
        return None
    log_info("Reading from stdin pipe...")
    data = stdin.read()
    if not data:
        log_info("Stdin pipe returned empty data")
        return None

    log_info(f"Read {len(data)} bytes from stdin pipe")
    return data


def should_stream_via_stdin(task_text: str, piped: bool) -> bool:
    """
    判定是否通过 stdin 传递任务：
    - 有管道输入
    - 文本包含换行
    - 文本包含反斜杠
    - 文本长度 > 800
    """
    if piped:
        return True
    if '\n' in task_text:
        return True
    if '\\' in task_text:
        return True
    if len(task_text) > 800:
        return True
    return False


def build_codex_args(params: dict, target_arg: str) -> list:
    """
    构建 codex CLI 参数

    Args:
        params: 参数字典
        target_arg: 最终传递给 codex 的参数（'-' 或具体 task 文本）
    """
    if params['mode'] == 'resume':
        return [
            'codex', 'e',
            '-m', DEFAULT_MODEL,
            '--skip-git-repo-check',
            '--json',
            'resume',
            params['session_id'],
            target_arg
        ]
    else:
        base_args = [
            'codex', 'e',
            '-m', DEFAULT_MODEL,
            '--dangerously-bypass-approvals-and-sandbox',
            '--skip-git-repo-check',
            '-C', params['workdir'],
            '--json',
            target_arg
        ]

        return base_args


def run_codex_process(codex_args, task_text: str, use_stdin: bool, timeout_sec: int):
    """
    启动 codex 子进程，处理 stdin / JSON 行输出和错误，成功时返回 (last_agent_message, thread_id)。
    失败路径上负责日志和退出码。
    """
    thread_id: Optional[str] = None
    last_agent_message: Optional[str] = None
    process: Optional[subprocess.Popen] = None

    try:
        # 启动 codex 子进程（文本模式管道）
        log_info(f"Starting codex with args: {' '.join(codex_args[:5])}...")
        process = subprocess.Popen(
            codex_args,
            stdin=subprocess.PIPE if use_stdin else None,
            stdout=subprocess.PIPE,
            stderr=sys.stderr,
            text=True,
            bufsize=1,
        )
        log_info(f"Process started with PID: {process.pid}")

        # 如果使用 stdin 模式，写入任务到 stdin 并关闭
        if use_stdin and process.stdin is not None:
            log_info(f"Writing {len(task_text)} chars to stdin...")
            process.stdin.write(task_text)
            process.stdin.flush()  # 强制刷新缓冲区，避免大任务死锁
            process.stdin.close()
            log_info("Stdin closed")

        # 逐行解析 JSON 输出
        if process.stdout is None:
            log_error('Codex stdout pipe not available')
            sys.exit(1)

        log_info("Reading stdout...")

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

        # 等待进程结束并检查退出码
        returncode = process.wait(timeout=timeout_sec)
        if returncode != 0:
            log_error(f'Codex exited with status {returncode}')
            sys.exit(returncode)

        if not last_agent_message:
            log_error('Codex completed without agent_message output')
            sys.exit(1)

        return last_agent_message, thread_id

    except subprocess.TimeoutExpired:
        log_error('Codex execution timeout')
        if process is not None:
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
        log_error("Codex interrupted by user")
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=FORCE_KILL_DELAY)
            except subprocess.TimeoutExpired:
                process.kill()
        sys.exit(130)


def main():
    log_info("Script started")
    params = parse_args()
    log_info(f"Parsed args: mode={params['mode']}, task_len={len(params['task'])}")
    timeout_sec = resolve_timeout()
    log_info(f"Timeout: {timeout_sec}s")

    explicit_stdin = params.get('explicit_stdin', False)

    if explicit_stdin:
        log_info("Explicit stdin mode: reading task from stdin")
        task_text = sys.stdin.read()
        if not task_text:
            log_error("Explicit stdin mode requires task input from stdin")
            sys.exit(1)
        piped = not sys.stdin.isatty()
    else:
        piped_task = read_piped_task()
        piped = piped_task is not None
        task_text = piped_task if piped else params['task']

    use_stdin = explicit_stdin or should_stream_via_stdin(task_text, piped)

    if use_stdin:
        reasons = []
        if piped:
            reasons.append('piped input')
        if explicit_stdin:
            reasons.append('explicit "-"')
        if '\n' in task_text:
            reasons.append('newline')
        if '\\' in task_text:
            reasons.append('backslash')
        if len(task_text) > 800:
            reasons.append('length>800')

        if reasons:
            log_warn(f"Using stdin mode for task due to: {', '.join(reasons)}")

    target_arg = '-' if use_stdin else params['task']
    codex_args = build_codex_args(params, target_arg)

    log_info('codex running...')

    last_agent_message, thread_id = run_codex_process(
        codex_args=codex_args,
        task_text=task_text,
        use_stdin=use_stdin,
        timeout_sec=timeout_sec,
    )

    # 输出 agent_message
    sys.stdout.write(f"{last_agent_message}\n")

    # 输出 session_id（如果存在）
    if thread_id:
        sys.stdout.write(f"\n---\nSESSION_ID: {thread_id}\n")

    sys.exit(0)


if __name__ == '__main__':
    main()
