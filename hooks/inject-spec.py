#!/usr/bin/env python3
"""
Global Spec Injection Hook (DEPRECATED).

Spec injection is now handled internally by codeagent-wrapper via the
per-task `skills:` field in parallel config and the `--skills` CLI flag.

This hook is kept as a no-op for backward compatibility.
"""

import sys

sys.exit(0)
