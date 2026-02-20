package planner_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/planner"
	"github.com/parth14193/ownbot/pkg/skills"
)

func setupEngine() (*planner.Engine, *skills.Registry) {
	r := skills.NewRegistry()
	_ = r.LoadBuiltins()
	return planner.NewEngine(r), r
}

func TestCreatePlan(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Test Plan", "A test plan")

	if plan.Name != "Test Plan" {
		t.Errorf("expected 'Test Plan', got '%s'", plan.Name)
	}
	if len(plan.Steps) != 0 {
		t.Errorf("expected 0 steps, got %d", len(plan.Steps))
	}
}

func TestAddStep(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Deploy Plan", "Deploy to staging")

	err := engine.AddStep(plan, "k8s.deploy", "Deploy app", map[string]interface{}{
		"namespace":  "default",
		"deployment": "app",
		"image":      "app:v1",
	})
	if err != nil {
		t.Fatalf("failed to add step: %v", err)
	}

	if len(plan.Steps) != 1 {
		t.Errorf("expected 1 step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].StepNumber != 1 {
		t.Errorf("expected step number 1, got %d", plan.Steps[0].StepNumber)
	}
	if plan.Steps[0].RiskLevel != core.RiskHigh {
		t.Errorf("expected HIGH risk, got %s", plan.Steps[0].RiskLevel)
	}
}

func TestAddStepInvalidSkill(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Bad Plan", "This will fail")

	err := engine.AddStep(plan, "nonexistent.skill", "Will fail", nil)
	if err == nil {
		t.Error("expected error for nonexistent skill")
	}
}

func TestAddConditionalStep(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Conditional Plan", "Deploy with rollback")

	_ = engine.AddStep(plan, "k8s.deploy", "Deploy", map[string]interface{}{
		"namespace": "default", "deployment": "app", "image": "app:v2",
	})

	err := engine.AddConditionalStep(plan,
		"error_rate > 2%",
		"k8s.rollback", "Rollback on error",
		"k8s.rollout.status", "Monitor rollout",
	)
	if err != nil {
		t.Fatalf("failed to add conditional step: %v", err)
	}

	if len(plan.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(plan.Steps))
	}
	if plan.Steps[1].SkillName != "CONDITIONAL" {
		t.Errorf("expected CONDITIONAL, got %s", plan.Steps[1].SkillName)
	}
}

func TestValidate(t *testing.T) {
	engine, _ := setupEngine()

	// Empty plan
	plan := engine.CreatePlan("Empty", "No steps")
	errs := engine.Validate(plan)
	if len(errs) == 0 {
		t.Error("expected validation errors for empty plan")
	}

	// Valid plan
	plan = engine.CreatePlan("Valid", "Valid plan")
	_ = engine.AddStep(plan, "aws.ec2.list", "List instances", nil)
	errs = engine.Validate(plan)
	if len(errs) != 0 {
		t.Errorf("expected no validation errors, got %d: %v", len(errs), errs)
	}
}

func TestOverallRiskCalculation(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Risk Test", "Test risk calc")

	_ = engine.AddStep(plan, "aws.ec2.list", "Low risk op", nil)
	if plan.OverallRisk != core.RiskLow {
		t.Errorf("expected LOW, got %s", plan.OverallRisk)
	}

	_ = engine.AddStep(plan, "k8s.deploy", "High risk op", map[string]interface{}{
		"namespace": "default", "deployment": "app", "image": "app:v1",
	})
	if plan.OverallRisk != core.RiskHigh {
		t.Errorf("expected HIGH, got %s", plan.OverallRisk)
	}
}

func TestStepsRequiringConfirmation(t *testing.T) {
	engine, _ := setupEngine()
	plan := engine.CreatePlan("Confirm Test", "Test confirmations")

	_ = engine.AddStep(plan, "aws.ec2.list", "Read-only", nil)
	_ = engine.AddStep(plan, "k8s.deploy", "Deploy", map[string]interface{}{
		"namespace": "default", "deployment": "app", "image": "img",
	})

	confirms := engine.StepsRequiringConfirmation(plan)
	if len(confirms) != 1 {
		t.Errorf("expected 1 confirmation, got %d", len(confirms))
	}
	if confirms[0] != 2 {
		t.Errorf("expected step 2 to require confirmation, got step %d", confirms[0])
	}
}
