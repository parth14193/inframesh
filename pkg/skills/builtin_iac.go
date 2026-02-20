package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// IaCSkills returns all Infrastructure-as-Code skill definitions.
func IaCSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "terraform.plan",
			Description: "Generate and display a Terraform plan",
			Provider:    core.ProviderTerraform,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "working_dir", Type: "string", Required: true, Description: "Terraform working directory"},
				{Name: "var_file", Type: "string", Required: false, Description: "Path to .tfvars file"},
				{Name: "target", Type: "string", Required: false, Description: "Specific resource to target"},
			},
			Outputs: []core.SkillOutput{
				{Name: "plan_output", Type: "string", Description: "Terraform plan output"},
				{Name: "resources_added", Type: "int", Description: "Resources to add"},
				{Name: "resources_changed", Type: "int", Description: "Resources to change"},
				{Name: "resources_destroyed", Type: "int", Description: "Resources to destroy"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "terraform plan -no-color",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation — plan does not mutate state"},
		},
		{
			Name:        "terraform.apply",
			Description: "Apply a Terraform plan (CRITICAL — requires explicit confirmation)",
			Provider:    core.ProviderTerraform,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "working_dir", Type: "string", Required: true, Description: "Terraform working directory"},
				{Name: "var_file", Type: "string", Required: false, Description: "Path to .tfvars file"},
				{Name: "auto_approve", Type: "bool", Required: false, Description: "Skip interactive approval"},
			},
			Outputs: []core.SkillOutput{
				{Name: "apply_output", Type: "string", Description: "Terraform apply output"},
				{Name: "resources_created", Type: "int", Description: "Resources created"},
				{Name: "resources_updated", Type: "int", Description: "Resources updated"},
				{Name: "resources_destroyed", Type: "int", Description: "Resources destroyed"},
			},
			RiskLevel:            core.RiskCritical,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "terraform apply -auto-approve",
				Timeout: 600 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "terraform destroy targeted resources or revert to previous state via terraform apply with prior .tfstate",
			},
		},
		{
			Name:        "pulumi.preview",
			Description: "Preview Pulumi stack changes",
			Provider:    core.ProviderPulumi,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "stack", Type: "string", Required: true, Description: "Pulumi stack name"},
				{Name: "working_dir", Type: "string", Required: false, Description: "Project directory"},
			},
			Outputs: []core.SkillOutput{
				{Name: "preview_output", Type: "string", Description: "Preview diff output"},
				{Name: "changes_count", Type: "int", Description: "Number of resource changes"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "pulumi preview --stack {stack}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only preview operation"},
		},
		{
			Name:        "helm.upgrade",
			Description: "Upgrade a Helm chart release",
			Provider:    core.ProviderHelm,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "release_name", Type: "string", Required: true, Description: "Helm release name"},
				{Name: "chart", Type: "string", Required: true, Description: "Chart name or path"},
				{Name: "namespace", Type: "string", Required: true, Description: "Kubernetes namespace"},
				{Name: "values_file", Type: "string", Required: false, Description: "Path to values.yaml"},
				{Name: "version", Type: "string", Required: false, Description: "Chart version"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Release status"},
				{Name: "revision", Type: "int", Description: "Release revision number"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "helm upgrade {release} {chart} -n {namespace} -f {values}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "helm rollback {release} {previous_revision} -n {namespace}",
			},
		},
		{
			Name:        "argocd.sync",
			Description: "Trigger ArgoCD application sync",
			Provider:    core.ProviderArgoCD,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "app_name", Type: "string", Required: true, Description: "ArgoCD application name"},
				{Name: "prune", Type: "bool", Required: false, Description: "Prune resources not in source"},
				{Name: "force", Type: "bool", Required: false, Description: "Force sync even if already synced"},
			},
			Outputs: []core.SkillOutput{
				{Name: "sync_status", Type: "string", Description: "Sync result status"},
				{Name: "health_status", Type: "string", Description: "Application health status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "argocd app sync {app_name}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "argocd app rollback {app_name} to previous revision",
			},
		},
	}
}
