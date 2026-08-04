package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Condition is a single path/equals check. Used both for `conditions` (when
// a requirement applies) and `requires` (what must hold once it does).
type Condition struct {
	Path   string      `yaml:"path" json:"path"`
	Equals interface{} `yaml:"equals" json:"equals"`
}

// Verify names how an external dependency could be checked. Only "manual"
// is implemented today; the field exists so a future automated checker can
// be added without changing the schema.
type Verify struct {
	Type string `yaml:"type" json:"type"`
}

// ExternalDependency is a prerequisite this tool cannot verify itself
// (owned by another repo/system). Always surfaced as a checklist, never
// affects exit status. ResolvedBy is an optional list of evidence links
// (e.g. a merged PR that created the Kafka topic or ingress rule) — set it
// once the prerequisite has actually been fulfilled, so the checklist shows
// it as resolved instead of pending, without ever deleting the record of
// what had to happen.
type ExternalDependency struct {
	ID          string   `yaml:"id" json:"id"`
	Description string   `yaml:"description" json:"description"`
	Owner       string   `yaml:"owner" json:"owner"`
	Verify      Verify   `yaml:"verify" json:"verify"`
	ResolvedBy  []string `yaml:"resolved_by,omitempty" json:"resolved_by,omitempty"`
}

// Requirement describes one conditional config rule: if all Conditions
// hold against a values file, all Requires must also hold.
type Requirement struct {
	ID                   string               `yaml:"id" json:"id"`
	Summary              string               `yaml:"summary" json:"summary"`
	Conditions           []Condition          `yaml:"conditions" json:"conditions"`
	Requires             []Condition          `yaml:"requires" json:"requires"`
	Remediation          string               `yaml:"remediation" json:"remediation"`
	ExternalDependencies []ExternalDependency `yaml:"external_dependencies" json:"external_dependencies,omitempty"`
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
	if err := yaml.Unmarshal(data, &rf); err != nil {
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
// requiring exact Go type matches.
func valuesEqual(actual, expected interface{}) bool {
	return fmt.Sprintf("%v", actual) == fmt.Sprintf("%v", expected)
}

func conditionHolds(values map[string]interface{}, c Condition) bool {
	v, ok := lookupPath(values, c.Path)
	if !ok {
		return false
	}
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
	ID          string   `json:"id"`
	Summary     string   `json:"summary"`
	Applicable  bool     `json:"applicable"`
	Satisfied   bool     `json:"satisfied"`
	UnmetPaths  []string `json:"unmet_paths,omitempty"`
	Remediation string   `json:"remediation,omitempty"`
}

// evaluateRequirement checks all Conditions (AND). If they don't all hold,
// the requirement doesn't apply to this values file (Applicable=false,
// Satisfied is meaningless/true so it never fails a check). If they do
// hold, every Requires entry is checked independently and unmet ones are
// reported by path.
func evaluateRequirement(values map[string]interface{}, r Requirement) RequirementResult {
	res := RequirementResult{ID: r.ID, Summary: r.Summary, Remediation: r.Remediation}

	for _, c := range r.Conditions {
		if !conditionHolds(values, c) {
			res.Applicable = false
			res.Satisfied = true
			return res
		}
	}
	res.Applicable = true

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
	RequirementID string   `json:"requirement_id"`
	ID            string   `json:"id"`
	Description   string   `json:"description"`
	Owner         string   `json:"owner"`
	VerifyType    string   `json:"verify_type"`
	Resolved      bool     `json:"resolved"`
	ResolvedBy    []string `json:"resolved_by,omitempty"`
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
				RequirementID: r.ID,
				ID:            d.ID,
				Description:   d.Description,
				Owner:         d.Owner,
				VerifyType:    d.Verify.Type,
				Resolved:      len(d.ResolvedBy) > 0,
				ResolvedBy:    d.ResolvedBy,
			})
		}
	}
	return deps
}
