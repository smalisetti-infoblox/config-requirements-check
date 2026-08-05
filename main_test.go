package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeTempFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", p, err)
	}
	return p
}

const testRequirements = `
requirements:
  - id: consolidated-health-enabled-toggle
    summary: consolidatedHealth.enabled must be set when redis.enabled is true.
    conditions:
      - path: redis.enabled
        equals: true
    requires:
      - path: consolidatedHealth.enabled
        equals: true
    remediation: "Set consolidatedHealth.enabled: true."
    external_dependencies:
      - id: kafka-topic
        description: topic must exist
        owner: platform-kafka
        verify:
          type: manual
`

func runCLI(t *testing.T, args []string) (int, string) {
	t.Helper()
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout")
	stderrPath := filepath.Join(dir, "stderr")
	stdout, err := os.Create(stdoutPath)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := os.Create(stderrPath)
	if err != nil {
		t.Fatal(err)
	}
	code := run(args, stdout, stderr)
	_ = stdout.Close()
	_ = stderr.Close()
	out, _ := os.ReadFile(stdoutPath)
	errOut, _ := os.ReadFile(stderrPath)
	return code, string(out) + string(errOut)
}

func TestCLI_CheckFailsOnMissingRequirement(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit code 1, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "MISCONFIGURED") {
		t.Fatalf("expected MISCONFIGURED in output, got:\n%s", out)
	}
}

func TestCLI_CheckPassesWhenSatisfied(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\nconsolidatedHealth:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-check", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "satisfied") {
		t.Fatalf("expected satisfied in output, got:\n%s", out)
	}
}

func TestCLI_DepsPrintedEvenWhenSatisfied(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\nconsolidatedHealth:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-deps", "-values", valuesPath, "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit code 0 (deps never fail), got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "kafka-topic") {
		t.Fatalf("expected kafka-topic dependency listed, got:\n%s", out)
	}
}

func TestCLI_DefaultModeShowsEverythingAndFailsOnViolation(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit code 1 in default mode with a violation, got %d", code)
	}
	if !strings.Contains(out, "Features:") || !strings.Contains(out, "Requirements:") || !strings.Contains(out, "External dependencies") {
		t.Fatalf("expected all three sections in default mode, got:\n%s", out)
	}
}

func TestCLI_FeatureFilter(t *testing.T) {
	dir := t.TempDir()
	multiReq := testRequirements + `
  - id: other-toggle
    conditions:
      - path: other.enabled
        equals: true
    requires:
      - path: other.required
        equals: true
`
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", multiReq)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\nother:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-check", "-feature", "consolidated-health-enabled-toggle", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d", code)
	}
	if strings.Contains(out, "other-toggle") {
		t.Fatalf("expected -feature to filter out other-toggle, got:\n%s", out)
	}
}

func TestCLI_JSONFormatIsValidAndMatchesData(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)
	valuesPath := writeTempFile(t, dir, "values.yaml", "redis:\n  enabled: true\n")

	code, out := runCLI(t, []string{"-format", "json", "-values", valuesPath, "-requirements", reqPath})
	if code != 1 {
		t.Fatalf("expected exit 1, got %d; output:\n%s", code, out)
	}
	var report Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("expected valid JSON, got error %v; output:\n%s", err, out)
	}
	if len(report.Requirements) != 1 || report.Requirements[0].Satisfied {
		t.Fatalf("expected one unsatisfied requirement in JSON report, got %+v", report.Requirements)
	}
	if len(report.Dependencies) != 1 {
		t.Fatalf("expected one dependency in JSON report, got %+v", report.Dependencies)
	}
}

func TestCLI_LintCleanFile(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)

	code, out := runCLI(t, []string{"-lint", "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "OK") {
		t.Fatalf("expected OK in output, got:\n%s", out)
	}
}

func TestCLI_LintCatchesTypoField(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", "requirements:\n  - id: test\n    conditons:\n      - path: a\n        equals: true\n")

	code, out := runCLI(t, []string{"-lint", "-requirements", reqPath})
	if code != 2 {
		t.Fatalf("expected exit code 2 (parse error), got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "conditons") {
		t.Fatalf("expected error to mention the typo'd field, got:\n%s", out)
	}
}

func TestCLI_LintNoValuesNeeded(t *testing.T) {
	dir := t.TempDir()
	reqPath := writeTempFile(t, dir, "config-requirements.yaml", testRequirements)

	// -lint must work without -values at all.
	code, _ := runCLI(t, []string{"-lint", "-requirements", reqPath})
	if code != 0 {
		t.Fatalf("expected -lint to succeed without -values, got exit %d", code)
	}
}

func TestCLI_InitPrintsStarterTemplate(t *testing.T) {
	code, out := runCLI(t, []string{"-init"})
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "requirements:") {
		t.Fatalf("expected starter template to contain 'requirements:', got:\n%s", out)
	}
}

func TestCLI_InitOutputLintsClean(t *testing.T) {
	_, out := runCLI(t, []string{"-init"})

	dir := t.TempDir()
	p := writeTempFile(t, dir, "config-requirements.yaml", out)

	code, lintOut := runCLI(t, []string{"-lint", "-requirements", p})
	if code != 0 {
		t.Fatalf("expected the -init output to lint clean, got exit %d; output:\n%s", code, lintOut)
	}
}

func TestCLI_InitNeedsNoOtherFlags(t *testing.T) {
	// -init must not require -values or a real -requirements file.
	code, _ := runCLI(t, []string{"-init"})
	if code != 0 {
		t.Fatalf("expected -init alone to succeed, got exit %d", code)
	}
}

func TestCLI_HelpExitsZeroAndShowsUsage(t *testing.T) {
	code, out := runCLI(t, []string{"-h"})
	if code != 0 {
		t.Fatalf("expected -h to exit 0, got %d; output:\n%s", code, out)
	}
	if !strings.Contains(out, "Usage:") || !strings.Contains(out, "Examples:") || !strings.Contains(out, "Exit codes:") {
		t.Fatalf("expected help output to contain Usage/Examples/Exit codes sections, got:\n%s", out)
	}
}

func TestCLI_UnknownFlagExitsTwo(t *testing.T) {
	code, _ := runCLI(t, []string{"-bogus-flag"})
	if code != 2 {
		t.Fatalf("expected an unrecognized flag to exit 2, got %d", code)
	}
}
