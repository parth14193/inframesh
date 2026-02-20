package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// CostSkills returns all built-in cost management skill definitions.
func CostSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "infracost.estimate",
			Description: "Estimate cost delta for Terraform changes",
			Provider:    core.ProviderInfracost,
			Category:    core.CategoryCost,
			Inputs: []core.SkillInput{
				{Name: "path", Type: "string", Required: true, Description: "Path to Terraform directory or plan file"},
				{Name: "format", Type: "string", Required: false, Description: "Output format (table, json, html)", Default: "table"},
			},
			Outputs: []core.SkillOutput{
				{Name: "monthly_cost", Type: "string", Description: "Estimated monthly cost"},
				{Name: "cost_delta", Type: "string", Description: "Cost change from current state"},
				{Name: "breakdown", Type: "list", Description: "Cost breakdown by resource"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "infracost breakdown --path {path} --format {format}",
				Timeout: 120 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only cost estimation"},
		},
	}
}
