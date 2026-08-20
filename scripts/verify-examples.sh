#!/bin/bash
# Verify example YAML files are valid and parseable

set -e

changed_yaml_files=$(git diff --cached --name-only | grep -E "examples/.*\.yaml$" || true)

if [ -z "$changed_yaml_files" ]; then
  exit 0
fi

echo "🔍 Verifying example YAML files..."

for file in $changed_yaml_files; do
  if [ ! -f "$file" ]; then
    continue
  fi

  echo "  Checking $file..."

  # Use Go's YAML parser to validate
  if ! go run . -lint -requirements "$file" > /dev/null 2>&1; then
    echo "❌ Example file not valid YAML: $file"
    exit 1
  fi

  # Check that examples can be parsed by the tool
  echo "  ✅ $file is valid"
done

echo "✅ All example files verified"
exit 0
