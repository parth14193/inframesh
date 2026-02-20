// Package safety provides the safety layer for evaluating risk, blast radius,
// and confirmation requirements before executing infrastructure actions.
package safety

import (
	"fmt"
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

// Layer evaluates the safety characteristics of skill executions.
type Layer struct{}

// NewLayer creates a new SafetyLayer.
func NewLayer() *Layer {
	return &Layer{}
}

// Evaluate produces a SafetyReport for a given skill and its parameters.
func (l *Layer) Evaluate(skill *core.Skill, params map[string]interface{}, env string) *core.SafetyReport {
	report := &core.SafetyReport{
		SkillName:           skill.Name,
		RiskLevel:           skill.RiskLevel,
		RequiresConfirmation: skill.RequiresConfirmation,
		RollbackAvailable:   skill.Rollback.Supported,
		RollbackProcedure:   skill.Rollback.Procedure,
	}

	// Blast radius analysis
	report.BlastRadius = l.estimateBlastRadius(skill, params)
	report.AffectedResources = l.identifyAffectedResources(skill, params)

	// Environment-based risk escalation
	if l.isProductionEnvironment(env) {
		report.EnvironmentWarning = "⚠️  TARGET ENVIRONMENT IS PRODUCTION — exercise extreme caution"
		if report.RiskLevel < core.RiskHigh {
			report.RiskLevel = core.RiskHigh
		}
		report.RequiresConfirmation = true
	}

	// Set dry run recommendation
	report.DryRunRecommended = l.shouldDryRun(skill)

	// Generate appropriate confirmation prompt
	report.ConfirmationPrompt = l.getConfirmationPrompt(report.RiskLevel)

	return report
}

// RequiresConfirmation returns whether a risk level requires user confirmation.
func (l *Layer) RequiresConfirmation(riskLevel core.RiskLevel) bool {
	return riskLevel >= core.RiskMedium
}

// GetConfirmationPrompt returns the confirmation prompt for a given risk level.
func (l *Layer) GetConfirmationPrompt(riskLevel core.RiskLevel) string {
	return l.getConfirmationPrompt(riskLevel)
}

func (l *Layer) getConfirmationPrompt(riskLevel core.RiskLevel) string {
	switch riskLevel {
	case core.RiskLow:
		return ""
	case core.RiskMedium:
		return `Type "yes" to proceed or "cancel" to abort`
	case core.RiskHigh:
		return `Type "yes, apply" to proceed or "cancel" to abort`
	case core.RiskCritical:
		return `Type "CONFIRM PRODUCTION" to proceed or "cancel" to abort`
	default:
		return `Type "yes" to proceed`
	}
}

// estimateBlastRadius estimates how many resources will be affected.
func (l *Layer) estimateBlastRadius(skill *core.Skill, params map[string]interface{}) int {
	// Heuristic-based estimation
	switch {
	case strings.Contains(skill.Name, ".list") || strings.Contains(skill.Name, ".audit") ||
		strings.Contains(skill.Name, ".query") || strings.Contains(skill.Name, ".report") ||
		strings.Contains(skill.Name, ".status") || strings.Contains(skill.Name, ".snapshot"):
		return 0 // Read-only operations
	case strings.Contains(skill.Name, ".deploy") || strings.Contains(skill.Name, ".upgrade"):
		return 1
	case strings.Contains(skill.Name, ".scale"):
		return estimateFromParam(params, "desired_capacity", 1)
	case strings.Contains(skill.Name, "terraform.apply"):
		return estimateFromParam(params, "resources_count", 5)
	case strings.Contains(skill.Name, ".sync") || strings.Contains(skill.Name, ".migrate"):
		return 10 // bulk operations
	default:
		return 1
	}
}

// identifyAffectedResources returns a list of resource descriptions that will be affected.
func (l *Layer) identifyAffectedResources(skill *core.Skill, params map[string]interface{}) []string {
	var resources []string

	provider := string(skill.Provider)
	category := string(skill.Category)
	resources = append(resources, fmt.Sprintf("%s/%s resources", provider, category))

	// Extract specific resource identifiers from params
	resourceKeys := []string{"instance_id", "bucket_name", "vpc_id", "deployment", "function_name",
		"asg_name", "secret_id", "release_name", "app_name", "vm_name", "zone", "image"}
	for _, key := range resourceKeys {
		if val, ok := params[key]; ok {
			resources = append(resources, fmt.Sprintf("%s=%v", key, val))
		}
	}

	return resources
}

// isProductionEnvironment checks if the given environment string indicates production.
func (l *Layer) isProductionEnvironment(env string) bool {
	env = strings.ToLower(env)
	return env == "production" || env == "prod" || env == "prd"
}

// shouldDryRun returns whether this skill type should default to dry-run mode.
func (l *Layer) shouldDryRun(skill *core.Skill) bool {
	// Destructive or mutating operations should dry-run first
	destructiveKeywords := []string{"apply", "deploy", "scale", "resize", "delete",
		"rotate", "sync", "migrate", "update", "upgrade", "rollback"}

	for _, keyword := range destructiveKeywords {
		if strings.Contains(strings.ToLower(skill.Name), keyword) {
			return true
		}
	}
	return false
}

// estimateFromParam extracts an integer estimation from a parameter map.
func estimateFromParam(params map[string]interface{}, key string, fallback int) int {
	if params == nil {
		return fallback
	}
	if val, ok := params[key]; ok {
		if v, ok := val.(int); ok {
			return v
		}
	}
	return fallback
}
