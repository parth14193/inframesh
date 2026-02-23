package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// GCPSkills returns all built-in Google Cloud Platform skill definitions.
func GCPSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "gcp.gce.snapshot",
			Description: "Create VM snapshots in Google Compute Engine",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryCompute,
			Inputs: []core.SkillInput{
				{Name: "instance", Type: "string", Required: true, Description: "GCE instance name"},
				{Name: "zone", Type: "string", Required: true, Description: "GCE zone"},
				{Name: "snapshot_name", Type: "string", Required: false, Description: "Custom snapshot name"},
				{Name: "project", Type: "string", Required: false, Description: "GCP project ID"},
			},
			Outputs: []core.SkillOutput{
				{Name: "snapshot_id", Type: "string", Description: "Created snapshot ID"},
				{Name: "size_gb", Type: "int", Description: "Snapshot size in GB"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud compute disks snapshot {instance} --zone={zone} --snapshot-names={name}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "gcloud compute snapshots delete {snapshot_name}",
			},
		},
		{
			Name:        "gcp.gcs.lifecycle",
			Description: "Set lifecycle rules on Google Cloud Storage buckets",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryStorage,
			Inputs: []core.SkillInput{
				{Name: "bucket", Type: "string", Required: true, Description: "GCS bucket name"},
				{Name: "lifecycle_file", Type: "string", Required: true, Description: "Path to lifecycle JSON config"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Apply status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gsutil lifecycle set {lifecycle_file} gs://{bucket}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "gsutil lifecycle set previous-lifecycle.json gs://{bucket}",
			},
		},
		{
			Name:        "gcp.gke.deploy",
			Description: "Deploy an application to Google Kubernetes Engine",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "cluster_name", Type: "string", Required: true, Description: "GKE cluster name"},
				{Name: "location", Type: "string", Required: true, Description: "GKE cluster region or zone"},
				{Name: "namespace", Type: "string", Required: false, Description: "Kubernetes namespace", Default: "default"},
				{Name: "deployment", Type: "string", Required: false, Description: "Deployment name", Default: "app"},
				{Name: "image", Type: "string", Required: false, Description: "Container image", Default: "gcr.io/project/app:latest"},
				{Name: "project", Type: "string", Required: false, Description: "GCP project ID"},
			},
			Outputs: []core.SkillOutput{
				{Name: "rollout_status", Type: "string", Description: "Deployment rollout status"},
				{Name: "cluster_endpoint", Type: "string", Description: "GKE API endpoint"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud container clusters get-credentials && kubectl apply -f deployment.yaml && kubectl rollout status",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "kubectl rollout undo deployment/{name} -n {namespace}"},
		},
		{
			Name:        "gcp.gce.deploy.cpu_optimized",
			Description: "Launch or update Google Compute Engine workloads with CPU-optimized machine types",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "instance", Type: "string", Required: true, Description: "GCE instance name"},
				{Name: "machine_type", Type: "string", Required: false, Description: "CPU-optimized machine type", Default: "c3-standard-4"},
				{Name: "zone", Type: "string", Required: true, Description: "GCE zone"},
				{Name: "image_family", Type: "string", Required: false, Description: "Image family", Default: "debian-12"},
				{Name: "project", Type: "string", Required: false, Description: "GCP project ID"},
			},
			Outputs: []core.SkillOutput{
				{Name: "instance", Type: "string", Description: "Deployed instance name"},
				{Name: "machine_type", Type: "string", Description: "Resolved CPU-optimized machine type"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud compute instances create {instance} --machine-type={machine_type} --zone={zone}",
				Timeout: 180 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "gcloud compute instances delete {instance} --zone={zone}"},
		},
		{
			Name:        "gcp.sql.deploy.secure",
			Description: "Launch a Cloud SQL instance with private networking, backups, and encryption defaults",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "instance", Type: "string", Required: true, Description: "Cloud SQL instance name"},
				{Name: "database_version", Type: "string", Required: false, Description: "Cloud SQL database version", Default: "POSTGRES_15"},
				{Name: "tier", Type: "string", Required: false, Description: "Cloud SQL machine tier", Default: "db-custom-2-7680"},
				{Name: "region", Type: "string", Required: true, Description: "Cloud SQL region"},
				{Name: "private_ip", Type: "bool", Required: false, Description: "Use private IP only", Default: "true"},
				{Name: "backup_enabled", Type: "bool", Required: false, Description: "Enable automated backups", Default: "true"},
				{Name: "project", Type: "string", Required: false, Description: "GCP project ID"},
			},
			Outputs: []core.SkillOutput{
				{Name: "instance_connection_name", Type: "string", Description: "Cloud SQL connection name"},
				{Name: "private_ip", Type: "string", Description: "Private IP address"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud sql instances create {instance} --database-version={database_version} --tier={tier} --region={region}",
				Timeout: 600 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "gcloud sql instances delete {instance}"},
		},
		{
			Name:        "gcp.service.deploy",
			Description: "Deploy a GCP service workload using a service name and deployment profile",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "service", Type: "string", Required: true, Description: "GCP service name (gke, compute, sql, cloudrun, functions, etc.)"},
				{Name: "profile", Type: "string", Required: false, Description: "Deployment profile such as secure, cpu_optimized, or standard", Default: "standard"},
				{Name: "environment", Type: "string", Required: false, Description: "Target environment", Default: "staging"},
				{Name: "project", Type: "string", Required: false, Description: "GCP project ID"},
			},
			Outputs: []core.SkillOutput{
				{Name: "service", Type: "string", Description: "Resolved GCP service"},
				{Name: "status", Type: "string", Description: "Deployment status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud <service> <deploy-command>",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "Run service-specific rollback or previous deployment version"},
		},
		{
			Name:        "gcp.billing.anomaly",
			Description: "Detect GCP billing anomalies",
			Provider:    core.ProviderGCP,
			Category:    core.CategoryCost,
			Inputs: []core.SkillInput{
				{Name: "project", Type: "string", Required: true, Description: "GCP project ID"},
				{Name: "threshold_percent", Type: "int", Required: false, Description: "Anomaly threshold percentage", Default: "20"},
			},
			Outputs: []core.SkillOutput{
				{Name: "anomalies", Type: "list", Description: "Detected billing anomalies"},
				{Name: "current_spend", Type: "string", Description: "Current period spend"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "gcloud billing budgets list --billing-account={account}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
	}
}
