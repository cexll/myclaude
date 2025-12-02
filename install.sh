#!/bin/bash
set -e

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
BINARY_NAME="codex-wrapper-${OS}-${ARCH}"
URL="https://github.com/${REPO}/releases/${VERSION}/download/${BINARY_NAME}"

echo "Downloading codex-wrapper from ${URL}..."
if ! curl -fsSL "$URL" -o /tmp/codex-wrapper; then
    echo "ERROR: failed to download binary" >&2
    exit 1
fi

mkdir -p "$HOME/bin"

mv /tmp/codex-wrapper "$HOME/bin/codex-wrapper"
chmod +x "$HOME/bin/codex-wrapper"

if "$HOME/bin/codex-wrapper" --version >/dev/null 2>&1; then
    echo "codex-wrapper installed successfully to ~/bin/codex-wrapper"
else
    echo "ERROR: installation verification failed" >&2
    exit 1
fi

if [[ ":$PATH:" != *":$HOME/bin:"* ]]; then
    echo ""
    echo "WARNING: ~/bin is not in your PATH"
    echo "Add this line to your ~/.bashrc or ~/.zshrc:"
    echo ""
    echo "    export PATH=\"\$HOME/bin:\$PATH\""
    echo ""
fi
