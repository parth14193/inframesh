package drift_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/drift"
)

func TestAnalyzeTerraformPlan(t *testing.T) {
	d := drift.NewDetector()
	planOutput := `
# aws_instance.web will be updated in-place
  ~ instance_type = "t3.micro" -> "t3.large"
# aws_s3_bucket.logs will be created
# aws_security_group.old will be destroyed
`
	report := d.AnalyzeTerraformPlan(planOutput)

	if report.Drifted != 1 {
		t.Errorf("expected 1 drifted, got %d", report.Drifted)
	}
	if report.New != 1 {
		t.Errorf("expected 1 new, got %d", report.New)
	}
	if report.Deleted != 1 {
		t.Errorf("expected 1 deleted, got %d", report.Deleted)
	}
	if len(report.Resources) != 3 {
		t.Errorf("expected 3 resources, got %d", len(report.Resources))
	}
}

func TestDetectManualChanges(t *testing.T) {
	d := drift.NewDetector()
	live := []string{"instance-1", "instance-2", "instance-3"}
	declared := []string{"instance-1", "instance-2", "instance-4"}

	report := d.DetectManualChanges("aws", "ec2.instance", live, declared)

	if report.InSync != 2 {
		t.Errorf("expected 2 in sync, got %d", report.InSync)
	}
	if report.New != 1 {
		t.Errorf("expected 1 new (instance-3), got %d", report.New)
	}
	if report.Deleted != 1 {
		t.Errorf("expected 1 deleted (instance-4), got %d", report.Deleted)
	}
}

func TestEmptyPlan(t *testing.T) {
	d := drift.NewDetector()
	report := d.AnalyzeTerraformPlan("")
	if len(report.Resources) != 0 {
		t.Error("empty plan should have no resources")
	}
}

func TestDriftReportRender(t *testing.T) {
	d := drift.NewDetector()
	report := d.AnalyzeTerraformPlan("# aws_instance.x will be created\n")
	rendered := report.Render()
	if len(rendered) == 0 {
		t.Error("render should produce output")
	}
}
