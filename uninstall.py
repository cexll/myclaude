#!/usr/bin/env python3
"""Uninstaller for myclaude - reads installed_modules.json for precise removal."""

from __future__ import annotations

import argparse
import json
import re
import shutil
import sys
from pathlib import Path
from typing import Any, Dict, List, Optional, Set

DEFAULT_INSTALL_DIR = "~/.claude"
SETTINGS_FILE = "settings.json"

# Files created by installer itself (not by modules)
INSTALLER_FILES = ["install.log", "installed_modules.json", "installed_modules.json.bak"]


def parse_args(argv: Optional[List[str]] = None) -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Uninstall myclaude")
    parser.add_argument(
        "--install-dir",
        default=DEFAULT_INSTALL_DIR,
        help="Installation directory (defaults to ~/.claude)",
    )
    parser.add_argument(
        "--module",
        help="Comma-separated modules to uninstall (default: all installed)",
    )
    parser.add_argument(
        "--list",
        action="store_true",
        help="List installed modules and exit",
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be removed without actually removing",
    )
    parser.add_argument(
        "--purge",
        action="store_true",
        help="Remove entire install directory (DANGEROUS: removes user files too)",
    )
    parser.add_argument(
        "-y", "--yes",
        action="store_true",
        help="Skip confirmation prompt",
    )
    return parser.parse_args(argv)


def load_installed_modules(install_dir: Path) -> Dict[str, Any]:
    """Load installed_modules.json to know what was installed."""
    status_file = install_dir / "installed_modules.json"
    if not status_file.exists():
        return {}
    try:
        with status_file.open("r", encoding="utf-8") as f:
            return json.load(f)
    except (json.JSONDecodeError, OSError):
        return {}


def load_config(install_dir: Path) -> Dict[str, Any]:
    """Try to load config.json from source repo to understand module structure."""
    # Look for config.json in common locations
    candidates = [
        Path(__file__).parent / "config.json",
        install_dir / "config.json",
    ]
    for path in candidates:
        if path.exists():
            try:
                with path.open("r", encoding="utf-8") as f:
                    return json.load(f)
            except (json.JSONDecodeError, OSError):
                continue
    return {}


def unmerge_hooks_from_settings(module_name: str, install_dir: Path) -> bool:
    """Remove hooks tagged with __module__=module_name from settings.json."""
    settings_path = install_dir / SETTINGS_FILE
    if not settings_path.exists():
        return False

    try:
        with settings_path.open("r", encoding="utf-8") as f:
            settings = json.load(f)
    except (json.JSONDecodeError, OSError):
        return False

    if "hooks" not in settings:
        return False

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
        with settings_path.open("w", encoding="utf-8") as f:
            json.dump(settings, f, indent=2, ensure_ascii=False)
            f.write("\n")

    return modified


def get_module_files(module_name: str, config: Dict[str, Any]) -> Set[str]:
    """Extract files/dirs that a module installs based on config.json operations."""
    files: Set[str] = set()
    modules = config.get("modules", {})
    module_cfg = modules.get(module_name, {})

    for op in module_cfg.get("operations", []):
        op_type = op.get("type", "")
        target = op.get("target", "")

        if op_type == "copy_file" and target:
            files.add(target)
        elif op_type == "copy_dir" and target:
            files.add(target)
        elif op_type == "merge_dir":
            # merge_dir merges subdirs like commands/, agents/ into install_dir
            source = op.get("source", "")
            source_path = Path(__file__).parent / source
            if source_path.exists():
                for subdir in source_path.iterdir():
                    if subdir.is_dir():
                        files.add(subdir.name)
        elif op_type == "run_command":
            # install.sh installs bin/codeagent-wrapper
            cmd = op.get("command", "")
            if "install.sh" in cmd or "install.bat" in cmd:
                files.add("bin/codeagent-wrapper")
                files.add("bin")

    return files


def cleanup_shell_config(rc_file: Path, bin_dir: Path) -> bool:
    """Remove PATH export added by installer from shell config."""
    if not rc_file.exists():
        return False

    content = rc_file.read_text(encoding="utf-8")
    original = content

    patterns = [
        r"\n?# Added by myclaude installer\n",
        rf'\nexport PATH="{re.escape(str(bin_dir))}:\$PATH"\n?',
    ]

    for pattern in patterns:
        content = re.sub(pattern, "\n", content)

    content = re.sub(r"\n{3,}$", "\n\n", content)

    if content != original:
        rc_file.write_text(content, encoding="utf-8")
        return True
    return False


def list_installed(install_dir: Path) -> None:
    """List installed modules."""
    status = load_installed_modules(install_dir)
    modules = status.get("modules", {})

    if not modules:
        print("No modules installed (installed_modules.json not found or empty)")
        return

    print(f"Installed modules in {install_dir}:")
    print(f"{'Module':<15} {'Status':<10} {'Installed At'}")
    print("-" * 50)
    for name, info in modules.items():
        st = info.get("status", "unknown")
        ts = info.get("installed_at", "unknown")[:19]
        print(f"{name:<15} {st:<10} {ts}")


def main(argv: Optional[List[str]] = None) -> int:
    args = parse_args(argv)
    install_dir = Path(args.install_dir).expanduser().resolve()
    bin_dir = install_dir / "bin"

    if not install_dir.exists():
        print(f"Install directory not found: {install_dir}")
        print("Nothing to uninstall.")
        return 0

    if args.list:
        list_installed(install_dir)
        return 0

    # Load installation status
    status = load_installed_modules(install_dir)
    installed_modules = status.get("modules", {})
    config = load_config(install_dir)

    # Determine which modules to uninstall
    if args.module:
        selected = [m.strip() for m in args.module.split(",") if m.strip()]
        # Validate
        for m in selected:
            if m not in installed_modules:
                print(f"Error: Module '{m}' is not installed")
                print("Use --list to see installed modules")
                return 1
    else:
        selected = list(installed_modules.keys())

    if not selected and not args.purge:
        print("No modules to uninstall.")
        print("Use --list to see installed modules, or --purge to remove everything.")
        return 0

    # Collect files to remove
    files_to_remove: Set[str] = set()
    for module_name in selected:
        files_to_remove.update(get_module_files(module_name, config))

    # Add installer files if removing all modules
    if set(selected) == set(installed_modules.keys()):
        files_to_remove.update(INSTALLER_FILES)

    # Show what will be removed
    print(f"Install directory: {install_dir}")
    if args.purge:
        print(f"\n⚠️  PURGE MODE: Will remove ENTIRE directory including user files!")
    else:
        print(f"\nModules to uninstall: {', '.join(selected)}")
        print(f"\nFiles/directories to remove:")
        for f in sorted(files_to_remove):
            path = install_dir / f
            exists = "✓" if path.exists() else "✗ (not found)"
            print(f"  {f} {exists}")

    # Confirmation
    if not args.yes and not args.dry_run:
        prompt = "\nProceed with uninstallation? [y/N] "
        response = input(prompt).strip().lower()
        if response not in ("y", "yes"):
            print("Aborted.")
            return 0

    if args.dry_run:
        print("\n[Dry run] No files were removed.")
        return 0

    print(f"\nUninstalling...")
    removed: List[str] = []

    if args.purge:
        shutil.rmtree(install_dir)
        print(f"  ✓ Removed {install_dir}")
        removed.append(str(install_dir))
    else:
        # Remove files/dirs in reverse order (files before parent dirs)
        for item in sorted(files_to_remove, key=lambda x: x.count("/"), reverse=True):
            path = install_dir / item
            if not path.exists():
                continue
            try:
                if path.is_dir():
                    # Only remove if empty or if it's a known module dir
                    if item in ("bin",):
                        # For bin, only remove codeagent-wrapper
                        wrapper = path / "codeagent-wrapper"
                        if wrapper.exists():
                            wrapper.unlink()
                            print(f"  ✓ Removed bin/codeagent-wrapper")
                            removed.append("bin/codeagent-wrapper")
                        # Remove bin if empty
                        if path.exists() and not any(path.iterdir()):
                            path.rmdir()
                            print(f"  ✓ Removed empty bin/")
                    else:
                        shutil.rmtree(path)
                        print(f"  ✓ Removed {item}/")
                        removed.append(item)
                else:
                    path.unlink()
                    print(f"  ✓ Removed {item}")
                    removed.append(item)
            except OSError as e:
                print(f"  ✗ Failed to remove {item}: {e}", file=sys.stderr)

        # Remove module hooks from settings.json
        for m in selected:
            if unmerge_hooks_from_settings(m, install_dir):
                print(f"  ✓ Removed hooks for module '{m}' from settings.json")

        # Update installed_modules.json
        status_file = install_dir / "installed_modules.json"
        if status_file.exists() and selected != list(installed_modules.keys()):
            # Partial uninstall: update status file
            for m in selected:
                installed_modules.pop(m, None)
            if installed_modules:
                with status_file.open("w", encoding="utf-8") as f:
                    json.dump({"modules": installed_modules}, f, indent=2)
                print(f"  ✓ Updated installed_modules.json")

        # Remove install dir if empty
        if install_dir.exists() and not any(install_dir.iterdir()):
            install_dir.rmdir()
            print(f"  ✓ Removed empty install directory")

    # Clean shell configs
    for rc_name in (".bashrc", ".zshrc"):
        rc_file = Path.home() / rc_name
        if cleanup_shell_config(rc_file, bin_dir):
            print(f"  ✓ Cleaned PATH from {rc_name}")

    print("")
    if removed:
        print(f"✓ Uninstallation complete ({len(removed)} items removed)")
    else:
        print("✓ Nothing to remove")

    if install_dir.exists() and any(install_dir.iterdir()):
        remaining = list(install_dir.iterdir())
        print(f"\nNote: {len(remaining)} items remain in {install_dir}")
        print("These are either user files or from other modules.")
        print("Use --purge to remove everything (DANGEROUS).")

    return 0


if __name__ == "__main__":
    sys.exit(main())
