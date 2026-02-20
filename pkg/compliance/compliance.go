// Package compliance provides codified compliance checks against
// security standards like CIS Benchmarks, SOC2, HIPAA, and PCI-DSS.
package compliance

import (
	"fmt"
	"strings"
	"time"
)

// Framework identifies a compliance standard.
type Framework string

const (
	FrameworkCIS    Framework = "CIS"
	FrameworkSOC2   Framework = "SOC2"
	FrameworkHIPAA  Framework = "HIPAA"
	FrameworkPCIDSS Framework = "PCI-DSS"
	FrameworkCustom Framework = "CUSTOM"
)

// CheckStatus indicates the result of a compliance check.
type CheckStatus string

const (
	StatusPass CheckStatus = "PASS"
	StatusFail CheckStatus = "FAIL"
	StatusWarn CheckStatus = "WARN"
	StatusSkip CheckStatus = "SKIP"
)

// Severity classifies check importance.
type Severity string

const (
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// Check defines a single compliance check.
type Check struct {
	ID          string    `json:"id"`
	Framework   Framework `json:"framework"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Severity    Severity  `json:"severity"`
	Category    string    `json:"category"`
	CheckFunc   func() CheckResult `json:"-"`
}

// CheckResult is the outcome of a single compliance check.
type CheckResult struct {
	ID          string      `json:"id"`
	Status      CheckStatus `json:"status"`
	Title       string      `json:"title"`
	Severity    Severity    `json:"severity"`
	Details     string      `json:"details"`
	Remediation string      `json:"remediation"`
}

// Report aggregates compliance check results for a framework.
type Report struct {
	Framework   Framework      `json:"framework"`
	Timestamp   time.Time      `json:"timestamp"`
	Results     []CheckResult  `json:"results"`
	TotalChecks int            `json:"total_checks"`
	Passed      int            `json:"passed"`
	Failed      int            `json:"failed"`
	Warnings    int            `json:"warnings"`
	Skipped     int            `json:"skipped"`
	Score       float64        `json:"score"` // percentage passed
}

// Auditor runs compliance audits against a specific framework.
type Auditor struct {
	checks map[Framework][]*Check
}

// NewAuditor creates a new ComplianceAuditor.
func NewAuditor() *Auditor {
	return &Auditor{
		checks: make(map[Framework][]*Check),
	}
}

// Register adds a check to the auditor.
func (a *Auditor) Register(check *Check) {
	a.checks[check.Framework] = append(a.checks[check.Framework], check)
}

// LoadCISBenchmarks registers all CIS AWS Foundation Benchmark checks.
func (a *Auditor) LoadCISBenchmarks() {
	for _, check := range CISBenchmarks() {
		a.Register(check)
	}
}

// LoadAll registers all built-in compliance checks across all frameworks.
func (a *Auditor) LoadAll() {
	a.LoadCISBenchmarks()
	for _, check := range SOC2Controls() {
		a.Register(check)
	}
	for _, check := range HIPAAControls() {
		a.Register(check)
	}
}

// RunAudit executes all checks for a given framework and returns a report.
func (a *Auditor) RunAudit(framework Framework) *Report {
	checks, exists := a.checks[framework]
	if !exists {
		return &Report{
			Framework: framework,
			Timestamp: time.Now(),
		}
	}

	report := &Report{
		Framework:   framework,
		Timestamp:   time.Now(),
		TotalChecks: len(checks),
	}

	for _, check := range checks {
		result := check.CheckFunc()
		result.ID = check.ID
		result.Title = check.Title
		result.Severity = check.Severity

		switch result.Status {
		case StatusPass:
			report.Passed++
		case StatusFail:
			report.Failed++
		case StatusWarn:
			report.Warnings++
		case StatusSkip:
			report.Skipped++
		}

		report.Results = append(report.Results, result)
	}

	if report.TotalChecks > 0 {
		report.Score = float64(report.Passed) / float64(report.TotalChecks) * 100
	}

	return report
}

// ListFrameworks returns all frameworks with registered checks.
func (a *Auditor) ListFrameworks() []Framework {
	var frameworks []Framework
	for f := range a.checks {
		frameworks = append(frameworks, f)
	}
	return frameworks
}

// Render formats a compliance report for display.
func (r *Report) Render() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("📋 COMPLIANCE REPORT: %s\n", r.Framework))
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Timestamp: %s\n", r.Timestamp.Format(time.RFC3339)))
	b.WriteString(fmt.Sprintf("Score:     %.1f%% (%d/%d passed)\n\n", r.Score, r.Passed, r.TotalChecks))

	// Summary bar
	b.WriteString(fmt.Sprintf("  ✅ Passed:   %d\n", r.Passed))
	b.WriteString(fmt.Sprintf("  ❌ Failed:   %d\n", r.Failed))
	b.WriteString(fmt.Sprintf("  ⚠️  Warnings: %d\n", r.Warnings))
	b.WriteString(fmt.Sprintf("  ⏭️  Skipped:  %d\n\n", r.Skipped))

	// Failed checks (show first)
	failedChecks := filterByStatus(r.Results, StatusFail)
	if len(failedChecks) > 0 {
		b.WriteString(fmt.Sprintf("❌ FAILED CHECKS (%d)\n", len(failedChecks)))
		b.WriteString("─────────────────────────────────────────\n")
		for _, c := range failedChecks {
			b.WriteString(fmt.Sprintf("  [%s] %s — %s\n", c.Severity, c.ID, c.Title))
			if c.Details != "" {
				b.WriteString(fmt.Sprintf("          Detail: %s\n", c.Details))
			}
			if c.Remediation != "" {
				b.WriteString(fmt.Sprintf("          Fix: %s\n", c.Remediation))
			}
		}
	}

	// Warning checks
	warnChecks := filterByStatus(r.Results, StatusWarn)
	if len(warnChecks) > 0 {
		b.WriteString(fmt.Sprintf("\n⚠️  WARNINGS (%d)\n", len(warnChecks)))
		for _, c := range warnChecks {
			b.WriteString(fmt.Sprintf("  [%s] %s — %s\n", c.Severity, c.ID, c.Title))
		}
	}

	return b.String()
}

func filterByStatus(results []CheckResult, status CheckStatus) []CheckResult {
	var filtered []CheckResult
	for _, r := range results {
		if r.Status == status {
			filtered = append(filtered, r)
		}
	}
	return filtered
}
