package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// AWSSkills returns all built-in AWS skill definitions.
func AWSSkills() []*core.Skill {
	return []*core.Skill{
		// ── Compute ──────────────────────────────────────────
		{
			Name:        "aws.ec2.list",
			Description: "List EC2 instances with filters (region, tag, state)",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryCompute,
			Inputs: []core.SkillInput{
				{Name: "region", Type: "string", Required: false, Description: "AWS region to query", Default: "us-east-1"},
				{Name: "tags", Type: "list", Required: false, Description: "Key=Value tag filters"},
				{Name: "state", Type: "string", Required: false, Description: "Instance state filter (running, stopped, etc.)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "instances", Type: "list", Description: "List of EC2 instance details"},
				{Name: "count", Type: "int", Description: "Total matching instance count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws ec2 describe-instances --filters",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation, no rollback needed"},
		},
		{
			Name:        "aws.ec2.scale",
			Description: "Scale Auto Scaling Groups up or down",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryCompute,
			Inputs: []core.SkillInput{
				{Name: "asg_name", Type: "string", Required: true, Description: "Auto Scaling Group name"},
				{Name: "desired_capacity", Type: "int", Required: true, Description: "Target instance count"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "previous_capacity", Type: "int", Description: "Previous desired capacity"},
				{Name: "new_capacity", Type: "int", Description: "New desired capacity"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws autoscaling update-auto-scaling-group --desired-capacity",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "Restore previous desired capacity via aws autoscaling update-auto-scaling-group"},
		},
		{
			Name:        "aws.lambda.deploy",
			Description: "Deploy or update Lambda functions",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryCompute,
			Inputs: []core.SkillInput{
				{Name: "function_name", Type: "string", Required: true, Description: "Lambda function name"},
				{Name: "zip_file", Type: "string", Required: false, Description: "Path to deployment package"},
				{Name: "image_uri", Type: "string", Required: false, Description: "Container image URI"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "function_arn", Type: "string", Description: "Deployed function ARN"},
				{Name: "version", Type: "string", Description: "Published version number"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws lambda update-function-code",
				Timeout: 120 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "Deploy previous version using aws lambda update-function-code with prior package"},
		},

		// ── Storage ──────────────────────────────────────────
		{
			Name:        "aws.s3.audit",
			Description: "Audit S3 buckets for public access, encryption, and versioning",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryStorage,
			Inputs: []core.SkillInput{
				{Name: "bucket_name", Type: "string", Required: false, Description: "Specific bucket to audit (all if omitted)"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "findings", Type: "list", Description: "Security findings per bucket"},
				{Name: "buckets_scanned", Type: "int", Description: "Number of buckets audited"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws s3api get-bucket-acl && get-bucket-encryption && get-bucket-versioning",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "aws.s3.sync",
			Description: "Sync S3 bucket contents between environments",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryStorage,
			Inputs: []core.SkillInput{
				{Name: "source", Type: "string", Required: true, Description: "Source bucket URI (s3://bucket/prefix)"},
				{Name: "destination", Type: "string", Required: true, Description: "Destination bucket URI"},
				{Name: "delete", Type: "bool", Required: false, Description: "Delete files in destination not in source"},
			},
			Outputs: []core.SkillOutput{
				{Name: "files_synced", Type: "int", Description: "Number of files synced"},
				{Name: "bytes_transferred", Type: "int", Description: "Total bytes transferred"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws s3 sync",
				Timeout: 600 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Reverse sync from destination back to source"},
		},

		// ── Networking ───────────────────────────────────────
		{
			Name:        "aws.vpc.inspect",
			Description: "Inspect VPC topology, subnets, route tables, and NACLs",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryNetworking,
			Inputs: []core.SkillInput{
				{Name: "vpc_id", Type: "string", Required: true, Description: "VPC ID to inspect"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "topology", Type: "object", Description: "VPC topology including subnets, routes, NACLs"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws ec2 describe-vpcs && describe-subnets && describe-route-tables",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "aws.sg.audit",
			Description: "Audit Security Groups for overly permissive rules (0.0.0.0/0)",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryNetworking,
			Inputs: []core.SkillInput{
				{Name: "vpc_id", Type: "string", Required: false, Description: "Limit to specific VPC"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "findings", Type: "list", Description: "Overly permissive security group rules"},
				{Name: "groups_scanned", Type: "int", Description: "Total security groups scanned"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws ec2 describe-security-groups",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},

		// ── Security ─────────────────────────────────────────
		{
			Name:        "aws.iam.audit",
			Description: "Audit IAM roles, policies, and unused permissions",
			Provider:    core.ProviderAWS,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "max_age_days", Type: "int", Required: false, Description: "Flag credentials unused for N days", Default: "90"},
			},
			Outputs: []core.SkillOutput{
				{Name: "unused_roles", Type: "list", Description: "IAM roles with no recent activity"},
				{Name: "overprivileged_policies", Type: "list", Description: "Policies with excessive permissions"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws iam get-credential-report && list-roles && list-policies",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "aws.secrets.rotate",
			Description: "Rotate secrets in AWS Secrets Manager",
			Provider:    core.ProviderAWS,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "secret_id", Type: "string", Required: true, Description: "Secret name or ARN"},
				{Name: "region", Type: "string", Required: false, Description: "AWS region", Default: "us-east-1"},
			},
			Outputs: []core.SkillOutput{
				{Name: "version_id", Type: "string", Description: "New secret version ID"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws secretsmanager rotate-secret",
				Timeout: 60 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "Restore previous secret version via aws secretsmanager update-secret-version-stage"},
		},
		{
			Name:        "aws.guardduty.report",
			Description: "Summarize GuardDuty findings by severity",
			Provider:    core.ProviderAWS,
			Category:    core.CategorySecurity,
			Inputs: []core.SkillInput{
				{Name: "detector_id", Type: "string", Required: true, Description: "GuardDuty detector ID"},
				{Name: "severity", Type: "string", Required: false, Description: "Minimum severity filter (LOW, MEDIUM, HIGH)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "findings", Type: "list", Description: "GuardDuty findings summary"},
				{Name: "count", Type: "int", Description: "Total finding count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws guardduty list-findings && get-findings",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},

		// ── Observability ────────────────────────────────────
		{
			Name:        "aws.cloudwatch.query",
			Description: "Query CloudWatch Logs Insights",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "log_group", Type: "string", Required: true, Description: "CloudWatch Log Group name"},
				{Name: "query", Type: "string", Required: true, Description: "Logs Insights query string"},
				{Name: "start_time", Type: "string", Required: false, Description: "Query start time (ISO 8601)"},
				{Name: "end_time", Type: "string", Required: false, Description: "Query end time (ISO 8601)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "results", Type: "list", Description: "Query result records"},
				{Name: "records_matched", Type: "int", Description: "Number of log records matched"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws logs start-query && get-query-results",
				Timeout: 120 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},

		// ── Cost Management ──────────────────────────────────
		{
			Name:        "aws.cost.report",
			Description: "Generate AWS Cost Explorer report by service or tag",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryCost,
			Inputs: []core.SkillInput{
				{Name: "granularity", Type: "string", Required: false, Description: "DAILY, MONTHLY, or HOURLY", Default: "MONTHLY"},
				{Name: "start_date", Type: "string", Required: true, Description: "Report start date (YYYY-MM-DD)"},
				{Name: "end_date", Type: "string", Required: true, Description: "Report end date (YYYY-MM-DD)"},
				{Name: "group_by", Type: "string", Required: false, Description: "Group by dimension (SERVICE, TAG, etc.)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "cost_data", Type: "list", Description: "Cost breakdown by group"},
				{Name: "total_cost", Type: "string", Description: "Total cost for the period"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws ce get-cost-and-usage",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "aws.rightsizing.suggest",
			Description: "Suggest EC2 rightsizing recommendations",
			Provider:    core.ProviderAWS,
			Category:    core.CategoryCost,
			Inputs: []core.SkillInput{
				{Name: "service", Type: "string", Required: false, Description: "Service to analyze (default: AmazonEC2)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "recommendations", Type: "list", Description: "Rightsizing recommendations with estimated savings"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "aws ce get-rightsizing-recommendation",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
	}
}
