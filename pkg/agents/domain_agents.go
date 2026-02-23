package agents

import (
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

type sreAgent struct{}
type platformAgent struct{}
type infraAgent struct{}

// NewSREAgent creates SRE domain agent.
func NewSREAgent() DomainAgent { return &sreAgent{} }

// NewPlatformAgent creates platform engineering domain agent.
func NewPlatformAgent() DomainAgent { return &platformAgent{} }

// NewInfraAgent creates infrastructure domain agent.
func NewInfraAgent() DomainAgent { return &infraAgent{} }

func (a *sreAgent) Role() Role { return RoleSRE }

func (a *sreAgent) Assess(ctx EvaluationContext, _ *MemoryStore) Assessment {
	proposals := []Proposal{
		{
			Agent:       RoleSRE,
			Summary:     "Assess burn rate and active alert surface before remediation",
			Skills:      []string{"datadog.slo.burnrate", "datadog.alert.list", "prometheus.query"},
			RiskLevel:   core.RiskLow,
			BlastRadius: 0,
			Params: map[string]interface{}{
				"slo_id":  ctx.Service,
				"window":  "1h",
				"service": ctx.Service,
			},
			RollbackPlan: "Read-only assessment",
		},
	}
	if strings.EqualFold(ctx.Urgency, "P0") || strings.Contains(strings.ToLower(strings.Join(ctx.Symptoms, " ")), "5xx") {
		proposals = append(proposals, Proposal{
			Agent:                RoleSRE,
			Summary:              "Execute bounded reliability auto-remediation playbook",
			Skills:               []string{"platform.reliability.autoremediate"},
			RiskLevel:            core.RiskHigh,
			BlastRadius:          1,
			RequiresConfirmation: true,
			Params: map[string]interface{}{
				"playbook":    "latency-spike-v1",
				"target":      ctx.Service,
				"environment": ctx.Environment,
			},
			RollbackPlan: "Use rollback token from platform.reliability.autoremediate",
		})
	}

	return Assessment{
		Agent:        RoleSRE,
		Confidence:   confidenceByUrgency(ctx.Urgency, 84),
		Observations: []string{"SLO and error budget should gate action sequencing"},
		Proposals:    proposals,
	}
}

func (a *platformAgent) Role() Role { return RolePlatform }

func (a *platformAgent) Assess(ctx EvaluationContext, _ *MemoryStore) Assessment {
	proposals := []Proposal{
		{
			Agent:       RolePlatform,
			Summary:     "Evaluate deployment risk and governance gate status",
			Skills:      []string{"github.pr.change_risk", "terraform.cloud.policy.check"},
			RiskLevel:   core.RiskLow,
			BlastRadius: 0,
			Params: map[string]interface{}{
				"repo":      "org/platform",
				"pr_number": 1,
			},
			RollbackPlan: "Read-only governance checks",
		},
	}
	if strings.EqualFold(ctx.Urgency, "P0") {
		proposals = append(proposals, Proposal{
			Agent:                RolePlatform,
			Summary:              "Restart affected workload revision if health remains degraded",
			Skills:               []string{"k8s.pod.restart"},
			RiskLevel:            core.RiskHigh,
			BlastRadius:          1,
			RequiresConfirmation: true,
			Params: map[string]interface{}{
				"namespace":  "default",
				"deployment": ctx.Service,
			},
			RollbackPlan: "kubectl rollout undo deployment/{deployment} -n {namespace}",
		})
	}

	return Assessment{
		Agent:        RolePlatform,
		Confidence:   confidenceByUrgency(ctx.Urgency, 78),
		Observations: []string{"Platform mutations should stay bounded to one service at a time"},
		Proposals:    proposals,
	}
}

func (a *infraAgent) Role() Role { return RoleInfra }

func (a *infraAgent) Assess(ctx EvaluationContext, _ *MemoryStore) Assessment {
	proposals := []Proposal{
		{
			Agent:       RoleInfra,
			Summary:     "Run infrastructure signal checks for root-cause isolation",
			Skills:      []string{"aws.cloudwatch.query", "cloudtrail.event.search"},
			RiskLevel:   core.RiskLow,
			BlastRadius: 0,
			Params: map[string]interface{}{
				"log_group": "/platform/" + ctx.Service,
				"query":     "fields @timestamp,@message | sort @timestamp desc | limit 50",
			},
			RollbackPlan: "Read-only diagnostics",
		},
	}
	if strings.EqualFold(ctx.Urgency, "P0") && strings.EqualFold(ctx.SignalType, "node") {
		proposals = append(proposals, Proposal{
			Agent:                RoleInfra,
			Summary:              "Cordon and drain unhealthy node after controller approval",
			Skills:               []string{"k8s.node.cordon", "k8s.node.drain"},
			RiskLevel:            core.RiskCritical,
			BlastRadius:          2,
			RequiresConfirmation: true,
			Params: map[string]interface{}{
				"node": ctx.Service + "-node-1",
			},
			RollbackPlan: "kubectl uncordon {node}",
		})
	}

	return Assessment{
		Agent:        RoleInfra,
		Confidence:   confidenceByUrgency(ctx.Urgency, 80),
		Observations: []string{"Infrastructure actions must follow read-before-write discipline"},
		Proposals:    proposals,
	}
}

func confidenceByUrgency(urgency string, base int) int {
	if strings.EqualFold(urgency, "P0") {
		return base + 5
	}
	if strings.EqualFold(urgency, "P1") {
		return base + 2
	}
	return base
}
