# Comprehensive Test Scenarios

This document describes all test scenarios available for validating config-requirements-check functionality.

## Test Examples Overview

### 1. Real-World Microservices Configuration
**Files:**
- `ex-real-world-microservices-requirements.yaml`
- `ex-real-world-microservices-values-valid.yaml`
- `ex-real-world-microservices-values-invalid.yaml`

**Scenario:** Production microservices deployment with realistic configuration requirements.

**What it tests:**
- ✅ Service discovery configuration (Consul)
- ✅ Database connection pooling (min/max sizes)
- ✅ Cache TTL bounds (60-3600 seconds)
- ✅ Logging configuration (level, JSON output, sampling)
- ✅ Metrics collection (Prometheus)
- ✅ Resource limits (memory, CPU)
- ✅ Timeout configuration (request, database, cache)
- ✅ External dependencies with environment filtering
- ✅ Remediation hints with structured guidance

**Typical Use:**
```bash
# Valid configuration (all requirements met)
go run . -values test-examples/ex-real-world-microservices-values-valid.yaml \
         -requirements test-examples/ex-real-world-microservices-requirements.yaml \
         -check

# Invalid configuration (multiple violations)
go run . -values test-examples/ex-real-world-microservices-values-invalid.yaml \
         -requirements test-examples/ex-real-world-microservices-requirements.yaml \
         -check
```

**Key Features Demonstrated:**
- Comparison operators (gte, lte)
- Array membership checks (contains)
- Unless conditions (forbidden states)
- Remediation hints with actionable advice
- Multiple requirement violations

**Expected Behavior:**
- Valid file: All 7 requirements satisfied
- Invalid file: All 7 requirements misconfigured with detailed hints

---

### 2. Advanced Conditional Logic
**Files:**
- `ex-advanced-conditional-logic-requirements.yaml`
- `ex-advanced-conditional-logic-values.yaml`

**Scenario:** Complex feature flag dependencies, mutual exclusivity, and version-dependent requirements.

**What it tests:**
- ✅ Feature flag dependency chains
- ✅ Mutually exclusive configurations
- ✅ Version-dependent requirements
- ✅ Production-specific security constraints
- ✅ Capacity-based resource allocation
- ✅ Multi-region deployment patterns
- ✅ Feature parity across environments
- ✅ Backwards compatibility constraints
- ✅ Data residency requirements
- ✅ Graceful deprecation paths

**Typical Use:**
```bash
# Check advanced configuration
go run . -values test-examples/ex-advanced-conditional-logic-values.yaml \
         -requirements test-examples/ex-advanced-conditional-logic-requirements.yaml \
         -check -features -deps
```

**Key Features Demonstrated:**
- Conditional requirements (if-then logic)
- Unless blocks (forbidden states)
- Version comparisons (gte operator)
- Array membership (contains)
- Multiple conditions working together
- Real-world business constraints

**Expected Behavior:**
- All 10 requirements should be satisfied
- Shows feature state (enabled/disabled/set)
- Lists applicable external dependencies

---

### 3. Edge Cases and Boundary Conditions
**Files:**
- `ex-edge-cases-requirements.yaml`
- `ex-edge-cases-values.yaml`

**Scenario:** Boundary values, null handling, type coercion, deeply nested paths.

**What it tests:**
- ✅ Numeric boundary values (0, max int)
- ✅ Empty array handling
- ✅ Deeply nested paths (5+ levels)
- ✅ Null/nil value semantics
- ✅ Special characters in strings
- ✅ Version parsing edge cases
- ✅ Float precision handling
- ✅ Missing optional configuration
- ✅ Type coercion (string to int, etc.)
- ✅ Single element arrays
- ✅ Whitespace handling in strings
- ✅ Zero and negative numbers

**Typical Use:**
```bash
# Test edge case handling
go run . -values test-examples/ex-edge-cases-values.yaml \
         -requirements test-examples/ex-edge-cases-requirements.yaml \
         -format json | jq .
```

**Key Features Demonstrated:**
- Deeply nested path traversal
- Null value handling
- Type conversion and coercion
- Boundary value validation
- Missing path behavior

**Expected Behavior:**
- All 12 requirements should be satisfied
- JSON output shows actual values found
- No errors on null values or missing paths

---

### 4. Performance and Stress Testing
**Files:**
- `ex-performance-large-requirements-requirements.yaml` (40+ requirements)
- `ex-performance-large-requirements-values.yaml`

**Scenario:** Large configuration with many requirements across multiple domains.

**What it tests:**
- ✅ Performance with 40+ requirements
- ✅ Multiple requirement categories:
  - Database service (5 requirements)
  - Cache service (5 requirements)
  - API Gateway (5 requirements)
  - Message Queue (5 requirements)
  - Observability (5 requirements)
  - Security (5 requirements)
- ✅ Evaluation speed
- ✅ Memory usage
- ✅ Output formatting

**Typical Use:**
```bash
# Large scale validation
time go run . -values test-examples/ex-performance-large-requirements-values.yaml \
             -requirements test-examples/ex-performance-large-requirements-requirements.yaml \
             -check -features

# With timing to measure performance
time go run . -values test-examples/ex-performance-large-requirements-values.yaml \
             -requirements test-examples/ex-performance-large-requirements-requirements.yaml \
             -format json > /tmp/large-check.json
```

**Key Features Demonstrated:**
- Scalability with many requirements
- Organized requirement groups
- Complex YAML files
- Performance under load

**Expected Behavior:**
- All 30+ requirements satisfied
- Should complete in < 1 second
- Minimal memory footprint
- Proper JSON output

---

## Test Execution Guide

### Running All Examples
```bash
#!/bin/bash

cd test-examples

echo "=== Testing Real-World Microservices ==="
go run .. -values ex-real-world-microservices-values-valid.yaml \
          -requirements ex-real-world-microservices-requirements.yaml -check

echo "=== Testing Advanced Logic ==="
go run .. -values ex-advanced-conditional-logic-values.yaml \
          -requirements ex-advanced-conditional-logic-requirements.yaml -check

echo "=== Testing Edge Cases ==="
go run .. -values ex-edge-cases-values.yaml \
          -requirements ex-edge-cases-requirements.yaml -format json

echo "=== Testing Performance ==="
time go run .. -values ex-performance-large-requirements-values.yaml \
               -requirements ex-performance-large-requirements-requirements.yaml -check
```

### Testing Specific Features

**Test Comparison Operators:**
```bash
go run . -values test-examples/ex-real-world-microservices-values-valid.yaml \
         -requirements test-examples/ex-real-world-microservices-requirements.yaml \
         -feature database-connection-pool -format json
```

**Test Array Support:**
```bash
go run . -values test-examples/ex-real-world-microservices-values-valid.yaml \
         -requirements test-examples/ex-real-world-microservices-requirements.yaml \
         -feature metrics-collection -format json
```

**Test Unless Conditions:**
```bash
go run . -values test-examples/ex-advanced-conditional-logic-values.yaml \
         -requirements test-examples/ex-advanced-conditional-logic-requirements.yaml \
         -feature mutually-exclusive-auth -format json
```

**Test Deeply Nested Paths:**
```bash
go run . -values test-examples/ex-edge-cases-values.yaml \
         -requirements test-examples/ex-edge-cases-requirements.yaml \
         -feature deeply-nested-configuration -format json
```

---

## Coverage Matrix

| Feature | Microservices | Advanced | Edge Cases | Performance |
|---------|---------------|----------|-----------|-------------|
| Comparison operators (gte, gt, lte, lt) | ✅ | ✅ | ✅ | ✅ |
| Array membership (contains) | ✅ | ✅ | ✅ | ❌ |
| Unless conditions | ✅ | ✅ | ❌ | ❌ |
| Nested conditions | ✅ | ✅ | ✅ | ✅ |
| External dependencies | ✅ | ❌ | ❌ | ❌ |
| Remediation hints | ✅ | ❌ | ❌ | ❌ |
| Null handling | ❌ | ❌ | ✅ | ❌ |
| Deep nesting (5+ levels) | ❌ | ❌ | ✅ | ❌ |
| Version comparisons | ❌ | ✅ | ❌ | ❌ |
| Float precision | ❌ | ❌ | ✅ | ❌ |
| Type coercion | ❌ | ❌ | ✅ | ✅ |
| Large configs | ❌ | ❌ | ❌ | ✅ |

---

## Expected Results

### Real-World Microservices (Valid)
```
Requirements:
  [satisfied]      service-discovery-enabled
  [satisfied]      database-connection-pool
  [satisfied]      cache-configuration
  [satisfied]      logging-level-production
  [satisfied]      metrics-collection
  [satisfied]      resource-limits-container
  [satisfied]      timeout-configuration
```

Exit code: 0 ✅

### Real-World Microservices (Invalid)
```
Requirements:
  [MISCONFIGURED]  service-discovery-enabled
                   unmet: [service_discovery.enabled]
                   actual values: map[service_discovery.enabled:false ...]
  [MISCONFIGURED]  database-connection-pool
                   unmet: [database.pool.min_connections database.pool.max_connections ...]
  ... (more violations)
```

Exit code: 1 ❌

---

## Adding New Test Examples

When adding new test scenarios:

1. **Create requirements file:**
   ```yaml
   # ex-category-feature-requirements.yaml
   requirements:
     - id: requirement-id
       summary: What this requirement does
       conditions:
         - path: some.path
           equals: value
       requires:
         - path: another.path
           gte: 10
   ```

2. **Create values file:**
   ```yaml
   # ex-category-feature-values.yaml
   some:
     path: value
   another:
     path: 15
   ```

3. **Document the scenario:**
   - What features does it test?
   - Why is it important?
   - What should pass/fail?

4. **Test it works:**
   ```bash
   go run . -values test-examples/ex-category-feature-values.yaml \
            -requirements test-examples/ex-category-feature-requirements.yaml -check
   ```

---

## Maintenance

These test examples should be updated when:
- New features are added to the tool
- New operators or comparison types are implemented
- Edge cases are discovered
- Real-world usage patterns emerge

Keep them as a living reference of the tool's capabilities!
