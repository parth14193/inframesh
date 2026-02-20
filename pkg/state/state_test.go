package state_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/state"
)

func TestNewManager(t *testing.T) {
	m := state.NewManager("test-session")
	s := m.GetState()

	if s.SessionID != "test-session" {
		t.Errorf("expected test-session, got %s", s.SessionID)
	}
	if s.ActiveEnvironment != "staging" {
		t.Errorf("expected default staging environment, got %s", s.ActiveEnvironment)
	}
	if s.ActiveProvider != core.ProviderAWS {
		t.Errorf("expected default AWS provider, got %s", s.ActiveProvider)
	}
}

func TestSetEnvironment(t *testing.T) {
	m := state.NewManager("sess")
	m.SetEnvironment("production")

	if m.GetEnvironment() != "production" {
		t.Errorf("expected production, got %s", m.GetEnvironment())
	}
}

func TestSetProvider(t *testing.T) {
	m := state.NewManager("sess")
	m.SetProvider(core.ProviderGCP)

	if m.GetProvider() != core.ProviderGCP {
		t.Errorf("expected gcp, got %s", m.GetProvider())
	}
}

func TestSetRegion(t *testing.T) {
	m := state.NewManager("sess")
	m.SetRegion("eu-west-1")

	if m.GetRegion() != "eu-west-1" {
		t.Errorf("expected eu-west-1, got %s", m.GetRegion())
	}
}

func TestAuditLog(t *testing.T) {
	m := state.NewManager("sess")

	m.AddToAuditLog("aws.ec2.list", "list", "staging/aws/us-east-1", core.StatusSuccess, core.RiskLow, "Listed instances")
	m.AddToAuditLog("k8s.deploy", "deploy", "staging/k8s", core.StatusDryRun, core.RiskHigh, "Dry run deploy")

	log := m.GetAuditLog()
	if len(log) != 2 {
		t.Errorf("expected 2 audit entries, got %d", len(log))
	}
	if log[0].SkillName != "aws.ec2.list" {
		t.Errorf("expected aws.ec2.list, got %s", log[0].SkillName)
	}
}

func TestResourceContext(t *testing.T) {
	m := state.NewManager("sess")

	err := m.UpdateResourceContext("cluster", "eks-prod")
	if err != nil {
		t.Fatalf("failed to update context: %v", err)
	}

	err = m.UpdateResourceContext("namespace", "payments")
	if err != nil {
		t.Fatalf("failed to update context: %v", err)
	}

	ctx := m.GetContext()
	if ctx.Cluster != "eks-prod" {
		t.Errorf("expected eks-prod, got %s", ctx.Cluster)
	}
	if ctx.Namespace != "payments" {
		t.Errorf("expected payments, got %s", ctx.Namespace)
	}
}

func TestResourceContextInvalidKey(t *testing.T) {
	m := state.NewManager("sess")
	err := m.UpdateResourceContext("invalid_key", "value")
	if err == nil {
		t.Error("expected error for invalid context key")
	}
}

func TestLoadSkill(t *testing.T) {
	m := state.NewManager("sess")

	m.LoadSkill("aws.ec2.list")
	m.LoadSkill("k8s.deploy")
	m.LoadSkill("aws.ec2.list") // duplicate — should not add again

	loaded := m.GetLoadedSkills()
	if len(loaded) != 2 {
		t.Errorf("expected 2 loaded skills (no duplicates), got %d", len(loaded))
	}
}

func TestCustomData(t *testing.T) {
	m := state.NewManager("sess")

	m.SetCustomData("last_action", "deploy")
	val, ok := m.GetCustomData("last_action")
	if !ok {
		t.Fatal("expected custom data to exist")
	}
	if val != "deploy" {
		t.Errorf("expected deploy, got %v", val)
	}

	_, ok = m.GetCustomData("nonexistent")
	if ok {
		t.Error("expected nonexistent key to not be found")
	}
}

func TestPendingConfirmations(t *testing.T) {
	m := state.NewManager("sess")

	m.AddPendingConfirmation("confirm terraform.apply")
	s := m.GetState()
	if len(s.PendingConfirmations) != 1 {
		t.Errorf("expected 1 pending confirmation, got %d", len(s.PendingConfirmations))
	}

	m.ClearPendingConfirmations()
	s = m.GetState()
	if len(s.PendingConfirmations) != 0 {
		t.Errorf("expected 0 pending confirmations after clear, got %d", len(s.PendingConfirmations))
	}
}
