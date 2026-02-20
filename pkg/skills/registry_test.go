package skills_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/skills"
)

func TestNewRegistry(t *testing.T) {
	r := skills.NewRegistry()
	if r.Count() != 0 {
		t.Errorf("expected empty registry, got %d skills", r.Count())
	}
}

func TestRegisterAndGet(t *testing.T) {
	r := skills.NewRegistry()
	skill := &core.Skill{
		Name:     "test.skill",
		Provider: core.ProviderAWS,
		Category: core.CategoryCompute,
	}

	err := r.Register(skill)
	if err != nil {
		t.Fatalf("failed to register skill: %v", err)
	}

	got, err := r.Get("test.skill")
	if err != nil {
		t.Fatalf("failed to get skill: %v", err)
	}
	if got.Name != "test.skill" {
		t.Errorf("expected test.skill, got %s", got.Name)
	}
}

func TestRegisterDuplicate(t *testing.T) {
	r := skills.NewRegistry()
	skill := &core.Skill{Name: "dup.skill"}

	_ = r.Register(skill)
	err := r.Register(skill)
	if err == nil {
		t.Error("expected error for duplicate registration")
	}
}

func TestGetNotFound(t *testing.T) {
	r := skills.NewRegistry()
	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("expected error for missing skill")
	}
}

func TestSearch(t *testing.T) {
	r := skills.NewRegistry()
	_ = r.Register(&core.Skill{Name: "aws.ec2.list", Provider: core.ProviderAWS, Description: "List EC2 instances"})
	_ = r.Register(&core.Skill{Name: "gcp.gce.snapshot", Provider: core.ProviderGCP, Description: "Create VM snapshot"})
	_ = r.Register(&core.Skill{Name: "k8s.deploy", Provider: core.ProviderKubernetes, Description: "Deploy workloads"})

	results := r.Search("aws")
	if len(results) != 1 {
		t.Errorf("expected 1 AWS result, got %d", len(results))
	}

	results = r.Search("snapshot")
	if len(results) != 1 {
		t.Errorf("expected 1 snapshot result, got %d", len(results))
	}

	results = r.Search("deploy")
	if len(results) != 1 {
		t.Errorf("expected 1 deploy result, got %d", len(results))
	}
}

func TestListByProvider(t *testing.T) {
	r := skills.NewRegistry()
	_ = r.Register(&core.Skill{Name: "aws.ec2.list", Provider: core.ProviderAWS})
	_ = r.Register(&core.Skill{Name: "aws.s3.audit", Provider: core.ProviderAWS})
	_ = r.Register(&core.Skill{Name: "gcp.gce.snapshot", Provider: core.ProviderGCP})

	awsSkills := r.ListByProvider(core.ProviderAWS)
	if len(awsSkills) != 2 {
		t.Errorf("expected 2 AWS skills, got %d", len(awsSkills))
	}
}

func TestListByCategory(t *testing.T) {
	r := skills.NewRegistry()
	_ = r.Register(&core.Skill{Name: "a", Category: core.CategoryCompute})
	_ = r.Register(&core.Skill{Name: "b", Category: core.CategoryCompute})
	_ = r.Register(&core.Skill{Name: "c", Category: core.CategoryStorage})

	compute := r.ListByCategory(core.CategoryCompute)
	if len(compute) != 2 {
		t.Errorf("expected 2 compute skills, got %d", len(compute))
	}
}

func TestLoadBuiltins(t *testing.T) {
	r := skills.NewRegistry()
	err := r.LoadBuiltins()
	if err != nil {
		t.Fatalf("failed to load builtins: %v", err)
	}

	count := r.Count()
	if count < 35 {
		t.Errorf("expected at least 35 built-in skills, got %d", count)
	}

	// Verify key skills exist
	keySkills := []string{
		"aws.ec2.list", "aws.ec2.scale", "aws.s3.audit",
		"k8s.deploy", "k8s.rollback",
		"terraform.plan", "terraform.apply",
		"gcp.gce.snapshot", "azure.vm.resize",
		"datadog.alert.list", "trivy.scan",
		"github.actions.trigger", "cloudflare.dns.manage",
		"infracost.estimate",
	}

	for _, name := range keySkills {
		if _, err := r.Get(name); err != nil {
			t.Errorf("expected built-in skill %s to exist: %v", name, err)
		}
	}
}
