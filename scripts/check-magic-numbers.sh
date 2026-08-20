#!/bin/bash
# Check for unexplained numeric literals (magic numbers)

set -e

changed_files=$(git diff --cached --name-only | grep -E "\.go$" | grep -v "_test.go$" | grep -v "vendor/" | grep -v "_test" || true)

if [ -z "$changed_files" ]; then
  exit 0
fi

echo "🔍 Checking for magic numbers..."

issues_found=false

for file in $changed_files; do
  if [ ! -f "$file" ]; then
    continue
  fi

  # Look for numeric literals that might be magic numbers
  # Common patterns: >= 2, < 5, == 65535, etc.
  # Exclude: len, cap, const definitions, errors with line numbers
  grep -n "[^A-Za-z_]\(65535\|1024\|5000\|3000\|4500\|100\|50\) " "$file" 2>/dev/null | \
    grep -v "const \|= \|//\|\"" | \
    grep -v "test\|Test\|^.*_test\.go" > /tmp/magic_numbers.txt 2>/dev/null || true

  if [ -s /tmp/magic_numbers.txt ]; then
    echo "⚠️  Potential magic numbers in $file:"
    cat /tmp/magic_numbers.txt | head -5
    issues_found=true
  fi

done

if [ "$issues_found" = true ]; then
  echo ""
  echo "Magic number guidelines:"
  echo "  Instead of:  if port < 65535"
  echo "  Use:         const maxPort = 65535; if port < maxPort"
  echo ""
  echo "  Instead of:  timeout := 5000"
  echo "  Use:         const defaultTimeout = 5 * time.Second"
  echo ""
  read -p "Review magic numbers and continue? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

exit 0
