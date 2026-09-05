#!/usr/bin/env python3
"""
Pre-Bash Hook - Block dangerous commands before execution.
"""

import sys

DANGEROUS_PATTERNS = [
    'rm -rf /',
    'rm -rf ~',
    'dd if=',
    ':(){:|:&};:',
    'mkfs.',
    '> /dev/sd',
]


def main():
    command = sys.argv[1] if len(sys.argv) > 1 else ''

    for pattern in DANGEROUS_PATTERNS:
        if pattern in command:
            print(f"[CWF] BLOCKED: Dangerous command detected: {pattern}", file=sys.stderr)
            sys.exit(1)

    sys.exit(0)


if __name__ == "__main__":
    main()
