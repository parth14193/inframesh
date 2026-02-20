package safety_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/safety"
)

func TestEvaluateReadOnly(t *testing.T) {
	layer := safety.NewLayer()
	skill := &core.Skill{
		Name:      "aws.ec2.list",
		RiskLevel: core.RiskLow,
		Rollback:  core.RollbackConfig{Supported: false},
	}

	report := layer.Evaluate(skill, nil, "staging")
	if report.RiskLevel != core.RiskLow {
		t.Errorf("expected LOW risk, got %s", report.RiskLevel)
	}
	if report.RequiresConfirmation {
		t.Error("read-only operations should not require confirmation")
	}
	if report.BlastRadius != 0 {
		t.Errorf("expected 0 blast radius for list operation, got %d", report.BlastRadius)
	}
}

func TestEvaluateProductionEscalation(t *testing.T) {
	layer := safety.NewLayer()
	skill := &core.Skill{
		Name:      "aws.ec2.scale",
		RiskLevel: core.RiskMedium,
		Rollback:  core.RollbackConfig{Supported: true, Procedure: "revert"},
	}

	report := layer.Evaluate(skill, nil, "production")
	if report.RiskLevel < core.RiskHigh {
		t.Errorf("production should escalate to at least HIGH, got %s", report.RiskLevel)
	}
	if !report.RequiresConfirmation {
		t.Error("production operations should always require confirmation")
	}
	if report.EnvironmentWarning == "" {
		t.Error("expected production environment warning")
	}
}

func TestEvaluateDryRunRecommendation(t *testing.T) {
	layer := safety.NewLayer()

	destructiveOps := []string{"k8s.deploy", "terraform.apply", "aws.ec2.scale"}
	for _, name := range destructiveOps {
		skill := &core.Skill{Name: name, RiskLevel: core.RiskHigh}
		report := layer.Evaluate(skill, nil, "staging")
		if !report.DryRunRecommended {
			t.Errorf("expected dry run recommended for %s", name)
		}
	}

	readOnlyOps := []string{"aws.ec2.list", "aws.s3.audit", "aws.cost.report"}
	for _, name := range readOnlyOps {
		skill := &core.Skill{Name: name, RiskLevel: core.RiskLow}
		report := layer.Evaluate(skill, nil, "staging")
		if report.DryRunRecommended {
			t.Errorf("did not expect dry run for read-only %s", name)
		}
	}
}

func TestConfirmationPrompts(t *testing.T) {
	layer := safety.NewLayer()

	tests := []struct {
		risk     core.RiskLevel
		contains string
	}{
		{core.RiskLow, ""},
		{core.RiskMedium, "yes"},
		{core.RiskHigh, "yes, apply"},
		{core.RiskCritical, "CONFIRM PRODUCTION"},
	}

	for _, tt := range tests {
		prompt := layer.GetConfirmationPrompt(tt.risk)
		if tt.contains == "" && prompt != "" {
			t.Errorf("expected empty prompt for %s, got %s", tt.risk, prompt)
		}
		if tt.contains != "" && !containsStr(prompt, tt.contains) {
			t.Errorf("expected prompt for %s to contain '%s', got '%s'", tt.risk, tt.contains, prompt)
		}
	}
}

func TestRequiresConfirmation(t *testing.T) {
	layer := safety.NewLayer()

	if layer.RequiresConfirmation(core.RiskLow) {
		t.Error("LOW should not require confirmation")
	}
	if !layer.RequiresConfirmation(core.RiskMedium) {
		t.Error("MEDIUM should require confirmation")
	}
	if !layer.RequiresConfirmation(core.RiskHigh) {
		t.Error("HIGH should require confirmation")
	}
	if !layer.RequiresConfirmation(core.RiskCritical) {
		t.Error("CRITICAL should require confirmation")
	}
}

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && contains(s, substr)
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
