package main

import (
	"encoding/json"
	"strings"
	"testing"
)

// ==== Improvement #1: Comparison Operators Tests ====

func TestNumericCompare_Gte(t *testing.T) {
	tests := []struct {
		actual   interface{}
		expected float64
		result   bool
	}{
		{5, 5.0, true},
		{5, 3.0, true},
		{3, 5.0, false},
		{"5", 5.0, true},
		{5.5, 5.0, true},
	}

	for _, tt := range tests {
		if numericCompare(tt.actual, tt.expected, "gte") != tt.result {
			t.Fatalf("numericCompare(%v, %v, gte) failed", tt.actual, tt.expected)
		}
	}
}

func TestNumericCompare_Gt(t *testing.T) {
	if !numericCompare(5, 3, "gt") {
		t.Fatalf("5 > 3 should be true")
	}
	if numericCompare(5, 5, "gt") {
		t.Fatalf("5 > 5 should be false")
	}
	if numericCompare(3, 5, "gt") {
		t.Fatalf("3 > 5 should be false")
	}
}

func TestNumericCompare_Lte(t *testing.T) {
	if !numericCompare(3, 5, "lte") {
		t.Fatalf("3 <= 5 should be true")
	}
	if !numericCompare(5, 5, "lte") {
		t.Fatalf("5 <= 5 should be true")
	}
	if numericCompare(5, 3, "lte") {
		t.Fatalf("5 <= 3 should be false")
	}
}

func TestNumericCompare_Lt(t *testing.T) {
	if !numericCompare(3, 5, "lt") {
		t.Fatalf("3 < 5 should be true")
	}
	if numericCompare(5, 5, "lt") {
		t.Fatalf("5 < 5 should be false")
	}
	if numericCompare(5, 3, "lt") {
		t.Fatalf("5 < 3 should be false")
	}
}

func TestConditionHolds_WithGte(t *testing.T) {
	req := Requirement{
		ID: "version-check",
		Conditions: []Condition{
			{Path: "api.version", Gte: 3},
		},
		Requires: []Condition{
			{Path: "auth.enabled", Equals: true},
		},
	}

	// Version exactly 3
	values1 := map[string]interface{}{
		"api": map[string]interface{}{"version": 3},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Applicable {
		t.Fatalf("version 3 >= 3 should apply requirement")
	}

	// Version 5
	values2 := map[string]interface{}{
		"api": map[string]interface{}{"version": 5},
	}
	res2 := evaluateRequirement(values2, req)
	if !res2.Applicable {
		t.Fatalf("version 5 >= 3 should apply requirement")
	}

	// Version 2
	values3 := map[string]interface{}{
		"api": map[string]interface{}{"version": 2},
	}
	res3 := evaluateRequirement(values3, req)
	if res3.Applicable {
		t.Fatalf("version 2 >= 3 should not apply requirement")
	}
}

func TestConditionHolds_WithLt(t *testing.T) {
	req := Requirement{
		ID: "timeout-check",
		Conditions: []Condition{
			{Path: "cache.timeout_ms", Lt: 5000},
		},
		Requires: []Condition{
			{Path: "cache.redis_enabled", Equals: true},
		},
	}

	// Timeout 3000 < 5000
	values1 := map[string]interface{}{
		"cache": map[string]interface{}{"timeout_ms": 3000},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Applicable {
		t.Fatalf("timeout 3000 < 5000 should apply requirement")
	}

	// Timeout 5000 < 5000 (false)
	values2 := map[string]interface{}{
		"cache": map[string]interface{}{"timeout_ms": 5000},
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Applicable {
		t.Fatalf("timeout 5000 < 5000 should not apply requirement")
	}
}

func TestConditionHolds_WithMultipleNumericConditions(t *testing.T) {
	req := Requirement{
		ID: "range-check",
		Conditions: []Condition{
			{Path: "port", Gte: 1024},
			{Path: "port", Lte: 65535},
		},
		Requires: []Condition{
			{Path: "tls.enabled", Equals: true},
		},
	}

	// Valid port 8080
	values1 := map[string]interface{}{
		"port": 8080,
		"tls":  map[string]interface{}{"enabled": true},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Satisfied {
		t.Fatalf("port 8080 should satisfy range requirements")
	}

	// Invalid port 80 (too low)
	values2 := map[string]interface{}{
		"port": 80,
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Applicable {
		t.Fatalf("port 80 < 1024 should not apply requirements")
	}
}

func TestToFloat64(t *testing.T) {
	tests := []struct {
		value  interface{}
		result float64
		ok     bool
	}{
		{5.0, 5.0, true},
		{5, 5.0, true},
		{"5", 5.0, true},
		{"5.5", 5.5, true},
		{true, 1.0, true},
		{false, 0.0, true},
		{"not-a-number", 0, false},
		{map[string]interface{}{}, 0, false},
	}

	for _, tt := range tests {
		result, ok := toFloat64(tt.value)
		if ok != tt.ok || (ok && result != tt.result) {
			t.Fatalf("toFloat64(%v) = %v, %v; want %v, %v", tt.value, result, ok, tt.result, tt.ok)
		}
	}
}

// ==== Improvement #2: Array Support Tests ====

func TestArrayContains_WithStringArray(t *testing.T) {
	arr := []interface{}{"https://example.com", "https://localhost", "https://api.example.com"}

	if !arrayContains(arr, "https://example.com") {
		t.Fatalf("array should contain 'https://example.com'")
	}
	if arrayContains(arr, "https://notinarray.com") {
		t.Fatalf("array should not contain 'https://notinarray.com'")
	}
}

func TestArrayContains_WithNumberArray(t *testing.T) {
	arr := []interface{}{1, 2, 3, 4, 5}

	if !arrayContains(arr, 3) {
		t.Fatalf("array should contain 3")
	}
	if !arrayContains(arr, "3") {
		t.Fatalf("array should contain '3' (matches 3 via string format)")
	}
	if arrayContains(arr, 10) {
		t.Fatalf("array should not contain 10")
	}
}

func TestArrayContains_NonArray(t *testing.T) {
	if arrayContains("not-an-array", "value") {
		t.Fatalf("non-array should not contain anything")
	}
	if arrayContains(nil, "value") {
		t.Fatalf("nil should not contain anything")
	}
}

func TestConditionHolds_WithContains(t *testing.T) {
	req := Requirement{
		ID: "allowed-origins-check",
		Conditions: []Condition{
			{Path: "cors.allowed_origins", Contains: "https://example.com"},
		},
		Requires: []Condition{
			{Path: "cors.enabled", Equals: true},
		},
	}

	// CORS enabled with allowed origin
	values1 := map[string]interface{}{
		"cors": map[string]interface{}{
			"enabled":           true,
			"allowed_origins":   []interface{}{"https://example.com", "https://localhost"},
		},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Applicable || !res1.Satisfied {
		t.Fatalf("should find origin in array")
	}

	// CORS with different origins
	values2 := map[string]interface{}{
		"cors": map[string]interface{}{
			"allowed_origins": []interface{}{"https://other.com"},
		},
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Applicable {
		t.Fatalf("should not find origin in array")
	}
}

// ==== Improvement #1: Lint Validation Tests ====

func TestLintRequirements_MultipleOperators(t *testing.T) {
	rf := &RequirementsFile{
		Requirements: []Requirement{
			{
				ID:         "invalid",
				Conditions: []Condition{{Path: "a", Equals: true, Gte: 5}},
				Requires:   []Condition{{Path: "b", Equals: true}},
			},
		},
	}
	issues := lintRequirements(rf)
	if len(issues) == 0 {
		t.Fatalf("should detect multiple operators in condition")
	}
	found := false
	for _, issue := range issues {
		if strings.Contains(issue, "exactly one operator") {
			found = true
		}
	}
	if !found {
		t.Fatalf("error should mention operator count, got %v", issues)
	}
}

func TestLintRequirements_ValidOperators(t *testing.T) {
	rf := &RequirementsFile{
		Requirements: []Requirement{
			{
				ID: "gte-check",
				Conditions: []Condition{{Path: "version", Gte: 3}},
				Requires:   []Condition{{Path: "feature", Equals: true}},
			},
			{
				ID: "array-check",
				Conditions: []Condition{{Path: "list", Contains: "item"}},
				Requires:   []Condition{{Path: "enabled", Equals: true}},
			},
		},
	}
	issues := lintRequirements(rf)
	if len(issues) != 0 {
		t.Fatalf("should not report issues for valid operators, got %v", issues)
	}
}

// ==== Integration test: CLI with new operators ====

func TestCLI_ComparisonOperators(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: port-range
    summary: "Port must be in valid range"
    conditions:
      - path: server.port
        gte: 1024
      - path: server.port
        lte: 65535
    requires:
      - path: server.tls
        equals: true
    remediation: "Configure TLS for the server"
  - id: version-check
    summary: "API version must be 2 or higher"
    conditions:
      - path: api.version
        gte: 2
    requires:
      - path: api.auth
        equals: true
    remediation: "Enable API authentication"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
server:
  port: 8080
  tls: true
api:
  version: 2
  auth: true
`)

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit 0 with valid config, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "satisfied") {
		t.Fatalf("expected both requirements satisfied, output:\n%s", out)
	}
}

func TestCLI_ArrayContains(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: cors-origin
    summary: "If CORS enabled, must allow example.com"
    conditions:
      - path: cors.enabled
        equals: true
    requires:
      - path: cors.allowed_origins
        contains: "https://example.com"
    remediation: "Add https://example.com to cors.allowed_origins"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
cors:
  enabled: true
  allowed_origins:
    - https://example.com
    - https://localhost
`)

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d; output:\n%s", code, out)
	}
}

// ==== Improvement #3: Actual Values in Errors Tests ====

func TestEvaluateRequirement_CapturesActualValues(t *testing.T) {
	req := Requirement{
		ID: "test",
		Conditions: []Condition{
			{Path: "redis.enabled", Equals: true},
		},
		Requires: []Condition{
			{Path: "redis.cluster_mode", Equals: true},
		},
	}

	values := map[string]interface{}{
		"redis": map[string]interface{}{
			"enabled":      true,
			"cluster_mode": false,
		},
	}

	res := evaluateRequirement(values, req)
	if res.ActualValues == nil {
		t.Fatalf("should capture actual values")
	}

	// Both paths should be captured
	if len(res.ActualValues) != 2 {
		t.Fatalf("expected 2 actual values, got %d: %v", len(res.ActualValues), res.ActualValues)
	}

	// Check captured values
	if res.ActualValues["redis.enabled"] != true {
		t.Fatalf("expected redis.enabled=true, got %v", res.ActualValues["redis.enabled"])
	}
	if res.ActualValues["redis.cluster_mode"] != false {
		t.Fatalf("expected redis.cluster_mode=false, got %v", res.ActualValues["redis.cluster_mode"])
	}
}

func TestEvaluateRequirement_ActualValuesNotCapturedForMissingPaths(t *testing.T) {
	req := Requirement{
		ID: "test",
		Conditions: []Condition{
			{Path: "feature.enabled", Equals: true},
		},
		Requires: []Condition{
			{Path: "feature.setting", Equals: "value"},
		},
	}

	values := map[string]interface{}{
		"feature": map[string]interface{}{
			"enabled": true,
			// setting is unset
		},
	}

	res := evaluateRequirement(values, req)

	// Only the set path should be in ActualValues
	if len(res.ActualValues) != 1 {
		t.Fatalf("expected 1 actual value (setting unset), got %d: %v", len(res.ActualValues), res.ActualValues)
	}
	if res.ActualValues["feature.enabled"] != true {
		t.Fatalf("expected feature.enabled=true in ActualValues")
	}
	if _, ok := res.ActualValues["feature.setting"]; ok {
		t.Fatalf("unset path should not appear in ActualValues")
	}
}

func TestCLI_ActualValuesInOutput(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: config-check
    summary: "Service must be configured"
    conditions:
      - path: service.enabled
        equals: true
    requires:
      - path: service.port
        gte: 1024
    remediation: "Configure service.port >= 1024"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
service:
  enabled: true
  port: 80
`)

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1 (misconfigured), got %d", code)
	}

	// Should show actual values in text output
	if !strings.Contains(out, "actual values") {
		t.Fatalf("expected 'actual values' in output for misconfigured requirement, got:\n%s", out)
	}
	if !strings.Contains(out, "service.enabled") {
		t.Fatalf("expected actual value for service.enabled in output")
	}
}

// ==== Improvement #5: Audit Trail Metadata Tests ====

func TestCLI_MetadataInJSON(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: test
    summary: "Test requirement"
    conditions:
      - path: enabled
        equals: true
    requires:
      - path: ready
        equals: true
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
enabled: true
ready: true
`)

	code, out := runCLI(t, []string{"-format", "json", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	var report interface{}
	err := json.Unmarshal([]byte(out), &report)
	if err != nil {
		t.Fatalf("expected valid JSON, got error %v", err)
	}

	// Check that metadata is present in JSON output
	reportMap := report.(map[string]interface{})
	if _, hasMetadata := reportMap["metadata"]; !hasMetadata {
		t.Fatalf("expected metadata field in JSON output")
	}

	metadata := reportMap["metadata"].(map[string]interface{})
	if _, hasTimestamp := metadata["timestamp"]; !hasTimestamp {
		t.Fatalf("expected timestamp in metadata")
	}
	if _, hasReqHash := metadata["requirements_hash"]; !hasReqHash {
		t.Fatalf("expected requirements_hash in metadata")
	}
	if _, hasValHash := metadata["values_hash"]; !hasValHash {
		t.Fatalf("expected values_hash in metadata")
	}
}

func TestCLI_NoMetadataInTextOutput(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: test
    summary: "Test"
    conditions:
      - path: enabled
        equals: true
    requires:
      - path: ready
        equals: true
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
enabled: true
ready: true
`)

	code, out := runCLI(t, []string{"-format", "text", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit 0, got %d", code)
	}

	// Metadata should not be in text output (only JSON)
	if strings.Contains(out, "timestamp") || strings.Contains(out, "requirements_hash") {
		t.Fatalf("metadata should not appear in text output")
	}
}

// ==== Improvement #4: Mutually Exclusive Conditions (Unless) Tests ====

func TestEvaluateRequirement_UnlessConditionPasses(t *testing.T) {
	req := Requirement{
		ID: "legacy-mode-removal",
		Conditions: []Condition{
			{Path: "redis.enabled", Equals: true},
		},
		Unless: []Condition{
			{Path: "redis.legacy_mode", Equals: true},
		},
		Requires: []Condition{
			{Path: "redis.cluster_mode", Equals: true},
		},
	}

	// Valid: redis enabled, no legacy mode, cluster mode enabled
	values1 := map[string]interface{}{
		"redis": map[string]interface{}{
			"enabled":      true,
			"legacy_mode":  false,
			"cluster_mode": true,
		},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Satisfied {
		t.Fatalf("should be satisfied when unless condition is false")
	}

	// Invalid: redis enabled, but legacy mode is also enabled (forbidden!)
	values2 := map[string]interface{}{
		"redis": map[string]interface{}{
			"enabled":      true,
			"legacy_mode":  true,
			"cluster_mode": true,
		},
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Satisfied {
		t.Fatalf("should fail when unless condition is true (forbidden state)")
	}
	if len(res2.UnmetPaths) == 0 {
		t.Fatalf("should report unmet path for forbidden state")
	}
	// Should have FORBIDDEN marker
	found := false
	for _, path := range res2.UnmetPaths {
		if strings.Contains(path, "FORBIDDEN") {
			found = true
		}
	}
	if !found {
		t.Fatalf("should mark forbidden paths, got %v", res2.UnmetPaths)
	}
}

func TestConditionHolds_UnlessWithMultipleConditions(t *testing.T) {
	req := Requirement{
		ID: "auth-mode",
		Conditions: []Condition{
			{Path: "auth.enabled", Equals: true},
		},
		Unless: []Condition{
			{Path: "auth.mfa_only", Equals: true},
			{Path: "auth.legacy_method", Equals: true},
		},
		Requires: []Condition{
			{Path: "auth.provider", Equals: "oauth2"},
		},
	}

	// Valid: auth enabled, neither forbidden mode enabled
	values1 := map[string]interface{}{
		"auth": map[string]interface{}{
			"enabled":         true,
			"mfa_only":        false,
			"legacy_method":   false,
			"provider":        "oauth2",
		},
	}
	res1 := evaluateRequirement(values1, req)
	if !res1.Satisfied {
		t.Fatalf("should satisfy when no unless conditions hold")
	}

	// Invalid: mfa_only is forbidden
	values2 := map[string]interface{}{
		"auth": map[string]interface{}{
			"enabled":         true,
			"mfa_only":        true,  // FORBIDDEN!
			"legacy_method":   false,
			"provider":        "oauth2",
		},
	}
	res2 := evaluateRequirement(values2, req)
	if res2.Satisfied {
		t.Fatalf("should fail when any unless condition holds")
	}
}

func TestCLI_UnlessConditions(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: no-legacy
    summary: "Legacy auth must not coexist with new auth"
    conditions:
      - path: auth.enabled
        equals: true
    unless:
      - path: auth.legacy_only
        equals: true
    requires:
      - path: auth.provider
        equals: "saml"
    remediation: "Remove auth.legacy_only or upgrade to SAML provider"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
auth:
  enabled: true
  legacy_only: true
  provider: "saml"
`)

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1 (forbidden condition), got %d", code)
	}
	if !strings.Contains(out, "MISCONFIGURED") {
		t.Fatalf("should show requirement as misconfigured")
	}
	if !strings.Contains(out, "FORBIDDEN") {
		t.Fatalf("should show FORBIDDEN marker for legacy_only")
	}
}

// ==== Improvement #8: Structured Remediation Hints Tests ====

func TestEvaluateRequirement_PassesThroughRemediationHints(t *testing.T) {
	hint := RemediationHint{
		Type:        "set_field",
		Path:        "cache.enabled",
		Value:       true,
		Description: "Enable cache to improve performance",
	}
	req := Requirement{
		ID:               "cache-check",
		Conditions:       []Condition{{Path: "redis.enabled", Equals: true}},
		Requires:         []Condition{{Path: "cache.enabled", Equals: true}},
		Remediation:      "Enable caching",
		RemediationHints: []RemediationHint{hint},
	}

	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": true},
	}

	res := evaluateRequirement(values, req)
	if len(res.RemediationHints) != 1 {
		t.Fatalf("expected remediation hints to be passed through, got %v", res.RemediationHints)
	}
	if res.RemediationHints[0].Type != "set_field" {
		t.Fatalf("expected hint type set_field, got %s", res.RemediationHints[0].Type)
	}
}

func TestCLI_RemediationHintsInOutput(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: database-migration
    summary: "Database credentials must be configured"
    conditions:
      - path: database.enabled
        equals: true
    requires:
      - path: database.user
        equals: null
      - path: database.password
        equals: null
    remediation: "Configure database credentials"
    remediation_hints:
      - type: set_field
        path: database.user
        value: "prod-user"
        description: "Use service account from secrets manager"
      - type: set_field
        path: database.password
        value: "USE_SECRETS_MANAGER"
        description: "Never commit passwords to version control"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
database:
  enabled: true
`)

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}

	// Check that hints are shown in text output
	if !strings.Contains(out, "hint:") {
		t.Fatalf("expected remediation hints in output")
	}
	if !strings.Contains(out, "set_field") {
		t.Fatalf("expected hint type in output")
	}
	if !strings.Contains(out, "database.user") {
		t.Fatalf("expected field path in hint output")
	}
}

func TestCLI_RemediationHintsInJSON(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "req.yaml", `
requirements:
  - id: api-config
    summary: "API key must be configured"
    conditions:
      - path: api.enabled
        equals: true
    requires:
      - path: api.key
        equals: null
    remediation: "Set api.key"
    remediation_hints:
      - type: set_field
        path: api.key
        value: "LOAD_FROM_VAULT"
`)
	valuesPath := writeTempFile(t, dir, "values.yaml", `
api:
  enabled: true
`)

	code, out := runCLI(t, []string{"-format", "json", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}

	var report interface{}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("expected valid JSON: %v", err)
	}

	reportMap := report.(map[string]interface{})
	requirements := reportMap["requirements"].([]interface{})
	if len(requirements) == 0 {
		t.Fatalf("expected requirements in JSON")
	}

	req := requirements[0].(map[string]interface{})
	if _, hasHints := req["remediation_hints"]; !hasHints {
		t.Fatalf("expected remediation_hints in JSON output")
	}

	hints := req["remediation_hints"].([]interface{})
	if len(hints) == 0 {
		t.Fatalf("expected at least one hint")
	}

	hint := hints[0].(map[string]interface{})
	if hint["type"] != "set_field" {
		t.Fatalf("expected hint type in JSON")
	}
}
