#!/usr/bin/env python3
"""Install/uninstall do skill to ~/.claude/skills/do"""
import argparse
import json
import os
import shutil
import sys
from pathlib import Path

SKILL_NAME = "do"
HOOK_PATH = "~/.claude/skills/do/hooks/stop-hook.py"

MODELS_JSON_TEMPLATE = {
    "agents": {
        "code-explorer": {
            "backend": "claude",
            "model": "claude-sonnet-4-5-20250929"
        },
        "code-architect": {
            "backend": "claude",
            "model": "claude-sonnet-4-5-20250929"
        },
        "code-reviewer": {
            "backend": "claude",
            "model": "claude-sonnet-4-5-20250929"
        }
    }
}

def get_settings_path() -> Path:
    return Path.home() / ".claude" / "settings.json"

def load_settings() -> dict:
    path = get_settings_path()
    if path.exists():
        with open(path, "r", encoding="utf-8") as f:
            return json.load(f)
    return {}

def save_settings(settings: dict):
    path = get_settings_path()
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(settings, f, indent=2)

def add_hook(settings: dict) -> dict:
    hook_command = str(Path(HOOK_PATH).expanduser())
    hook_entry = {
        "type": "command",
        "command": f"python3 {hook_command}"
    }

    if "hooks" not in settings:
        settings["hooks"] = {}
    if "Stop" not in settings["hooks"]:
        settings["hooks"]["Stop"] = []

    stop_hooks = settings["hooks"]["Stop"]

    for item in stop_hooks:
        if "hooks" in item:
            for h in item["hooks"]:
                if "stop-hook" in h.get("command", "") and "do" in h.get("command", ""):
                    h["command"] = f"python3 {hook_command}"
                    return settings

    stop_hooks.append({"hooks": [hook_entry]})
    return settings

def remove_hook(settings: dict) -> dict:
    if "hooks" not in settings or "Stop" not in settings["hooks"]:
        return settings

    stop_hooks = settings["hooks"]["Stop"]
    new_stop_hooks = []

    for item in stop_hooks:
        if "hooks" in item:
            filtered = [h for h in item["hooks"]
                       if "stop-hook" not in h.get("command", "")
                       or "do" not in h.get("command", "")]
            if filtered:
                item["hooks"] = filtered
                new_stop_hooks.append(item)
        else:
            new_stop_hooks.append(item)

    settings["hooks"]["Stop"] = new_stop_hooks
    if not settings["hooks"]["Stop"]:
        del settings["hooks"]["Stop"]
    if not settings["hooks"]:
        del settings["hooks"]

    return settings

def install_models_json():
    """Install ~/.codeagent/models.json if not exists"""
    path = Path.home() / ".codeagent" / "models.json"
    if path.exists():
        print(f"⚠ {path} already exists, skipping")
        return

    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w", encoding="utf-8") as f:
        json.dump(MODELS_JSON_TEMPLATE, f, indent=2)
    print(f"✓ Created {path}")

def install():
    src = Path(__file__).parent.resolve()
    dest = Path.home() / ".claude" / "skills" / SKILL_NAME

    dest.mkdir(parents=True, exist_ok=True)

    exclude = {".git", "__pycache__", ".DS_Store", "install.py"}

    for item in src.iterdir():
        if item.name in exclude:
            continue
        target = dest / item.name
        if target.exists():
            if target.is_dir():
                shutil.rmtree(target)
            else:
                target.unlink()
        if item.is_dir():
            shutil.copytree(item, target)
        else:
            shutil.copy2(item, target)

    settings = load_settings()
    settings = add_hook(settings)
    save_settings(settings)

    install_models_json()

    print(f"✓ Installed to {dest}")
    print(f"✓ Hook added to settings.json")

def uninstall():
    dest = Path.home() / ".claude" / "skills" / SKILL_NAME

    settings = load_settings()
    settings = remove_hook(settings)
    save_settings(settings)
    print(f"✓ Hook removed from settings.json")

    if dest.exists():
        shutil.rmtree(dest)
        print(f"✓ Removed {dest}")
    else:
        print(f"⚠ {dest} not found")

def main():
    parser = argparse.ArgumentParser(description="Install/uninstall do skill")
    parser.add_argument("--uninstall", "-u", action="store_true", help="Uninstall the skill")
    args = parser.parse_args()

    if args.uninstall:
        uninstall()
    else:
        install()

if __name__ == "__main__":
    main()
