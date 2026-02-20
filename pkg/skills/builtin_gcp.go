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
