# Config-Requirements-Check: Comprehensive Examples

This document shows how to use all available flags with real examples.

## Quick Reference: All Flags

| Flag | Type | Purpose |
|------|------|---------|
| `-values <path>` | Input | Path to values YAML file to check |
| `-values-dir <path>` | Input | Directory with values.yaml files (batch check) |
| `-requirements <path>` | Input | Path to requirements registry (default: config-requirements.yaml) |
| `-init` | Mode | Print starter config-requirements.yaml template |
| `-lint` | Mode | Validate requirements registry structure |
| `-check` | Output | Validate conditions/requires and report violations |
| `-features` | Output | Print resolved feature-gate states |
| `-deps` | Output | Print external-dependency checklist |
| `-feature <id>` | Filter | Restrict output to single requirement id |
| `-environment <env>` | Filter | Environment name (skips dependencies for this env) |
| `-format <fmt>` | Format | Output format: text (default) or json |

---

## 1. Getting Started

### Initialize a new config-requirements.yaml

```bash
# Print starter template to stdout
config-requirements-check -init

# Or save directly to file
config-requirements-check -init > config-requirements.yaml
```

Output: A comprehensive template with all available fields and examples.

---

## 2. Validate Your Requirements File

### Lint the requirements registry

```bash
# Validate structure and schema
config-requirements-check -lint -requirements config-requirements.yaml
```

Output:
```
OK: no issues found
```

Detects:
- Typos in field names (e.g., `conditons` instead of `conditions`)
- Empty required fields
- Duplicate IDs
- Invalid `verify.type` values

---

## 3. Check a Single Environment

### Show all information (default mode)

```bash
# Default: shows features, requirements, and dependencies
config-requirements-check -values envs/prod/values.yaml
```

Output:
```
Features:
  redis.enabled=true (enabled)
  server.port=8080 (set)

Requirements:
  [satisfied]      redis-cluster-required
  [not-applicable] legacy-auth-mode

External dependencies (not verified...):
  [redis-cluster-required] redis-cluster-setup: Setup Redis cluster...
```

Exit code: 0 (all requirements satisfied or not applicable)

---

## 4. Specific Output Modes

### Show only feature states

```bash
config-requirements-check -values values.yaml -features
```

Output:
```
Features:
  cache.enabled=true (enabled)
  cache.ttl_ms=3000 (set)
  database.max_connections=500 (set)
```

Labels:
- `(enabled)` - Boolean true
- `(disabled)` - Boolean false
- `(set)` - Any other value
- `(unset)` - Path not found

---

### Show only requirement validation

```bash
config-requirements-check -values values.yaml -check
```

Output:
```
Requirements:
  [satisfied]      port-range-validation
  [satisfied]      cors-origins-configured
  [MISCONFIGURED]  cache-ttl-requirement
                   Cache TTL must be less than 5000ms
                   unmet: [cache.ttl_ms]
                   actual values: map[cache.ttl_ms:6000]
                   fix: Reduce cache.ttl_ms to below 5000ms
                   hint: set_field cache.ttl_ms = 4000
```

Status values:
- `[satisfied]` - All requirements met
- `[not-applicable]` - Conditions don't apply to this environment
- `[MISCONFIGURED]` - Conditions apply but requirements not met

---

### Show only external dependencies

```bash
config-requirements-check -values values.yaml -deps
```

Output:
```
External dependencies (not verified...):
  [cors-origins-configured] api-auth-service: Auth service must be available
                    known implementation in dev: https://github.com/org/...
  [database-cluster] kafka-topic: Kafka topic must be created
                    no known implementations on record — verify manually
```

---

## 5. Environment-Specific Checking

### Skip dependencies for specific environment

```bash
# Prod: skip dependencies marked skip_in_environments: ["prod"]
config-requirements-check -values envs/prod/values.yaml -environment prod -deps

# Dev: show all dependencies
config-requirements-check -values envs/dev/values.yaml -environment dev -deps
```

Use case: Prod environment has infrastructure pre-setup, dev environments need full checklist.

---

## 6. Filter Output

### Check single requirement

```bash
config-requirements-check -values values.yaml \
  -feature consolidated-health-enabled-toggle
```

Output: Only shows that one requirement's status.

Useful: Large registries with many requirements.

---

## 7. Output Formats

### Text format (default, human-readable)

```bash
config-requirements-check -values values.yaml -format text
```

Output:
```
Features:
  redis.enabled=true (enabled)
Requirements:
  [satisfied] redis-configured
External dependencies (not verified...):
  (none apply)
```

Exit code: 0

---

### JSON format (machine-readable, CI/CD friendly)

```bash
config-requirements-check -values values.yaml -format json
```

Output:
```json
{
  "metadata": {
    "timestamp": "2026-08-05T05:59:58Z",
    "requirements_hash": "db227727a8bb...",
    "values_hash": "5663ce00bbd0..."
  },
  "features": [
    {
      "path": "redis.enabled",
      "status": "enabled",
      "value": true
    }
  ],
  "requirements": [
    {
      "id": "redis-configured",
      "summary": "...",
      "applicable": true,
      "satisfied": true,
      "actual_values": {
        "redis.enabled": true
      }
    }
  ],
  "dependencies": [...]
}
```

Includes:
- `metadata`: Timestamp and file hashes for audit trail
- `actual_values`: Current values when misconfigured
- Structured output for automation

---

## 8. Batch Checking Multiple Environments

### Check all environments at once

```bash
# Recursive search for all values.yaml files
config-requirements-check -values-dir envs/ -requirements requirements.yaml

# With text output
config-requirements-check -values-dir envs/ -format text
```

Output:
```
Batch check: 3 files, 2 passed, 1 failed
  ✓ PASS  /tmp/monorepo/prod/values.yaml
  ✓ PASS  /tmp/monorepo/staging/values.yaml
  ✗ FAIL  /tmp/monorepo/dev/values.yaml
           (2/5 requirements failed)
```

Exit code: 1 (one environment failed)

---

### Batch checking with JSON

```bash
config-requirements-check -values-dir envs/ -format json
```

Output:
```json
{
  "files": [
    {
      "path": "/envs/prod/values.yaml",
      "passed": true,
      "requirements_failed": 0,
      "requirements_total": 5
    },
    {
      "path": "/envs/dev/values.yaml",
      "passed": false,
      "requirements_failed": 2,
      "requirements_total": 5
    }
  ],
  "total": {
    "checked": 3,
    "passed": 2,
    "failed": 1,
    "not_applicable": 0
  }
}
```

Perfect for: CI/CD aggregation, monitoring dashboards, compliance reporting.

---

## 9. Complex Examples

### Full CI/CD pipeline check

```bash
# 1. Validate requirements file
config-requirements-check -lint -requirements requirements.yaml
if [ $? -ne 0 ]; then echo "Requirements file invalid"; exit 1; fi

# 2. Check production environment
config-requirements-check -values envs/prod/values.yaml \
  -environment prod \
  -check -deps \
  -format json > /tmp/prod-report.json

# 3. Check all environments
config-requirements-check -values-dir envs/ \
  -format json > /tmp/batch-report.json

# 4. Parse JSON for automation
FAILED=$(jq '.total.failed' /tmp/batch-report.json)
if [ $FAILED -gt 0 ]; then
  echo "❌ $FAILED environment(s) failed validation"
  exit 1
fi
```

---

### Real-world requirement with all features

**requirements.yaml:**
```yaml
requirements:
  - id: database-cluster-setup
    summary: "Postgres clustering requires 100+ connections"
    conditions:
      - path: database.cluster.enabled
        equals: true
    requires:
      - path: database.max_connections
        gte: 100
    remediation: "Set database.max_connections to at least 100"
    remediation_hints:
      - type: set_field
        path: database.max_connections
        value: 200
        description: "Recommended for 3-node cluster"
    external_dependencies:
      - id: postgres-replication
        description: "Setup streaming replication on all nodes"
        owner: database-platform-team
        skip_in_environments: [prod, staging]
        verify:
          type: manual
        known_implementations:
          - environment: dev
            url: https://github.com/org/setup-guide/pull/123
    references:
      - label: "Database architecture RFC"
        url: https://wiki.example.com/db-cluster
```

**Test commands:**

```bash
# Dev: check with all dependencies listed
config-requirements-check -values envs/dev/values.yaml -environment dev

# Prod: skip already-setup dependencies
config-requirements-check -values envs/prod/values.yaml -environment prod

# Machine-readable for automation
config-requirements-check -values envs/prod/values.yaml -format json
```

---

## 10. Exit Codes for Automation

```bash
config-requirements-check [flags]
```

| Exit Code | Meaning |
|-----------|---------|
| 0 | ✅ All applicable requirements satisfied |
| 1 | ❌ One or more requirements misconfigured |
| 2 | ⚠️ Usage error (bad flags, missing file, parse error) |

Use in scripts:
```bash
if config-requirements-check -values values.yaml; then
  echo "✅ Configuration valid"
  deploy
else
  echo "❌ Configuration invalid, skipping deploy"
  exit 1
fi
```

---

## 11. Flag Combinations

### Most Common Patterns

#### Pattern 1: Quick validation (default mode)
```bash
config-requirements-check -values values.yaml
# Shows everything: features, requirements, dependencies
```

#### Pattern 2: CI check pass/fail
```bash
config-requirements-check -values values.yaml -check
# Returns exit code 0/1, useful for CI gates
```

#### Pattern 3: Monorepo batch validation
```bash
config-requirements-check -values-dir ./envs -format json
# Check all environments, parse JSON for automation
```

#### Pattern 4: Environment-aware validation
```bash
config-requirements-check -values envs/prod/values.yaml \
  -environment prod \
  -check -deps -format json
# Skip prod-specific dependencies, output JSON for dashboards
```

#### Pattern 5: Single requirement debug
```bash
config-requirements-check -values values.yaml \
  -feature some-requirement-id \
  -format json
# Deep dive into one requirement with actual values
```

#### Pattern 6: Lint before commit (pre-commit hook)
```bash
config-requirements-check -lint -requirements config-requirements.yaml
# Catches typos and structural errors before commit
```

---

## 12. Comparison Operators and Advanced Features

All flags work with modern requirement features:

```yaml
requirements:
  # Improvement #1: Comparison operators
  - id: port-validation
    conditions:
      - path: server.port
        gte: 1024
      - path: server.port
        lte: 65535
    requires:
      - path: server.tls
        equals: true

  # Improvement #2: Array support
  - id: cors-origins
    conditions:
      - path: cors.enabled
        equals: true
    requires:
      - path: cors.allowed_origins
        contains: "https://example.com"

  # Improvement #4: Unless (forbidden states)
  - id: no-legacy-auth
    conditions:
      - path: auth.enabled
        equals: true
    unless:
      - path: auth.legacy_only
        equals: true
    requires:
      - path: auth.provider
        equals: "oauth2"

  # Improvement #8: Remediation hints
  - id: cache-config
    conditions:
      - path: cache.enabled
        equals: true
    requires:
      - path: cache.ttl_ms
        lt: 5000
    remediation_hints:
      - type: set_field
        path: cache.ttl_ms
        value: 3000
```

All features work with all output flags and formats.

---

## Summary

| Use Case | Command |
|----------|---------|
| Start new project | `config-requirements-check -init > config-requirements.yaml` |
| Validate requirements file | `config-requirements-check -lint -requirements config-requirements.yaml` |
| Check one environment | `config-requirements-check -values values.yaml` |
| CI/CD validation | `config-requirements-check -values values.yaml -check -format json` |
| Check all environments | `config-requirements-check -values-dir envs/ -format json` |
| Prod with skipped deps | `config-requirements-check -values values.yaml -environment prod -deps` |
| Debug one requirement | `config-requirements-check -values values.yaml -feature requirement-id -format json` |

All flags can be combined for maximum flexibility!
