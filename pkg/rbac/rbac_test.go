package rbac_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/rbac"
)

func TestViewerPermissions(t *testing.T) {
	e := rbac.NewEngine()
	e.AddUser("viewer1", rbac.RoleViewer, nil)

	lowSkill := &core.Skill{Name: "aws.ec2.list", RiskLevel: core.RiskLow}
	ok, _ := e.CanExecute("viewer1", lowSkill, "staging")
	if !ok {
		t.Error("viewer should access LOW risk in staging")
	}

	highSkill := &core.Skill{Name: "k8s.deploy", RiskLevel: core.RiskHigh}
	ok, reason := e.CanExecute("viewer1", highSkill, "staging")
	if ok {
		t.Error("viewer should NOT access HIGH risk")
	}
	if reason == "" {
		t.Error("should provide denial reason")
	}

	ok, _ = e.CanExecute("viewer1", lowSkill, "production")
	if ok {
		t.Error("viewer should NOT access production")
	}
}

func TestOperatorPermissions(t *testing.T) {
	e := rbac.NewEngine()
	e.AddUser("ops1", rbac.RoleOperator, []string{"platform"})

	medSkill := &core.Skill{Name: "aws.ec2.scale", RiskLevel: core.RiskMedium}
	ok, _ := e.CanExecute("ops1", medSkill, "staging")
	if !ok {
		t.Error("operator should access MEDIUM risk in staging")
	}

	ok, _ = e.CanExecute("ops1", medSkill, "production")
	if ok {
		t.Error("operator should NOT access production")
	}
}

func TestAdminPermissions(t *testing.T) {
	e := rbac.NewEngine()
	e.AddUser("admin1", rbac.RoleAdmin, nil)

	highSkill := &core.Skill{Name: "k8s.deploy", RiskLevel: core.RiskHigh}
	ok, _ := e.CanExecute("admin1", highSkill, "production")
	if !ok {
		t.Error("admin should access HIGH risk in production")
	}

	critSkill := &core.Skill{Name: "terraform.apply", RiskLevel: core.RiskCritical}
	ok, _ = e.CanExecute("admin1", critSkill, "production")
	if ok {
		t.Error("admin should NOT access CRITICAL risk")
	}

	if !e.CanApprove("admin1") {
		t.Error("admin should be able to approve")
	}
}

func TestSuperAdminPermissions(t *testing.T) {
	e := rbac.NewEngine()
	e.AddUser("super1", rbac.RoleSuperAdmin, nil)

	critSkill := &core.Skill{Name: "terraform.apply", RiskLevel: core.RiskCritical}
	ok, _ := e.CanExecute("super1", critSkill, "production")
	if !ok {
		t.Error("superadmin should access CRITICAL risk in production")
	}
}

func TestUnknownUser(t *testing.T) {
	e := rbac.NewEngine()
	skill := &core.Skill{Name: "aws.ec2.list", RiskLevel: core.RiskLow}
	ok, _ := e.CanExecute("nonexistent", skill, "staging")
	if ok {
		t.Error("unknown user should be denied")
	}
}

func TestDisabledRBAC(t *testing.T) {
	e := rbac.NewEngine()
	e.SetEnabled(false)

	skill := &core.Skill{Name: "terraform.apply", RiskLevel: core.RiskCritical}
	ok, _ := e.CanExecute("nobody", skill, "production")
	if !ok {
		t.Error("disabled RBAC should allow everything")
	}
}
