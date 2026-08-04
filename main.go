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
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

// starterTemplate is a generic, placeholder-filled config-requirements.yaml
// emitted by -init. Embedding the actual file (rather than a Go string
// literal) means the CI-tested copy in examples/ and the copy -init prints
// are guaranteed to be the same bytes — no drift between docs and reality.
//
//go:embed examples/starter.yaml
var starterTemplate string

// Report is the full, flag-filtered result of one run, shared by the text
// and JSON printers.
type Report struct {
	Features     []FeatureState      `json:"features,omitempty"`
	Requirements []RequirementResult `json:"requirements,omitempty"`
	Dependencies []DependencyEntry   `json:"dependencies,omitempty"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr *os.File) int {
	fs := flag.NewFlagSet("config-requirements-check", flag.ContinueOnError)
	fs.SetOutput(stderr)

	valuesPath := fs.String("values", "", "path to the values YAML file to inspect (required unless -lint)")
	requirementsPath := fs.String("requirements", "config-requirements.yaml", "path to the requirements registry YAML file")
	showFeaturesFlag := fs.Bool("features", false, "print resolved feature-gate states")
	showCheckFlag := fs.Bool("check", false, "validate conditions/requires and report violations")
	showDepsFlag := fs.Bool("deps", false, "print external-dependency checklist")
	featureID := fs.String("feature", "", "restrict output to a single requirement id")
	format := fs.String("format", "text", "output format: text|json")
	lint := fs.Bool("lint", false, "validate the requirements registry's own schema/structure and exit; no -values needed")
	initFlag := fs.Bool("init", false, "print a starter config-requirements.yaml to stdout and exit; no -values needed")

	if err := fs.Parse(args); err != nil {
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

	var report Report
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
		report.Dependencies = applicableDependencies(values, reqs)
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
				fmt.Fprintf(w, "                   fix: %s\n", req.Remediation)
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
