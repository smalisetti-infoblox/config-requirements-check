#!/bin/bash
# Ensure TODO/FIXME comments have issue references

set -e

changed_files=$(git diff --cached --name-only | grep -E "\.(go|md|yaml)$" || true)

if [ -z "$changed_files" ]; then
  exit 0
fi

found_bare_todos=false

for file in $changed_files; do
  if [ ! -f "$file" ]; then
    continue
  fi

  # Look for TODO/FIXME without issue references
  if grep -n "// TODO\|// FIXME\|# TODO\|# FIXME" "$file" | grep -v "#[0-9]" | grep -v "github.com" > /dev/null; then
    echo "❌ Found unresolved TODO/FIXME comments in $file:"
    grep -n "// TODO\|// FIXME\|# TODO\|# FIXME" "$file" | grep -v "#[0-9]" | grep -v "github.com" || true
    found_bare_todos=true
  fi
done

if [ "$found_bare_todos" = true ]; then
  echo ""
  echo "TODOs must reference a GitHub issue, e.g.:"
  echo "  // TODO: Fix this later (see #123)"
  echo "  // FIXME: Handle error case (GitHub issue #456)"
  exit 1
fi

exit 0
