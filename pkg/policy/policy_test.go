package policy_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/policy"
)

func TestLoadBuiltins(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementWarn)
	e.LoadBuiltins()
	if len(e.ListPolicies()) < 6 {
		t.Errorf("expected at least 6 built-in policies, got %d", len(e.ListPolicies()))
	}
}

func TestNoPublicS3(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementDeny)
	e.LoadBuiltins()

	skill := &core.Skill{Name: "aws.s3.sync", RiskLevel: core.RiskMedium}
	publicParams := map[string]interface{}{"acl": "public-read"}
	result := e.Evaluate(skill, publicParams, "staging")
	if result.Passed {
		t.Error("should block public S3 ACL")
	}

	privateParams := map[string]interface{}{"acl": "private"}
	result = e.Evaluate(skill, privateParams, "staging")
	if !result.Passed {
		t.Error("should allow private ACL")
	}
}

func TestNoWideOpenSG(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementDeny)
	e.LoadBuiltins()

	skill := &core.Skill{Name: "aws.sg.audit", RiskLevel: core.RiskMedium}
	badParams := map[string]interface{}{"cidr": "0.0.0.0/0", "port": "22"}
	result := e.Evaluate(skill, badParams, "staging")
	if result.Passed {
		t.Error("should block SSH open to 0.0.0.0/0")
	}
}

func TestMaxBlastRadius(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementDeny)
	e.LoadBuiltins()

	skill := &core.Skill{Name: "aws.ec2.scale", RiskLevel: core.RiskHigh}
	bigParams := map[string]interface{}{"_resource_count": 100}
	result := e.Evaluate(skill, bigParams, "staging")
	if result.Passed {
		t.Error("should deny >50 resource blast radius")
	}
}

func TestWarnMode(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementWarn)
	e.LoadBuiltins()

	skill := &core.Skill{Name: "aws.s3.sync", RiskLevel: core.RiskMedium}
	result := e.Evaluate(skill, map[string]interface{}{"acl": "public-read"}, "staging")
	if result.Denied {
		t.Error("warn mode should not deny — only warn")
	}
}

func TestPolicyAppliesPatternMatching(t *testing.T) {
	e := policy.NewEngine(policy.EnforcementDeny)
	e.Register(&policy.Policy{
		Name:        "test_pattern",
		Enforcement: policy.EnforcementDeny,
		Severity:    policy.SeverityCritical,
		AppliesTo:   []string{"aws.s3.*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			return true, "always fails"
		},
	})

	// Should match
	s3Skill := &core.Skill{Name: "aws.s3.sync"}
	result := e.Evaluate(s3Skill, nil, "staging")
	if result.Passed {
		t.Error("should match aws.s3.* pattern")
	}

	// Should not match
	ec2Skill := &core.Skill{Name: "aws.ec2.list"}
	result = e.Evaluate(ec2Skill, nil, "staging")
	if !result.Passed {
		t.Error("should not match aws.s3.* for ec2 skill")
	}
}
