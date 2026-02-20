package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// ObservabilitySkills returns all built-in observability skill definitions.
func ObservabilitySkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "datadog.alert.list",
			Description: "List active Datadog alerts",
			Provider:    core.ProviderDatadog,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "priority", Type: "string", Required: false, Description: "Filter by priority (P1-P5)"},
				{Name: "tags", Type: "list", Required: false, Description: "Filter by tags"},
			},
			Outputs: []core.SkillOutput{
				{Name: "alerts", Type: "list", Description: "List of active alerts"},
				{Name: "count", Type: "int", Description: "Total active alert count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /api/v1/monitor?monitor_tags={tags}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "datadog.dashboard.create",
			Description: "Generate a Datadog dashboard from a metric spec",
			Provider:    core.ProviderDatadog,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "title", Type: "string", Required: true, Description: "Dashboard title"},
				{Name: "metrics", Type: "list", Required: true, Description: "List of metric queries to include"},
				{Name: "layout_type", Type: "string", Required: false, Description: "ordered or free", Default: "ordered"},
			},
			Outputs: []core.SkillOutput{
				{Name: "dashboard_id", Type: "string", Description: "Created dashboard ID"},
				{Name: "dashboard_url", Type: "string", Description: "Dashboard URL"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "POST /api/v1/dashboard",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "DELETE /api/v1/dashboard/{dashboard_id}",
			},
		},
		{
			Name:        "grafana.snapshot",
			Description: "Export a Grafana dashboard snapshot",
			Provider:    core.ProviderGrafana,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "dashboard_uid", Type: "string", Required: true, Description: "Grafana dashboard UID"},
				{Name: "time_range", Type: "string", Required: false, Description: "Time range (e.g., last-6h)", Default: "last-6h"},
			},
			Outputs: []core.SkillOutput{
				{Name: "snapshot_url", Type: "string", Description: "Snapshot URL"},
				{Name: "snapshot_key", Type: "string", Description: "Snapshot key for deletion"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "POST /api/snapshots",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "DELETE /api/snapshots/{snapshot_key}",
			},
		},
		{
			Name:        "pagerduty.incident.status",
			Description: "Get current PagerDuty incident status",
			Provider:    core.ProviderPagerDuty,
			Category:    core.CategoryObservability,
			Inputs: []core.SkillInput{
				{Name: "service_id", Type: "string", Required: false, Description: "Filter by PagerDuty service ID"},
				{Name: "status", Type: "string", Required: false, Description: "Filter by status (triggered, acknowledged, resolved)"},
				{Name: "urgency", Type: "string", Required: false, Description: "Filter by urgency (high, low)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "incidents", Type: "list", Description: "Active incidents"},
				{Name: "count", Type: "int", Description: "Incident count"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "GET /incidents?statuses[]={status}&service_ids[]={service_id}",
				Timeout: 15 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
	}
}
