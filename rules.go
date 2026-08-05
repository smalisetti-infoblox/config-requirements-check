package main

import (
	"bytes"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Condition is a single path check with optional comparison operators. Used both
// for `conditions` (when a requirement applies) and `requires` (what must hold once it does).
// Exactly one comparison operator (Equals, Gte, Gt, Lte, Lt, Contains) must be set.
type Condition struct {
	Path     string      `yaml:"path" json:"path"`
	Equals   interface{} `yaml:"equals,omitempty" json:"equals,omitempty"`
	Gte      interface{} `yaml:"gte,omitempty" json:"gte,omitempty"`       // >= comparison
	Gt       interface{} `yaml:"gt,omitempty" json:"gt,omitempty"`         // > comparison
	Lte      interface{} `yaml:"lte,omitempty" json:"lte,omitempty"`       // <= comparison
	Lt       interface{} `yaml:"lt,omitempty" json:"lt,omitempty"`         // < comparison
	Contains interface{} `yaml:"contains,omitempty" json:"contains,omitempty"` // array membership
}

// Verify names how an external dependency could be checked. Only "manual"
// is implemented today; the field exists so a future automated checker can
// be added without changing the schema.
type Verify struct {
	Type string `yaml:"type" json:"type"`
}

// KnownImplementation points at a concrete case where this dependency was
// actually fulfilled in one named environment (e.g. the PR that created a
// Kafka topic for "us-dev-2"). This is not verification that the dependency
// holds for whatever values file is being checked right now — environments
// don't share infrastructure just because they share a chart. It's a
// pointer for a human setting up a *new* environment: "here's how this was
// done elsewhere, copy the pattern."
type KnownImplementation struct {
	Environment string `yaml:"environment" json:"environment"`
	URL         string `yaml:"url" json:"url"`
}

// ExternalDependency is a prerequisite this tool cannot verify itself
// (owned by another repo/system). Always surfaced as a checklist, never
// affects exit status. KnownImplementations lists environments where this
// was already done, purely so other environments have something to copy —
// it does not mean the current values file's environment has it.
type ExternalDependency struct {
	ID                   string                `yaml:"id" json:"id"`
	Description          string                `yaml:"description" json:"description"`
	Owner                string                `yaml:"owner" json:"owner"`
	Verify               Verify                `yaml:"verify" json:"verify"`
	KnownImplementations []KnownImplementation `yaml:"known_implementations,omitempty" json:"known_implementations,omitempty"`
}

// Reference is a link to supporting history for a requirement — e.g. the PR
// that originally turned on the condition (redis.enabled=true) in a given
// environment. Purely informational: an audit trail of "why does this
// requirement exist / where did the condition come from," not a check.
type Reference struct {
	Label string `yaml:"label" json:"label"`
	URL   string `yaml:"url" json:"url"`
}

// RemediationHint provides machine-parseable guidance on how to fix a misconfigured requirement.
type RemediationHint struct {
	Type        string      `yaml:"type" json:"type"`   // set_field, remove_field, etc.
	Path        string      `yaml:"path" json:"path"`   // Field to modify
	Value       interface{} `yaml:"value,omitempty" json:"value,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
}

// Requirement describes one conditional config rule: if all Conditions
// hold AND none of the Unless conditions hold, then all Requires must also hold.
type Requirement struct {
	ID                   string               `yaml:"id" json:"id"`
	Summary              string               `yaml:"summary" json:"summary"`
	Conditions           []Condition          `yaml:"conditions" json:"conditions"`
	Unless               []Condition          `yaml:"unless,omitempty" json:"unless,omitempty"` // Forbidden conditions
	Requires             []Condition          `yaml:"requires" json:"requires"`
	Remediation          string               `yaml:"remediation" json:"remediation"`
	RemediationHints     []RemediationHint    `yaml:"remediation_hints,omitempty" json:"remediation_hints,omitempty"`
	ExternalDependencies []ExternalDependency `yaml:"external_dependencies" json:"external_dependencies,omitempty"`
	References           []Reference          `yaml:"references,omitempty" json:"references,omitempty"`
}

// RequirementsFile is the top-level shape of a requirements registry.
type RequirementsFile struct {
	Requirements []Requirement `yaml:"requirements"`
}

func loadRequirements(path string) (*RequirementsFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading requirements file %q: %w", path, err)
	}
	var rf RequirementsFile
	// KnownFields(true) makes an unrecognized key (e.g. a typo like
	// "conditons") a hard parse error instead of being silently dropped.
	// Without this, a typo'd field just produces an empty/zero value with
	// no error — exactly the kind of silent gap this tool exists to catch,
	// except in its own config.
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&rf); err != nil {
		return nil, fmt.Errorf("parsing requirements file %q: %w", path, err)
	}
	// YAML folded block scalars (">") retain a trailing newline; trim so
	// printed/JSON output doesn't carry stray whitespace from formatting
	// choices in the source file.
	for i := range rf.Requirements {
		r := &rf.Requirements[i]
		r.Summary = strings.TrimSpace(r.Summary)
		r.Remediation = strings.TrimSpace(r.Remediation)
		for j := range r.ExternalDependencies {
			d := &r.ExternalDependencies[j]
			d.Description = strings.TrimSpace(d.Description)
			d.Owner = strings.TrimSpace(d.Owner)
		}
	}
	return &rf, nil
}

func loadValues(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading values file %q: %w", path, err)
	}
	var values map[string]interface{}
	if err := yaml.Unmarshal(data, &values); err != nil {
		return nil, fmt.Errorf("parsing values file %q: %w", path, err)
	}
	if values == nil {
		values = map[string]interface{}{}
	}
	return values, nil
}

// lookupPath resolves a dot-separated path (e.g. "redis.enabled") against a
// nested map decoded from YAML. Returns ok=false if any segment is absent
// or not a map, never panics.
func lookupPath(values map[string]interface{}, path string) (interface{}, bool) {
	var cur interface{} = values
	for _, part := range strings.Split(path, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil, false
		}
		v, ok := m[part]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

// valuesEqual compares a resolved YAML value against a requirement's
// expected value. Formatting-based comparison keeps this working across the
// scalar types YAML unmarshals to (bool, string, int, float64) without
// requiring exact Go type matches. Both values are converted to strings via
// Sprintf, so true matches "true", 1 matches "1", etc.
func valuesEqual(actual, expected interface{}) bool {
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

// numericCompare performs numeric comparison between actual and expected values.
// Both values are converted to float64 for comparison. Returns false if conversion fails.
func numericCompare(actual, expected interface{}, op string) bool {
	actualNum, ok := toFloat64(actual)
	if !ok {
		return false
	}
	expectedNum, ok := toFloat64(expected)
	if !ok {
		return false
	}
	switch op {
	case "gte":
		return actualNum >= expectedNum
	case "gt":
		return actualNum > expectedNum
	case "lte":
		return actualNum <= expectedNum
	case "lt":
		return actualNum < expectedNum
	default:
		return false
	}
}

// toFloat64 converts a value to float64 for numeric comparison.
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	case bool:
		if val {
			return 1.0, true
		}
		return 0.0, true
	default:
		return 0, false
	}
}

// arrayContains checks if an array contains a specific value.
func arrayContains(arr interface{}, target interface{}) bool {
	switch a := arr.(type) {
	case []interface{}:
		for _, item := range a {
			if valuesEqual(item, target) {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func conditionHolds(values map[string]interface{}, c Condition) bool {
	v, ok := lookupPath(values, c.Path)
	if !ok {
		return false
	}

	// Try comparison operators in priority order
	if c.Gte != nil {
		return numericCompare(v, c.Gte, "gte")
	}
	if c.Gt != nil {
		return numericCompare(v, c.Gt, "gt")
	}
	if c.Lte != nil {
		return numericCompare(v, c.Lte, "lte")
	}
	if c.Lt != nil {
		return numericCompare(v, c.Lt, "lt")
	}
	if c.Contains != nil {
		return arrayContains(v, c.Contains)
	}

	// Default: use Equals (even if it's nil, for null comparison)
	return valuesEqual(v, c.Equals)
}

// FeatureState is the resolved state of one path referenced anywhere in a
// requirements file's conditions/requires.
type FeatureState struct {
	Path   string      `json:"path"`
	Status string      `json:"status"` // "enabled" | "disabled" | "unset" | "set"
	Value  interface{} `json:"value,omitempty"`
}

// featureStates collects every distinct path referenced by the given
// requirements (conditions and requires) and resolves its current state
// against values. No separate list to maintain — the requirements file is
// the source of truth for which paths are "features".
func featureStates(values map[string]interface{}, reqs []Requirement) []FeatureState {
	seen := map[string]bool{}
	var paths []string
	for _, r := range reqs {
		for _, c := range r.Conditions {
			if !seen[c.Path] {
				seen[c.Path] = true
				paths = append(paths, c.Path)
			}
		}
		for _, c := range r.Requires {
			if !seen[c.Path] {
				seen[c.Path] = true
				paths = append(paths, c.Path)
			}
		}
	}
	sort.Strings(paths)

	states := make([]FeatureState, 0, len(paths))
	for _, p := range paths {
		v, ok := lookupPath(values, p)
		if !ok {
			states = append(states, FeatureState{Path: p, Status: "unset"})
			continue
		}
		if b, ok := v.(bool); ok {
			if b {
				states = append(states, FeatureState{Path: p, Status: "enabled", Value: v})
			} else {
				states = append(states, FeatureState{Path: p, Status: "disabled", Value: v})
			}
			continue
		}
		states = append(states, FeatureState{Path: p, Status: "set", Value: v})
	}
	return states
}

// RequirementResult is the outcome of evaluating one requirement against a
// values file.
type RequirementResult struct {
	ID                string                 `json:"id"`
	Summary           string                 `json:"summary"`
	Applicable        bool                   `json:"applicable"`
	Satisfied         bool                   `json:"satisfied"`
	UnmetPaths        []string               `json:"unmet_paths,omitempty"`
	Remediation       string                 `json:"remediation,omitempty"`
	RemediationHints  []RemediationHint      `json:"remediation_hints,omitempty"`
	References        []Reference            `json:"references,omitempty"`
	ActualValues      map[string]interface{} `json:"actual_values,omitempty"` // Values for all paths referenced in conditions/requires
}

// evaluateRequirement checks all Conditions (AND). If they don't all hold,
// the requirement doesn't apply to this values file (Applicable=false,
// Satisfied is meaningless/true so it never fails a check). If they do
// hold, every Requires entry is checked independently and unmet ones are
// reported by path. Also captures actual values for all paths mentioned.
func evaluateRequirement(values map[string]interface{}, r Requirement) RequirementResult {
	res := RequirementResult{
		ID:               r.ID,
		Summary:          r.Summary,
		Remediation:      r.Remediation,
		RemediationHints: r.RemediationHints,
		References:       r.References,
	}

	// Collect all paths referenced in conditions, unless, and requires
	pathsToCapture := make(map[string]bool)
	for _, c := range r.Conditions {
		pathsToCapture[c.Path] = true
	}
	for _, u := range r.Unless {
		pathsToCapture[u.Path] = true
	}
	for _, req := range r.Requires {
		pathsToCapture[req.Path] = true
	}

	// Capture actual values for all referenced paths
	res.ActualValues = make(map[string]interface{})
	for path := range pathsToCapture {
		if v, ok := lookupPath(values, path); ok {
			res.ActualValues[path] = v
		}
		// If path not found, we don't include it in ActualValues to keep output clean
	}

	for _, c := range r.Conditions {
		if !conditionHolds(values, c) {
			res.Applicable = false
			res.Satisfied = true
			return res
		}
	}
	res.Applicable = true

	// Check unless conditions: if any hold, the requirement is violated
	for _, u := range r.Unless {
		if conditionHolds(values, u) {
			// Unless condition holds - this violates the requirement
			res.UnmetPaths = append(res.UnmetPaths, "FORBIDDEN: "+u.Path)
		}
	}

	// Check requires conditions
	for _, req := range r.Requires {
		if !conditionHolds(values, req) {
			res.UnmetPaths = append(res.UnmetPaths, req.Path)
		}
	}
	res.Satisfied = len(res.UnmetPaths) == 0
	return res
}

// DependencyEntry flattens one requirement's external dependency into a
// report row, tagged with which requirement it came from.
type DependencyEntry struct {
	RequirementID        string                `json:"requirement_id"`
	ID                   string                `json:"id"`
	Description          string                `json:"description"`
	Owner                string                `json:"owner"`
	VerifyType           string                `json:"verify_type"`
	KnownImplementations []KnownImplementation `json:"known_implementations,omitempty"`
}

// applicableDependencies returns the external dependencies of every
// requirement whose Conditions currently hold, regardless of whether its
// Requires are satisfied — the checklist is about "you're using this
// feature," not "you configured it correctly."
func applicableDependencies(values map[string]interface{}, reqs []Requirement) []DependencyEntry {
	var deps []DependencyEntry
	for _, r := range reqs {
		applies := true
		for _, c := range r.Conditions {
			if !conditionHolds(values, c) {
				applies = false
				break
			}
		}
		if !applies {
			continue
		}
		for _, d := range r.ExternalDependencies {
			deps = append(deps, DependencyEntry{
				RequirementID:        r.ID,
				ID:                   d.ID,
				Description:          d.Description,
				Owner:                d.Owner,
				VerifyType:           d.Verify.Type,
				KnownImplementations: d.KnownImplementations,
			})
		}
	}
	return deps
}

// isValidConditionOperator checks that at most one comparison operator is set.
// We reject conditions where multiple operators are explicitly set (e.g., both Equals and Gte).
// Note: When no operators are set (all nil), we consider it valid for backward compatibility.
func isValidConditionOperator(c Condition) bool {
	// Count how many non-nil operators are set
	nonNilCount := 0

	if c.Equals != nil {
		nonNilCount++
	}
	if c.Gte != nil {
		nonNilCount++
	}
	if c.Gt != nil {
		nonNilCount++
	}
	if c.Lte != nil {
		nonNilCount++
	}
	if c.Lt != nil {
		nonNilCount++
	}
	if c.Contains != nil {
		nonNilCount++
	}

	// Allow 0 (no operators, backward compatible) or 1 (exactly one operator)
	return nonNilCount <= 1
}

// knownVerifyTypes lists every verify.type this tool actually knows how to
// interpret. Kept as a single source of truth so lintRequirements and any
// future automated checker agree on what's valid.
var knownVerifyTypes = map[string]bool{
	"manual": true,
}

// lintRequirements checks structural invariants that strict YAML decoding
// alone can't catch: empty required fields, duplicate ids, and unrecognized
// verify.type values (a likely typo, since "manual" is the only type this
// tool understands today). Returns one message per problem found; an empty
// slice means the file is structurally sound.
func lintRequirements(rf *RequirementsFile) []string {
	var issues []string
	seenReqID := map[string]bool{}

	for i, r := range rf.Requirements {
		loc := fmt.Sprintf("requirements[%d]", i)
		if r.ID != "" {
			loc = fmt.Sprintf("requirements[%d] (id=%s)", i, r.ID)
		}

		if r.ID == "" {
			issues = append(issues, fmt.Sprintf("%s: missing id", loc))
		} else if seenReqID[r.ID] {
			issues = append(issues, fmt.Sprintf("%s: duplicate requirement id %q", loc, r.ID))
		} else {
			seenReqID[r.ID] = true
		}

		if len(r.Conditions) == 0 {
			issues = append(issues, fmt.Sprintf("%s: conditions is empty — this requirement will never apply to anything", loc))
		}
		for j, c := range r.Conditions {
			if c.Path == "" {
				issues = append(issues, fmt.Sprintf("%s: conditions[%d] has an empty path", loc, j))
			}
			if !isValidConditionOperator(c) {
				issues = append(issues, fmt.Sprintf("%s: conditions[%d] must have exactly one operator (equals, gte, gt, lte, lt, or contains)", loc, j))
			}
		}

		for j, u := range r.Unless {
			if u.Path == "" {
				issues = append(issues, fmt.Sprintf("%s: unless[%d] has an empty path", loc, j))
			}
			if !isValidConditionOperator(u) {
				issues = append(issues, fmt.Sprintf("%s: unless[%d] must have exactly one operator (equals, gte, gt, lte, lt, or contains)", loc, j))
			}
		}

		if len(r.Requires) == 0 {
			issues = append(issues, fmt.Sprintf("%s: requires is empty — this requirement can never be misconfigured", loc))
		}
		for j, req := range r.Requires {
			if req.Path == "" {
				issues = append(issues, fmt.Sprintf("%s: requires[%d] has an empty path", loc, j))
			}
			if !isValidConditionOperator(req) {
				issues = append(issues, fmt.Sprintf("%s: requires[%d] must have exactly one operator (equals, gte, gt, lte, lt, or contains)", loc, j))
			}
		}

		seenDepID := map[string]bool{}
		for j, d := range r.ExternalDependencies {
			depLoc := fmt.Sprintf("%s.external_dependencies[%d]", loc, j)
			if d.ID != "" {
				depLoc = fmt.Sprintf("%s.external_dependencies[%d] (id=%s)", loc, j, d.ID)
			}
			if d.ID == "" {
				issues = append(issues, fmt.Sprintf("%s: missing id", depLoc))
			} else if seenDepID[d.ID] {
				issues = append(issues, fmt.Sprintf("%s: duplicate external_dependencies id %q within this requirement", depLoc, d.ID))
			} else {
				seenDepID[d.ID] = true
			}
			if d.Description == "" {
				issues = append(issues, fmt.Sprintf("%s: missing description", depLoc))
			}
			if d.Verify.Type == "" {
				issues = append(issues, fmt.Sprintf("%s: missing verify.type", depLoc))
			} else if !knownVerifyTypes[d.Verify.Type] {
				issues = append(issues, fmt.Sprintf("%s: unrecognized verify.type %q (known: manual) — likely a typo", depLoc, d.Verify.Type))
			}
			for k, impl := range d.KnownImplementations {
				implLoc := fmt.Sprintf("%s.known_implementations[%d]", depLoc, k)
				if impl.Environment == "" {
					issues = append(issues, fmt.Sprintf("%s: missing environment", implLoc))
				}
				if impl.URL == "" {
					issues = append(issues, fmt.Sprintf("%s: missing url", implLoc))
				}
			}
		}

		for j, ref := range r.References {
			refLoc := fmt.Sprintf("%s.references[%d]", loc, j)
			if ref.Label == "" {
				issues = append(issues, fmt.Sprintf("%s: missing label", refLoc))
			}
			if ref.URL == "" {
				issues = append(issues, fmt.Sprintf("%s: missing url", refLoc))
			}
		}
	}
	return issues
}
