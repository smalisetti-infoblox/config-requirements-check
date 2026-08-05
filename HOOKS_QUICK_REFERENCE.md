# Pre-Commit Hooks - Quick Reference

## Installation (One-time)
```bash
pip install pre-commit  # or: brew install pre-commit
./scripts/setup-hooks.sh
```

## What Happens When You Commit

```
$ git commit -m "Add new feature"

✅ Formatting (auto-fixed)
  - gofmt: Format Go code
  - end-of-file-fixer: Fix file endings
  
✅ Validation (must pass)
  - golangci-lint: Lint Go code
  - Check YAML syntax
  - Find private keys
  
✅ Tests (must pass)
  - require-tests: Code + tests must change together
  - test-suite: All tests pass
  - race-detector: No data races
  - coverage-threshold: Coverage >= 70%
  
⚠️  Documentation (warns or blocks)
  - require-docs: Documentation files updated?
  - verify-help-output: Help text includes new flags?
  - verify-examples: Example files valid?
  - check-starter-yaml-sync: Starter template updated with new features?
  
✅ Code Quality (must pass)
  - no-debug-statements: No println/fmt.Println
  - no-bare-todos: TODOs reference issues (#123)
  - check-error-handling: Errors properly handled
  - check-error-msgs: Error messages are descriptive
  - check-magic-numbers: Numbers have constants
  
✅ Commit Message (must pass)
  - validate-commit-msg: 10+ chars, capital letter, meaningful
```

## Common Hook Failures & Solutions

| Issue | Solution |
|-------|----------|
| Code changed but no tests | `git add *_test.go && git commit` |
| Debug println left in | Remove `fmt.Println()` statements |
| TODO without issue ref | Change `// TODO` to `// TODO: (#123)` |
| Undocumented flag | Add to help text in main.go |
| New requirement not in starter.yaml | Add example to `examples/starter.yaml` |
| Test coverage dropped | Write more tests for new code |
| Commit message too short | Use at least 10 characters, be descriptive |
| Error message quality | Start with lowercase: `"value is invalid"` not `"Error"` |
| Magic number (e.g., `5000`) | Use constant: `const defaultTimeout = 5000` |

## Useful Commands

```bash
# Install hooks
./scripts/setup-hooks.sh

# Run all hooks manually
pre-commit run --all-files

# Run specific hook
pre-commit run gofmt --all-files
pre-commit run require-tests --all-files

# Skip hooks for one commit (use sparingly)
git commit --no-verify

# Check what's staged
git diff --cached

# Run tests locally before committing
go test -v ./...

# Check coverage locally
go test -cover ./...

# Format code before commit
go fmt ./...

# Lint code
golangci-lint run
```

## Pre-Commit Workflow

```
1. Make code changes
   $ vim rules.go

2. Add corresponding test changes
   $ vim rules_test.go

3. Update documentation if needed
   $ vim EXAMPLES.md

4. Stage changes
   $ git add rules.go rules_test.go EXAMPLES.md

5. Commit (hooks run automatically)
   $ git commit -m "Add feature X for requirement Y"

6. Hooks check:
   ✅ Formatting
   ✅ Tests included
   ✅ Tests pass
   ✅ Coverage >= 70%
   ✅ No debug code
   ✅ Good error messages
   ✅ Meaningful commit message

7. Commit succeeds!
   [main abc123d] Add feature X for requirement Y
```

## Disable/Bypass (Emergency Only)

```bash
# Skip all hooks for one commit
git commit --no-verify

# Disable specific hook temporarily
# Edit .pre-commit-config.yaml and comment out the hook:
# - id: no-debug-statements  # Disabled temporarily
#   name: No debug statements

# Re-enable after
pre-commit install --install-hooks
```

## Hook Categories at a Glance

| Category | Prevents | Allows |
|----------|----------|--------|
| **Formatting** | Inconsistent code style | All properly formatted code |
| **Testing** | Untested code commits | Code + tests together |
| **Documentation** | Out-of-sync docs | Updated docs |
| **Code Quality** | Debug statements, bad errors | Clean, maintainable code |
| **Commit Messages** | Vague messages | Descriptive, 10+ char messages |

## Key Rules

✅ **MUST**: Tests update with code changes
✅ **MUST**: All tests pass
✅ **MUST**: No debug statements (fmt.Println, println)
✅ **MUST**: TODOs reference GitHub issues
✅ **MUST**: Meaningful commit messages (10+ chars)
✅ **SHOULD**: Update documentation with changes
✅ **SHOULD**: Error messages are descriptive
✅ **SHOULD**: Coverage stays >= 70%
✅ **SHOULD**: No magic numbers

## Statistics

- **14 hooks** enforcing quality
- **Tests checked**: Unit tests, race conditions, coverage
- **Documentation verified**: Help text, examples, docs sync
- **Code quality**: Format, lint, error handling, clarity
- **Commit quality**: Message validation
- **Total files checked**: Go code, YAML, examples, documentation

## Still Need Help?

```bash
# See all hook details
cat PRE_COMMIT_HOOKS.md

# See what each hook does
cat .pre-commit-config.yaml

# Check installed hooks
git hook list

# Debug a failing hook
pre-commit run --all-files --verbose
```
