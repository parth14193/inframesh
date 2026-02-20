package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// CICDSkills returns all built-in CI/CD skill definitions.
func CICDSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "github.actions.trigger",
			Description: "Trigger GitHub Actions workflows",
			Provider:    core.ProviderGitHub,
			Category:    core.CategoryCICD,
			Inputs: []core.SkillInput{
				{Name: "repo", Type: "string", Required: true, Description: "Repository (owner/name)"},
				{Name: "workflow", Type: "string", Required: true, Description: "Workflow file name or ID"},
				{Name: "ref", Type: "string", Required: false, Description: "Branch or tag ref", Default: "main"},
				{Name: "inputs", Type: "list", Required: false, Description: "Workflow input parameters"},
			},
			Outputs: []core.SkillOutput{
				{Name: "run_id", Type: "string", Description: "Workflow run ID"},
				{Name: "status", Type: "string", Description: "Trigger status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "Cancel run via POST /repos/{owner}/{repo}/actions/runs/{run_id}/cancel",
			},
		},
		{
			Name:        "github.actions.status",
			Description: "Get GitHub Actions workflow run status",
			Provider:    core.ProviderGitHub,
			Category:    core.CategoryCICD,
			Inputs: []core.SkillInput{
				{Name: "repo", Type: "string", Required: true, Description: "Repository (owner/name)"},
				{Name: "run_id", Type: "string", Required: false, Description: "Specific run ID (latest if omitted)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Run status (queued, in_progress, completed)"},
				{Name: "conclusion", Type: "string", Description: "Run conclusion (success, failure, etc.)"},
				{Name: "duration", Type: "string", Description: "Run duration"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /repos/{owner}/{repo}/actions/runs/{run_id}",
				Timeout: 15 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "gitlab.pipeline.status",
			Description: "Get GitLab CI/CD pipeline status",
			Provider:    core.ProviderGitLab,
			Category:    core.CategoryCICD,
			Inputs: []core.SkillInput{
				{Name: "project_id", Type: "string", Required: true, Description: "GitLab project ID or path"},
				{Name: "pipeline_id", Type: "string", Required: false, Description: "Specific pipeline ID (latest if omitted)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Pipeline status"},
				{Name: "stages", Type: "list", Description: "Stage statuses"},
				{Name: "duration", Type: "string", Description: "Pipeline duration"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /api/v4/projects/{id}/pipelines/{pipeline_id}",
				Timeout: 15 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "jenkins.job.trigger",
			Description: "Trigger Jenkins jobs",
			Provider:    core.ProviderJenkins,
			Category:    core.CategoryCICD,
			Inputs: []core.SkillInput{
				{Name: "job_name", Type: "string", Required: true, Description: "Jenkins job name"},
				{Name: "parameters", Type: "list", Required: false, Description: "Job parameters"},
			},
			Outputs: []core.SkillOutput{
				{Name: "build_number", Type: "int", Description: "Triggered build number"},
				{Name: "queue_id", Type: "int", Description: "Queue item ID"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "POST /job/{job_name}/build",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "POST /job/{job_name}/{build_number}/stop to abort the build",
			},
		},
	}
}
