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

mkdir -p "$HOME/bin"

mv /tmp/codeagent-wrapper "$HOME/bin/codeagent-wrapper"
chmod +x "$HOME/bin/codeagent-wrapper"

if "$HOME/bin/codeagent-wrapper" --version >/dev/null 2>&1; then
    echo "codeagent-wrapper installed successfully to ~/bin/codeagent-wrapper"
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
