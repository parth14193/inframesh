package main

import (
	"testing"

	"github.com/parth14193/inframesh/pkg/planner"
	"github.com/parth14193/inframesh/pkg/skills"
)

func setupPlanEngine(t *testing.T) *planner.Engine {
	t.Helper()
	registry := skills.NewRegistry()
	if err := registry.LoadBuiltins(); err != nil {
		t.Fatalf("failed to load builtins: %v", err)
	}
	return planner.NewEngine(registry)
}

func TestPopulatePlanFromDescription_EKSDeploy(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("EKS", "deploy eks with autoscaling")

	ok := populatePlanFromDescription(engine, plan, "deploy eks with autoscaling")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "aws.eks.deploy" {
		t.Fatalf("expected first step aws.eks.deploy, got %s", plan.Steps[0].SkillName)
	}
}

func TestPopulatePlanFromDescription_EC2CPUOptimized(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("EC2", "deploy ec2 with cpu optimised")

	ok := populatePlanFromDescription(engine, plan, "deploy ec2 with cpu optimised")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "aws.ec2.deploy.cpu_optimized" {
		t.Fatalf("expected first step aws.ec2.deploy.cpu_optimized, got %s", plan.Steps[0].SkillName)
	}
}

func TestPopulatePlanFromDescription_RDSSecure(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("RDS", "launch rds in secure way")

	ok := populatePlanFromDescription(engine, plan, "launch rds in secure way")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "aws.rds.launch.secure" {
		t.Fatalf("expected first step aws.rds.launch.secure, got %s", plan.Steps[0].SkillName)
	}
}

func TestPopulatePlanFromDescription_Unknown(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("Unknown", "do something random")

	ok := populatePlanFromDescription(engine, plan, "do something random")
	if ok {
		t.Fatal("expected plan generation to fail")
	}
}

func TestPopulatePlanFromDescription_GKEDeploy(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("GKE", "deploy gke service")

	ok := populatePlanFromDescription(engine, plan, "deploy gke service")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "gcp.gke.deploy" {
		t.Fatalf("expected first step gcp.gke.deploy, got %s", plan.Steps[0].SkillName)
	}
}

func TestPopulatePlanFromDescription_AKSDeploy(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("AKS", "deploy aks app")

	ok := populatePlanFromDescription(engine, plan, "deploy aks app")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "azure.aks.deploy" {
		t.Fatalf("expected first step azure.aks.deploy, got %s", plan.Steps[0].SkillName)
	}
}

func TestPopulatePlanFromDescription_AWSGenericServiceDeploy(t *testing.T) {
	engine := setupPlanEngine(t)
	plan := engine.CreatePlan("AWS", "deploy aws ecs in secure way")

	ok := populatePlanFromDescription(engine, plan, "deploy aws ecs in secure way")
	if !ok {
		t.Fatal("expected plan generation to succeed")
	}
	if len(plan.Steps) < 1 {
		t.Fatalf("expected at least one step, got %d", len(plan.Steps))
	}
	if plan.Steps[0].SkillName != "aws.service.deploy" {
		t.Fatalf("expected first step aws.service.deploy, got %s", plan.Steps[0].SkillName)
	}
}
