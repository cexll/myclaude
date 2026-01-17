#!/bin/bash
set -e

if [ -z "${SKIP_WARNING:-}" ]; then
  echo "⚠️  WARNING: install.sh is LEGACY and will be removed in future versions."
  echo "Please use the new installation method:"
  echo "  python3 install.py --install-dir ~/.claude"
  echo ""
  echo "Set SKIP_WARNING=1 to bypass this message"
  echo "Continuing with legacy installation in 5 seconds..."
  sleep 5
fi

# Detect platform
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Normalize architecture names
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Build download URL
REPO="cexll/myclaude"
VERSION="latest"
BINARY_NAME="codeagent-wrapper-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/${VERSION}/download/${BINARY_NAME}"

echo "Downloading codeagent-wrapper from ${URL}..."
if ! curl -fsSL "$URL" -o /tmp/codeagent-wrapper; then
    echo "ERROR: failed to download binary" >&2
    exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-$HOME/.claude}"
BIN_DIR="${INSTALL_DIR}/bin"
mkdir -p "$BIN_DIR"

mv /tmp/codeagent-wrapper "${BIN_DIR}/codeagent-wrapper"
chmod +x "${BIN_DIR}/codeagent-wrapper"

if "${BIN_DIR}/codeagent-wrapper" --version >/dev/null 2>&1; then
    echo "codeagent-wrapper installed successfully to ${BIN_DIR}/codeagent-wrapper"
else
    echo "ERROR: installation verification failed" >&2
    exit 1
fi

# Auto-add to shell config files with idempotency
if [[ ":${PATH}:" != *":${BIN_DIR}:"* ]]; then
    echo ""
    echo "WARNING: ${BIN_DIR} is not in your PATH"

    # Detect shell and set config files
    if [ -n "$ZSH_VERSION" ]; then
        RC_FILE="$HOME/.zshrc"
        PROFILE_FILE="$HOME/.zprofile"
    else
        RC_FILE="$HOME/.bashrc"
        PROFILE_FILE="$HOME/.profile"
    fi

    # Idempotent add: check if complete export statement already exists
    EXPORT_LINE="export PATH=\"${BIN_DIR}:\$PATH\""
    FILES_TO_UPDATE=("$RC_FILE" "$PROFILE_FILE")

    for FILE in "${FILES_TO_UPDATE[@]}"; do
        if [ -f "$FILE" ] && grep -qF "${EXPORT_LINE}" "$FILE" 2>/dev/null; then
            echo "  ${BIN_DIR} already in ${FILE}, skipping."
        else
            echo "  Adding to ${FILE}..."
            echo "" >> "$FILE"
            echo "# Added by myclaude installer" >> "$FILE"
            echo "${EXPORT_LINE}" >> "$FILE"
        fi
    done

    echo "  Done. Restart your shell or run:"
    echo "    source ${PROFILE_FILE}"
    echo "    source ${RC_FILE}"
    echo ""
fi
