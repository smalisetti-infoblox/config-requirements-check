#!/bin/bash
# Validate that README.md documentation matches actual code implementation
# Checks: all flags are documented, all operators are mentioned, key fields are listed

set -e

# Only check if main.go or rules.go changed
if ! git diff --cached --name-only | grep -qE "^(main|rules)\.go$"; then
  exit 0
fi

errors=0

echo "🔍 Validating documentation accuracy against code..."

# Extract all flags from main.go
flags=$(grep -oP 'fs\.(?:String|Bool)\("([^"]+)"' main.go | cut -d'"' -f2 | sort | uniq)

echo "Checking flags are in README.md..."
for flag in $flags; do
  if ! grep -q "\-$flag" README.md; then
    echo "❌ Flag -$flag defined in main.go but not mentioned in README.md"
    errors=$((errors + 1))
  fi
done

# Check for required condition operators in README
echo "Checking condition operators are documented..."
operators=("gte" "gt" "lte" "lt" "contains" "between" "not_equals")
for op in "${operators[@]}"; do
  if ! grep -q "$op" README.md; then
    echo "❌ Condition operator '$op' not documented in README.md"
    errors=$((errors + 1))
  fi
done

# Check for required requirement fields in README
echo "Checking requirement fields are documented..."
fields=("unless" "skip_if" "severity" "remediation_hints")
for field in "${fields[@]}"; do
  if ! grep -q "$field" README.md; then
    echo "❌ Requirement field '$field' not documented in README.md"
    errors=$((errors + 1))
  fi
done

# Check that exit codes are documented
echo "Checking exit codes are documented..."
if ! grep -qE "(exit.*0|exit.*1|exit.*2|Exit code)" README.md; then
  echo "❌ Exit codes not documented in README.md"
  errors=$((errors + 1))
fi

if [ $errors -gt 0 ]; then
  echo ""
  echo "❌ Documentation accuracy check failed: $errors issue(s) found"
  echo "Update README.md to match the actual code implementation"
  exit 1
fi

echo "✅ Documentation accuracy check passed"
exit 0
