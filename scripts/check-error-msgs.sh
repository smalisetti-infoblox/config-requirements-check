#!/bin/bash
# Check for high-quality error messages

set -e

changed_files=$(git diff --cached --name-only | grep -E "\.go$" | grep -v "_test.go$" | grep -v "vendor/" || true)

if [ -z "$changed_files" ]; then
  exit 0
fi

echo "🔍 Checking error message quality..."

issues_found=false

for file in $changed_files; do
  if [ ! -f "$file" ]; then
    continue
  fi

  # Check for error messages that start with capital letter (should be lowercase for wrapping)
  if grep -n "errors\.New(\"[A-Z]\|fmt\.Errorf(\"[A-Z]" "$file" > /dev/null; then
    echo "⚠️  Error messages should start with lowercase in $file:"
    grep -n "errors\.New(\"[A-Z]\|fmt\.Errorf(\"[A-Z]" "$file" | head -3
    issues_found=true
  fi

  # Check for error messages that are too vague
  if grep -n "errors\.New(\"error\|fmt\.Errorf(\"error\|\"Error" "$file" > /dev/null; then
    echo "⚠️  Error messages too vague in $file (just 'error'):"
    grep -n "errors\.New(\"error\|fmt\.Errorf(\"error\|\"Error" "$file" | head -3
    issues_found=true
  fi

  # Check for error messages missing context
  if grep -n "errors\.New(\".\{1,5\}\"\)" "$file" > /dev/null; then
    echo "⚠️  Error messages too short (missing context) in $file:"
    grep -n "errors\.New(\".\{1,5\}\"\)" "$file" | head -3
    issues_found=true
  fi

done

if [ "$issues_found" = true ]; then
  echo ""
  echo "Error message guidelines:"
  echo "  ✓ Start with lowercase (for error wrapping)"
  echo "  ✓ Be specific about what went wrong"
  echo "  ✓ Include context: variable names, expected values"
  echo ""
  echo "  Good: fmt.Errorf(\"invalid port %d: must be 1024-65535\", port)"
  echo "  Bad:  errors.New(\"Error\")"
  echo ""
  read -p "Continue? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

exit 0
