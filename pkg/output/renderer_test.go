package output_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/output"
)

func TestRenderTable(t *testing.T) {
	r := output.NewRenderer()

	headers := []string{"Name", "Value"}
	rows := [][]string{
		{"cpu", "4 cores"},
		{"memory", "16 GB"},
	}

	result := r.RenderTable(headers, rows)

	if !strings.Contains(result, "Name") {
		t.Error("table should contain header 'Name'")
	}
	if !strings.Contains(result, "4 cores") {
		t.Error("table should contain '4 cores'")
	}
	if !strings.Contains(result, "┌") {
		t.Error("table should have box-drawing characters")
	}
}

func TestRenderTableEmpty(t *testing.T) {
	r := output.NewRenderer()
	result := r.RenderTable([]string{}, nil)
	if result != "" {
		t.Error("empty headers should produce empty output")
	}
}

func TestRenderQuery(t *testing.T) {
	r := output.NewRenderer()
	result := r.RenderQuery("aws.ec2.list", "staging", "aws", "us-east-1", "Found 5 instances", 1240, 5)

	if !strings.Contains(result, "aws.ec2.list") {
		t.Error("should contain skill name")
	}
	if !strings.Contains(result, "staging") {
		t.Error("should contain environment")
	}
	if !strings.Contains(result, "1240ms") {
		t.Error("should contain duration")
	}
	if !strings.Contains(result, "5 resources") {
		t.Error("should contain resource count")
	}
}

func TestRenderMutation(t *testing.T) {
	r := output.NewRenderer()
	result := r.RenderMutation(
		"Scale ASG",
		"staging", "aws", "us-east-1",
		3,
		"desired_capacity: 2",
		"desired_capacity: 5",
		core.RiskHigh,
		"Restore previous capacity",
	)

	if !strings.Contains(result, "BEFORE") {
		t.Error("should contain BEFORE section")
	}
	if !strings.Contains(result, "AFTER") {
		t.Error("should contain AFTER section")
	}
	if !strings.Contains(result, "yes, apply") {
		t.Error("HIGH risk should show 'yes, apply' prompt")
	}
}

func TestRenderPlan(t *testing.T) {
	r := output.NewRenderer()
	plan := &core.Plan{
		Name:        "Deploy Plan",
		Description: "Deploy v2.0",
		Steps: []core.PlanStep{
			{StepNumber: 1, SkillName: "k8s.deploy", Description: "Deploy new image", RiskLevel: core.RiskHigh},
			{StepNumber: 2, SkillName: "k8s.rollout.status", Description: "Watch rollout", RiskLevel: core.RiskLow},
		},
		EstimatedTime: "2m30s",
		OverallRisk:   core.RiskHigh,
	}

	result := r.RenderPlan(plan)

	if !strings.Contains(result, "2 steps") {
		t.Error("should show step count")
	}
	if !strings.Contains(result, "Requires confirmation") {
		t.Error("HIGH risk steps should note confirmation requirement")
	}
	if !strings.Contains(result, "HIGH") {
		t.Error("should show overall risk level")
	}
}

func TestRenderSkillInfo(t *testing.T) {
	r := output.NewRenderer()
	skill := &core.Skill{
		Name:        "k8s.deploy",
		Description: "Deploy Kubernetes workloads",
		Provider:    core.ProviderKubernetes,
		Category:    core.CategoryDeployment,
		RiskLevel:   core.RiskHigh,
		Inputs: []core.SkillInput{
			{Name: "namespace", Type: "string", Required: true, Description: "Target namespace"},
		},
		Outputs: []core.SkillOutput{
			{Name: "status", Type: "string", Description: "Rollout status"},
		},
	}

	result := r.RenderSkillInfo(skill)

	if !strings.Contains(result, "k8s.deploy") {
		t.Error("should contain skill name")
	}
	if !strings.Contains(result, "namespace") {
		t.Error("should list inputs")
	}
	if !strings.Contains(result, "status") {
		t.Error("should list outputs")
	}
}

func TestRenderSuccessErrorWarning(t *testing.T) {
	r := output.NewRenderer()

	if !strings.Contains(r.RenderSuccess("done"), "✅") {
		t.Error("success should have ✅")
	}
	if !strings.Contains(r.RenderError(fmt.Errorf("fail")), "❌") {
		t.Error("error should have ❌")
	}
	if !strings.Contains(r.RenderWarning("caution"), "⚠") {
		t.Error("warning should have ⚠")
	}
}

func TestRenderSafetyReport(t *testing.T) {
	r := output.NewRenderer()
	report := &core.SafetyReport{
		SkillName:           "terraform.apply",
		RiskLevel:           core.RiskCritical,
		BlastRadius:         15,
		RequiresConfirmation: true,
		RollbackAvailable:   true,
		RollbackProcedure:   "terraform destroy",
		DryRunRecommended:   true,
		ConfirmationPrompt:  "CONFIRM PRODUCTION",
	}

	result := r.RenderSafetyReport(report)
	if !strings.Contains(result, "CRITICAL") {
		t.Error("should show CRITICAL risk level")
	}
	if !strings.Contains(result, "15") {
		t.Error("should show blast radius")
	}
}
