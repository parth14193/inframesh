package agents

import (
	"sort"
	"strconv"
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

// Controller coordinates all domain agents and emits an execution decision.
type Controller struct {
	agents []DomainAgent
	memory *MemoryStore
}

// NewController creates a default multi-agent controller.
func NewController(memory *MemoryStore) *Controller {
	if memory == nil {
		memory = NewMemoryStore()
	}
	return &Controller{
		agents: []DomainAgent{
			NewSREAgent(),
			NewPlatformAgent(),
			NewInfraAgent(),
		},
		memory: memory,
	}
}

// Decide runs all domain assessments and synthesizes a coordinated decision.
func (c *Controller) Decide(ctx EvaluationContext) Decision {
	assessments := make([]Assessment, 0, len(c.agents))
	for _, agent := range c.agents {
		assessments = append(assessments, agent.Assess(ctx, c.memory))
	}

	selected := selectProposals(assessments, ctx)
	requiresHuman := requiresHumanApproval(ctx, selected)
	reasoning := buildReasoning(ctx, assessments, selected, requiresHuman)

	decision := Decision{
		Controller:            RoleController,
		Context:               ctx,
		Assessments:           assessments,
		SelectedProposals:     selected,
		RequiresHumanApproval: requiresHuman,
		Reasoning:             reasoning,
	}
	c.memory.RecordDecision(decision)
	return decision
}

func selectProposals(assessments []Assessment, ctx EvaluationContext) []Proposal {
	// Observe-first: always include low-risk diagnostic proposal from each agent.
	selected := make([]Proposal, 0, len(assessments)*2)
	for _, a := range assessments {
		if p := firstByRisk(a.Proposals, core.RiskLow); p != nil {
			selected = append(selected, *p)
		}
	}

	// In P0, include the top mutation candidate by confidence and bounded blast radius.
	if strings.EqualFold(ctx.Urgency, "P0") {
		type candidate struct {
			proposal   Proposal
			confidence int
		}
		var candidates []candidate
		for _, a := range assessments {
			for _, p := range a.Proposals {
				if p.RiskLevel >= core.RiskHigh && p.BlastRadius <= 2 {
					candidates = append(candidates, candidate{proposal: p, confidence: a.Confidence})
				}
			}
		}
		sort.Slice(candidates, func(i, j int) bool {
			if candidates[i].confidence == candidates[j].confidence {
				return candidates[i].proposal.RiskLevel < candidates[j].proposal.RiskLevel
			}
			return candidates[i].confidence > candidates[j].confidence
		})
		if len(candidates) > 0 {
			selected = append(selected, candidates[0].proposal)
		}
	}
	return selected
}

func firstByRisk(proposals []Proposal, risk core.RiskLevel) *Proposal {
	for _, p := range proposals {
		if p.RiskLevel == risk {
			cp := p
			return &cp
		}
	}
	return nil
}

func requiresHumanApproval(ctx EvaluationContext, proposals []Proposal) bool {
	if isProduction(ctx.Environment) {
		return true
	}
	for _, p := range proposals {
		if p.RequiresConfirmation || p.RiskLevel >= core.RiskHigh {
			return true
		}
	}
	return false
}

func isProduction(env string) bool {
	e := strings.ToLower(strings.TrimSpace(env))
	return e == "production" || e == "prod" || e == "prd"
}

func buildReasoning(ctx EvaluationContext, assessments []Assessment, proposals []Proposal, requiresHuman bool) []string {
	reasoning := []string{
		"Controller collected parallel assessments from SRE, Platform, and Infra agents.",
		"Controller enforced observe-before-act by selecting low-risk diagnostics first.",
	}
	if strings.EqualFold(ctx.Urgency, "P0") {
		reasoning = append(reasoning, "P0 urgency enabled one bounded remediation candidate after diagnostic proposals.")
	}
	if requiresHuman {
		reasoning = append(reasoning, "Human approval required due to production context or high-risk mutation.")
	}
	if len(proposals) == 0 {
		reasoning = append(reasoning, "No safe proposal selected; fallback is manual triage.")
	}
	for _, a := range assessments {
		reasoning = append(reasoning, string(a.Agent)+" confidence="+strconv.Itoa(a.Confidence))
	}
	return reasoning
}
