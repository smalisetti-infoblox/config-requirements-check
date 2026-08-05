# Pre-Commit Hooks Configuration

This project uses comprehensive pre-commit hooks to enforce code quality, maintainability, and debuggability standards. Hooks run automatically before each commit and prevent committing code that violates project standards.

## Installation

### Quick Start
```bash
# Install pre-commit framework
pip install pre-commit
# or with Homebrew
brew install pre-commit

# Install hooks for this project
./scripts/setup-hooks.sh
```

### Manual Installation
```bash
# From project root
pre-commit install
pre-commit install --hook-type commit-msg

# Verify installation
git hook list
```

## Hook Categories

### 1. **Code Formatting Hooks** (Automatic)

These hooks automatically fix formatting issues and prevent committing improperly formatted code.

#### gofmt - Go Code Formatting
```bash
# What it does:
- Formats all Go code consistently
- Enforces standard Go style
- Runs on all .go files

# Failures prevent commit
# Example:
if x==1 {  # ❌ Spacing wrong
    return y
}
# Fixed to:
if x == 1 {  # ✅ Proper spacing
    return y
}
```

#### golangci-lint - Go Linting
```bash
# What it does:
- Checks for code smells
- Detects unused variables/imports
- Ensures idiomatic Go patterns
- Runs multiple linters: errcheck, ineffassign, etc.

# Issues reported prevent commit
# Example:
var unused int  # ❌ Unused variable
return x

# Must remove or use it
```

#### end-of-file-fixer - File Ending
```bash
# What it does:
- Ensures files end with newline
- Removes trailing whitespace

# Auto-fixes before commit
```

#### YAML Validation
```bash
# What it does:
- Validates all YAML syntax
- Detects structural errors

# Example:
requirements:
  - id: test
    summary: Missing closing quote
# ❌ Fails YAML validation
```

#### Large File Detection
```bash
# What it does:
- Prevents committing files > 500KB
- Avoids bloating repository

# Example:
Commit rejected: binary.zip is 2.5MB
```

#### Private Key Detection
```bash
# What it does:
- Scans for accidentally committed secrets
- Detects AWS keys, private keys, etc.

# Example:
AWS_SECRET_ACCESS_KEY=... ❌ Prevented!
```

---

### 2. **Testing Hooks** (Fail on Violation)

These hooks ensure tests are maintained alongside code changes.

#### require-tests - Enforce Test Updates
```bash
# What it does:
- Detects when .go code changes
- Verifies corresponding _test.go files also changed
- Prevents untested code from being committed

# Success:
✅ modified:    rules.go
✅ modified:    rules_test.go
Commit allowed

# Failure:
❌ modified:    rules.go
❌ NO TEST CHANGES DETECTED
Commit prevented until tests updated
```

#### test-suite - Run All Tests
```bash
# What it does:
- Runs go test -v ./... before commit
- All tests must pass
- Tests cannot hang or timeout

# Failure output:
--- FAIL: TestSomething (0.00s)
    rule_test.go:42: unexpected result
Commit prevented

# Success:
ok      github.com/org/repo    1.234s
Commit allowed
```

#### race-detector - Detect Race Conditions
```bash
# What it does:
- Runs go test -race ./...
- Detects concurrent access to shared memory
- Prevents race condition bugs

# Failure:
WARNING: DATA RACE
Write at ... by goroutine 1:
Read at ... by goroutine 2:
Commit prevented
```

#### coverage-threshold - Maintain Coverage
```bash
# What it does:
- Generates coverage report
- Ensures coverage >= 70% (configurable)
- Warns if coverage decreases

# Failure:
❌ Test coverage below threshold
Current: 65%
Minimum: 70%

Required coverage breakdown:
  main.go: 45% (too low)
  rules.go: 92% (good)
```

---

### 3. **Documentation Hooks** (Warn or Block)

These hooks ensure documentation stays synchronized with code.

#### require-docs - Check Documentation Updates
```bash
# What it does:
- Detects when Go code changes
- Warns if documentation not updated
- Requires confirmation to proceed

# Warning:
⚠️  Go code changed but documentation files not updated:
   - EXAMPLES.md: For usage examples
   - IMPROVEMENTS.md: For implementation details
   - README.md: For high-level documentation

Continue without updating docs? (y/n)

# Best practice: Always update relevant docs
```

#### verify-help-output - Ensure Flag Documentation
```bash
# What it does:
- Checks if all CLI flags documented in -h output
- Prevents undocumented flags
- Ensures help text completeness

# Failure:
❌ Flag -new-flag defined but not documented
All flags must be documented in help text

# Fix: Add flag to Examples section in main.go
```

---

### 4. **Code Quality Hooks** (Warn or Block)

These hooks enforce patterns that improve maintainability.

#### no-debug-statements - Prevent Debug Code
```bash
# What it does:
- Detects fmt.Println in production code
- Detects log.Println in production code
- Allows in _test.go files

# Failure:
❌ Debug statements found in rules.go:
  fmt.Println("DEBUG: value is", x)

# Fix: Remove or replace with proper logging
//log.Printf("Processing value: %v", x)  // Add context
```

#### no-bare-todos - Require Issue References
```bash
# What it does:
- Finds // TODO comments
- Requires GitHub issue reference (#123)
- Prevents orphaned TODOs

# Failure:
❌ Unresolved TODO/FIXME found in rules.go:
  // TODO: Handle this edge case

# Fix: Add issue reference
// TODO: Handle edge case (see #456)
```

#### check-error-handling - Proper Error Handling
```bash
# What it does:
- Detects errors being ignored (_ = func())
- Warns about missing error checks
- Suggests proper error handling

# Warning:
⚠️  Ignoring error in rules.go:
  _ = someFunc()

# Fix:
if err := someFunc(); err != nil {
    return err
}
```

#### check-error-msgs - Quality Error Messages
```bash
# What it does:
- Ensures errors start with lowercase
- Requires descriptive messages
- Prevents vague "error" messages

# Failure:
❌ Error message should start with lowercase:
  errors.New("File not found")

❌ Error message too vague:
  errors.New("error")

# Fix:
fmt.Errorf("file not found: %s", path)
fmt.Errorf("invalid port %d: must be 1024-65535", port)
```

#### check-magic-numbers - Explain Numeric Literals
```bash
# What it does:
- Detects unexplained numeric literals
- Requires constants for magic numbers
- Improves code readability

# Warning:
⚠️  Magic number found in rules.go:
  if port < 65535

# Fix:
const maxPort = 65535
if port < maxPort
```

---

### 5. **Commit Message Hooks** (Block on Violation)

These hooks enforce meaningful commit messages.

#### validate-commit-msg - Quality Commit Messages
```bash
# What it does:
- Minimum 10 characters (prevents "fix" or "wip")
- Requires capital first letter
- Prefers imperative mood
- Suggests issue references

# Failure:
❌ Commit message too short: "fix"
Minimum 10 characters required

# Fix:
"Fix race condition in feature flag evaluation"

# Also warns:
⚠️  Consider using imperative mood
   "Added tests" -> "Add tests"
   "Fixing bug" -> "Fix bug"

# And suggests:
ℹ️  Consider referencing issue
   "Fix bug (see #123)"
```

---

### 6. **Example/Configuration Hooks**

#### verify-examples - Validate Example Files
```bash
# What it does:
- Checks example YAML files are valid
- Runs through the tool's validator
- Prevents invalid examples

# Failure:
❌ Example file not valid YAML: examples/bad.yaml
Syntax error at line 5

# Fix: Ensure YAML is valid
```

---

## Hook Behavior

### When Hooks Run

| Stage | When | Hooks |
|-------|------|-------|
| **Pre-commit** | Before creating commit | Formatting, linting, tests |
| **Commit-msg** | After commit message entered | Message validation |
| **Post-commit** | After successful commit | (Can be added) |

### Bypassing Hooks

```bash
# Skip all hooks (use cautiously!)
git commit --no-verify

# Skip specific hooks (with pre-commit)
pre-commit run --hook-id gofmt --all-files

# Manually run hooks on specific files
pre-commit run --files rules.go
```

### Running Hooks Manually

```bash
# Run all hooks on all files
pre-commit run --all-files

# Run specific hook
pre-commit run gofmt --all-files
pre-commit run require-tests --all-files

# Run hooks on staged files only
pre-commit run
```

---

## Common Scenarios

### Scenario 1: Modified Code Without Tests

```bash
$ git add rules.go && git commit -m "Add new comparison operator"

❌ Go code changed but no test changes detected

Solution:
$ git add rules_test.go
$ git commit -m "Add new comparison operator"
✅ Commit successful
```

### Scenario 2: Forgot Documentation

```bash
$ git commit -m "Add -new-flag option"

⚠️  Go code changed but documentation files not updated:
   - EXAMPLES.md
   - README.md

Continue without updating? (y/n) y
✅ Commit allowed (with warning)
```

### Scenario 3: Debug Code Left In

```bash
$ git add rules.go && git commit -m "Debug value checking"

❌ Debug statements found in code
   fmt.Println("DEBUG: value is", x)

Solution:
- Remove or replace with proper logging
- Re-stage and commit
```

### Scenario 4: Test Failure

```bash
$ git commit -m "Fix issue"

--- FAIL: TestSomethingCritical
    Fix test failure or revert code change

❌ Commit prevented
```

### Scenario 5: Coverage Dropped

```bash
$ git commit -m "Add feature"

❌ Test coverage below threshold
Current: 65%
Minimum: 70%

Solution:
- Add tests for new code
- Run: pre-commit run coverage-threshold --all-files
- Commit again
```

---

## Configuration

### Modify Hook Strictness

Edit `.pre-commit-config.yaml`:

```yaml
# Make coverage threshold stricter
- id: coverage-threshold
  entry: bash -c './.git/hooks/check-coverage.sh'
  stages: [commit]
  
# Add before calling script to change threshold to 80%
```

### Disable Specific Hooks

```yaml
# Comment out in .pre-commit-config.yaml
# - id: no-bare-todos
#   name: No unresolved TODOs
```

### Add New Hooks

```yaml
# Add to .pre-commit-config.yaml
- repo: local
  hooks:
    - id: my-custom-hook
      name: My custom check
      entry: bash -c './scripts/my-check.sh'
      language: system
      types: [go]
      pass_filenames: false
```

---

## Troubleshooting

### Hooks Not Running

```bash
# Verify installation
git hook list

# Reinstall
pre-commit install --install-hooks

# Check .git/hooks directory
ls -la .git/hooks/ | grep pre-commit
```

### Pre-commit Not Found

```bash
# Install it
pip install --upgrade pre-commit

# Verify
pre-commit --version
```

### Specific Hook Failing

```bash
# Run just that hook with verbose output
pre-commit run specific-hook-id --all-files --verbose

# Debug script
bash -x .git/hooks/check-coverage.sh
```

### False Positives

Edit `.pre-commit-config.yaml` to exclude files:

```yaml
- id: some-hook
  exclude: ^vendor/|^\.git/
```

---

## Benefits of Pre-Commit Hooks

✅ **Quality**
- Prevents untested code
- Catches formatting issues early
- Ensures error handling
- Maintains code consistency

✅ **Documentation**
- Keeps docs in sync with code
- Prevents undocumented features
- Maintains example validity

✅ **Debuggability**
- No debug statements leak to repo
- TODOs have references
- Error messages are clear
- Magic numbers explained

✅ **Maintainability**
- Code reviews focus on logic, not style
- Consistent commit messages
- Tests always accompany changes
- Coverage doesn't decrease

✅ **Team Alignment**
- Everyone follows same rules
- No style arguments in PRs
- Shared quality standards
- Better onboarding

---

## Next Steps

1. **Install hooks**: `./scripts/setup-hooks.sh`
2. **Try the hooks**: `pre-commit run --all-files`
3. **Understand output**: Read the hook descriptions above
4. **Configure**: Adjust thresholds in `.pre-commit-config.yaml`
5. **Commit**: Your first commit will be auto-validated!

---

## See Also

- [pre-commit documentation](https://pre-commit.com/)
- [Go code style guide](https://golang.org/doc/effective_go)
- [Conventional commits](https://www.conventionalcommits.org/)
- Project's `.pre-commit-config.yaml` for full configuration
