package cost

import (
	"testing"

	"github.com/parth14193/inframesh/pkg/core"
)

func TestEstimateSkillWithKnownInstance(t *testing.T) {
	estimator := NewEstimator()
	skill := &core.Skill{
		Name:     "aws.ec2.deploy",
		Provider: core.ProviderAWS,
		Category: core.CategoryCompute,
	}
	params := map[string]interface{}{
		"instance_type": "m6i.large",
	}

	est := estimator.EstimateSkill(skill, params)
	if est.HourlyCost == 0 {
		t.Fatal("expected non-zero hourly cost for m6i.large")
	}
	if est.MonthlyCost == 0 {
		t.Fatal("expected non-zero monthly cost")
	}
	if est.Confidence != "high" {
		t.Fatalf("expected high confidence, got %s", est.Confidence)
	}
	if est.PricingTier != "m6i.large" {
		t.Fatalf("expected pricing tier m6i.large, got %s", est.PricingTier)
	}
}

func TestEstimateSkillWithUnknownInstance(t *testing.T) {
	estimator := NewEstimator()
	skill := &core.Skill{
		Name:     "custom.action",
		Provider: core.ProviderCustom,
		Category: core.CategoryCompute,
	}

	est := estimator.EstimateSkill(skill, nil)
	if est.Confidence != "low" {
		t.Fatalf("expected low confidence for unknown resource, got %s", est.Confidence)
	}
}

func TestEstimateSkillWithMultiplier(t *testing.T) {
	estimator := NewEstimator()
	skill := &core.Skill{
		Name:     "aws.ec2.deploy",
		Provider: core.ProviderAWS,
		Category: core.CategoryCompute,
	}
	paramsSingle := map[string]interface{}{
		"instance_type": "t3.large",
	}
	paramsMulti := map[string]interface{}{
		"instance_type":    "t3.large",
		"desired_capacity": 3,
	}

	single := estimator.EstimateSkill(skill, paramsSingle)
	multi := estimator.EstimateSkill(skill, paramsMulti)

	if multi.MonthlyCost <= single.MonthlyCost {
		t.Fatalf("expected multi-instance cost > single, got %.2f vs %.2f",
			multi.MonthlyCost, single.MonthlyCost)
	}
}

func TestEstimatePlan(t *testing.T) {
	estimator := NewEstimator()
	skills := []*core.Skill{
		{Name: "aws.ec2.deploy", Provider: core.ProviderAWS, Category: core.CategoryCompute},
		{Name: "aws.rds.launch", Provider: core.ProviderAWS, Category: core.CategoryStorage},
	}
	params := []map[string]interface{}{
		{"instance_type": "m6i.large"},
		{"instance_class": "db.m6i.large"},
	}

	report := estimator.EstimatePlan("test-plan", skills, params)
	if report.TotalMonthly == 0 {
		t.Fatal("expected non-zero total monthly cost")
	}
	if report.TotalAnnual != report.TotalMonthly*12 {
		t.Fatal("annual should be 12x monthly")
	}
	if len(report.Estimates) != 2 {
		t.Fatalf("expected 2 estimates, got %d", len(report.Estimates))
	}
}

func TestRenderEstimate(t *testing.T) {
	est := &Estimate{
		SkillName:    "aws.ec2.deploy",
		Provider:     "aws",
		ResourceType: "compute",
		PricingTier:  "m6i.large",
		HourlyCost:   0.096,
		MonthlyCost:  70.08,
		Confidence:   "high",
	}
	output := RenderEstimate(est)
	if output == "" {
		t.Fatal("expected non-empty render")
	}
}

func TestGCPPricing(t *testing.T) {
	estimator := NewEstimator()
	skill := &core.Skill{
		Name:     "gcp.gce.deploy",
		Provider: core.ProviderGCP,
		Category: core.CategoryCompute,
	}
	params := map[string]interface{}{
		"machine_type": "c3-standard-4",
	}

	est := estimator.EstimateSkill(skill, params)
	if est.Confidence != "high" {
		t.Fatalf("expected high confidence for GCP c3-standard-4, got %s", est.Confidence)
	}
}
