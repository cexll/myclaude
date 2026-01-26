#!/bin/bash
set -e

INSTALL_DIR="${INSTALL_DIR:-$HOME/.claude}"
BIN_DIR="${INSTALL_DIR}/bin"
STATUS_FILE="${INSTALL_DIR}/installed_modules.json"
DRY_RUN=false
PURGE=false
YES=false
LIST_ONLY=false
MODULES=""

usage() {
    cat <<EOF
Usage: $0 [OPTIONS]

Uninstall myclaude modules.

Options:
  --install-dir DIR   Installation directory (default: ~/.claude)
  --module MODULES    Comma-separated modules to uninstall (default: all)
  --list              List installed modules and exit
  --dry-run           Show what would be removed without removing
  --purge             Remove entire install directory (DANGEROUS)
  -y, --yes           Skip confirmation prompt
  -h, --help          Show this help

Examples:
  $0 --list                    # List installed modules
  $0 --dry-run                 # Preview what would be removed
  $0 --module do               # Uninstall only 'do' module
  $0 -y                        # Uninstall all without confirmation
  $0 --purge -y                # Remove everything (DANGEROUS)
EOF
    exit 0
}

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --install-dir) INSTALL_DIR="$2"; BIN_DIR="${INSTALL_DIR}/bin"; STATUS_FILE="${INSTALL_DIR}/installed_modules.json"; shift 2 ;;
        --module) MODULES="$2"; shift 2 ;;
        --list) LIST_ONLY=true; shift ;;
        --dry-run) DRY_RUN=true; shift ;;
        --purge) PURGE=true; shift ;;
        -y|--yes) YES=true; shift ;;
        -h|--help) usage ;;
        *) echo "Unknown option: $1" >&2; exit 1 ;;
    esac
done

# Check if install dir exists
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Install directory not found: $INSTALL_DIR"
    echo "Nothing to uninstall."
    exit 0
fi

# List installed modules
list_modules() {
    if [ ! -f "$STATUS_FILE" ]; then
        echo "No modules installed (installed_modules.json not found)"
        return
    fi
    echo "Installed modules in $INSTALL_DIR:"
    echo "Module          Status     Installed At"
    echo "--------------------------------------------------"
    # Parse JSON with basic tools (no jq dependency)
    python3 -c "
import json, sys
try:
    with open('$STATUS_FILE') as f:
        data = json.load(f)
    for name, info in data.get('modules', {}).items():
        status = info.get('status', 'unknown')
        ts = info.get('installed_at', 'unknown')[:19]
        print(f'{name:<15} {status:<10} {ts}')
except Exception as e:
    print(f'Error reading status file: {e}', file=sys.stderr)
    sys.exit(1)
"
}

if [ "$LIST_ONLY" = true ]; then
    list_modules
    exit 0
fi

# Get installed modules from status file
get_installed_modules() {
    if [ ! -f "$STATUS_FILE" ]; then
        echo ""
        return
    fi
    python3 -c "
import json
try:
    with open('$STATUS_FILE') as f:
        data = json.load(f)
    print(' '.join(data.get('modules', {}).keys()))
except:
    print('')
"
}

INSTALLED=$(get_installed_modules)

# Determine modules to uninstall
if [ -n "$MODULES" ]; then
    SELECTED="$MODULES"
else
    SELECTED="$INSTALLED"
fi

if [ -z "$SELECTED" ] && [ "$PURGE" != true ]; then
    echo "No modules to uninstall."
    echo "Use --list to see installed modules, or --purge to remove everything."
    exit 0
fi

echo "Install directory: $INSTALL_DIR"

if [ "$PURGE" = true ]; then
    echo ""
    echo "⚠️  PURGE MODE: Will remove ENTIRE directory including user files!"
else
    echo ""
    echo "Modules to uninstall: $SELECTED"
    echo ""
    echo "Files/directories that may be removed:"
    for item in commands agents skills docs bin CLAUDE.md install.log installed_modules.json; do
        if [ -e "${INSTALL_DIR}/${item}" ]; then
            echo "  $item ✓"
        fi
    done
fi

# Confirmation
if [ "$YES" != true ] && [ "$DRY_RUN" != true ]; then
    echo ""
    read -p "Proceed with uninstallation? [y/N] " response
    case "$response" in
        [yY]|[yY][eE][sS]) ;;
        *) echo "Aborted."; exit 0 ;;
    esac
fi

if [ "$DRY_RUN" = true ]; then
    echo ""
    echo "[Dry run] No files were removed."
    exit 0
fi

echo ""
echo "Uninstalling..."

if [ "$PURGE" = true ]; then
    rm -rf "$INSTALL_DIR"
    echo "  ✓ Removed $INSTALL_DIR"
else
    # Remove codeagent-wrapper binary
    if [ -f "${BIN_DIR}/codeagent-wrapper" ]; then
        rm -f "${BIN_DIR}/codeagent-wrapper"
        echo "  ✓ Removed bin/codeagent-wrapper"
    fi

    # Remove bin directory if empty
    if [ -d "$BIN_DIR" ] && [ -z "$(ls -A "$BIN_DIR" 2>/dev/null)" ]; then
        rmdir "$BIN_DIR"
        echo "  ✓ Removed empty bin/"
    fi

    # Remove installed directories
    for dir in commands agents skills docs; do
        if [ -d "${INSTALL_DIR}/${dir}" ]; then
            rm -rf "${INSTALL_DIR}/${dir}"
            echo "  ✓ Removed ${dir}/"
        fi
    done

    # Remove installed files
    for file in CLAUDE.md install.log installed_modules.json installed_modules.json.bak; do
        if [ -f "${INSTALL_DIR}/${file}" ]; then
            rm -f "${INSTALL_DIR}/${file}"
            echo "  ✓ Removed ${file}"
        fi
    done

    # Remove install directory if empty
    if [ -d "$INSTALL_DIR" ] && [ -z "$(ls -A "$INSTALL_DIR" 2>/dev/null)" ]; then
        rmdir "$INSTALL_DIR"
        echo "  ✓ Removed empty install directory"
    fi
fi

# Clean up PATH from shell config files
cleanup_shell_config() {
    local rc_file="$1"
    if [ -f "$rc_file" ]; then
        if grep -q "# Added by myclaude installer" "$rc_file" 2>/dev/null; then
            # Create backup
            cp "$rc_file" "${rc_file}.bak"
            # Remove myclaude lines
            grep -v "# Added by myclaude installer" "$rc_file" | \
            grep -v "export PATH=\"${BIN_DIR}:\$PATH\"" > "${rc_file}.tmp"
            mv "${rc_file}.tmp" "$rc_file"
            echo "  ✓ Cleaned PATH from $(basename "$rc_file")"
        fi
    fi
}

cleanup_shell_config "$HOME/.bashrc"
cleanup_shell_config "$HOME/.zshrc"

echo ""
echo "✓ Uninstallation complete"

# Check for remaining files
if [ -d "$INSTALL_DIR" ] && [ -n "$(ls -A "$INSTALL_DIR" 2>/dev/null)" ]; then
    remaining=$(ls -1 "$INSTALL_DIR" 2>/dev/null | wc -l | tr -d ' ')
    echo ""
    echo "Note: $remaining items remain in $INSTALL_DIR"
    echo "These are either user files or from other modules."
    echo "Use --purge to remove everything (DANGEROUS)."
fi
