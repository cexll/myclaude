#!/usr/bin/env python3
# /// script
# requires-python = ">=3.8"
# dependencies = []
# ///
"""
Gemini CLI wrapper with cross-platform support.

Usage:
    uv run gemini.py -m <model> -p "<prompt>" [workdir]
    python3 gemini.py -m <model> -p "<prompt>"
    ./gemini.py -m gemini-3-pro-preview -p "your prompt"
"""
import subprocess
import sys
import os
import argparse

DEFAULT_MODEL = os.environ.get('GEMINI_MODEL', 'gemini-3-pro-preview')
DEFAULT_WORKDIR = '.'
DEFAULT_TIMEOUT = 7200  # 2 hours in seconds
FORCE_KILL_DELAY = 5


def log_error(message: str):
    """输出错误信息到 stderr"""
    sys.stderr.write(f"ERROR: {message}\n")


def log_warn(message: str):
    """输出警告信息到 stderr"""
    sys.stderr.write(f"WARN: {message}\n")


def resolve_timeout() -> int:
    """解析超时配置（秒）"""
    raw = os.environ.get('GEMINI_TIMEOUT', '')
    if not raw:
        return DEFAULT_TIMEOUT

    try:
        parsed = int(raw)
        if parsed <= 0:
            log_warn(f"Invalid GEMINI_TIMEOUT '{raw}', falling back to {DEFAULT_TIMEOUT}s")
            return DEFAULT_TIMEOUT
        # 环境变量是毫秒，转换为秒
        return parsed // 1000 if parsed > 10000 else parsed
    except ValueError:
        log_warn(f"Invalid GEMINI_TIMEOUT '{raw}', falling back to {DEFAULT_TIMEOUT}s")
        return DEFAULT_TIMEOUT


def parse_args():
    """解析命令行参数"""
    parser = argparse.ArgumentParser(
        description='Gemini CLI wrapper for Claude Code integration',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument(
        '-m', '--model',
        default=DEFAULT_MODEL,
        help=f'Gemini model to use (default: {DEFAULT_MODEL})'
    )
    parser.add_argument(
        '-p', '--prompt',
        required=True,
        help='Prompt to send to Gemini'
    )
    parser.add_argument(
        'workdir',
        nargs='?',
        default=DEFAULT_WORKDIR,
        help='Working directory (default: current directory)'
    )

    return parser.parse_args()


def build_gemini_args(args) -> list:
    """构建 gemini CLI 参数"""
    return [
        'gemini',
        '-m', args.model,
        '-p', args.prompt
    ]


def main():
    args = parse_args()
    gemini_args = build_gemini_args(args)
    timeout_sec = resolve_timeout()

    # 如果指定了工作目录，切换到该目录
    if args.workdir != DEFAULT_WORKDIR:
        try:
            os.chdir(args.workdir)
        except FileNotFoundError:
            log_error(f"Working directory not found: {args.workdir}")
            sys.exit(1)
        except PermissionError:
            log_error(f"Permission denied: {args.workdir}")
            sys.exit(1)

    try:
        # 启动 gemini 子进程，直接透传 stdout 和 stderr
        process = subprocess.Popen(
            gemini_args,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1  # 行缓冲
        )

        # 实时输出 stdout
        stdout_lines = []
        for line in process.stdout:
            sys.stdout.write(line)
            sys.stdout.flush()
            stdout_lines.append(line)

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
        process.terminate()
        try:
            process.wait(timeout=FORCE_KILL_DELAY)
        except subprocess.TimeoutExpired:
            process.kill()
        sys.exit(130)


if __name__ == '__main__':
    main()
