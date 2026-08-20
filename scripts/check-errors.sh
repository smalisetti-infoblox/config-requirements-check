#!/bin/bash
# Check for unchecked errors and missing error handling

set -e

changed_files=$(git diff --cached --name-only | grep -E "\.go$" | grep -v "_test.go$" | grep -v "vendor/" || true)

if [ -z "$changed_files" ]; then
  exit 0
fi

echo "🔍 Checking error handling patterns..."

issues_found=false

for file in $changed_files; do
  if [ ! -f "$file" ]; then
    continue
  fi

  # Check for error result that's ignored (common pattern: _ = someFunc())
  if grep -n "_ = .*\.\|_ = ioutil\|_ = os\|_ = fmt" "$file" | grep -v "test\|Test" > /dev/null; then
    echo "⚠️  Ignoring error return value in $file:"
    grep -n "_ = " "$file" | head -5
    issues_found=true
  fi

  # Check for defer without checking if file opened
  if grep -B2 "defer.*Close" "$file" | grep -v "if err" > /dev/null; then
    echo "⚠️  Potential error handling issue in $file (defer without err check):"
    grep -n "defer.*Close" "$file" | head -3
  fi

done

if [ "$issues_found" = true ]; then
  echo ""
  echo "⚠️  Please review error handling:"
  echo "   - Never use _ = to ignore errors"
  echo "   - Prefer: if err != nil { return err }"
  echo "   - Or log the error and continue if appropriate"
  echo ""
  read -p "Continue anyway? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

exit 0
