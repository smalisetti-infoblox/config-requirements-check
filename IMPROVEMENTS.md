# Project Review: config-requirements-check

## Executive Summary

`config-requirements-check` is a well-designed CLI tool that catches silent configuration gaps when breaking changes are introduced. It compares a requirements registry against a values YAML file and reports feature states, requirement satisfaction, and external dependencies. The codebase is clean, defensive, and testable. This document outlines strengths, improvement opportunities, and comprehensive test examples.

---

## Project Strengths

1. **Excellent Separation of Concerns**: CLI logic (main.go), data structures (rules.go types), and business logic (rules.go functions) are cleanly separated.

2. **Strict Schema Enforcement**: The `KnownFields(true)` YAML decoder at rules.go:90 catches typos in the requirements file itself—solving the exact problem the tool exists to prevent. This is meta-awareness.

3. **Comprehensive Test Coverage**: 20+ tests covering happy paths, errors, edge cases, and CLI integration. Both text and JSON formats validated.

4. **Clear, Actionable Output**: Hierarchical display (features → requirements → dependencies) with visual indicators ([satisfied], [MISCONFIGURED], [not-applicable]). JSON output is well-structured for automation.

5. **Extensible Dependency Verification**: The `Verify` struct (rules.go:23-25) designed for future automated checkers without schema changes. `KnownImplementations` documents how other environments solved similar problems.

6. **Defensive YAML Handling**: `lookupPath()` (rules.go:128-142) never panics on malformed structures—gracefully returns ok=false.

7. **Smart Feature State Labeling**: `featureStates()` (rules.go:172-209) labels booleans as enabled/disabled but other values as "set", clearly differentiating conditions.

---

## Key Improvement Opportunities

### 1. Missing Comparison Operators
**Current Limitation**: Only exact equality (`equals: true`) is supported.

**Why It Matters**: Real migrations often need threshold checks (port > 1024, timeout <= 30, version >= 13).

**Suggestion**: Extend Condition struct to support optional `gte`, `lte`, `gt`, `lt` fields:
```go
type Condition struct {
    Path   string      `yaml:"path"`
    Equals interface{} `yaml:"equals,omitempty"`
    Gte    interface{} `yaml:"gte,omitempty"`      // New
    Lte    interface{} `yaml:"lte,omitempty"`      // New
    Gt     interface{} `yaml:"gt,omitempty"`       // New
    Lt     interface{} `yaml:"lt,omitempty"`       // New
}
```
Add lint warnings for incompatible combinations (e.g., both `equals` and `gte`).

---

### 2. Array/List Support
**Current Limitation**: Paths can only resolve to scalars or maps. Arrays cause graceful failure.

**Why It Matters**: Configs often use arrays (allowed origins, service list, ingress rules).

**Suggestion**: Extend path syntax to support array membership:
```yaml
requirements:
  - conditions:
      - path: allowedOrigins
        contains: "https://example.com"
    requires:
      - path: corsEnabled
        equals: true
```

Implement `contains` comparison mode and update `lookupPath()` to handle array traversal.

---

### 3. Incomplete Error Context for Users
**Current Limitation**: When a requirement fails, only unmet paths are shown, not actual values.

**Why It Matters**: Debugging is harder without seeing what values ARE set.

**Suggestion**: Add `ActualValues map[string]interface{}` to `RequirementResult`:
```json
{
  "id": "consolidated-health-enabled-toggle",
  "applicable": true,
  "satisfied": false,
  "unmet_paths": ["consolidatedHealth.enabled"],
  "actual_values": {
    "redis.enabled": true,
    "consolidatedHealth.enabled": null
  }
}
```

---

### 4. Mutually Exclusive Conditions
**Current Limitation**: Only AND logic; can't express "X must NOT be set when Y is set".

**Why It Matters**: Breaking changes sometimes forbid old flags alongside new ones.

**Suggestion**: Add optional `unless` block to requirements:
```yaml
requirements:
  - id: legacy-mode-removal
    conditions:
      - path: redis.enabled
        equals: true
    unless:  # Fail if these hold
      - path: redis_legacy_mode
        equals: true
    requires:
      - path: redis.cluster_enabled
        equals: true
```

---

### 5. Audit Trail for Decisions
**Current Limitation**: Tool prints references but doesn't track evaluation metadata.

**Why It Matters**: CI/CD pipelines need audit evidence (when, by what version, which requirements were checked).

**Suggestion**: Add optional metadata envelope to JSON output:
```json
{
  "metadata": {
    "timestamp": "2026-08-05T10:00:00Z",
    "tool_version": "v1.2.3",
    "requirements_file_hash": "sha256:abc123...",
    "environment": "us-prod-1"
  },
  "features": [...],
  "requirements": [...],
  "dependencies": [...]
}
```

---

### 6. Environment-Scoped Dependency Skips
**Current Limitation**: `known_implementations` lists environments but doesn't let users skip dependencies for their specific environment.

**Why It Matters**: In production, some dependencies are always satisfied (no need to list them every time).

**Suggestion**: Add optional `skip_in_environments` to external dependencies:
```yaml
external_dependencies:
  - id: kafka-topic
    description: "Topic must exist"
    owner: "platform-team"
    skip_in_environments: [prod, staging]  # Already satisfied platform-wide
    verify:
      type: manual
```

---

### 7. Batch Checking for Monorepos
**Current Limitation**: One values file at a time; monorepos must call the tool once per environment.

**Why It Matters**: Multi-environment setups can't aggregate results or parallelize checks.

**Suggestion**: Add `-values-dir` flag for recursive checking:
```bash
config-requirements-check -values-dir envs/ -format json
# Outputs per-file status and aggregate results
```

---

### 8. Machine-Parseable Remediation
**Current Limitation**: Remediation is free-form prose (users must parse text).

**Why It Matters**: Automation can't act on remediation without human interpretation.

**Suggestion**: Extend requirements with structured remediation hints:
```yaml
requirements:
  - id: consolidated-health-enabled-toggle
    remediation: "Set consolidatedHealth.enabled: true"
    remediation_hints:  # New
      - type: "set_field"
        path: "consolidatedHealth.enabled"
        value: true
      - type: "reference"
        url: "https://wiki.example.com/config-migration#consolidated-health"
```

---

### 9. Numeric Comparison Precision
**Current Limitation**: `valuesEqual()` uses string formatting, which could silently equate `0` and `0.0`.

**Why It Matters**: Real-world configs distinguish between types (port 8080 vs. "8080").

**Suggestion**: Document the behavior clearly and consider stricter comparison:
```go
// Document current behavior in valuesEqual comment:
// Compares via string formatting to handle YAML's polymorphic scalars.
// Note: true and "true" both format to "true" and will match.

// For future: add optional "strict" mode for type-exact comparison
type ConditionComparison string
const (
    ComparisonFormatted = "formatted"  // Current: string format
    ComparisonStrict    = "strict"     // New: exact type match
)
```

---

### 10. Performance Optimization for Large Registries
**Current Limitation**: Evaluates every requirement linearly (O(n) for each check).

**Why It Matters**: Projects with 100+ requirements may have slow checks in CI.

**Suggestion**: Build an index at load time:
```go
type RequirementsIndex struct {
    ByConditionPath map[string][]int  // path -> requirement indices
    ByID            map[string]int     // id -> index
}

// Use index to skip requirements whose conditions can't possibly hold
// if a path is unset in the values file
```

---

## Comprehensive Test Examples

### Test 1: Numeric Type Handling
**Purpose**: Ensure equality comparison works across int, float, string.

**Example**:
```yaml
conditions:
  - path: server.port
    equals: 8080
requires:
  - path: server.ssl
    equals: true
```

**Values (passing)**:
```yaml
server:
  port: 8080
  ssl: true
```

**Values (also passing)**: String "8080" matches int 8080 due to string formatting.

**Why Important**: YAML can deserialize the same value as int or string; users need clarity on equality semantics.

---

### Test 2: Deeply Nested Paths
**Purpose**: Verify `lookupPath()` handles 5+ levels without errors.

**Example**:
```yaml
conditions:
  - path: a.b.c.d.e.f.flag
    equals: true
```

**Values (passing)**:
```yaml
a:
  b:
    c:
      d:
        e:
          f:
            flag: true
```

**Why Important**: Real configs often have 4-6 levels (services.kafka.producers.main.enabled).

---

### Test 3: Null vs. Unset vs. Empty String
**Purpose**: Distinguish between three different states.

**Example**:
```yaml
conditions:
  - path: config.nullable
    equals: null
```

**Values (case 1 - unset)**:
```yaml
config:
  required: true
  # nullable is missing
```
Result: Not applicable (unset ≠ null).

**Values (case 2 - null)**:
```yaml
config:
  nullable: null
  required: true
```
Result: Applicable (null == null).

**Why Important**: Kubernetes/Helm use `null` for "default"; empty string means "explicitly disabled".

---

### Test 4: Multiple Conditions (AND Logic)
**Purpose**: Verify ALL conditions must hold; failing any makes requirement not applicable.

**Example**:
```yaml
conditions:
  - path: redis.enabled
    equals: true
  - path: cache.type
    equals: "redis"
  - path: cache.timeout_ms
    equals: 5000
requires:
  - path: redis.cluster_mode
    equals: true
```

**Values (one condition fails)**:
```yaml
redis:
  enabled: true
cache:
  type: "redis"
  timeout_ms: 3000  # Fails condition
```
Result: Not applicable (short-circuits on timeout mismatch).

**Why Important**: Users must trust AND semantics; non-applicable means the requirement truly doesn't apply.

---

### Test 5: Multiple Requirements to Satisfy
**Purpose**: When condition holds, all requires are checked independently.

**Example**:
```yaml
conditions:
  - path: feature.enabled
    equals: true
requires:
  - path: auth.enabled
    equals: true
  - path: auth.timeout_seconds
    equals: 300
  - path: auth.mfa
    equals: true
```

**Values (partially unmet)**:
```yaml
feature:
  enabled: true
auth:
  enabled: true
  timeout_seconds: 300
  # mfa is unset
```
Result: Applicable=true, Satisfied=false, UnmetPaths=["auth.mfa"].

**Why Important**: Precise diagnostics identify exactly which config is missing.

---

### Test 6: External Dependencies Checklist
**Purpose**: Ensure dependencies are listed when conditions hold, regardless of requirement satisfaction.

**Example**:
```yaml
conditions:
  - path: kafka.enabled
    equals: true
requires:
  - path: kafka.topic_id
    equals: "prod-topic"
external_dependencies:
  - id: kafka-topic
    description: "Topic must exist"
    owner: "platform-kafka"
    verify:
      type: manual
    known_implementations:
      - environment: us-west-2
        url: "https://github.com/org/kafka-topics/pull/456"
```

**Values (condition holds, requirement fails)**:
```yaml
kafka:
  enabled: true
  # topic_id is missing
```
Result: Requirements shows MISCONFIGURED; Dependencies lists kafka-topic checklist (because condition held).

**Why Important**: Checklist is about "you're using this feature", not "you configured it correctly".

---

### Test 7: JSON Output Validity
**Purpose**: Verify JSON output is parseable and contains all expected fields.

**Command**:
```bash
config-requirements-check -format json -values values.yaml
```

**Expected output structure**:
```json
{
  "features": [
    {
      "path": "redis.enabled",
      "status": "enabled",
      "value": true
    }
  ],
  "requirements": [
    {
      "id": "req-id",
      "summary": "...",
      "applicable": true,
      "satisfied": false,
      "unmet_paths": ["consolidatedHealth.enabled"],
      "remediation": "...",
      "references": [...]
    }
  ],
  "dependencies": [...]
}
```

**Why Important**: JSON is used in CI/CD; must be parseable and contain all information.

---

### Test 8: Lint Detects Structural Problems
**Purpose**: Verify `-lint` catches typos, empty fields, duplicates, unknown verify.type.

**Example (malformed)**:
```yaml
requirements:
  - id: req1
    conditons:  # typo: should be "conditions"
      - path: feature.enabled
        equals: true
    requires:
      - path: feature.set
        equals: true
  - id: req1  # duplicate ID
    conditions:
      - path: other.flag
        equals: true
    requires: []  # empty requires
```

**Expected behavior**:
```bash
$ config-requirements-check -lint
Found 4 issue(s):
  - requirements[0]: unknown field "conditons"
  - requirements[0] (id=req1): conditions is empty
  - requirements[1] (id=req1): duplicate requirement id "req1"
  - requirements[1] (id=req1): requires is empty
```

**Why Important**: The tool's own config is as susceptible to silent gaps; strict validation prevents broken registries.

---

### Test 9: Large Feature Set Performance
**Purpose**: Verify the tool handles 50+ features without slowdown.

**Expected**: Completes in <100ms.

**Why Important**: Enterprise configs have hundreds of flags; tool must not be a CI bottleneck.

---

### Test 10: Feature Filtering
**Purpose**: Ensure `-feature <id>` restricts output to a single requirement.

**Command**:
```bash
config-requirements-check -feature req1 -values values.yaml
```

**Expected**: Only req1 is evaluated and reported; req2 and others are ignored.

**Why Important**: Users troubleshooting one requirement shouldn't be distracted by others.

---

### Test 11: Case Sensitivity
**Purpose**: Verify string comparisons are case-sensitive.

**Example**:
```yaml
conditions:
  - path: env
    equals: "Production"
```

**Values (lowercase)**:
```yaml
env: "production"
```

**Result**: Condition does NOT hold (case-sensitive comparison).

**Why Important**: Real configs distinguish between "Production" and "production"; must not accidentally normalize.

---

### Test 12: References (Audit Trail)
**Purpose**: Verify references are captured and displayed.

**Example**:
```yaml
requirements:
  - id: with-refs
    references:
      - label: "Redis migration (2024-Q2)"
        url: "https://github.com/org/ops/pull/5678"
```

**Expected output**:
```
[satisfied]      with-refs
                 ref: Redis migration (2024-Q2) (https://github.com/org/ops/pull/5678)
```

**Why Important**: References provide context for why a breaking change exists.

---

### Test 13: Known Implementations Display
**Purpose**: Verify known_implementations are shown with environment labels.

**Expected output**:
```
[external-dep] kafka-topic: Topic must exist... (owner: platform-team)
  known implementation in us-west-2: https://github.com/org/topics/pull/100
  known implementation in eu-central-1: https://github.com/org/topics/pull/156
```

**Why Important**: Users setting up new environments can copy patterns from similar ones.

---

### Test 14: Error Handling (Malformed YAML)
**Purpose**: Ensure bad YAML produces clear error messages.

**Example**:
```yaml
requirements:
  - id: test
    summary: "Bad YAML"
    conditions:
      - path: a
        equals: true
    requires  # Missing colon
      - path: b
        equals: true
```

**Expected**: Exit code 2, clear error message.

**Why Important**: Users will make YAML syntax errors; tool must fail fast with actionable messages.

---

### Test 15: Mixed Data Types
**Purpose**: Verify feature state labels are correct for booleans, numbers, strings.

**Example**:
```yaml
requirements:
  - id: mixed
    conditions:
      - path: bool.true_flag
        equals: true
      - path: bool.false_flag
        equals: false
      - path: string.name
        equals: "myname"
      - path: int.count
        equals: 42
```

**Expected output**:
```
bool.true_flag=true (enabled)
bool.false_flag=false (disabled)
string.name="myname" (set)
int.count=42 (set)
```

**Why Important**: Status labels must be intuitive; enabled/disabled only for booleans.

---

## Code Quality Observations

### Best Practices Present
1. **main.go:39-92**: `run()` function is well-structured; flag parsing separated from logic.
2. **rules.go:78-107**: `loadRequirements()` demonstrates defensive programming with strict YAML decoding and deliberate whitespace trimming.
3. **rules.go:128-142**: `lookupPath()` never panics on malformed structures.
4. **Test organization**: Separate files for CLI tests (main_test.go) and logic tests (rules_test.go).

### Areas for Enhancement
1. **Input validation**: Paths in Condition should be validated (non-empty, valid dot notation). Add to `lintRequirements()`.
2. **Error messages**: Could be more specific (e.g., distinguish YAML syntax errors from strict field violations).
3. **Documentation**: Add inline comments explaining type coercion in `valuesEqual()`.
4. **Performance**: For 100+ requirements, consider building a condition-path index at load time.

---

## Testing Strategy

The codebase has good test coverage. Recommended additions:

1. **Edge cases test file** (edge_cases_test.go): 30+ tests for numeric types, deeply nested paths, null values, etc.
2. **Performance benchmarks**: Ensure large registries remain sub-100ms.
3. **Integration scenarios**: Multiple requirements with overlapping conditions.
4. **Round-trip JSON**: Verify JSON → struct → JSON preserves all data.

---

## Conclusion

`config-requirements-check` is a well-executed tool that solves a real problem. The codebase is clean, defensive, and testable. The improvements outlined above would:

1. **Expand applicability** (arrays, comparisons, negation).
2. **Improve debuggability** (actual values in output, structured remediation).
3. **Enable automation** (batch checking, audit trails, environment skipping).
4. **Scale better** (performance optimization for large registries).

Each improvement is optional and orthogonal; the tool is immediately useful in its current form.

