#!/bin/bash
# Install pre-commit hooks for this project
# Usage: ./scripts/setup-hooks.sh

set -e

echo "🔧 Setting up pre-commit hooks..."

# Check if pre-commit is installed
if ! command -v pre-commit &> /dev/null; then
  echo "❌ pre-commit is not installed"
  echo ""
  echo "Install pre-commit with:"
  echo "  pip install pre-commit"
  echo "  # or with Homebrew:"
  echo "  brew install pre-commit"
  echo ""
  exit 1
fi

# Install the git hooks
pre-commit install
pre-commit install --hook-type commit-msg

echo ""
echo "✅ Pre-commit hooks installed!"
echo ""
echo "Hooks will run automatically on:"
echo "  • git commit (before creating the commit)"
echo "  • Commit messages (validation)"
echo ""
echo "To run hooks manually on all files:"
echo "  pre-commit run --all-files"
echo ""
echo "To bypass hooks for a specific commit:"
echo "  git commit --no-verify"
echo ""
echo "Hook checks include:"
echo "  ✓ Go formatting (gofmt)"
echo "  ✓ Go linting (golangci-lint)"
echo "  ✓ YAML validation"
echo "  ✓ Tests updated with code changes"
echo "  ✓ Documentation updated"
echo "  ✓ No debug statements (println, fmt.Println)"
echo "  ✓ No unresolved TODO/FIXME"
echo "  ✓ Error handling verification"
echo "  ✓ Error message quality"
echo "  ✓ Commit message validation"
echo "  ✓ Test coverage threshold"
echo "  ✓ Race condition detection"
echo "  ✓ Example file validation"
echo ""
