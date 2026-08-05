# Test Examples

This directory contains comprehensive test scenarios for config-requirements-check.

Each example demonstrates real-world use cases and edge cases.

## Example Categories

### Basic Examples
- Simple equality checks
- Feature flag validation
- Basic external dependencies

### Intermediate Examples  
- Comparison operators (gte, gt, lte, lt)
- Array membership (contains)
- Multiple conditions (AND logic)
- Conditional requirements

### Advanced Examples
- Semantic version comparisons
- Complex nested conditions
- Unless conditions (forbidden states)
- Multi-environment validation
- Batch checking scenarios

### Edge Cases
- Null/missing value handling
- Type coercion scenarios
- Deeply nested paths
- Empty arrays/strings
- Boundary conditions

## Running Examples

Each example has:
- `*-requirements.yaml` - Requirement definitions
- `*-values.yaml` - Configuration values to check
- `README.md` - Expected behavior and use case

```bash
# Test a specific example
config-requirements-check \
  -requirements test-examples/example-name/requirements.yaml \
  -values test-examples/example-name/values.yaml
```

## Example Naming Convention

- `ex-basic-*.yaml` - Basic examples (equality, simple conditions)
- `ex-operators-*.yaml` - Comparison operators and arithmetic
- `ex-advanced-*.yaml` - Complex scenarios
- `ex-edge-*.yaml` - Edge cases and boundary conditions
- `ex-real-world-*.yaml` - Realistic production scenarios
