package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// PlatformAutomanageSkills returns production engineering, SRE, governance, and compliance skill packs.
func PlatformAutomanageSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "prometheus.query",
			Description: "Run Prometheus instant or range queries for SLO and health signals",
			Provider:    core.ProviderPrometheus,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "query", Type: "string", Required: true, Description: "PromQL query expression"},
				{Name: "window", Type: "string", Required: false, Description: "Query window (e.g., 5m, 1h)", Default: "5m"},
			},
			Outputs: []core.SkillOutput{
				{Name: "series", Type: "list", Description: "Result time series"},
				{Name: "sample_count", Type: "int", Description: "Number of returned samples"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /api/v1/query_range?query={query}&window={window}",
				Timeout: 20 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "datadog.slo.burnrate",
			Description: "Evaluate SLO burn rate and error budget consumption",
			Provider:    core.ProviderDatadog,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "slo_id", Type: "string", Required: true, Description: "Datadog SLO identifier"},
				{Name: "window", Type: "string", Required: false, Description: "Evaluation window", Default: "1h"},
			},
			Outputs: []core.SkillOutput{
				{Name: "burn_rate", Type: "string", Description: "Current burn rate"},
				{Name: "error_budget_remaining", Type: "string", Description: "Remaining error budget percentage"},
				{Name: "recommendation", Type: "string", Description: "Action recommendation"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /api/v1/slo/{slo_id}/burn_rate?window={window}",
				Timeout: 20 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "sre.incident.timeline.generate",
			Description: "Generate incident timeline from alerts, deployments, and events",
			Provider:    core.ProviderCustom,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "incident_id", Type: "string", Required: true, Description: "Incident identifier"},
				{Name: "start_time", Type: "string", Required: true, Description: "Incident start timestamp"},
				{Name: "end_time", Type: "string", Required: false, Description: "Incident end timestamp"},
			},
			Outputs: []core.SkillOutput{
				{Name: "timeline", Type: "list", Description: "Chronological event timeline"},
				{Name: "root_cause_hypothesis", Type: "string", Description: "Best-effort root cause hypothesis"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore internal timeline generate --incident {incident_id} --start {start_time} --end {end_time}",
				Timeout: 45 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only synthesis operation"},
		},
		{
			Name:        "k8s.pod.restart",
			Description: "Restart a Kubernetes deployment with safety checks",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "namespace", Type: "string", Required: true, Description: "Kubernetes namespace"},
				{Name: "deployment", Type: "string", Required: true, Description: "Deployment name"},
				{Name: "context", Type: "string", Required: false, Description: "kubectl context"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Restart operation status"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl rollout restart deployment/{deployment} -n {namespace} --context {context}",
				Timeout: 90 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "kubectl rollout undo deployment/{deployment} -n {namespace}",
			},
		},
		{
			Name:        "k8s.node.cordon",
			Description: "Mark node unschedulable before maintenance",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "node", Type: "string", Required: true, Description: "Node name"},
				{Name: "context", Type: "string", Required: false, Description: "kubectl context"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Cordon status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl cordon {node} --context {context}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "kubectl uncordon {node} --context {context}",
			},
		},
		{
			Name:        "k8s.node.drain",
			Description: "Drain node safely for maintenance or failure remediation",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "node", Type: "string", Required: true, Description: "Node name"},
				{Name: "ignore_daemonsets", Type: "bool", Required: false, Description: "Ignore daemonsets", Default: "true"},
				{Name: "delete_emptydir_data", Type: "bool", Required: false, Description: "Delete emptyDir data", Default: "false"},
				{Name: "context", Type: "string", Required: false, Description: "kubectl context"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Drain status"},
			},
			RiskLevel:            core.RiskCritical,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl drain {node} --ignore-daemonsets={ignore_daemonsets} --delete-emptydir-data={delete_emptydir_data} --context {context}",
				Timeout: 180 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "kubectl uncordon {node} --context {context}",
			},
		},
		{
			Name:        "terraform.cloud.policy.check",
			Description: "Evaluate Terraform Cloud policy checks for a workspace run",
			Provider:    core.ProviderTerraform,
			Category:    core.CategoryGovernance,
			Inputs: []core.SkillInput{
				{Name: "workspace", Type: "string", Required: true, Description: "Terraform Cloud workspace"},
				{Name: "run_id", Type: "string", Required: true, Description: "Run identifier"},
			},
			Outputs: []core.SkillOutput{
				{Name: "passed", Type: "bool", Description: "Policy check result"},
				{Name: "violations", Type: "list", Description: "Policy violations"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /api/v2/runs/{run_id}/policy-checks",
				Timeout: 20 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "github.pr.change_risk",
			Description: "Assess PR deployment risk from touched infra paths and service criticality",
			Provider:    core.ProviderGitHub,
			Category:    core.CategoryGovernance,
			Inputs: []core.SkillInput{
				{Name: "repo", Type: "string", Required: true, Description: "Repository owner/name"},
				{Name: "pr_number", Type: "int", Required: true, Description: "Pull request number"},
			},
			Outputs: []core.SkillOutput{
				{Name: "risk_level", Type: "string", Description: "Derived risk level"},
				{Name: "requires_peer_review", Type: "bool", Description: "Whether peer approval is required"},
				{Name: "affected_services", Type: "list", Description: "Service ownership map"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /repos/{repo}/pulls/{pr_number}/files",
				Timeout: 20 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only analysis"},
		},
		{
			Name:        "governance.change.approve",
			Description: "Apply governance approval gate for production changes",
			Provider:    core.ProviderCustom,
			Category:    core.CategoryGovernance,
			Inputs: []core.SkillInput{
				{Name: "change_ticket", Type: "string", Required: true, Description: "Approved change ticket ID"},
				{Name: "approver", Type: "string", Required: true, Description: "Approver identity"},
				{Name: "environment", Type: "string", Required: true, Description: "Target environment"},
			},
			Outputs: []core.SkillOutput{
				{Name: "approved", Type: "bool", Description: "Approval outcome"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore internal governance approve --ticket {change_ticket} --approver {approver} --env {environment}",
				Timeout: 15 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "infracore internal governance revoke --ticket {change_ticket}",
			},
		},
		{
			Name:        "compliance.evidence.collect",
			Description: "Collect compliance evidence artifacts for audits",
			Provider:    core.ProviderCustom,
			Category:    core.CategoryGovernance,
			Inputs: []core.SkillInput{
				{Name: "framework", Type: "string", Required: true, Description: "Framework name (CIS, SOC2, HIPAA)"},
				{Name: "period", Type: "string", Required: false, Description: "Evidence period", Default: "30d"},
			},
			Outputs: []core.SkillOutput{
				{Name: "artifacts", Type: "list", Description: "Collected evidence references"},
				{Name: "missing_controls", Type: "list", Description: "Missing or incomplete controls"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore compliance collect --framework {framework} --period {period}",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only collection"},
		},
		{
			Name:        "cloudtrail.event.search",
			Description: "Search CloudTrail for governance and forensics investigations",
			Provider:    core.ProviderAWS,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "lookup_attribute", Type: "string", Required: true, Description: "Lookup attribute key"},
				{Name: "value", Type: "string", Required: true, Description: "Lookup value"},
				{Name: "start_time", Type: "string", Required: false, Description: "Start time"},
				{Name: "end_time", Type: "string", Required: false, Description: "End time"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "events", Type: "list", Description: "Matching CloudTrail events"},
				{Name: "count", Type: "int", Description: "Event count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws cloudtrail lookup-events --lookup-attributes AttributeKey={lookup_attribute},AttributeValue={value} --start-time {start_time} --end-time {end_time} --region {region}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "security.iam.least_privilege.diff",
			Description: "Diff current IAM permissions against least-privilege policy baseline",
			Provider:    core.ProviderAWS,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "principal_arn", Type: "string", Required: true, Description: "IAM principal ARN"},
				{Name: "policy_baseline", Type: "string", Required: true, Description: "Baseline policy document path"},
			},
			Outputs: []core.SkillOutput{
				{Name: "excess_permissions", Type: "list", Description: "Permissions outside baseline"},
				{Name: "missing_permissions", Type: "list", Description: "Permissions required but missing"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore security iam-diff --principal {principal_arn} --baseline {policy_baseline}",
				Timeout: 45 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only analysis"},
		},
		{
			Name:        "security.secrets.exposure.scan",
			Description: "Scan repositories and runtime logs for secret exposure indicators",
			Provider:    core.ProviderCustom,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "scope", Type: "string", Required: true, Description: "Scan scope (repo, logs, both)"},
				{Name: "target", Type: "string", Required: true, Description: "Repository name, log group, or identifier"},
			},
			Outputs: []core.SkillOutput{
				{Name: "findings", Type: "list", Description: "Potential secret exposures"},
				{Name: "severity", Type: "string", Description: "Highest severity detected"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore security secrets-scan --scope {scope} --target {target}",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only scan"},
		},
		{
			Name:        "platform.capacity.forecast",
			Description: "Forecast resource saturation and recommend scaling actions",
			Provider:    core.ProviderCustom,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "service", Type: "string", Required: true, Description: "Service name"},
				{Name: "horizon", Type: "string", Required: false, Description: "Forecast horizon", Default: "7d"},
			},
			Outputs: []core.SkillOutput{
				{Name: "forecast", Type: "list", Description: "Capacity forecast points"},
				{Name: "recommended_action", Type: "string", Description: "Scale recommendation"},
				{Name: "confidence", Type: "string", Description: "Forecast confidence"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore sre forecast --service {service} --horizon {horizon}",
				Timeout: 45 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only analysis"},
		},
		{
			Name:        "platform.reliability.autoremediate",
			Description: "Execute a bounded auto-remediation action with rollback metadata",
			Provider:    core.ProviderCustom,
			Category:    core.CategoryReliability,
			Inputs: []core.SkillInput{
				{Name: "playbook", Type: "string", Required: true, Description: "Approved remediation playbook ID"},
				{Name: "target", Type: "string", Required: true, Description: "Target resource or service"},
				{Name: "environment", Type: "string", Required: true, Description: "Target environment"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Execution status"},
				{Name: "rollback_token", Type: "string", Description: "Rollback reference token"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecScript,
				Command: "infracore sre autoremediate --playbook {playbook} --target {target} --env {environment}",
				Timeout: 120 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "infracore sre rollback --token {rollback_token}",
			},
		},
	}
}
