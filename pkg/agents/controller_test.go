package agents_test

import (
	"testing"
	"time"

	"github.com/parth14193/ownbot/pkg/agents"
)

func TestControllerCoordinatesAllAgents(t *testing.T) {
	controller := agents.NewController(nil)
	ctx := agents.EvaluationContext{
		Environment: "staging",
		Urgency:     "P1",
		Service:     "checkout-api",
		SignalType:  "latency",
		Symptoms:    []string{"latency spike"},
		Timestamp:   time.Now(),
	}

	decision := controller.Decide(ctx)
	if len(decision.Assessments) != 3 {
		t.Fatalf("expected 3 agent assessments, got %d", len(decision.Assessments))
	}
	if len(decision.SelectedProposals) < 3 {
		t.Fatalf("expected observe-first proposals from all agents, got %d", len(decision.SelectedProposals))
	}
}

func TestControllerRequiresHumanForProduction(t *testing.T) {
	controller := agents.NewController(nil)
	ctx := agents.EvaluationContext{
		Environment: "production",
		Urgency:     "P2",
		Service:     "checkout-api",
		SignalType:  "latency",
		Timestamp:   time.Now(),
	}

	decision := controller.Decide(ctx)
	if !decision.RequiresHumanApproval {
		t.Fatal("expected production decisions to require human approval")
	}
}

func TestControllerAddsBoundedMutationForP0(t *testing.T) {
	controller := agents.NewController(nil)
	ctx := agents.EvaluationContext{
		Environment: "staging",
		Urgency:     "P0",
		Service:     "checkout-api",
		SignalType:  "latency",
		Symptoms:    []string{"5xx spike"},
		Timestamp:   time.Now(),
	}

	decision := controller.Decide(ctx)
	foundMutation := false
	for _, p := range decision.SelectedProposals {
		if p.RiskLevel.String() == "HIGH" || p.RiskLevel.String() == "CRITICAL" {
			foundMutation = true
			break
		}
	}
	if !foundMutation {
		t.Fatal("expected bounded mutation proposal for P0 context")
	}
}
