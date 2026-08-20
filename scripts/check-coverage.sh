#!/bin/bash
# Check that test coverage meets minimum threshold

set -e

# Run tests with coverage
echo "📊 Checking test coverage..."

coverage_output=$(go test -coverprofile=/tmp/coverage.out ./... 2>&1 || true)

if [ ! -f "/tmp/coverage.out" ]; then
  echo "⚠️  Could not generate coverage report"
  exit 0
fi

# Extract coverage percentage
coverage=$(go tool cover -func=/tmp/coverage.out | grep total | awk '{print $3}' | sed 's/%//')

# Minimum threshold (adjust as needed)
min_coverage=70

if (( $(echo "$coverage < $min_coverage" | bc -l) )); then
  echo "❌ Test coverage below minimum threshold"
  echo "   Current: ${coverage}%"
  echo "   Minimum: ${min_coverage}%"
  echo ""
  echo "Coverage by package:"
  go tool cover -func=/tmp/coverage.out | grep -v "total"
  echo ""
  read -p "Continue without meeting coverage? (y/n) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    exit 1
  fi
else
  echo "✅ Test coverage: ${coverage}% (threshold: ${min_coverage}%)"
fi

exit 0
