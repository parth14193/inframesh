package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// SecuritySkills returns all built-in security skill definitions.
func SecuritySkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "vault.policy.update",
			Description: "Update HashiCorp Vault policies",
			Provider:    core.ProviderVault,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "policy_name", Type: "string", Required: true, Description: "Vault policy name"},
				{Name: "policy_file", Type: "string", Required: true, Description: "Path to HCL policy file"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Policy update status"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "vault policy write {policy_name} {policy_file}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "vault policy write {policy_name} previous-policy.hcl",
			},
		},
		{
			Name:        "trivy.scan",
			Description: "Run container image vulnerability scans",
			Provider:    core.ProviderTrivy,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "image", Type: "string", Required: true, Description: "Container image to scan (e.g., nginx:latest)"},
				{Name: "severity", Type: "string", Required: false, Description: "Minimum severity filter", Default: "HIGH,CRITICAL"},
				{Name: "format", Type: "string", Required: false, Description: "Output format (table, json)", Default: "table"},
			},
			Outputs: []core.SkillOutput{
				{Name: "vulnerabilities", Type: "list", Description: "List of found vulnerabilities"},
				{Name: "total_count", Type: "int", Description: "Total vulnerability count"},
				{Name: "critical_count", Type: "int", Description: "Critical vulnerability count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "trivy image --severity {severity} --format {format} {image}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only scan operation"},
		},
	}
}
