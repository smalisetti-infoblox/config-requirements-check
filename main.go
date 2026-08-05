// config-requirements-check reports, for a given values file, which
// conditional config requirements from a requirements registry are
// satisfied, which named flags/features are enabled or disabled, and which
// external (non-configurable-here) prerequisites need manual verification.
//
// It has no knowledge of any specific product, chart, or schema beyond the
// generic "path/equals" shape defined in rules.go.
package main

import (
	_ "embed"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"time"
)

// starterTemplate is a generic, placeholder-filled config-requirements.yaml
// emitted by -init. Embedding the actual file (rather than a Go string
// literal) means the CI-tested copy in examples/ and the copy -init prints
// are guaranteed to be the same bytes — no drift between docs and reality.
//
//go:embed examples/starter.yaml
var starterTemplate string

// Metadata contains audit trail information about the check execution.
type Metadata struct {
	Timestamp            string `json:"timestamp"`           // ISO8601 timestamp of execution
	RequirementsFileHash string `json:"requirements_hash"`   // SHA256 hash of requirements file
	ValuesFileHash       string `json:"values_hash"`         // SHA256 hash of values file
}

// Report is the full, flag-filtered result of one run, shared by the text
// and JSON printers.
type Report struct {
	Metadata     *Metadata           `json:"metadata,omitempty"`
	Features     []FeatureState      `json:"features,omitempty"`
	Requirements []RequirementResult `json:"requirements,omitempty"`
	Dependencies []DependencyEntry   `json:"dependencies,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// fileHash computes SHA256 hash of a file's contents
func fileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("config-requirements-check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	valuesPath := fs.String("values", "", "path to the values YAML file to inspect (required unless -lint or -init)")
	requirementsPath := fs.String("requirements", "config-requirements.yaml", "path to the requirements registry YAML file")
	showFeaturesFlag := fs.Bool("features", false, "print resolved feature-gate states")
	showCheckFlag := fs.Bool("check", false, "validate conditions/requires and report violations")
	showDepsFlag := fs.Bool("deps", false, "print external-dependency checklist")
	featureID := fs.String("feature", "", "restrict output to a single requirement id")
	format := fs.String("format", "text", "output format: text|json")
	environment := fs.String("environment", "", "environment name (e.g., 'prod', 'staging'); skips dependencies marked skip_in_environments")
	lint := fs.Bool("lint", false, "validate the requirements registry's own schema/structure and exit; no -values needed")
	initFlag := fs.Bool("init", false, "print a starter config-requirements.yaml to stdout and exit; no -values or -requirements needed")

	fs.Usage = func() {
		out := fs.Output()
		fmt.Fprint(out, `config-requirements-check catches silent config-migration gaps: cases where
a breaking change makes an old config combination invalid, but nothing
crashes or errors -- the affected feature just quietly stops working.

Given a requirements registry (YAML) and a values file (YAML), it reports
which named config paths are enabled/disabled, which conditional
requirements are satisfied/misconfigured, and which external (cross-repo)
prerequisites need manual verification.

Usage:
  config-requirements-check -values <path> [flags]
  config-requirements-check -lint [-requirements <path>] [-format text|json]
  config-requirements-check -init

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprint(out, `
Examples:
  config-requirements-check -init > config-requirements.yaml
  config-requirements-check -lint -requirements config-requirements.yaml
  config-requirements-check -values envs/us-dev-2/values.yaml
  config-requirements-check -values envs/prod/values.yaml -environment prod
  config-requirements-check -check -deps -format json -values envs/us-dev-2/values.yaml

Exit codes:
  0  no violations found (or -features/-deps/-lint/-init ran successfully)
  1  -check (or the no-flag default) found a misconfigured requirement,
     or -lint found a structural problem
  2  usage error: bad flags, missing -values, or an unreadable/unparsable file
`)
	}

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}

	if *initFlag {
		fmt.Fprint(stdout, starterTemplate)
		return 0
	}

	if *format != "text" && *format != "json" {
		fmt.Fprintf(stderr, "error: -format must be \"text\" or \"json\", got %q\n", *format)
		return 2
	}

	if *lint {
		return runLint(*requirementsPath, *format, stdout, stderr)
	}

	if *valuesPath == "" {
		fmt.Fprintln(stderr, "error: -values is required")
		return 2
	}

	values, err := loadValues(*valuesPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	rf, err := loadRequirements(*requirementsPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}

	reqs := rf.Requirements
	if *featureID != "" {
		var filtered []Requirement
		for _, r := range reqs {
			if r.ID == *featureID {
				filtered = append(filtered, r)
			}
		}
		reqs = filtered
		if len(reqs) == 0 {
			fmt.Fprintf(stderr, "error: no requirement with id %q found in %s\n", *featureID, *requirementsPath)
			return 2
		}
	}

	// With no output-selecting flag, show everything (the useful default).
	defaultMode := !*showFeaturesFlag && !*showCheckFlag && !*showDepsFlag
	showFeatures := *showFeaturesFlag || defaultMode
	showCheck := *showCheckFlag || defaultMode
	showDeps := *showDepsFlag || defaultMode

	// Build metadata
	var metadata *Metadata
	if *format == "json" {
		metadata = &Metadata{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		// Compute file hashes for audit trail
		if hash, err := fileHash(*requirementsPath); err == nil {
			metadata.RequirementsFileHash = hash
		}
		if hash, err := fileHash(*valuesPath); err == nil {
			metadata.ValuesFileHash = hash
		}
	}

	var report Report
	report.Metadata = metadata
	if showFeatures {
		report.Features = featureStates(values, reqs)
	}
	failed := false
	if showCheck {
		for _, r := range reqs {
			res := evaluateRequirement(values, r)
			report.Requirements = append(report.Requirements, res)
			if res.Applicable && !res.Satisfied {
				failed = true
			}
		}
	}
	if showDeps {
		report.Dependencies = applicableDependencies(values, reqs, *environment)
	}

	if *format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(report); err != nil {
			fmt.Fprintln(stderr, "error encoding report:", err)
			return 2
		}
	} else {
		printText(stdout, report, showFeatures, showCheck, showDeps)
	}

	if showCheck && failed {
		return 1
	}
	return 0
}

// runLint validates a requirements registry's own structure — the file's
// schema, not any values file. Catches the failure modes strict YAML
// decoding can't: empty required fields, duplicate ids, unrecognized
// verify.type values. Exits non-zero iff any issue is found.
func runLint(requirementsPath, format string, stdout, stderr *os.File) int {
	rf, err := loadRequirements(requirementsPath)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 2
	}
	issues := lintRequirements(rf)

	if format == "json" {
		enc := json.NewEncoder(stdout)
		enc.SetIndent("", "  ")
		if err := enc.Encode(struct {
			Issues []string `json:"issues"`
		}{Issues: issues}); err != nil {
			fmt.Fprintln(stderr, "error encoding report:", err)
			return 2
		}
	} else if len(issues) == 0 {
		fmt.Fprintln(stdout, "OK: no issues found")
	} else {
		fmt.Fprintf(stdout, "Found %d issue(s) in %s:\n", len(issues), requirementsPath)
		for _, issue := range issues {
			fmt.Fprintf(stdout, "  - %s\n", issue)
		}
	}

	if len(issues) > 0 {
		return 1
	}
	return 0
}

func printText(w *os.File, r Report, showFeatures, showCheck, showDeps bool) {
	if showFeatures {
		fmt.Fprintln(w, "Features:")
		if len(r.Features) == 0 {
			fmt.Fprintln(w, "  (none referenced by the requirements file)")
		}
		for _, f := range r.Features {
			if f.Status == "unset" {
				fmt.Fprintf(w, "  %s: unset\n", f.Path)
			} else {
				fmt.Fprintf(w, "  %s=%v (%s)\n", f.Path, f.Value, f.Status)
			}
		}
		fmt.Fprintln(w)
	}

	if showCheck {
		fmt.Fprintln(w, "Requirements:")
		if len(r.Requirements) == 0 {
			fmt.Fprintln(w, "  (none)")
		}
		for _, req := range r.Requirements {
			switch {
			case !req.Applicable:
				fmt.Fprintf(w, "  [not-applicable] %s\n", req.ID)
			case req.Satisfied:
				fmt.Fprintf(w, "  [satisfied]      %s\n", req.ID)
			default:
				fmt.Fprintf(w, "  [MISCONFIGURED]  %s\n", req.ID)
				fmt.Fprintf(w, "                   %s\n", req.Summary)
				fmt.Fprintf(w, "                   unmet: %v\n", req.UnmetPaths)
				// Show actual values for debugging
				if len(req.ActualValues) > 0 {
					fmt.Fprintf(w, "                   actual values: %v\n", req.ActualValues)
				}
				fmt.Fprintf(w, "                   fix: %s\n", req.Remediation)
				// Show structured remediation hints if available
				for _, hint := range req.RemediationHints {
					fmt.Fprintf(w, "                   hint: %s %s = %v", hint.Type, hint.Path, hint.Value)
					if hint.Description != "" {
						fmt.Fprintf(w, " (%s)", hint.Description)
					}
					fmt.Fprintf(w, "\n")
				}
			}
			for _, ref := range req.References {
				fmt.Fprintf(w, "                   ref: %s (%s)\n", ref.Label, ref.URL)
			}
		}
		fmt.Fprintln(w)
	}

	if showDeps {
		fmt.Fprintln(w, "External dependencies (not verified against this values file — set up per environment):")
		if len(r.Dependencies) == 0 {
			fmt.Fprintln(w, "  (none apply)")
		}
		for _, d := range r.Dependencies {
			fmt.Fprintf(w, "  [%s] %s: %s (owner: %s)\n", d.RequirementID, d.ID, d.Description, d.Owner)
			if len(d.KnownImplementations) == 0 {
				fmt.Fprintln(w, "                    no known implementations on record — verify manually")
			}
			for _, impl := range d.KnownImplementations {
				fmt.Fprintf(w, "                    known implementation in %s: %s\n", impl.Environment, impl.URL)
			}
		}
	}
}
