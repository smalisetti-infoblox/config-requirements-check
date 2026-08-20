#!/bin/bash
# Verify all flags are documented in help output

set -e

# Only check if main.go changed
if ! git diff --cached --name-only | grep -q "^main\.go$"; then
  exit 0
fi

# Get all flag definitions from current main.go
flags=$(grep -oP 'fs\.(?:String|Bool|Int)\("([^"]+)"' main.go | cut -d'"' -f2 | sort | uniq)

if [ -z "$flags" ]; then
  exit 0
fi

echo "🔍 Checking if all flags are documented in help text..."

missing_docs=false

for flag in $flags; do
  if ! grep -q "\-$flag" main.go; then
    echo "❌ Flag -$flag is defined but not documented in help text"
    missing_docs=true
  fi
done

if [ "$missing_docs" = true ]; then
  echo ""
  echo "All flags must be documented in the help text shown by -h"
  echo "Add your flag to the Examples section in main.go"
  exit 1
fi

echo "✅ All flags documented in help output"
exit 0
