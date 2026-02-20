package runbook_test

import (
	"testing"

	"github.com/parth14193/ownbot/pkg/runbook"
)

func TestLoadBuiltins(t *testing.T) {
	e := runbook.NewEngine()
	e.LoadBuiltins()
	rbs := e.List()
	if len(rbs) < 5 {
		t.Errorf("expected at least 5 built-in runbooks, got %d", len(rbs))
	}
}

func TestGetRunbook(t *testing.T) {
	e := runbook.NewEngine()
	e.LoadBuiltins()

	rb, err := e.Get("high-cpu-response")
	if err != nil {
		t.Fatalf("expected high-cpu-response: %v", err)
	}
	if len(rb.Steps) < 5 {
		t.Error("high-cpu runbook should have multiple steps")
	}
}

func TestGetMissing(t *testing.T) {
	e := runbook.NewEngine()
	_, err := e.Get("nonexistent")
	if err == nil {
		t.Error("should error for missing runbook")
	}
}

func TestRegisterDuplicate(t *testing.T) {
	e := runbook.NewEngine()
	rb := &runbook.Runbook{Name: "test", Steps: []runbook.Step{{Name: "s1", Type: runbook.StepManual}}}
	_ = e.Register(rb)
	err := e.Register(rb)
	if err == nil {
		t.Error("should error on duplicate")
	}
}

func TestValidate(t *testing.T) {
	e := runbook.NewEngine()
	empty := &runbook.Runbook{Name: ""}
	errs := e.Validate(empty)
	if len(errs) == 0 {
		t.Error("should have validation errors")
	}

	valid := &runbook.Runbook{Name: "test", Steps: []runbook.Step{{Name: "s1", Type: runbook.StepManual, Description: "do thing"}}}
	errs = e.Validate(valid)
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestSimulateRun(t *testing.T) {
	e := runbook.NewEngine()
	e.LoadBuiltins()
	rb, _ := e.Get("deployment-rollback")
	log := e.SimulateRun(rb)
	if log.Status != "simulated" {
		t.Errorf("expected simulated, got %s", log.Status)
	}
	if len(log.StepResults) != len(rb.Steps) {
		t.Errorf("expected %d step results, got %d", len(rb.Steps), len(log.StepResults))
	}
}

func TestRunbookRender(t *testing.T) {
	e := runbook.NewEngine()
	e.LoadBuiltins()
	rb, _ := e.Get("incident-response")
	rendered := rb.Render()
	if len(rendered) == 0 {
		t.Error("render should produce output")
	}
}
