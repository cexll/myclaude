#!/usr/bin/env python3
# /// script
# requires-python = ">=3.8"
# dependencies = []
# ///
"""
Gemini CLI wrapper with cross-platform support.

Usage:
    uv run gemini.py "<prompt>" [workdir]
    python3 gemini.py "<prompt>"
    ./gemini.py "your prompt"
"""
import subprocess
import sys
import os

DEFAULT_MODEL = os.environ.get('GEMINI_MODEL', 'gemini-3-pro-preview')
DEFAULT_WORKDIR = '.'
TIMEOUT_MS = 7_200_000  # 固定 2 小时，毫秒
DEFAULT_TIMEOUT = TIMEOUT_MS // 1000
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


def parse_args():
    """解析位置参数"""
    if len(sys.argv) < 2:
        log_error('Prompt required')
        sys.exit(1)

    return {
        'prompt': sys.argv[1],
        'workdir': sys.argv[2] if len(sys.argv) > 2 else DEFAULT_WORKDIR
    }


def build_gemini_args(args) -> list:
    """构建 gemini CLI 参数"""
    return [
        'gemini',
        '-m', DEFAULT_MODEL,
        '-p', args['prompt']
    ]


def main():
    log_info('Script started')
    args = parse_args()
    log_info(f"Prompt length: {len(args['prompt'])}")
    log_info(f"Working dir: {args['workdir']}")
    gemini_args = build_gemini_args(args)
    timeout_sec = DEFAULT_TIMEOUT
    log_info(f"Timeout: {timeout_sec}s")

    # 如果指定了工作目录，切换到该目录
    if args['workdir'] != DEFAULT_WORKDIR:
        try:
            os.chdir(args['workdir'])
        except FileNotFoundError:
            log_error(f"Working directory not found: {args['workdir']}")
            sys.exit(1)
        except PermissionError:
            log_error(f"Permission denied: {args['workdir']}")
            sys.exit(1)
        log_info('Changed working directory')

    try:
        log_info(f"Starting gemini with model {DEFAULT_MODEL}")
        process = None
        # 启动 gemini 子进程，直接透传 stdout 和 stderr
        process = subprocess.Popen(
            gemini_args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1  # 行缓冲
        )

        # 实时输出 stdout
        for line in process.stdout:
            sys.stdout.write(line)
            sys.stdout.flush()

        # 等待进程结束
        returncode = process.wait(timeout=timeout_sec)

        # 读取 stderr
        stderr_output = process.stderr.read()
        if stderr_output:
            sys.stderr.write(stderr_output)

        # 检查退出码
        if returncode != 0:
            log_error(f'Gemini exited with status {returncode}')
            sys.exit(returncode)

        sys.exit(0)

    except subprocess.TimeoutExpired:
        log_error(f'Gemini execution timeout ({timeout_sec}s)')
        if process is not None:
            process.kill()
            try:
                process.wait(timeout=FORCE_KILL_DELAY)
            except subprocess.TimeoutExpired:
                pass
        sys.exit(124)

    except FileNotFoundError:
        log_error("gemini command not found in PATH")
        log_error("Please install Gemini CLI: https://github.com/google/generative-ai-python")
        sys.exit(127)

    except KeyboardInterrupt:
        if process is not None:
            process.terminate()
            try:
                process.wait(timeout=FORCE_KILL_DELAY)
            except subprocess.TimeoutExpired:
                process.kill()
        sys.exit(130)


if __name__ == '__main__':
    main()
