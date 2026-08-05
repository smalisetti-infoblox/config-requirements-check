# Implementation Plan: All 10 Improvements

## Phase 1: Foundational Comparison Logic (Commits 1-3)
- [ ] **#1: Add comparison operators** (gte, lte, gt, lt)
  - Update Condition struct
  - Implement conditionHolds() logic for comparisons
  - Update lintRequirements() to validate operators
  - Add tests
  
- [ ] **#2: Array/list support** (contains)
  - Extend Condition struct with Contains field
  - Update lookupPath() for array traversal
  - Implement array element checking
  - Add tests
  
- [ ] **#9: Type precision & documentation**
  - Add detailed comments to valuesEqual()
  - Document string formatting behavior in README
  - Add comparison mode options

## Phase 2: Output & Error Improvements (Commits 4-5)
- [ ] **#3: Actual values in errors**
  - Add ActualValues to RequirementResult
  - Capture actual values during evaluation
  - Display in text output
  - Include in JSON output
  - Add tests
  
- [ ] **#5: Audit trail metadata**
  - Add Metadata struct
  - Capture timestamp, version, hash
  - Include in JSON Report
  - Document in README
  - Add tests

## Phase 3: Schema Extensions (Commits 6-7)
- [ ] **#4: Mutually exclusive conditions** (unless)
  - Add Unless []Condition to Requirement
  - Implement unless logic in evaluateRequirement()
  - Update lintRequirements() validation
  - Update starter template
  - Add tests
  
- [ ] **#8: Structured remediation hints**
  - Add RemediationHint struct
  - Add RemediationHints []RemediationHint to Requirement
  - Update text output to show hints
  - Include in JSON
  - Update starter template
  - Add tests

## Phase 4: CLI & Operational Features (Commits 8-9)
- [ ] **#6: Environment-scoped dependency skips**
  - Add SkipInEnvironments []string to ExternalDependency
  - Add -environment flag to CLI
  - Filter dependencies based on environment
  - Update text output
  - Include in JSON
  - Add tests
  
- [ ] **#7: Batch checking** (-values-dir)
  - Add -values-dir flag
  - Implement recursive directory walking
  - Aggregate results from multiple files
  - Add per-file status reporting
  - New output format for batch results
  - Add tests

## Phase 5: Performance Optimization (Commit 10)
- [ ] **#10: Performance index**
  - Build RequirementsIndex on load
  - Index requirements by condition paths
  - Use index to skip non-applicable requirements
  - Benchmark before/after
  - Add performance tests

## Implementation Order
1. Start with Phase 1 (foundational) - needed for other logic
2. Move to Phase 2 (affects all output)
3. Phase 3 (schema extensions, backward compatible)
4. Phase 4 (CLI features)
5. Phase 5 (performance, no user-visible changes)

Total estimated effort: ~15-20 commits, 3-4 hours
