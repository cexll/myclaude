#!/usr/bin/env python3
"""Unit tests for harness hook scripts.

Tests the activation guard (.harness-active marker), task state logic,
and edge cases for all 4 hooks: Stop, SessionStart, TeammateIdle, SubagentStop.
"""
from __future__ import annotations

import json
import os
import subprocess
import sys
import tempfile
import unittest
from pathlib import Path

HOOKS_DIR = Path(__file__).resolve().parent.parent / "hooks"
STOP_HOOK = HOOKS_DIR / "harness-stop.py"
SESSION_HOOK = HOOKS_DIR / "harness-sessionstart.py"
IDLE_HOOK = HOOKS_DIR / "harness-teammateidle.py"
SUBAGENT_HOOK = HOOKS_DIR / "harness-subagentstop.py"


def build_hook_env(env_extra: dict | None = None) -> dict[str, str]:
    """Build an isolated environment for hook subprocesses."""
    env = os.environ.copy()
    # Clear harness env vars to avoid interference
    env.pop("HARNESS_STATE_ROOT", None)
    env.pop("HARNESS_WORKER_ID", None)
    env.pop("CLAUDE_PROJECT_DIR", None)
    if env_extra:
        env.update(env_extra)
    return env


def run_hook(script: Path, payload: dict, env_extra: dict | None = None) -> tuple[int, str, str]:
    """Run a hook script with JSON payload on stdin. Returns (exit_code, stdout, stderr)."""
    env = build_hook_env(env_extra)
    proc = subprocess.run(
        [sys.executable, str(script)],
        input=json.dumps(payload),
        capture_output=True,
        text=True,
        timeout=10,
        env=env,
    )
    return proc.returncode, proc.stdout.strip(), proc.stderr.strip()


def write_tasks(root: Path, tasks: list[dict], **extra) -> None:
    state = {"tasks": tasks, **extra}
    (root / "harness-tasks.json").write_text(json.dumps(state), encoding="utf-8")


def activate(root: Path) -> None:
    (root / ".harness-active").touch()


def deactivate(root: Path) -> None:
    p = root / ".harness-active"
    if p.exists():
        p.unlink()


# ---------------------------------------------------------------------------
# Activation Guard Tests (shared across all hooks)
# ---------------------------------------------------------------------------
class TestActivationGuard(unittest.TestCase):
    """All hooks must be no-ops when .harness-active is absent."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)
        write_tasks(self.root, [
            {"id": "t1", "title": "Pending task", "status": "pending", "priority": "P0", "depends_on": []},
        ])
        (self.root / "harness-progress.txt").write_text("[SESSION-1] INIT\n")

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def _payload(self, **extra):
        return {"cwd": self.tmpdir, **extra}

    def test_stop_inactive_allows(self):
        """Stop hook allows stop when .harness-active is absent."""
        deactivate(self.root)
        code, stdout, stderr = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_stop_active_blocks(self):
        """Stop hook blocks when .harness-active is present and tasks remain."""
        activate(self.root)
        code, stdout, stderr = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")

    def test_sessionstart_inactive_noop(self):
        """SessionStart hook produces no output when inactive."""
        deactivate(self.root)
        code, stdout, stderr = run_hook(SESSION_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_sessionstart_active_injects(self):
        """SessionStart hook injects context when active."""
        activate(self.root)
        code, stdout, stderr = run_hook(SESSION_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertIn("additionalContext", data.get("hookSpecificOutput", {}))

    def test_teammateidle_inactive_allows(self):
        """TeammateIdle hook allows idle when inactive."""
        deactivate(self.root)
        code, stdout, stderr = run_hook(IDLE_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stderr, "")

    def test_teammateidle_active_blocks(self):
        """TeammateIdle hook blocks idle when active and tasks remain."""
        activate(self.root)
        code, stdout, stderr = run_hook(IDLE_HOOK, self._payload())
        self.assertEqual(code, 2)
        self.assertIn("HARNESS", stderr)

    def test_subagentstop_inactive_allows(self):
        """SubagentStop hook allows stop when inactive."""
        deactivate(self.root)
        code, stdout, stderr = run_hook(SUBAGENT_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_subagentstop_active_blocks(self):
        """SubagentStop hook blocks when active and tasks in progress."""
        write_tasks(self.root, [
            {"id": "t1", "title": "Working task", "status": "in_progress", "priority": "P0", "depends_on": []},
        ])
        activate(self.root)
        code, stdout, stderr = run_hook(SUBAGENT_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")


# ---------------------------------------------------------------------------
# No Harness Root Tests
# ---------------------------------------------------------------------------
class TestNoHarnessRoot(unittest.TestCase):
    """All hooks must be no-ops when no harness-tasks.json exists."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_stop_no_root(self):
        code, stdout, _ = run_hook(STOP_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_sessionstart_no_root(self):
        code, stdout, _ = run_hook(SESSION_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_teammateidle_no_root(self):
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stderr, "")

    def test_subagentstop_no_root(self):
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")


# ---------------------------------------------------------------------------
# Stop Hook — Task State Logic
# ---------------------------------------------------------------------------
class TestStopHookTaskLogic(unittest.TestCase):
    """Stop hook task selection, completion detection, and safety valve."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)
        (self.root / "harness-progress.txt").write_text("")
        activate(self.root)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def _payload(self, **extra):
        return {"cwd": self.tmpdir, **extra}

    def test_all_completed_allows_stop(self):
        """When all tasks are completed, stop is allowed and .harness-reflect created."""
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "completed"},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")
        self.assertFalse((self.root / ".harness-active").exists())
        self.assertTrue(
            (self.root / ".harness-reflect").exists(),
            ".harness-reflect should be created when all tasks complete",
        )

    def test_pending_with_unmet_deps_allows_stop(self):
        """Pending tasks with unmet dependencies don't block stop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": 3, "max_attempts": 3},
            {"id": "t2", "status": "pending", "depends_on": ["t1"]},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_retryable_failed_blocks(self):
        """Failed task with attempts < max_attempts blocks stop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": 1, "max_attempts": 3, "priority": "P0", "depends_on": [], "title": "Retry me"},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        self.assertIn("Retry me", data["reason"])

    def test_exhausted_retries_allows_stop(self):
        """Failed task with attempts >= max_attempts allows stop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": 3, "max_attempts": 3, "depends_on": []},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_in_progress_blocks(self):
        """In-progress tasks block stop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "priority": "P0"},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")

    def test_session_limit_allows_stop(self):
        """Session limit reached allows stop even with pending tasks."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "priority": "P0"},
        ], session_count=5, session_config={"max_sessions": 5})
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_max_tasks_per_session_limit_allows_stop(self):
        """Per-session completed-task cap allows stop when reached."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "priority": "P0"},
        ], session_count=2, session_config={"max_tasks_per_session": 1})
        (self.root / "harness-progress.txt").write_text("[SESSION-2] Completed [task-1]\n")
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_concurrent_other_worker_in_progress_allows_stop(self):
        """Concurrent mode should not block on another worker's in-progress task."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "worker-a", "priority": "P0"},
        ], session_config={"concurrency_mode": "concurrent"})
        code, stdout, _ = run_hook(
            STOP_HOOK, self._payload(),
            env_extra={"HARNESS_WORKER_ID": "worker-b"},
        )
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_priority_ordering_in_block_reason(self):
        """Block reason shows highest priority task as next."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "priority": "P2", "depends_on": [], "title": "Low"},
            {"id": "t2", "status": "pending", "priority": "P0", "depends_on": [], "title": "High"},
        ])
        code, stdout, _ = run_hook(STOP_HOOK, self._payload())
        data = json.loads(stdout)
        self.assertIn("t2", data["reason"])
        self.assertIn("High", data["reason"])

    def test_stop_hook_active_safety_valve(self):
        """After MAX_CONSECUTIVE_BLOCKS with stop_hook_active, allows stop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "priority": "P0"},
        ])
        (self.root / ".harness-stop-counter").write_text("9,0")
        code, stdout, stderr = run_hook(STOP_HOOK, self._payload(stop_hook_active=True))
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")
        self.assertIn("WARN", stderr)

    def test_stop_hook_active_below_threshold_blocks(self):
        """Below MAX_CONSECUTIVE_BLOCKS with stop_hook_active still blocks."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "priority": "P0"},
        ])
        (self.root / ".harness-stop-counter").write_text("2,0")
        code, stdout, _ = run_hook(STOP_HOOK, self._payload(stop_hook_active=True))
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")

    def test_progress_resets_block_counter(self):
        """When completed count increases, block counter resets."""
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "pending", "depends_on": [], "priority": "P0"},
        ])
        (self.root / ".harness-stop-counter").write_text("7,0")
        code, stdout, _ = run_hook(STOP_HOOK, self._payload(stop_hook_active=True))
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        counter = (self.root / ".harness-stop-counter").read_text().strip()
        self.assertEqual(counter, "1,1")

    def test_corrupt_json_with_stop_hook_active_allows(self):
        """Corrupt config + stop_hook_active should allow stop to avoid loop."""
        (self.root / "harness-tasks.json").write_text("{invalid json")
        code, stdout, stderr = run_hook(STOP_HOOK, self._payload(stop_hook_active=True))
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")
        self.assertIn("WARN", stderr)


# ---------------------------------------------------------------------------
# SessionStart Hook — Context Injection
# ---------------------------------------------------------------------------
class TestSessionStartHook(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)
        activate(self.root)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def _payload(self):
        return {"cwd": self.tmpdir}

    def test_summary_includes_counts(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "pending", "depends_on": ["t1"]},
            {"id": "t3", "status": "failed", "depends_on": []},
        ])
        (self.root / "harness-progress.txt").write_text("[SESSION-1] STATS total=3\n")
        code, stdout, _ = run_hook(SESSION_HOOK, self._payload())
        data = json.loads(stdout)
        ctx = data["hookSpecificOutput"]["additionalContext"]
        self.assertIn("completed=1", ctx)
        self.assertIn("pending=1", ctx)
        self.assertIn("failed=1", ctx)
        self.assertIn("total=3", ctx)

    def test_next_task_hint(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "pending", "priority": "P0", "depends_on": ["t1"], "title": "Do stuff"},
        ])
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(SESSION_HOOK, self._payload())
        data = json.loads(stdout)
        ctx = data["hookSpecificOutput"]["additionalContext"]
        self.assertIn("next=t2", ctx)
        self.assertIn("Do stuff", ctx)

    def test_empty_tasks_no_crash(self):
        write_tasks(self.root, [])
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(SESSION_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertIn("total=0", data["hookSpecificOutput"]["additionalContext"])

    def test_corrupt_json_reports_error(self):
        (self.root / "harness-tasks.json").write_text("{invalid json")
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(SESSION_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertIn("error", data["hookSpecificOutput"]["additionalContext"].lower())

    def test_invalid_attempt_fields_no_crash(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": "oops", "max_attempts": "bad", "depends_on": []},
        ])
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(SESSION_HOOK, self._payload())
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertIn("total=1", data["hookSpecificOutput"]["additionalContext"])


# ---------------------------------------------------------------------------
# TeammateIdle Hook — Ownership & Task State
# ---------------------------------------------------------------------------
class TestTeammateIdleHook(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)
        activate(self.root)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_owned_in_progress_blocks(self):
        """Teammate with in-progress task is blocked from going idle."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "alice", "title": "My task"},
        ])
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir, "teammate_name": "alice"})
        self.assertEqual(code, 2)
        self.assertIn("t1", stderr)

    def test_unowned_in_progress_allows(self):
        """Teammate without owned tasks and no pending allows idle."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "bob"},
        ])
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir, "teammate_name": "alice"})
        self.assertEqual(code, 0)

    def test_pending_tasks_block(self):
        """Pending eligible tasks block idle even without ownership."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "title": "Next up"},
        ])
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 2)
        self.assertIn("t1", stderr)

    def test_all_completed_allows(self):
        """All tasks completed allows idle."""
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "completed"},
        ])
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stderr, "")

    def test_failed_retryable_blocks(self):
        """Retryable failed tasks block idle."""
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": 1, "max_attempts": 3, "depends_on": [], "title": "Retry"},
        ])
        code, _, stderr = run_hook(IDLE_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 2)
        self.assertIn("t1", stderr)

    def test_worker_id_env_matches(self):
        """HARNESS_WORKER_ID env var matches claimed_by."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "w-123"},
        ])
        code, _, stderr = run_hook(
            IDLE_HOOK, {"cwd": self.tmpdir},
            env_extra={"HARNESS_WORKER_ID": "w-123"},
        )
        self.assertEqual(code, 2)
        self.assertIn("t1", stderr)


# ---------------------------------------------------------------------------
# SubagentStop Hook — Stop Guard & stop_hook_active
# ---------------------------------------------------------------------------
class TestSubagentStopHook(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)
        activate(self.root)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_in_progress_blocks(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "title": "Working"},
        ])
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        self.assertIn("Working", data["reason"])

    def test_pending_allows(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "pending", "depends_on": ["t1"], "title": "Next"},
        ])
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_all_done_allows(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
            {"id": "t2", "status": "completed"},
        ])
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_stop_hook_active_allows(self):
        """stop_hook_active=True bypasses all checks to prevent infinite loop."""
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress"},
        ])
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir, "stop_hook_active": True})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_blocked_deps_not_counted(self):
        """Pending tasks with unmet deps don't trigger block."""
        write_tasks(self.root, [
            {"id": "t1", "status": "failed", "attempts": 3, "max_attempts": 3},
            {"id": "t2", "status": "pending", "depends_on": ["t1"]},
        ])
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_concurrent_owned_in_progress_blocks(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "worker-a", "title": "Mine"},
        ], session_config={"concurrency_mode": "concurrent"})
        code, stdout, _ = run_hook(
            SUBAGENT_HOOK, {"cwd": self.tmpdir},
            env_extra={"HARNESS_WORKER_ID": "worker-a"},
        )
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        self.assertIn("Mine", data["reason"])

    def test_concurrent_other_worker_in_progress_allows(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "worker-a", "title": "Other"},
        ], session_config={"concurrency_mode": "concurrent"})
        code, stdout, _ = run_hook(
            SUBAGENT_HOOK, {"cwd": self.tmpdir},
            env_extra={"HARNESS_WORKER_ID": "worker-b"},
        )
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_concurrent_missing_identity_blocks(self):
        write_tasks(self.root, [
            {"id": "t1", "status": "in_progress", "claimed_by": "worker-a", "title": "Other"},
        ], session_config={"concurrency_mode": "concurrent"})
        code, stdout, _ = run_hook(SUBAGENT_HOOK, {"cwd": self.tmpdir})
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        self.assertIn("worker identity", data["reason"])


# ---------------------------------------------------------------------------
# Edge Cases
# ---------------------------------------------------------------------------
class TestEdgeCases(unittest.TestCase):

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)

    def test_empty_stdin(self):
        """Hooks handle empty stdin gracefully."""
        write_tasks(self.root, [{"id": "t1", "status": "pending", "depends_on": []}])
        activate(self.root)
        for hook in [STOP_HOOK, SESSION_HOOK, IDLE_HOOK, SUBAGENT_HOOK]:
            proc = subprocess.run(
                [sys.executable, str(hook)],
                input="",
                capture_output=True, text=True, timeout=10,
                cwd=self.tmpdir,
                env=build_hook_env(),
            )
            self.assertIn(proc.returncode, {0, 2}, f"{hook.name} failed on empty stdin")
            self.assertNotIn("Traceback", proc.stderr)

    def test_invalid_json_stdin(self):
        """Hooks handle invalid JSON stdin gracefully."""
        write_tasks(self.root, [{"id": "t1", "status": "pending", "depends_on": []}])
        activate(self.root)
        for hook in [STOP_HOOK, SESSION_HOOK, IDLE_HOOK, SUBAGENT_HOOK]:
            proc = subprocess.run(
                [sys.executable, str(hook)],
                input="not json at all",
                capture_output=True, text=True, timeout=10,
                cwd=self.tmpdir,
                env=build_hook_env(),
            )
            self.assertIn(proc.returncode, {0, 2}, f"{hook.name} crashed on invalid JSON")
            self.assertNotIn("Traceback", proc.stderr)

    def test_harness_state_root_env(self):
        """HARNESS_STATE_ROOT env var is respected."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": [], "priority": "P0"},
        ])
        activate(self.root)
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(
            STOP_HOOK, {"cwd": "/nonexistent"},
            env_extra={"HARNESS_STATE_ROOT": self.tmpdir},
        )
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")

    def test_tasks_not_a_list(self):
        """Hooks handle tasks field being non-list."""
        (self.root / "harness-tasks.json").write_text('{"tasks": "not a list"}')
        activate(self.root)
        (self.root / "harness-progress.txt").write_text("")
        code, stdout, _ = run_hook(STOP_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")


# ---------------------------------------------------------------------------
# Self-Reflect Stop Hook — Only triggers after harness completes
# ---------------------------------------------------------------------------
REFLECT_HOOK = HOOKS_DIR / "self-reflect-stop.py"


class TestSelfReflectStopHook(unittest.TestCase):
    """self-reflect-stop.py must only trigger when .harness-reflect marker exists."""

    def setUp(self):
        self.tmpdir = tempfile.mkdtemp()
        self.root = Path(self.tmpdir)

    def tearDown(self):
        import shutil
        shutil.rmtree(self.tmpdir, ignore_errors=True)
        # Clean up counter files
        for p in Path(tempfile.gettempdir()).glob("claude-reflect-test-*"):
            try:
                p.unlink()
            except Exception:
                pass

    def _payload(self, session_id="test-reflect-001", **extra):
        return {"cwd": self.tmpdir, "session_id": session_id, **extra}

    def _set_reflect(self):
        """Create .harness-reflect marker (simulates harness completion)."""
        (self.root / ".harness-reflect").touch()

    def test_no_harness_root_is_noop(self):
        """When harness-tasks.json doesn't exist, hook is a complete no-op."""
        code, stdout, stderr = run_hook(REFLECT_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "", "Should produce no output when harness never used")

    def test_harness_active_no_reflect_marker(self):
        """When .harness-active exists but no .harness-reflect, hook is no-op."""
        write_tasks(self.root, [
            {"id": "t1", "status": "pending", "depends_on": []},
        ])
        activate(self.root)
        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload())
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "", "Should not self-reflect while harness is active")

    def test_stale_tasks_without_reflect_marker_is_noop(self):
        """Stale harness-tasks.json without .harness-reflect does NOT trigger (fixes false positive)."""
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
        ])
        deactivate(self.root)
        # No .harness-reflect marker — this is a stale file from a previous run
        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload(session_id="test-stale"))
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "", "Stale harness-tasks.json should NOT trigger self-reflect")

    def test_harness_completed_triggers_reflection(self):
        """When .harness-reflect marker exists, triggers self-reflection."""
        write_tasks(self.root, [
            {"id": "t1", "status": "completed"},
        ])
        deactivate(self.root)
        self._set_reflect()
        sid = "test-reflect-trigger"
        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload(session_id=sid))
        self.assertEqual(code, 0)
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")
        self.assertIn("Self-Reflect", data["reason"])

    def test_counter_increments(self):
        """Each invocation increments the iteration counter."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        deactivate(self.root)
        self._set_reflect()
        sid = "test-reflect-counter"

        # First call: iteration 1
        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload(session_id=sid))
        data = json.loads(stdout)
        self.assertIn("1/5", data["reason"])

        # Second call: iteration 2
        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload(session_id=sid))
        data = json.loads(stdout)
        self.assertIn("2/5", data["reason"])

    def test_max_iterations_allows_stop_and_cleans_marker(self):
        """After max iterations, hook allows stop and removes .harness-reflect."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        deactivate(self.root)
        self._set_reflect()
        sid = "test-reflect-max"

        # Write counter at max
        counter_path = Path(tempfile.gettempdir()) / f"claude-reflect-{sid}"
        counter_path.write_text("5", encoding="utf-8")

        code, stdout, _ = run_hook(REFLECT_HOOK, self._payload(session_id=sid))
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "", "Should allow stop after max iterations")
        self.assertFalse(
            (self.root / ".harness-reflect").exists(),
            ".harness-reflect should be cleaned up after max iterations",
        )

    def test_disabled_via_env(self):
        """REFLECT_MAX_ITERATIONS=0 disables self-reflection."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        deactivate(self.root)
        self._set_reflect()
        code, stdout, _ = run_hook(
            REFLECT_HOOK,
            self._payload(session_id="test-reflect-disabled"),
            env_extra={"REFLECT_MAX_ITERATIONS": "0"},
        )
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "", "Should be disabled when max=0")

    def test_no_session_id_is_noop(self):
        """Missing session_id makes hook a no-op."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        deactivate(self.root)
        self._set_reflect()
        code, stdout, _ = run_hook(REFLECT_HOOK, {"cwd": self.tmpdir})
        self.assertEqual(code, 0)
        self.assertEqual(stdout, "")

    def test_empty_stdin_no_crash(self):
        """Empty stdin doesn't crash."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        self._set_reflect()
        proc = subprocess.run(
            [sys.executable, str(REFLECT_HOOK)],
            input="",
            capture_output=True, text=True, timeout=10,
            cwd=self.tmpdir,
            env=build_hook_env(),
        )
        self.assertEqual(proc.returncode, 0)
        self.assertNotIn("Traceback", proc.stderr)

    def test_harness_state_root_env_respected(self):
        """HARNESS_STATE_ROOT env var is used for root discovery."""
        write_tasks(self.root, [{"id": "t1", "status": "completed"}])
        deactivate(self.root)
        self._set_reflect()
        sid = "test-reflect-env"
        code, stdout, _ = run_hook(
            REFLECT_HOOK,
            {"cwd": "/nonexistent", "session_id": sid},
            env_extra={"HARNESS_STATE_ROOT": self.tmpdir},
        )
        data = json.loads(stdout)
        self.assertEqual(data["decision"], "block")


if __name__ == "__main__":
    unittest.main()
