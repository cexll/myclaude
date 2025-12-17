#!/bin/bash
# Example pre-commit hook
# This hook runs before git commit to validate code quality

set -e

# Get staged files
STAGED_FILES="$(git diff --cached --name-only --diff-filter=ACM)"

if [ -z "$STAGED_FILES" ]; then
  echo "No files to validate"
  exit 0
fi

echo "Running pre-commit checks..."

# Check Go files
GO_FILES="$(printf '%s\n' "$STAGED_FILES" | grep '\.go$' || true)"
if [ -n "$GO_FILES" ]; then
  echo "Checking Go files..."

  if ! command -v gofmt &> /dev/null; then
    echo "❌ gofmt not found. Please install Go (gofmt is included with the Go toolchain)."
    exit 1
  fi

  # Format check
  GO_FILE_ARGS=()
  while IFS= read -r file; do
    if [ -n "$file" ]; then
      GO_FILE_ARGS+=("$file")
    fi
  done <<< "$GO_FILES"

  if [ "${#GO_FILE_ARGS[@]}" -gt 0 ]; then
    UNFORMATTED="$(gofmt -l "${GO_FILE_ARGS[@]}")"
    if [ -n "$UNFORMATTED" ]; then
      echo "❌ The following files need formatting:"
      echo "$UNFORMATTED"
      echo "Run: gofmt -w <file>"
      exit 1
    fi
  fi

  # Run tests
  if command -v go &> /dev/null; then
    echo "Running go tests..."
    go test ./... -short || {
      echo "❌ Tests failed"
      exit 1
    }
  fi
fi

# Check JSON files
JSON_FILES="$(printf '%s\n' "$STAGED_FILES" | grep '\.json$' || true)"
if [ -n "$JSON_FILES" ]; then
  echo "Validating JSON files..."
  if ! command -v jq &> /dev/null; then
    echo "❌ jq not found. Please install jq to validate JSON files."
    exit 1
  fi
  while IFS= read -r file; do
    if [ -z "$file" ]; then
      continue
    fi
    if ! jq empty "$file" 2>/dev/null; then
      echo "❌ Invalid JSON: $file"
      exit 1
    fi
  done <<< "$JSON_FILES"
fi

# Check Markdown files
MD_FILES="$(printf '%s\n' "$STAGED_FILES" | grep '\.md$' || true)"
if [ -n "$MD_FILES" ]; then
  echo "Checking markdown files..."
  # Add markdown linting if needed
fi

echo "✅ All pre-commit checks passed"
exit 0
