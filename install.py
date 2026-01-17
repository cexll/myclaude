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

try:
    import jsonschema
except ImportError:  # pragma: no cover
    jsonschema = None

DEFAULT_INSTALL_DIR = "~/.claude"
SETTINGS_FILE = "settings.json"


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
        help="Comma-separated modules to install/uninstall, or 'all'",
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
        "--status",
        action="store_true",
        help="Show installation status of all modules",
    )
    parser.add_argument(
        "--uninstall",
        action="store_true",
        help="Uninstall specified modules",
    )
    parser.add_argument(
        "--force",
        action="store_true",
        help="Force overwrite existing files",
    )
    parser.add_argument(
        "--verbose", "-v",
        action="store_true",
        help="Enable verbose output to terminal",
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


def _save_json(path: Path, data: Any) -> None:
    """Save data to JSON file with proper formatting."""
    path.parent.mkdir(parents=True, exist_ok=True)
    with path.open("w", encoding="utf-8") as fh:
        json.dump(data, fh, indent=2, ensure_ascii=False)
        fh.write("\n")


# =============================================================================
# Hooks Management
# =============================================================================

def load_settings(ctx: Dict[str, Any]) -> Dict[str, Any]:
    """Load settings.json from install directory."""
    settings_path = ctx["install_dir"] / SETTINGS_FILE
    if settings_path.exists():
        try:
            return _load_json(settings_path)
        except (ValueError, FileNotFoundError):
            return {}
    return {}


def save_settings(ctx: Dict[str, Any], settings: Dict[str, Any]) -> None:
    """Save settings.json to install directory."""
    settings_path = ctx["install_dir"] / SETTINGS_FILE
    _save_json(settings_path, settings)


def find_module_hooks(module_name: str, cfg: Dict[str, Any], ctx: Dict[str, Any]) -> Optional[Dict[str, Any]]:
    """Find hooks.json for a module if it exists."""
    # Check for hooks in operations (copy_dir targets)
    for op in cfg.get("operations", []):
        if op.get("type") == "copy_dir":
            target_dir = ctx["install_dir"] / op["target"]
            hooks_file = target_dir / "hooks" / "hooks.json"
            if hooks_file.exists():
                try:
                    return _load_json(hooks_file)
                except (ValueError, FileNotFoundError):
                    pass

    # Also check source directory during install
    for op in cfg.get("operations", []):
        if op.get("type") == "copy_dir":
            source_dir = ctx["config_dir"] / op["source"]
            hooks_file = source_dir / "hooks" / "hooks.json"
            if hooks_file.exists():
                try:
                    return _load_json(hooks_file)
                except (ValueError, FileNotFoundError):
                    pass

    return None


def _create_hook_marker(module_name: str) -> str:
    """Create a marker to identify hooks from a specific module."""
    return f"__module:{module_name}__"


def merge_hooks_to_settings(module_name: str, hooks_config: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    """Merge module hooks into settings.json."""
    settings = load_settings(ctx)
    settings.setdefault("hooks", {})

    module_hooks = hooks_config.get("hooks", {})
    marker = _create_hook_marker(module_name)

    for hook_type, hook_entries in module_hooks.items():
        settings["hooks"].setdefault(hook_type, [])

        for entry in hook_entries:
            # Add marker to identify this hook's source module
            entry_copy = dict(entry)
            entry_copy["__module__"] = module_name

            # Check if already exists (avoid duplicates)
            exists = False
            for existing in settings["hooks"][hook_type]:
                if existing.get("__module__") == module_name:
                    # Same module, check if same hook
                    if _hooks_equal(existing, entry_copy):
                        exists = True
                        break

            if not exists:
                settings["hooks"][hook_type].append(entry_copy)

    save_settings(ctx, settings)
    write_log({"level": "INFO", "message": f"Merged hooks for module: {module_name}"}, ctx)


def unmerge_hooks_from_settings(module_name: str, ctx: Dict[str, Any]) -> None:
    """Remove module hooks from settings.json."""
    settings = load_settings(ctx)

    if "hooks" not in settings:
        return

    modified = False
    for hook_type in list(settings["hooks"].keys()):
        original_len = len(settings["hooks"][hook_type])
        settings["hooks"][hook_type] = [
            entry for entry in settings["hooks"][hook_type]
            if entry.get("__module__") != module_name
        ]
        if len(settings["hooks"][hook_type]) < original_len:
            modified = True

        # Remove empty hook type arrays
        if not settings["hooks"][hook_type]:
            del settings["hooks"][hook_type]

    if modified:
        save_settings(ctx, settings)
        write_log({"level": "INFO", "message": f"Removed hooks for module: {module_name}"}, ctx)


def _hooks_equal(hook1: Dict[str, Any], hook2: Dict[str, Any]) -> bool:
    """Compare two hooks ignoring the __module__ marker."""
    h1 = {k: v for k, v in hook1.items() if k != "__module__"}
    h2 = {k: v for k, v in hook2.items() if k != "__module__"}
    return h1 == h2


def load_config(path: str) -> Dict[str, Any]:
    """Load config and validate against JSON Schema.

    Schema is searched in the config directory first, then alongside this file.
    """

    config_path = Path(path).expanduser().resolve()
    config = _load_json(config_path)

    if jsonschema is None:
        print(
            "WARNING: python package 'jsonschema' is not installed; "
            "skipping config validation. To enable validation run:\n"
            "  python3 -m pip install jsonschema\n",
            file=sys.stderr,
        )

        if not isinstance(config, dict):
            raise ValueError(
                f"Config must be a dict, got {type(config).__name__}. "
                "Check your config.json syntax."
            )

        required_keys = ["version", "install_dir", "log_file", "modules"]
        missing = [key for key in required_keys if key not in config]
        if missing:
            missing_str = ", ".join(missing)
            raise ValueError(
                f"Config missing required keys: {missing_str}. "
                "Install jsonschema for better validation: "
                "python3 -m pip install jsonschema"
            )

        return config

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
        "verbose": bool(getattr(args, "verbose", False)),
        "applied_paths": [],
        "status_backup": None,
    }


def list_modules(config: Dict[str, Any]) -> None:
    print("Available Modules:")
    print(f"{'#':<3} {'Name':<15} {'Default':<8} Description")
    print("-" * 65)
    for idx, (name, cfg) in enumerate(config.get("modules", {}).items(), 1):
        default = "✓" if cfg.get("enabled", False) else "✗"
        desc = cfg.get("description", "")
        print(f"{idx:<3} {name:<15} {default:<8} {desc}")
    print("\n✓ = installed by default when no --module specified")


def load_installed_status(ctx: Dict[str, Any]) -> Dict[str, Any]:
    """Load installed modules status from status file."""
    status_path = Path(ctx["status_file"])
    if status_path.exists():
        try:
            return _load_json(status_path)
        except (ValueError, FileNotFoundError):
            return {"modules": {}}
    return {"modules": {}}


def check_module_installed(name: str, cfg: Dict[str, Any], ctx: Dict[str, Any]) -> bool:
    """Check if a module is installed by verifying its files exist."""
    install_dir = ctx["install_dir"]

    for op in cfg.get("operations", []):
        op_type = op.get("type")
        if op_type in ("copy_dir", "copy_file"):
            target = (install_dir / op["target"]).expanduser().resolve()
            if target.exists():
                return True
    return False


def get_installed_modules(config: Dict[str, Any], ctx: Dict[str, Any]) -> Dict[str, bool]:
    """Get installation status of all modules by checking files."""
    result = {}
    modules = config.get("modules", {})

    # First check status file
    status = load_installed_status(ctx)
    status_modules = status.get("modules", {})

    for name, cfg in modules.items():
        # Check both status file and filesystem
        in_status = name in status_modules
        files_exist = check_module_installed(name, cfg, ctx)
        result[name] = in_status or files_exist

    return result


def list_modules_with_status(config: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    """List modules with installation status."""
    installed_status = get_installed_modules(config, ctx)
    status_data = load_installed_status(ctx)
    status_modules = status_data.get("modules", {})

    print("\n" + "=" * 70)
    print("Module Status")
    print("=" * 70)
    print(f"{'#':<3} {'Name':<15} {'Status':<15} {'Installed At':<20} Description")
    print("-" * 70)

    for idx, (name, cfg) in enumerate(config.get("modules", {}).items(), 1):
        desc = cfg.get("description", "")[:25]
        if installed_status.get(name, False):
            status = "✅ Installed"
            installed_at = status_modules.get(name, {}).get("installed_at", "")[:16]
        else:
            status = "⬚ Not installed"
            installed_at = ""
        print(f"{idx:<3} {name:<15} {status:<15} {installed_at:<20} {desc}")

    total = len(config.get("modules", {}))
    installed_count = sum(1 for v in installed_status.values() if v)
    print(f"\nTotal: {installed_count}/{total} modules installed")
    print(f"Install dir: {ctx['install_dir']}")


def select_modules(config: Dict[str, Any], module_arg: Optional[str]) -> Dict[str, Any]:
    modules = config.get("modules", {})
    if not module_arg:
        # No --module specified: show interactive selection
        return interactive_select_modules(config)

    if module_arg.strip().lower() == "all":
        return dict(modules.items())

    selected: Dict[str, Any] = {}
    for name in (part.strip() for part in module_arg.split(",")):
        if not name:
            continue
        if name not in modules:
            raise ValueError(f"Module '{name}' not found")
        selected[name] = modules[name]
    return selected


def interactive_select_modules(config: Dict[str, Any]) -> Dict[str, Any]:
    """Interactive module selection when no --module is specified."""
    modules = config.get("modules", {})
    module_names = list(modules.keys())

    print("\n" + "=" * 65)
    print("Welcome to Claude Plugin Installer")
    print("=" * 65)
    print("\nNo modules specified. Please select modules to install:\n")

    list_modules(config)

    print("\nEnter module numbers or names (comma-separated), or:")
    print("  'all'  - Install all modules")
    print("  'q'    - Quit without installing")
    print()

    while True:
        try:
            user_input = input("Select modules: ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nInstallation cancelled.")
            sys.exit(0)

        if not user_input:
            print("No input. Please enter module numbers, names, 'all', or 'q'.")
            continue

        if user_input.lower() == "q":
            print("Installation cancelled.")
            sys.exit(0)

        if user_input.lower() == "all":
            print(f"\nSelected all {len(modules)} modules.")
            return dict(modules.items())

        # Parse selection
        selected: Dict[str, Any] = {}
        parts = [p.strip() for p in user_input.replace(" ", ",").split(",") if p.strip()]

        try:
            for part in parts:
                # Try as number first
                if part.isdigit():
                    idx = int(part) - 1
                    if 0 <= idx < len(module_names):
                        name = module_names[idx]
                        selected[name] = modules[name]
                    else:
                        print(f"Invalid number: {part}. Valid range: 1-{len(module_names)}")
                        selected = {}
                        break
                # Try as name
                elif part in modules:
                    selected[part] = modules[part]
                else:
                    print(f"Module not found: '{part}'")
                    selected = {}
                    break

            if selected:
                names = ", ".join(selected.keys())
                print(f"\nSelected {len(selected)} module(s): {names}")
                return selected

        except ValueError:
            print("Invalid input. Please try again.")
            continue


def uninstall_module(name: str, cfg: Dict[str, Any], ctx: Dict[str, Any]) -> Dict[str, Any]:
    """Uninstall a module by removing its files and hooks."""
    result: Dict[str, Any] = {
        "module": name,
        "status": "success",
        "uninstalled_at": datetime.now().isoformat(),
    }

    install_dir = ctx["install_dir"]
    removed_paths = []

    for op in cfg.get("operations", []):
        op_type = op.get("type")
        try:
            if op_type in ("copy_dir", "copy_file"):
                target = (install_dir / op["target"]).expanduser().resolve()
                if target.exists():
                    if target.is_dir():
                        shutil.rmtree(target)
                    else:
                        target.unlink()
                    removed_paths.append(str(target))
                    write_log({"level": "INFO", "message": f"Removed: {target}"}, ctx)
            # merge_dir and merge_json are harder to uninstall cleanly, skip
        except Exception as exc:
            write_log({"level": "WARNING", "message": f"Failed to remove {op.get('target', 'unknown')}: {exc}"}, ctx)

    # Remove module hooks from settings.json
    try:
        unmerge_hooks_from_settings(name, ctx)
        result["hooks_removed"] = True
    except Exception as exc:
        write_log({"level": "WARNING", "message": f"Failed to remove hooks for {name}: {exc}"}, ctx)

    result["removed_paths"] = removed_paths
    return result


def update_status_after_uninstall(uninstalled_modules: List[str], ctx: Dict[str, Any]) -> None:
    """Remove uninstalled modules from status file."""
    status = load_installed_status(ctx)
    modules = status.get("modules", {})

    for name in uninstalled_modules:
        if name in modules:
            del modules[name]

    status["modules"] = modules
    status["updated_at"] = datetime.now().isoformat()

    status_path = Path(ctx["status_file"])
    with status_path.open("w", encoding="utf-8") as fh:
        json.dump(status, fh, indent=2, ensure_ascii=False)


def interactive_manage(config: Dict[str, Any], ctx: Dict[str, Any]) -> int:
    """Interactive module management menu."""
    while True:
        installed_status = get_installed_modules(config, ctx)
        modules = config.get("modules", {})
        module_names = list(modules.keys())

        print("\n" + "=" * 70)
        print("Claude Plugin Manager")
        print("=" * 70)
        print(f"{'#':<3} {'Name':<15} {'Status':<15} Description")
        print("-" * 70)

        for idx, (name, cfg) in enumerate(modules.items(), 1):
            desc = cfg.get("description", "")[:30]
            if installed_status.get(name, False):
                status = "✅ Installed"
            else:
                status = "⬚ Not installed"
            print(f"{idx:<3} {name:<15} {status:<15} {desc}")

        total = len(modules)
        installed_count = sum(1 for v in installed_status.values() if v)
        print(f"\nInstalled: {installed_count}/{total} | Dir: {ctx['install_dir']}")

        print("\nCommands:")
        print("  i <num/name>  - Install module(s)")
        print("  u <num/name>  - Uninstall module(s)")
        print("  q             - Quit")
        print()

        try:
            user_input = input("Enter command: ").strip()
        except (EOFError, KeyboardInterrupt):
            print("\nExiting.")
            return 0

        if not user_input:
            continue

        if user_input.lower() == "q":
            print("Goodbye!")
            return 0

        parts = user_input.split(maxsplit=1)
        cmd = parts[0].lower()
        args = parts[1] if len(parts) > 1 else ""

        if cmd == "i":
            # Install
            selected = _parse_module_selection(args, modules, module_names)
            if selected:
                # Filter out already installed
                to_install = {k: v for k, v in selected.items() if not installed_status.get(k, False)}
                if not to_install:
                    print("All selected modules are already installed.")
                    continue
                print(f"\nInstalling: {', '.join(to_install.keys())}")
                results = []
                for name, cfg in to_install.items():
                    try:
                        results.append(execute_module(name, cfg, ctx))
                        print(f"  ✓ {name} installed")
                    except Exception as exc:
                        print(f"  ✗ {name} failed: {exc}")
                # Update status
                current_status = load_installed_status(ctx)
                for r in results:
                    if r.get("status") == "success":
                        current_status.setdefault("modules", {})[r["module"]] = r
                current_status["updated_at"] = datetime.now().isoformat()
                with Path(ctx["status_file"]).open("w", encoding="utf-8") as fh:
                    json.dump(current_status, fh, indent=2, ensure_ascii=False)

        elif cmd == "u":
            # Uninstall
            selected = _parse_module_selection(args, modules, module_names)
            if selected:
                # Filter to only installed ones
                to_uninstall = {k: v for k, v in selected.items() if installed_status.get(k, False)}
                if not to_uninstall:
                    print("None of the selected modules are installed.")
                    continue
                print(f"\nUninstalling: {', '.join(to_uninstall.keys())}")
                confirm = input("Confirm? (y/N): ").strip().lower()
                if confirm != "y":
                    print("Cancelled.")
                    continue
                for name, cfg in to_uninstall.items():
                    try:
                        uninstall_module(name, cfg, ctx)
                        print(f"  ✓ {name} uninstalled")
                    except Exception as exc:
                        print(f"  ✗ {name} failed: {exc}")
                update_status_after_uninstall(list(to_uninstall.keys()), ctx)

        else:
            print(f"Unknown command: {cmd}. Use 'i', 'u', or 'q'.")


def _parse_module_selection(
    args: str, modules: Dict[str, Any], module_names: List[str]
) -> Dict[str, Any]:
    """Parse module selection from user input."""
    if not args:
        print("Please specify module number(s) or name(s).")
        return {}

    if args.lower() == "all":
        return dict(modules.items())

    selected: Dict[str, Any] = {}
    parts = [p.strip() for p in args.replace(",", " ").split() if p.strip()]

    for part in parts:
        if part.isdigit():
            idx = int(part) - 1
            if 0 <= idx < len(module_names):
                name = module_names[idx]
                selected[name] = modules[name]
            else:
                print(f"Invalid number: {part}")
                return {}
        elif part in modules:
            selected[part] = modules[part]
        else:
            print(f"Module not found: '{part}'")
            return {}

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
            elif op_type == "merge_json":
                op_merge_json(op, ctx)
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

    # Handle hooks: find and merge module hooks into settings.json
    hooks_config = find_module_hooks(name, cfg, ctx)
    if hooks_config:
        try:
            merge_hooks_to_settings(name, hooks_config, ctx)
            result["operations"].append({"type": "merge_hooks", "status": "success"})
            result["has_hooks"] = True
        except Exception as exc:
            write_log({"level": "WARNING", "message": f"Failed to merge hooks for {name}: {exc}"}, ctx)
            result["operations"].append({"type": "merge_hooks", "status": "failed", "error": str(exc)})

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


def op_merge_json(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    """Merge JSON from source into target, supporting nested key paths."""
    src = _source_path(op, ctx)
    dst = _target_path(op, ctx)
    merge_key = op.get("merge_key")

    if not src.exists():
        raise FileNotFoundError(f"Source JSON not found: {src}")

    src_data = _load_json(src)

    dst.parent.mkdir(parents=True, exist_ok=True)
    if dst.exists():
        dst_data = _load_json(dst)
    else:
        dst_data = {}
        _record_created(dst, ctx)

    if merge_key:
        # Merge into specific key
        keys = merge_key.split(".")
        target = dst_data
        for key in keys[:-1]:
            target = target.setdefault(key, {})

        last_key = keys[-1]
        if isinstance(src_data, dict) and isinstance(target.get(last_key), dict):
            # Deep merge for dicts
            target[last_key] = {**target.get(last_key, {}), **src_data}
        else:
            target[last_key] = src_data
    else:
        # Merge at root level
        if isinstance(src_data, dict) and isinstance(dst_data, dict):
            dst_data = {**dst_data, **src_data}
        else:
            dst_data = src_data

    with dst.open("w", encoding="utf-8") as fh:
        json.dump(dst_data, fh, indent=2, ensure_ascii=False)
        fh.write("\n")

    write_log({"level": "INFO", "message": f"Merged JSON {src} -> {dst} (key: {merge_key or 'root'})"}, ctx)


def op_run_command(op: Dict[str, Any], ctx: Dict[str, Any]) -> None:
    env = os.environ.copy()
    for key, value in op.get("env", {}).items():
        env[key] = value.replace("${install_dir}", str(ctx["install_dir"]))

    command = op.get("command", "")
    if sys.platform == "win32" and command.strip() == "bash install.sh":
        command = "cmd /c install.bat"

    # Stream output in real-time while capturing for logging
    process = subprocess.Popen(
        command,
        shell=True,
        cwd=ctx["config_dir"],
        env=env,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
    )

    stdout_lines: List[str] = []
    stderr_lines: List[str] = []

    # Read stdout and stderr in real-time
    if sys.platform == "win32":
        # On Windows, use threads instead of selectors (pipes aren't selectable)
        import threading

        def read_output(pipe, lines, file=None):
            for line in iter(pipe.readline, ''):
                lines.append(line)
                print(line, end="", flush=True, file=file)
            pipe.close()

        stdout_thread = threading.Thread(target=read_output, args=(process.stdout, stdout_lines))
        stderr_thread = threading.Thread(target=read_output, args=(process.stderr, stderr_lines, sys.stderr))

        stdout_thread.start()
        stderr_thread.start()

        stdout_thread.join()
        stderr_thread.join()
        process.wait()
    else:
        # On Unix, use selectors for more efficient I/O
        import selectors
        sel = selectors.DefaultSelector()
        sel.register(process.stdout, selectors.EVENT_READ)  # type: ignore[arg-type]
        sel.register(process.stderr, selectors.EVENT_READ)  # type: ignore[arg-type]

        while process.poll() is None or sel.get_map():
            for key, _ in sel.select(timeout=0.1):
                line = key.fileobj.readline()  # type: ignore[union-attr]
                if not line:
                    sel.unregister(key.fileobj)
                    continue
                if key.fileobj == process.stdout:
                    stdout_lines.append(line)
                    print(line, end="", flush=True)
                else:
                    stderr_lines.append(line)
                    print(line, end="", file=sys.stderr, flush=True)

        sel.close()
        process.wait()

    write_log(
        {
            "level": "INFO",
            "message": f"Command: {command}",
            "stdout": "".join(stdout_lines),
            "stderr": "".join(stderr_lines),
            "returncode": process.returncode,
        },
        ctx,
    )

    if process.returncode != 0:
        raise RuntimeError(f"Command failed with code {process.returncode}: {command}")


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

    # Terminal output when verbose
    if ctx.get("verbose"):
        prefix = {"INFO": "ℹ️ ", "WARNING": "⚠️ ", "ERROR": "❌"}.get(level, "")
        print(f"{prefix}[{level}] {message}")
        if entry.get("stdout"):
            print(f"  stdout: {entry['stdout'][:500]}")
        if entry.get("stderr"):
            print(f"  stderr: {entry['stderr'][:500]}", file=sys.stderr)
        if entry.get("returncode") is not None:
            print(f"  returncode: {entry['returncode']}")


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

    # Handle --list-modules
    if getattr(args, "list_modules", False):
        list_modules(config)
        return 0

    # Handle --status
    if getattr(args, "status", False):
        list_modules_with_status(config, ctx)
        return 0

    # Handle --uninstall
    if getattr(args, "uninstall", False):
        if not args.module:
            print("Error: --uninstall requires --module to specify which modules to uninstall")
            return 1
        modules = config.get("modules", {})
        installed = load_installed_status(ctx)
        installed_modules = installed.get("modules", {})

        selected = select_modules(config, args.module)
        to_uninstall = {k: v for k, v in selected.items() if k in installed_modules}

        if not to_uninstall:
            print("None of the specified modules are installed.")
            return 0

        print(f"Uninstalling {len(to_uninstall)} module(s): {', '.join(to_uninstall.keys())}")
        for name, cfg in to_uninstall.items():
            try:
                uninstall_module(name, cfg, ctx)
                print(f"  ✓ {name} uninstalled")
            except Exception as exc:
                print(f"  ✗ {name} failed: {exc}", file=sys.stderr)

        update_status_after_uninstall(list(to_uninstall.keys()), ctx)
        print(f"\n✓ Uninstall complete")
        return 0

    # No --module specified: enter interactive management mode
    if not args.module:
        try:
            ensure_install_dir(ctx["install_dir"])
        except Exception as exc:
            print(f"Failed to prepare install dir: {exc}", file=sys.stderr)
            return 1
        return interactive_manage(config, ctx)

    # Install specified modules
    modules = select_modules(config, args.module)

    try:
        ensure_install_dir(ctx["install_dir"])
    except Exception as exc:  # noqa: BLE001
        print(f"Failed to prepare install dir: {exc}", file=sys.stderr)
        return 1

    prepare_status_backup(ctx)

    total = len(modules)
    print(f"Installing {total} module(s) to {ctx['install_dir']}...")

    results: List[Dict[str, Any]] = []
    for idx, (name, cfg) in enumerate(modules.items(), 1):
        print(f"[{idx}/{total}] Installing module: {name}...")
        try:
            results.append(execute_module(name, cfg, ctx))
            print(f"  ✓ {name} installed successfully")
        except Exception as exc:  # noqa: BLE001
            print(f"  ✗ {name} failed: {exc}", file=sys.stderr)
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

    # Merge with existing status
    current_status = load_installed_status(ctx)
    for r in results:
        if r.get("status") == "success":
            current_status.setdefault("modules", {})[r["module"]] = r
    current_status["updated_at"] = datetime.now().isoformat()
    with Path(ctx["status_file"]).open("w", encoding="utf-8") as fh:
        json.dump(current_status, fh, indent=2, ensure_ascii=False)

    # Summary
    success = sum(1 for r in results if r.get("status") == "success")
    failed = len(results) - success
    if failed == 0:
        print(f"\n✓ Installation complete: {success} module(s) installed")
        print(f"  Log file: {ctx['log_file']}")
    else:
        print(f"\n⚠ Installation finished with errors: {success} success, {failed} failed")
        print(f"  Check log file for details: {ctx['log_file']}")
        if not args.force:
            return 1

    return 0


if __name__ == "__main__":  # pragma: no cover
    sys.exit(main())
