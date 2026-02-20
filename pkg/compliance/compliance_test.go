package compliance_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/compliance"
)

func TestCISAudit(t *testing.T) {
	a := compliance.NewAuditor()
	a.LoadCISBenchmarks()

	report := a.RunAudit(compliance.FrameworkCIS)
	if report.TotalChecks == 0 {
		t.Error("expected CIS checks to be registered")
	}
	if report.TotalChecks < 10 {
		t.Errorf("expected at least 10 CIS checks, got %d", report.TotalChecks)
	}
}

func TestSOC2Audit(t *testing.T) {
	a := compliance.NewAuditor()
	a.LoadAll()

	report := a.RunAudit(compliance.FrameworkSOC2)
	if report.TotalChecks == 0 {
		t.Error("expected SOC2 checks")
	}
}

func TestHIPAAAudit(t *testing.T) {
	a := compliance.NewAuditor()
	a.LoadAll()

	report := a.RunAudit(compliance.FrameworkHIPAA)
	if report.TotalChecks == 0 {
		t.Error("expected HIPAA checks")
	}
}

func TestNonexistentFramework(t *testing.T) {
	a := compliance.NewAuditor()
	report := a.RunAudit("NONEXISTENT")
	if report.TotalChecks != 0 {
		t.Error("unknown framework should return empty report")
	}
}

func TestReportScoring(t *testing.T) {
	a := compliance.NewAuditor()
	a.Register(&compliance.Check{
		ID: "TEST-1", Framework: "TEST", Title: "Pass test",
		CheckFunc: func() compliance.CheckResult {
			return compliance.CheckResult{Status: compliance.StatusPass}
		},
	})
	a.Register(&compliance.Check{
		ID: "TEST-2", Framework: "TEST", Title: "Fail test",
		CheckFunc: func() compliance.CheckResult {
			return compliance.CheckResult{Status: compliance.StatusFail}
		},
	})

	report := a.RunAudit("TEST")
	if report.Passed != 1 {
		t.Errorf("expected 1 pass, got %d", report.Passed)
	}
	if report.Failed != 1 {
		t.Errorf("expected 1 fail, got %d", report.Failed)
	}
	if report.Score != 50.0 {
		t.Errorf("expected 50%% score, got %.1f%%", report.Score)
	}
}

func TestReportRender(t *testing.T) {
	a := compliance.NewAuditor()
	a.LoadCISBenchmarks()
	report := a.RunAudit(compliance.FrameworkCIS)
	rendered := report.Render()
	if len(rendered) == 0 {
		t.Error("render should produce output")
	}
}
