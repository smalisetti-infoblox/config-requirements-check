package main

import (
	"reflect"
	"testing"
)

func mkReq() Requirement {
	return Requirement{
		ID:      "consolidated-health-enabled-toggle",
		Summary: "consolidatedHealth.enabled now explicitly gates the feature.",
		Conditions: []Condition{
			{Path: "redis.enabled", Equals: true},
		},
		Requires: []Condition{
			{Path: "consolidatedHealth.enabled", Equals: true},
		},
		Remediation: "Set consolidatedHealth.enabled: true.",
		ExternalDependencies: []ExternalDependency{
			{ID: "kafka-topic", Description: "topic must exist", Owner: "platform-kafka", Verify: Verify{Type: "manual"}},
		},
	}
}

func TestEvaluateRequirement_AllConditionsMet_RequirementMissing(t *testing.T) {
	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": true},
	}
	res := evaluateRequirement(values, mkReq())
	if !res.Applicable {
		t.Fatalf("expected applicable=true")
	}
	if res.Satisfied {
		t.Fatalf("expected satisfied=false when consolidatedHealth.enabled is unset")
	}
	if !reflect.DeepEqual(res.UnmetPaths, []string{"consolidatedHealth.enabled"}) {
		t.Fatalf("unexpected unmet paths: %v", res.UnmetPaths)
	}
}

func TestEvaluateRequirement_AllConditionsMet_RequirementSatisfied(t *testing.T) {
	values := map[string]interface{}{
		"redis":              map[string]interface{}{"enabled": true},
		"consolidatedHealth": map[string]interface{}{"enabled": true},
	}
	res := evaluateRequirement(values, mkReq())
	if !res.Applicable || !res.Satisfied {
		t.Fatalf("expected applicable and satisfied, got %+v", res)
	}
	if len(res.UnmetPaths) != 0 {
		t.Fatalf("expected no unmet paths, got %v", res.UnmetPaths)
	}
}

func TestEvaluateRequirement_ConditionNotMet_NotApplicable(t *testing.T) {
	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": false},
	}
	res := evaluateRequirement(values, mkReq())
	if res.Applicable {
		t.Fatalf("expected applicable=false when redis.enabled is false")
	}
	if !res.Satisfied {
		t.Fatalf("a non-applicable requirement must never report unsatisfied")
	}
}

func TestEvaluateRequirement_MultipleRequiresPartiallyUnmet(t *testing.T) {
	req := Requirement{
		ID:         "multi-requires",
		Conditions: []Condition{{Path: "a.enabled", Equals: true}},
		Requires: []Condition{
			{Path: "b.enabled", Equals: true},
			{Path: "c.enabled", Equals: true},
		},
	}
	values := map[string]interface{}{
		"a": map[string]interface{}{"enabled": true},
		"b": map[string]interface{}{"enabled": true},
		// c.enabled left unset
	}
	res := evaluateRequirement(values, req)
	if !res.Applicable {
		t.Fatalf("expected applicable=true")
	}
	if res.Satisfied {
		t.Fatalf("expected satisfied=false")
	}
	if !reflect.DeepEqual(res.UnmetPaths, []string{"c.enabled"}) {
		t.Fatalf("expected only c.enabled unmet, got %v", res.UnmetPaths)
	}
}

func TestEvaluateRequirement_MultipleConditionsRequireAll(t *testing.T) {
	req := Requirement{
		ID: "multi-condition",
		Conditions: []Condition{
			{Path: "a.enabled", Equals: true},
			{Path: "b.enabled", Equals: true},
		},
		Requires: []Condition{{Path: "c.enabled", Equals: true}},
	}
	// Only one of the two conditions holds -> not applicable.
	values := map[string]interface{}{
		"a": map[string]interface{}{"enabled": true},
		"b": map[string]interface{}{"enabled": false},
	}
	res := evaluateRequirement(values, req)
	if res.Applicable {
		t.Fatalf("expected applicable=false when not all conditions hold")
	}
}

func TestLookupPath_MalformedOrMissing(t *testing.T) {
	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": true},
		"flat":  "not-a-map",
	}
	if _, ok := lookupPath(values, "does.not.exist"); ok {
		t.Fatalf("expected missing path to report ok=false")
	}
	if _, ok := lookupPath(values, "flat.nested"); ok {
		t.Fatalf("expected traversal through non-map to report ok=false, not panic")
	}
	v, ok := lookupPath(values, "redis.enabled")
	if !ok || v != true {
		t.Fatalf("expected redis.enabled=true, got %v ok=%v", v, ok)
	}
}

func TestFeatureStates(t *testing.T) {
	reqs := []Requirement{mkReq()}
	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": true},
		// consolidatedHealth.enabled intentionally unset
	}
	states := featureStates(values, reqs)
	got := map[string]string{}
	for _, s := range states {
		got[s.Path] = s.Status
	}
	if got["redis.enabled"] != "enabled" {
		t.Fatalf("expected redis.enabled=enabled, got %v", got["redis.enabled"])
	}
	if got["consolidatedHealth.enabled"] != "unset" {
		t.Fatalf("expected consolidatedHealth.enabled=unset, got %v", got["consolidatedHealth.enabled"])
	}
}

func TestApplicableDependencies(t *testing.T) {
	reqs := []Requirement{mkReq()}

	// Conditions hold -> dependency checklist present, regardless of Requires outcome.
	values := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": true},
	}
	deps := applicableDependencies(values, reqs)
	if len(deps) != 1 || deps[0].ID != "kafka-topic" {
		t.Fatalf("expected 1 dependency for applicable requirement, got %v", deps)
	}

	// Conditions don't hold -> no dependency checklist.
	values2 := map[string]interface{}{
		"redis": map[string]interface{}{"enabled": false},
	}
	deps2 := applicableDependencies(values2, reqs)
	if len(deps2) != 0 {
		t.Fatalf("expected no dependencies when conditions don't hold, got %v", deps2)
	}
}
