package agents

import (
	"time"

	"github.com/parth14193/inframesh/pkg/core"
)

// Role identifies a coordinating or domain agent.
type Role string

const (
	RoleController Role = "controller"
	RoleSRE        Role = "sre-agent"
	RolePlatform   Role = "platform-agent"
	RoleInfra      Role = "infra-agent"
)

// EvaluationContext is the shared context broadcast to all agents.
type EvaluationContext struct {
	Environment    string
	Urgency        string
	Service        string
	SignalType     string
	Symptoms       []string
	DesiredOutcome string
	Timestamp      time.Time
}

// Proposal is an executable recommendation from a domain agent.
type Proposal struct {
	Agent                Role
	Summary              string
	Skills               []string
	Params               map[string]interface{}
	RiskLevel            core.RiskLevel
	BlastRadius          int
	RequiresConfirmation bool
	RollbackPlan         string
}

// Assessment is the full domain-agent response back to controller.
type Assessment struct {
	Agent        Role
	Confidence   int
	Observations []string
	Proposals    []Proposal
}

// Decision is the controller's coordinated output.
type Decision struct {
	Controller           Role
	Context              EvaluationContext
	Assessments          []Assessment
	SelectedProposals    []Proposal
	RequiresHumanApproval bool
	Reasoning            []string
}

// DomainAgent describes a specialist agent.
type DomainAgent interface {
	Role() Role
	Assess(ctx EvaluationContext, memory *MemoryStore) Assessment
}
