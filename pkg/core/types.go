// Package core defines the foundational types and interfaces for the InfraCore agent framework.
package core

import (
	"fmt"
	"time"
)

// RiskLevel defines the risk classification for infrastructure operations.
type RiskLevel int

const (
	RiskLow      RiskLevel = iota // Execute immediately, log result
	RiskMedium                     // Show plan, ask for confirmation
	RiskHigh                       // Show plan + blast radius, require typed confirmation
	RiskCritical                   // Show plan + blast radius + rollback plan, require CONFIRM PRODUCTION
)

// String returns the string representation of a RiskLevel.
func (r RiskLevel) String() string {
	switch r {
	case RiskLow:
		return "LOW"
	case RiskMedium:
		return "MEDIUM"
	case RiskHigh:
		return "HIGH"
	case RiskCritical:
		return "CRITICAL"
	default:
		return "UNKNOWN"
	}
}

// ParseRiskLevel converts a string to RiskLevel.
func ParseRiskLevel(s string) (RiskLevel, error) {
	switch s {
	case "LOW":
		return RiskLow, nil
	case "MEDIUM":
		return RiskMedium, nil
	case "HIGH":
		return RiskHigh, nil
	case "CRITICAL":
		return RiskCritical, nil
	default:
		return RiskLow, fmt.Errorf("unknown risk level: %s", s)
	}
}

// Provider represents a cloud or platform provider.
type Provider string

const (
	ProviderAWS        Provider = "aws"
	ProviderGCP        Provider = "gcp"
	ProviderAzure      Provider = "azure"
	ProviderKubernetes Provider = "k8s"
	ProviderTerraform  Provider = "terraform"
	ProviderPulumi     Provider = "pulumi"
	ProviderHelm       Provider = "helm"
	ProviderArgoCD     Provider = "argocd"
	ProviderDatadog    Provider = "datadog"
	ProviderGrafana    Provider = "grafana"
	ProviderPagerDuty  Provider = "pagerduty"
	ProviderVault      Provider = "vault"
	ProviderTrivy      Provider = "trivy"
	ProviderGitHub     Provider = "github"
	ProviderGitLab     Provider = "gitlab"
	ProviderJenkins    Provider = "jenkins"
	ProviderCloudflare Provider = "cloudflare"
	ProviderInfracost  Provider = "infracost"
	ProviderCustom     Provider = "custom"
)

// SkillCategory represents a functional category for skills.
type SkillCategory string

const (
	CategoryCompute       SkillCategory = "compute"
	CategoryStorage       SkillCategory = "storage"
	CategoryNetworking    SkillCategory = "networking"
	CategoryDeployment    SkillCategory = "deployment"
	CategorySecurity      SkillCategory = "security"
	CategoryObservability SkillCategory = "observability"
	CategoryCost          SkillCategory = "cost"
	CategoryCICD          SkillCategory = "cicd"
)

// ExecutionType defines how a skill is executed.
type ExecutionType string

const (
	ExecCLI       ExecutionType = "cli"
	ExecAPI       ExecutionType = "api"
	ExecTerraform ExecutionType = "terraform"
	ExecScript    ExecutionType = "script"
)

// SkillInput defines a parameter that a skill accepts.
type SkillInput struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"` // string, int, bool, list
	Required    bool   `json:"required" yaml:"required"`
	Description string `json:"description" yaml:"description"`
	Default     string `json:"default,omitempty" yaml:"default,omitempty"`
}

// SkillOutput defines what a skill returns.
type SkillOutput struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
}

// ExecutionConfig defines how a skill is executed.
type ExecutionConfig struct {
	Type    ExecutionType `json:"type" yaml:"type"`
	Command string        `json:"command" yaml:"command"`
	Timeout time.Duration `json:"timeout,omitempty" yaml:"timeout,omitempty"`
}

// RollbackConfig defines how to undo a skill's action.
type RollbackConfig struct {
	Supported bool   `json:"supported" yaml:"supported"`
	Procedure string `json:"procedure" yaml:"procedure"`
}

// Skill represents a modular capability unit in the InfraCore framework.
type Skill struct {
	Name                 string          `json:"name" yaml:"name"`
	Description          string          `json:"description" yaml:"description"`
	Provider             Provider        `json:"provider" yaml:"provider"`
	Category             SkillCategory   `json:"category" yaml:"category"`
	Inputs               []SkillInput    `json:"inputs" yaml:"inputs"`
	Outputs              []SkillOutput   `json:"outputs" yaml:"outputs"`
	RiskLevel            RiskLevel       `json:"risk_level" yaml:"risk_level"`
	RequiresConfirmation bool            `json:"requires_confirmation" yaml:"requires_confirmation"`
	Execution            ExecutionConfig `json:"execution" yaml:"execution"`
	Rollback             RollbackConfig  `json:"rollback" yaml:"rollback"`
}

// ExecutionStatus represents the outcome status of a skill execution.
type ExecutionStatus string

const (
	StatusSuccess   ExecutionStatus = "success"
	StatusFailed    ExecutionStatus = "failed"
	StatusDryRun    ExecutionStatus = "dry_run"
	StatusCancelled ExecutionStatus = "cancelled"
	StatusPending   ExecutionStatus = "pending"
)

// ExecutionResult captures the outcome of executing a skill.
type ExecutionResult struct {
	SkillName string                 `json:"skill_name"`
	Status    ExecutionStatus        `json:"status"`
	Output    map[string]interface{} `json:"output,omitempty"`
	Message   string                 `json:"message,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Error     string                 `json:"error,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// ConditionType defines the type of conditional logic in a plan step.
type ConditionType string

const (
	ConditionNone      ConditionType = ""
	ConditionIfThen    ConditionType = "if_then"
	ConditionIfElse    ConditionType = "if_else"
)

// PlanStep represents a single step in a multi-step execution plan.
type PlanStep struct {
	StepNumber    int                    `json:"step_number"`
	SkillName     string                 `json:"skill_name"`
	Description   string                 `json:"description"`
	Params        map[string]interface{} `json:"params,omitempty"`
	RiskLevel     RiskLevel              `json:"risk_level"`
	Condition     ConditionType          `json:"condition,omitempty"`
	ConditionExpr string                 `json:"condition_expr,omitempty"`
	OnTrue        *PlanStep              `json:"on_true,omitempty"`
	OnFalse       *PlanStep              `json:"on_false,omitempty"`
}

// Plan represents a multi-step execution plan.
type Plan struct {
	Name            string     `json:"name"`
	Description     string     `json:"description"`
	Steps           []PlanStep `json:"steps"`
	EstimatedTime   string     `json:"estimated_time"`
	OverallRisk     RiskLevel  `json:"overall_risk"`
	CreatedAt       time.Time  `json:"created_at"`
}

// ResourceContext tracks the active infrastructure context.
type ResourceContext struct {
	Cluster        string `json:"cluster,omitempty"`
	Namespace      string `json:"namespace,omitempty"`
	LastDeployment string `json:"last_deployment,omitempty"`
}

// AuditEntry records a single action taken during the session.
type AuditEntry struct {
	Timestamp   time.Time       `json:"timestamp"`
	SkillName   string          `json:"skill_name"`
	Action      string          `json:"action"`
	Target      string          `json:"target"`
	Status      ExecutionStatus `json:"status"`
	RiskLevel   RiskLevel       `json:"risk_level"`
	Details     string          `json:"details,omitempty"`
}

// SessionState maintains the full session context.
type SessionState struct {
	SessionID           string                 `json:"session_id"`
	ActiveEnvironment   string                 `json:"active_environment"`
	ActiveProvider      Provider               `json:"active_provider"`
	ActiveRegion        string                 `json:"active_region"`
	LoadedSkills        []string               `json:"loaded_skills"`
	ResourceContext     ResourceContext         `json:"resource_context"`
	PendingConfirmations []string              `json:"pending_confirmations"`
	AuditLog            []AuditEntry           `json:"audit_log"`
	CustomData          map[string]interface{} `json:"custom_data,omitempty"`
}

// SafetyReport is the result of a safety evaluation.
type SafetyReport struct {
	SkillName           string    `json:"skill_name"`
	RiskLevel           RiskLevel `json:"risk_level"`
	BlastRadius         int       `json:"blast_radius"`
	AffectedResources   []string  `json:"affected_resources"`
	RequiresConfirmation bool     `json:"requires_confirmation"`
	ConfirmationPrompt  string    `json:"confirmation_prompt"`
	RollbackAvailable   bool      `json:"rollback_available"`
	RollbackProcedure   string    `json:"rollback_procedure"`
	DryRunRecommended   bool      `json:"dry_run_recommended"`
	EnvironmentWarning  string    `json:"environment_warning,omitempty"`
}
