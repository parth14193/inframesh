package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// NetworkingSkills returns all built-in networking skill definitions.
func NetworkingSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "cloudflare.dns.manage",
			Description: "Manage DNS records via Cloudflare API",
			Provider:    core.ProviderCloudflare,
			Category:    core.CategoryNetworking,
			Inputs: []core.SkillInput{
				{Name: "zone", Type: "string", Required: true, Description: "Cloudflare zone name or ID"},
				{Name: "action", Type: "string", Required: true, Description: "Action: list, create, update, delete"},
				{Name: "record_type", Type: "string", Required: false, Description: "DNS record type (A, CNAME, TXT, etc.)"},
				{Name: "name", Type: "string", Required: false, Description: "Record name"},
				{Name: "content", Type: "string", Required: false, Description: "Record value/content"},
				{Name: "ttl", Type: "int", Required: false, Description: "TTL in seconds", Default: "1"},
				{Name: "proxied", Type: "bool", Required: false, Description: "Whether to proxy through Cloudflare"},
			},
			Outputs: []core.SkillOutput{
				{Name: "records", Type: "list", Description: "DNS records (for list action)"},
				{Name: "record_id", Type: "string", Description: "Created/updated record ID"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecAPI,
				Command: "Cloudflare API /zones/{zone_id}/dns_records",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "Restore previous DNS record configuration via Cloudflare API",
			},
		},
	}
}
