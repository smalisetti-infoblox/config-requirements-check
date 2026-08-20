#!/bin/bash
# Ensure documentation is updated when relevant code changes

set -e

changed_go_files=$(git diff --cached --name-only | grep -E "\.go$" | grep -v "_test.go$" | grep -v "vendor/" || true)

if [ -z "$changed_go_files" ]; then
  exit 0
fi

docs_changed=false

# Check if any relevant docs files were modified
for doc_file in "EXAMPLES.md" "IMPROVEMENTS.md" "README.md" "IMPLEMENTATION_PLAN.md"; do
  if git diff --cached --name-only | grep -q "^$doc_file$"; then
    docs_changed=true
    break
  fi
done

# Check if help output would have changed (main.go modifications)
if echo "$changed_go_files" | grep -q "^main\.go$"; then
  # If main.go changed, help output likely changed
  if ! git diff --cached main.go | grep -q "Examples:\|Usage:\|Flags:"; then
    echo "ℹ️  main.go modified - did you update the help output?"
    read -p "Continue anyway? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      exit 1
    fi
  fi
fi

# Warn if documentation should be updated
if [ "$docs_changed" = false ] && [ ! -z "$changed_go_files" ]; then
  echo "⚠️  Go code changed but documentation files were not updated:"
  echo "   - EXAMPLES.md: For usage examples"
  echo "   - IMPROVEMENTS.md: For implementation details"
  echo "   - README.md: For high-level documentation"
  echo ""
  read -p "Continue without updating docs? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
fi

exit 0
