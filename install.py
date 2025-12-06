#!/usr/bin/env python3
"""JSON-driven modular installer.

Keep it simple: validate config, expand paths, run three operation types,
and record what happened. Designed to be small, readable, and predictable.
"""

from __future__ import annotations

import argparse
import json
import os
import shutil
import subprocess
import sys
from datetime import datetime
from pathlib import Path
from typing import Any, Dict, Iterable, List, Optional

import jsonschema

DEFAULT_INSTALL_DIR = "~/.claude"


def _ensure_list(ctx: Dict[str, Any], key: str) -> List[Any]:
    ctx.setdefault(key, [])
    return ctx[key]


def parse_args(argv: Optional[Iterable[str]] = None) -> argparse.Namespace:
    """Parse CLI arguments.

    The default install dir must remain "~/.claude" to match docs/tests.
    """

    parser = argparse.ArgumentParser(
        description="JSON-driven modular installation system"
    )
    parser.add_argument(
        "--install-dir",
        default=DEFAULT_INSTALL_DIR,
        help="Installation directory (defaults to ~/.claude)",
    )
    parser.add_argument(
        "--module",
        help="Comma-separated modules to install, or 'all' for all enabled",
    )
    parser.add_argument(
        "--config",
        default="config.json",
        help="Path to configuration file",
    )
    parser.add_argument(
        "--list-modules",
        action="store_true",
        help="List available modules and exit",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Force overwrite existing files",
    )
    return parser.parse_args(argv)


def _load_json(path: Path) -> Any:
    try:
        with path.open("r", encoding="utf-8") as fh:
            return json.load(fh)
    except FileNotFoundError as exc:
        raise FileNotFoundError(f"File not found: {path}") from exc
    except json.JSONDecodeError as exc:
        raise ValueError(f"Invalid JSON in {path}: {exc}") from exc


def load_config(path: str) -> Dict[str, Any]:
    """Load config and validate against JSON Schema.

    Schema is searched in the config directory first, then alongside this file.
    """

    config_path = Path(path).expanduser().resolve()
    config = _load_json(config_path)

    schema_candidates = [
        config_path.parent / "config.schema.json",
        Path(__file__).resolve().with_name("config.schema.json"),
    ]
    schema_path = next((p for p in schema_candidates if p.exists()), None)
    if schema_path is None:
        raise FileNotFoundError("config.schema.json not found")

    schema = _load_json(schema_path)
    try:
        jsonschema.validate(config, schema)
    except jsonschema.ValidationError as exc:
        raise ValueError(f"Config validation failed: {exc.message}") from exc

    return config


def resolve_paths(config: Dict[str, Any], args: argparse.Namespace) -> Dict[str, Any]:
    """Resolve all filesystem paths to absolute Path objects."""

    config_dir = Path(args.config).expanduser().resolve().parent

    if args.install_dir and args.install_dir != DEFAULT_INSTALL_DIR:
        install_dir_raw = args.install_dir
    elif config.get("install_dir"):
        install_dir_raw = config.get("install_dir")
    else:
        install_dir_raw = DEFAULT_INSTALL_DIR

    install_dir = Path(install_dir_raw).expanduser().resolve()

    log_file_raw = config.get("log_file", "install.log")
    log_file = Path(log_file_raw).expanduser()
    if not log_file.is_absolute():
        log_file = install_dir / log_file

    return {
        "install_dir": install_dir,
        "log_file": log_file,
        "status_file": install_dir / "installed_modules.json",
        "config_dir": config_dir,
        "force": bool(getattr(args, "force", False)),
        "applied_paths": [],
        "status_backup": None,
    }


def list_modules(config: Dict[str, Any]) -> None:
    print("Available Modules:")
    print(f"{'Name':<15} {'Enabled':<8} Description")
    print("-" * 60)
    for name, cfg in config.get("modules", {}).items():
        enabled = "✓" if cfg.get("enabled", False) else "✗"
        desc = cfg.get("description", "")
        print(f"{name:<15} {enabled:<8} {desc}")


def select_modules(config: Dict[str, Any], module_arg: Optional[str]) -> Dict[str, Any]:
    modules = config.get("modules", {})
    if not module_arg:
        return {k: v for k, v in modules.items() if v.get("enabled", False)}

    if module_arg.strip().lower() == "all":
        return {k: v for k, v in modules.items() if v.get("enabled", False)}

    selected: Dict[str, Any] = {}
    for name in (part.strip() for part in module_arg.split(",")):
        if not name:
            continue
        if name not in modules:
            raise ValueError(f"Module '{name}' not found")
        selected[name] = modules[name]
    return selected


def ensure_install_dir(path: Path) -> None:
    path = Path(path)
    if path.exists() and not path.is_dir():
        raise NotADirectoryError(f"Install path exists and is not a directory: {path}")
    path.mkdir(parents=True, exist_ok=True)
    if not os.access(path, os.W_OK):
        raise PermissionError(f"No write permission for install dir: {path}")


def execute_module(name: str, cfg: Dict[str, Any], ctx: Dict[str, Any]) -> Dict[str, Any]:
    result: Dict[str, Any] = {
        "module": name,
        "status": "success",
        "operations": [],
        "installed_at": datetime.now().isoformat(),
    }

    for op in cfg.get("operations", []):
        op_type = op.get("type")
        try:
            if op_type == "copy_dir":
                op_copy_dir(op, ctx)
            elif op_type == "copy_file":
                op_copy_file(op, ctx)
            elif op_type == "merge_dir":
                op_merge_dir(op, ctx)
            elif op_type == "run_command":
                op_run_command(op, ctx)
            else:
                raise ValueError(f"Unknown operation type: {op_type}")

            result["operations"].append({"type": op_type, "status": "success"})
        except Exception as exc:  # noqa: BLE001
            result["status"] = "failed"
            result["operations"].append(
                {"type": op_type, "status": "failed", "error": str(exc)}
            )
            write_log(
                {
                    "level": "ERROR",
                    "message": f"Module {name} failed on {op_type}: {exc}",
                },
                ctx,
            )
            raise

    return result


def _source_path(op: Dict[str, Any], ctx: Dict[str, Any]) -> Path:
    return (ctx["config_dir"] / op["source"]).expanduser().resolve()


def _target_path(op: Dict[str, Any], ctx: Dict[str, Any]) -> Path:
    return (ctx["install_dir"] / op["target"]).expanduser().resolve()


def _record_created(path: Path, ctx: Dict[str, Any]) -> None:
    install_dir = Path(ctx["install_dir"]).resolve()
    resolved = Path(path).resolve()
    if resolved == install_dir or install_dir not in resolved.parents:
        return
    applied = _ensure_list(ctx, "applied_paths")
    if resolved not in applied:
        applied.append(resolved)


def op_copy_dir(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    src = _source_path(op, ctx)
    dst = _target_path(op, ctx)

    existed_before = dst.exists()
    if existed_before and not ctx.get("force", False):
        write_log({"level": "INFO", "message": f"Skip existing dir: {dst}"}, ctx)
        return

    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(src, dst, dirs_exist_ok=True)
    if not existed_before:
        _record_created(dst, ctx)
    write_log({"level": "INFO", "message": f"Copied dir {src} -> {dst}"}, ctx)


def op_merge_dir(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    """Merge source dir's subdirs (commands/, agents/, etc.) into install_dir."""
    src = _source_path(op, ctx)
    install_dir = ctx["install_dir"]
    force = ctx.get("force", False)
    merged = []

    for subdir in src.iterdir():
        if not subdir.is_dir():
            continue
        target_subdir = install_dir / subdir.name
        target_subdir.mkdir(parents=True, exist_ok=True)
        for f in subdir.iterdir():
            if f.is_file():
                dst = target_subdir / f.name
                if dst.exists() and not force:
                    continue
                shutil.copy2(f, dst)
                merged.append(f"{subdir.name}/{f.name}")

    write_log({"level": "INFO", "message": f"Merged {src.name}: {', '.join(merged) or 'no files'}"}, ctx)


def op_copy_file(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    src = _source_path(op, ctx)
    dst = _target_path(op, ctx)

    existed_before = dst.exists()
    if existed_before and not ctx.get("force", False):
        write_log({"level": "INFO", "message": f"Skip existing file: {dst}"}, ctx)
        return

    dst.parent.mkdir(parents=True, exist_ok=True)
    shutil.copy2(src, dst)
    if not existed_before:
        _record_created(dst, ctx)
    write_log({"level": "INFO", "message": f"Copied file {src} -> {dst}"}, ctx)


def op_run_command(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    env = os.environ.copy()
    for key, value in op.get("env", {}).items():
        env[key] = value.replace("${install_dir}", str(ctx["install_dir"]))

    command = op.get("command", "")
    if sys.platform == "win32" and command.strip() == "bash install.sh":
        command = "cmd /c install.bat"
    result = subprocess.run(
        command,
        shell=True,
        cwd=ctx["config_dir"],
        env=env,
        capture_output=True,
        text=True,
    )

    write_log(
        {
            "level": "INFO",
            "message": f"Command: {command}",
            "stdout": result.stdout,
            "stderr": result.stderr,
            "returncode": result.returncode,
        },
        ctx,
    )

    if result.returncode != 0:
        raise RuntimeError(f"Command failed with code {result.returncode}: {command}")


def write_log(entry: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    log_path = Path(ctx["log_file"])
    log_path.parent.mkdir(parents=True, exist_ok=True)

    ts = datetime.now().isoformat()
    level = entry.get("level", "INFO")
    message = entry.get("message", "")

    with log_path.open("a", encoding="utf-8") as fh:
        fh.write(f"[{ts}] {level}: {message}\n")
        for key in ("stdout", "stderr", "returncode"):
            if key in entry and entry[key] not in (None, ""):
                fh.write(f"  {key}: {entry[key]}\n")


def write_status(results: List[Dict[str, Any]], ctx: Dict[str, Any]) -> None:
    status = {
        "installed_at": datetime.now().isoformat(),
        "modules": {item["module"]: item for item in results},
    }

    status_path = Path(ctx["status_file"])
    status_path.parent.mkdir(parents=True, exist_ok=True)
    with status_path.open("w", encoding="utf-8") as fh:
        json.dump(status, fh, indent=2, ensure_ascii=False)


def prepare_status_backup(ctx: Dict[str, Any]) -> None:
    status_path = Path(ctx["status_file"])
    if status_path.exists():
        backup = status_path.with_suffix(".json.bak")
        backup.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(status_path, backup)
        ctx["status_backup"] = backup


def rollback(ctx: Dict[str, Any]) -> None:
    write_log({"level": "WARNING", "message": "Rolling back installation"}, ctx)

    install_dir = Path(ctx["install_dir"]).resolve()
    for path in reversed(ctx.get("applied_paths", [])):
        resolved = Path(path).resolve()
        try:
            if resolved == install_dir or install_dir not in resolved.parents:
                continue
            if resolved.is_dir():
                shutil.rmtree(resolved, ignore_errors=True)
            else:
                resolved.unlink(missing_ok=True)
        except Exception as exc:  # noqa: BLE001
            write_log(
                {
                    "level": "ERROR",
                    "message": f"Rollback skipped {resolved}: {exc}",
                },
                ctx,
            )

    backup = ctx.get("status_backup")
    if backup and Path(backup).exists():
        shutil.copy2(backup, ctx["status_file"])

    write_log({"level": "INFO", "message": "Rollback completed"}, ctx)


def main(argv: Optional[Iterable[str]] = None) -> int:
    args = parse_args(argv)
    try:
        config = load_config(args.config)
    except Exception as exc:  # noqa: BLE001
        print(f"Error loading config: {exc}", file=sys.stderr)
        return 1

    ctx = resolve_paths(config, args)

    if getattr(args, "list_modules", False):
        list_modules(config)
        return 0

    modules = select_modules(config, args.module)

    try:
        ensure_install_dir(ctx["install_dir"])
    except Exception as exc:  # noqa: BLE001
        print(f"Failed to prepare install dir: {exc}", file=sys.stderr)
        return 1

    prepare_status_backup(ctx)

    results: List[Dict[str, Any]] = []
    for name, cfg in modules.items():
        try:
            results.append(execute_module(name, cfg, ctx))
        except Exception:  # noqa: BLE001
            if not args.force:
                rollback(ctx)
                return 1
            rollback(ctx)
            results.append(
                {
                    "module": name,
                    "status": "failed",
                    "operations": [],
                    "installed_at": datetime.now().isoformat(),
                }
            )
            break

    write_status(results, ctx)
    return 0


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
