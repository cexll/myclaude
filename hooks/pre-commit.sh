#!/bin/bash
# Example pre-commit hook
# This hook runs before git commit to validate code quality

set -e

# Get staged files
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)

if [ -z "$STAGED_FILES" ]; then
  echo "No files to validate"
  exit 0
fi

echo "Running pre-commit checks..."

# Check Go files
GO_FILES=$(echo "$STAGED_FILES" | grep '\.go$' || true)
if [ -n "$GO_FILES" ]; then
  echo "Checking Go files..."

  # Format check
  gofmt -l $GO_FILES | while read -r file; do
    if [ -n "$file" ]; then
      echo "❌ $file needs formatting (run: gofmt -w $file)"
      exit 1
    fi
  done

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
JSON_FILES=$(echo "$STAGED_FILES" | grep '\.json$' || true)
if [ -n "$JSON_FILES" ]; then
  echo "Validating JSON files..."
  for file in $JSON_FILES; do
    if ! jq empty "$file" 2>/dev/null; then
      echo "❌ Invalid JSON: $file"
      exit 1
    fi
  done
fi

# Check Markdown files
MD_FILES=$(echo "$STAGED_FILES" | grep '\.md$' || true)
if [ -n "$MD_FILES" ]; then
  echo "Checking markdown files..."
  # Add markdown linting if needed
fi

echo "✅ All pre-commit checks passed"
exit 0
