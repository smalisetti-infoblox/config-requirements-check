package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestNumericValues_IntegerCondition tests requirements with integer values
func TestNumericValues_IntegerCondition(t *testing.T) {
	req := Requirement{
		ID: "numeric-version-check",
		Conditions: []Condition{
			{Path: "api.version", Equals: 3},
		},
		Requires: []Condition{
			{Path: "auth.provider", Equals: "oauth2"},
		},
	}
	values := map[string]interface{}{
		"api": map[string]interface{}{"version": 3},
		"auth": map[string]interface{}{"provider": "oauth2"},
	}
	res := evaluateRequirement(values, req)
	if !res.Applicable || !res.Satisfied {
		t.Fatalf("numeric condition should work with integers, got %+v", res)
	}
}

// TestNumericValues_FloatCondition tests floating-point comparisons
func TestNumericValues_FloatCondition(t *testing.T) {
	req := Requirement{
		ID: "float-threshold",
		Conditions: []Condition{
			{Path: "cache.ratio", Equals: 0.8},
		},
		Requires: []Condition{
			{Path: "cache.enabled", Equals: true},
		},
	}
	values := map[string]interface{}{
		"cache": map[string]interface{}{
			"ratio":   0.8,
			"enabled": true,
		},
	}
	res := evaluateRequirement(values, req)
	if !res.Satisfied {
		t.Fatalf("float condition comparison failed")
	}
}

// TestNumericValues_FloatStringCast tests that 3.0 == "3.0" in string format
func TestNumericValues_FloatStringCast(t *testing.T) {
	req := Requirement{
		ID: "float-string-cast",
		Conditions: []Condition{
			// YAML unmarshals 3.0 as float64(3.0), but equals uses string comparison
			{Path: "threshold", Equals: 3.0},
		},
		Requires: []Condition{
			{Path: "enabled", Equals: true},
		},
	}
	values := map[string]interface{}{
		"threshold": 3.0,
		"enabled": true,
	}
	res := evaluateRequirement(values, req)
	if !res.Satisfied {
		t.Fatalf("float condition should match when string-formatted equal")
	}
}

// TestStringValues tests various string conditions
func TestStringValues_ExactMatch(t *testing.T) {
	req := Requirement{
		ID: "env-type-check",
		Conditions: []Condition{
			{Path: "environment.type", Equals: "production"},
		},
		Requires: []Condition{
			{Path: "tls.enforced", Equals: true},
		},
	}
	values := map[string]interface{}{
		"environment": map[string]interface{}{"type": "production"},
		"tls": map[string]interface{}{"enforced": true},
	}
	res := evaluateRequirement(values, req)
	if !res.Applicable || !res.Satisfied {
		t.Fatalf("string condition failed, got %+v", res)
	}
}

// TestStringValues_CaseSensitive tests that string comparisons are case-sensitive
func TestStringValues_CaseSensitive(t *testing.T) {
	req := Requirement{
		ID: "case-test",
		Conditions: []Condition{
			{Path: "env", Equals: "Production"},
		},
		Requires: []Condition{
			{Path: "required", Equals: true},
		},
	}
	values := map[string]interface{}{
		"env": "production",  // lowercase
		"required": true,
	}
	res := evaluateRequirement(values, req)
	if res.Applicable {
		t.Fatalf("string comparison should be case-sensitive, but condition matched")
	}
}

// TestDeeplyNestedPaths tests path resolution through many levels
func TestDeeplyNestedPaths_FiveLevels(t *testing.T) {
	req := Requirement{
		ID: "deep-nesting",
		Conditions: []Condition{
			{Path: "a.b.c.d.e", Equals: "value"},
		},
		Requires: []Condition{
			{Path: "x.y.z", Equals: true},
		},
	}
	values := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				"c": map[string]interface{}{
					"d": map[string]interface{}{
						"e": "value",
					},
				},
			},
		},
		"x": map[string]interface{}{
			"y": map[string]interface{}{
				"z": true,
			},
		},
	}
	res := evaluateRequirement(values, req)
	if !res.Satisfied {
		t.Fatalf("deeply nested paths should resolve correctly")
	}
}

// TestDeeplyNestedPaths_MissingMiddleSegment tests path failure in deep nesting
func TestDeeplyNestedPaths_MissingMiddleSegment(t *testing.T) {
	values := map[string]interface{}{
		"a": map[string]interface{}{
			"b": map[string]interface{}{
				// "c" is missing
				"d": "value",
			},
		},
	}
	_, ok := lookupPath(values, "a.b.c.d")
	if ok {
		t.Fatalf("lookup should fail when intermediate segment is missing")
	}
}

// TestEmptyAndNullValues tests behavior with null/empty values
func TestEmptyAndNullValues_NullValue(t *testing.T) {
	values := map[string]interface{}{
		"flag": nil,
	}
	v, ok := lookupPath(values, "flag")
	if !ok {
		t.Fatalf("null value should be found (ok=true), got %v", ok)
	}
	if v != nil {
		t.Fatalf("null value should be nil, got %v", v)
	}
}

// TestEmptyAndNullValues_NullVsUnset tests difference between null and unset
func TestEmptyAndNullValues_NullVsUnset(t *testing.T) {
	req := Requirement{
		ID: "null-test",
		Conditions: []Condition{
			{Path: "setting", Equals: nil},
		},
		Requires: []Condition{
			{Path: "required", Equals: true},
		},
	}

	// Case 1: explicitly set to null
	values1 := map[string]interface{}{
		"setting": nil,
		"required": true,
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Applicable {
		t.Fatalf("explicitly null value should match nil condition")
	}

	// Case 2: unset (missing key)
	values2 := map[string]interface{}{
		"required": true,
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Applicable {
		t.Fatalf("unset value should not match nil condition")
	}
}

// TestEmptyAndNullValues_EmptyString tests empty string vs unset
func TestEmptyAndNullValues_EmptyString(t *testing.T) {
	req := Requirement{
		ID: "empty-string-test",
		Conditions: []Condition{
			{Path: "name", Equals: ""},
		},
		Requires: []Condition{
			{Path: "required", Equals: true},
		},
	}

	values := map[string]interface{}{
		"name": "",
		"required": true,
	}
	res := evaluateRequirement(values, req)
	if !res.Applicable {
		t.Fatalf("empty string should match empty string condition")
	}
}

// TestMultipleConditionsAndRequires_ComplexScenario tests a real-world-like requirement
func TestMultipleConditionsAndRequires_ComplexScenario(t *testing.T) {
	req := Requirement{
		ID: "database-migration",
		Conditions: []Condition{
			{Path: "database.engine", Equals: "postgres"},
			{Path: "database.version", Equals: 13},
			{Path: "features.caching", Equals: true},
		},
		Requires: []Condition{
			{Path: "cache.backend", Equals: "redis"},
			{Path: "cache.persistence", Equals: true},
			{Path: "monitoring.enabled", Equals: true},
		},
	}

	values := map[string]interface{}{
		"database": map[string]interface{}{
			"engine": "postgres",
			"version": 13,
		},
		"features": map[string]interface{}{
			"caching": true,
		},
		"cache": map[string]interface{}{
			"backend": "redis",
			"persistence": true,
		},
		"monitoring": map[string]interface{}{
			"enabled": true,
		},
	}

	res := evaluateRequirement(values, req)
	if !res.Applicable || !res.Satisfied {
		t.Fatalf("complex multi-condition requirement failed: %+v", res)
	}
}

// TestMultipleConditionsAndRequires_PartiallyUnmet tests partial violations
func TestMultipleConditionsAndRequires_PartiallyUnmet(t *testing.T) {
	req := Requirement{
		ID: "partial-unmet",
		Conditions: []Condition{
			{Path: "feature.enabled", Equals: true},
		},
		Requires: []Condition{
			{Path: "dep1.enabled", Equals: true},
			{Path: "dep2.enabled", Equals: true},
			{Path: "dep3.enabled", Equals: true},
		},
	}

	values := map[string]interface{}{
		"feature": map[string]interface{}{"enabled": true},
		"dep1": map[string]interface{}{"enabled": true},
		// dep2 unset
		"dep3": map[string]interface{}{"enabled": false},
	}

	res := evaluateRequirement(values, req)
	if !res.Applicable || res.Satisfied {
		t.Fatalf("should detect unmet requirements")
	}
	if len(res.UnmetPaths) != 2 {
		t.Fatalf("expected 2 unmet paths (dep2, dep3), got %v", res.UnmetPaths)
	}
}

// TestFeatureStates_MixedTypes tests feature state detection with various value types
func TestFeatureStates_MixedTypes(t *testing.T) {
	reqs := []Requirement{
		{
			ID: "test",
			Conditions: []Condition{
				{Path: "bool.flag", Equals: true},
				{Path: "num.value", Equals: 42},
				{Path: "str.value", Equals: "active"},
			},
			Requires: []Condition{},
		},
	}

	values := map[string]interface{}{
		"bool": map[string]interface{}{"flag": true},
		"num": map[string]interface{}{"value": 42},
		"str": map[string]interface{}{"value": "active"},
	}

	states := featureStates(values, reqs)
	if len(states) != 3 {
		t.Fatalf("expected 3 feature states, got %d", len(states))
	}

	statusMap := make(map[string]string)
	for _, s := range states {
		statusMap[s.Path] = s.Status
	}

	if statusMap["bool.flag"] != "enabled" {
		t.Fatalf("boolean true should have status 'enabled'")
	}
	if statusMap["num.value"] != "set" {
		t.Fatalf("numeric value should have status 'set'")
	}
	if statusMap["str.value"] != "set" {
		t.Fatalf("string value should have status 'set'")
	}
}

// TestFeatureStates_BooleanFalse tests that false is reported as "disabled" not "unset"
func TestFeatureStates_BooleanFalse(t *testing.T) {
	reqs := []Requirement{
		{
			ID: "test",
			Conditions: []Condition{
				{Path: "feature.enabled", Equals: false},
			},
			Requires: []Condition{},
		},
	}

	values := map[string]interface{}{
		"feature": map[string]interface{}{"enabled": false},
	}

	states := featureStates(values, reqs)
	if len(states) != 1 {
		t.Fatalf("expected 1 state, got %d", len(states))
	}
	if states[0].Status != "disabled" {
		t.Fatalf("false should have status 'disabled', got %q", states[0].Status)
	}
	if states[0].Value != false {
		t.Fatalf("value should be false")
	}
}

// TestJSONOutput_ValidStructure tests that JSON output can be unmarshaled
func TestJSONOutput_ValidStructure(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: test-req
    summary: Test summary
    conditions:
      - path: feature.enabled
        equals: true
    requires:
      - path: setting.required
        equals: true
    external_dependencies:
      - id: external-dep
        description: Test dependency
        owner: test-team
        verify:
          type: manual
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
feature:
  enabled: true
setting:
  required: true
`)

	code, out := runCLI(t, []string{"-format", "json", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	var report Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON output should be valid: %v", err)
	}

	if len(report.Features) != 2 {
		t.Fatalf("expected 2 features, got %d", len(report.Features))
	}
	if len(report.Requirements) != 1 {
		t.Fatalf("expected 1 requirement, got %d", len(report.Requirements))
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("expected 1 dependency, got %d", len(report.Dependencies))
	}
}

// TestConditionHolds_WithBooleanStrings tests that string "true" DOES match boolean true
// because valuesEqual uses string formatting (both format to "true")
func TestConditionHolds_WithBooleanStrings(t *testing.T) {
	req := Requirement{
		ID: "bool-string-test",
		Conditions: []Condition{
			{Path: "setting", Equals: true},
		},
		Requires: []Condition{},
	}

	// Actual boolean
	values1 := map[string]interface{}{"setting": true}
	if !conditionHolds(values1, req.Conditions[0]) {
		t.Fatalf("actual boolean true should match")
	}

	// String "true" DOES match boolean true via string formatting
	values2 := map[string]interface{}{"setting": "true"}
	if !conditionHolds(values2, req.Conditions[0]) {
		t.Fatalf("string 'true' should match boolean true due to string formatting")
	}
}

// TestConditionHolds_NumericStringMatch tests that string "1" matches int 1
// because valuesEqual uses string formatting (both format to "1")
func TestConditionHolds_NumericStringMatch(t *testing.T) {
	values := map[string]interface{}{"val": "1"}
	cond := Condition{Path: "val", Equals: 1}
	// Both format to "1" via Sprintf, so they match
	if !conditionHolds(values, cond) {
		t.Fatalf("string '1' and int 1 should match due to string formatting")
	}
}

// TestValuesEqual_StringFormatComparison confirms the string formatting behavior.
// Note: valuesEqual uses Sprintf which means true and "true" both become "true".
func TestValuesEqual_StringFormatComparison(t *testing.T) {
	tests := []struct {
		actual interface{}
		expected interface{}
		shouldMatch bool
	}{
		{true, true, true},
		{false, false, true},
		{1, 1, true},
		{1.0, 1.0, true},
		{"hello", "hello", true},
		{true, "true", true},  // Both format to "true"
		{1, "1", true},        // Both format to "1"
		{nil, nil, true},
		{0, false, false},     // "0" != "false"
		{1, true, false},      // "1" != "true"
	}

	for i, tt := range tests {
		result := valuesEqual(tt.actual, tt.expected)
		if result != tt.shouldMatch {
			t.Fatalf("test %d: valuesEqual(%v, %v) = %v, want %v",
				i, tt.actual, tt.expected, result, tt.shouldMatch)
		}
	}
}

// TestLintRequirements_SummaryAndRemediationRequired tests required text fields
func TestLintRequirements_MissingSummary(t *testing.T) {
	rf := &RequirementsFile{
		Requirements: []Requirement{
			{
				ID:         "test",
				Summary:    "",  // missing
				Conditions: []Condition{{Path: "a", Equals: true}},
				Requires:   []Condition{{Path: "b", Equals: true}},
			},
		},
	}
	issues := lintRequirements(rf)
	// Note: current lint doesn't check Summary/Remediation for emptiness
	// This test documents current behavior; could be improved
	if len(issues) != 0 {
		t.Fatalf("current lint doesn't check Summary/Remediation, got %v", issues)
	}
}

// TestApplicableDependencies_NoConditionsHold tests that deps aren't reported when conditions fail
func TestApplicableDependencies_NoConditionsHold(t *testing.T) {
	req := Requirement{
		ID: "dep-test",
		Conditions: []Condition{
			{Path: "enabled", Equals: true},
		},
		Requires: []Condition{},
		ExternalDependencies: []ExternalDependency{
			{
				ID: "kafka-topic",
				Description: "Topic must exist",
				Owner: "kafka-team",
				Verify: Verify{Type: "manual"},
			},
		},
	}

	values := map[string]interface{}{
		"enabled": false,
	}

	deps := applicableDependencies(values, []Requirement{req})
	if len(deps) != 0 {
		t.Fatalf("dependencies should not be listed when conditions don't hold")
	}
}

// TestApplicableDependencies_WithPartialRequirements tests deps shown even if requires unmet
func TestApplicableDependencies_WithPartialRequirements(t *testing.T) {
	req := Requirement{
		ID: "dep-test",
		Conditions: []Condition{
			{Path: "enabled", Equals: true},
		},
		Requires: []Condition{
			{Path: "required", Equals: true},
		},
		ExternalDependencies: []ExternalDependency{
			{
				ID: "external-thing",
				Description: "Must be configured",
				Owner: "other-team",
				Verify: Verify{Type: "manual"},
			},
		},
	}

	values := map[string]interface{}{
		"enabled": true,
		// "required" is unset, so requirement will be unsatisfied
	}

	deps := applicableDependencies(values, []Requirement{req})
	if len(deps) != 1 {
		t.Fatalf("dependencies should be listed when conditions hold, even if requires unset")
	}
}

// TestTextOutput_FormattingConsistency tests that text output is stable
func TestTextOutput_FormattingConsistency(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: test
    summary: Test requirement
    conditions:
      - path: setting
        equals: true
    requires:
      - path: other
        equals: true
    remediation: "Fix it"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
setting: true
`)

	code1, out1 := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	code2, out2 := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})

	if code1 != code2 || out1 != out2 {
		t.Fatalf("output should be deterministic")
	}
}

// TestPathLookup_WithListValues tests that arrays/lists in values are handled gracefully
func TestPathLookup_WithListValues(t *testing.T) {
	values := map[string]interface{}{
		"items": []interface{}{"a", "b", "c"},
	}
	// Trying to traverse through a list should fail gracefully
	_, ok := lookupPath(values, "items.0")
	if ok {
		t.Fatalf("list traversal should not work (lists aren't maps)")
	}
}

// TestPathLookup_WithMixedNesting tests complex mixed structures
func TestPathLookup_WithMixedNesting(t *testing.T) {
	values := map[string]interface{}{
		"config": map[string]interface{}{
			"services": map[string]interface{}{
				"api": map[string]interface{}{
					"port": 8080,
					"tls": map[string]interface{}{
						"enabled": true,
					},
				},
			},
		},
	}

	tests := []struct {
		path string
		shouldExist bool
	}{
		{"config.services.api.port", true},
		{"config.services.api.tls.enabled", true},
		{"config.services.api.tls", true},
		{"config.services.web", false},
		{"config.services.api.nonexistent", false},
	}

	for _, tt := range tests {
		_, ok := lookupPath(values, tt.path)
		if ok != tt.shouldExist {
			t.Fatalf("path %q: expected ok=%v, got ok=%v", tt.path, tt.shouldExist, ok)
		}
	}
}

// TestCLI_RequirementsNotFound tests error handling for missing requirements file
func TestCLI_RequirementsNotFound(t *testing.T) {
	dir := t.TempDir()
	valuesPath := writeTempFile(t, dir, "values.yaml", "setting: true")

	code, out := runCLI(t, []string{
		"-values", valuesPath,
		"-requirements", "/nonexistent/path/config-requirements.yaml",
	})

	if code != 2 {
		t.Fatalf("missing file should exit 2, got %d", code)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("error message should be present")
	}
}

// TestCLI_ValuesNotFound tests error handling for missing values file
func TestCLI_ValuesNotFound(t *testing.T) {
	code, out := runCLI(t, []string{
		"-values", "/nonexistent/path/values.yaml",
	})

	if code != 2 {
		t.Fatalf("missing values should exit 2, got %d", code)
	}
	if !strings.Contains(out, "error") {
		t.Fatalf("error message should be present")
	}
}

// TestCLI_EmptyValuesFile tests behavior with empty/blank values
func TestCLI_EmptyValuesFile(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: test
    conditions:
      - path: enabled
        equals: true
    requires:
      - path: required
        equals: true
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", "")  // Empty file

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("empty values file should not cause error, exit code %d", code)
	}
	if !strings.Contains(out, "not-applicable") {
		t.Fatalf("requirement should not apply to empty values")
	}
}

// TestMultipleRequirements_IndependentTracking tests multiple requirements don't interfere
func TestMultipleRequirements_IndependentTracking(t *testing.T) {
	rf := &RequirementsFile{
		Requirements: []Requirement{
			{
				ID: "first",
				Conditions: []Condition{{Path: "a", Equals: true}},
				Requires: []Condition{{Path: "x", Equals: true}},
			},
			{
				ID: "second",
				Conditions: []Condition{{Path: "b", Equals: true}},
				Requires: []Condition{{Path: "y", Equals: true}},
			},
		},
	}

	values := map[string]interface{}{
		"a": true,
		"x": true,
		"b": true,
		// y is unset
	}

	first := evaluateRequirement(values, rf.Requirements[0])
	second := evaluateRequirement(values, rf.Requirements[1])

	if !first.Satisfied {
		t.Fatalf("first requirement should be satisfied")
	}
	if second.Satisfied {
		t.Fatalf("second requirement should not be satisfied")
	}
}
