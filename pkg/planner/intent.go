// Package planner provides intent parsing to convert natural language
// infrastructure commands into executable Plans.
package planner

import (
	"fmt"
	"strings"

	"github.com/parth14193/inframesh/pkg/core"
	"github.com/parth14193/inframesh/pkg/skills"
)

// IntentParser converts a natural language string into a Plan.
type IntentParser interface {
	Parse(text string) (*core.Plan, error)
}

// intentRule maps a set of trigger keywords to a plan-building recipe.
type intentRule struct {
	triggers    []string               // lowercase keywords that activate this rule
	name        string                 // human-readable plan name
	description string                 // plan description
	steps       []intentStep           // ordered steps to add
}

// intentStep describes a single step to be added when a rule fires.
type intentStep struct {
	skillName   string
	description string
	// paramExtractors pull values from the parsed tokens
	paramExtractors []paramExtractor
}

// paramExtractor extracts a named parameter from the token list.
type paramExtractor struct {
	paramKey      string
	defaultValue  string
	// extractFn receives the full lowercased token list and returns the value.
	extractFn func(tokens []string) string
}

// KeywordIntentParser is a rule-based NL intent parser.
// It matches keyword triggers in the input text and maps them to
// pre-defined plan recipes referencing registered skills.
type KeywordIntentParser struct {
	registry *skills.Registry
	rules    []intentRule
}

// NewKeywordIntentParser creates a parser wired to the given skill registry.
func NewKeywordIntentParser(registry *skills.Registry) *KeywordIntentParser {
	p := &KeywordIntentParser{registry: registry}
	p.loadRules()
	return p
}

// Parse converts natural language text into a Plan.
// Returns an error if no matching rule is found or a referenced skill is missing.
func (p *KeywordIntentParser) Parse(text string) (*core.Plan, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("intent text must not be empty")
	}

	tokens := tokenise(text)
	rule := p.matchRule(tokens)
	if rule == nil {
		return nil, fmt.Errorf(
			"no intent rule matched %q — try keywords like: deploy, rollback, scale, restart, scan, audit, drift, incident",
			text,
		)
	}

	engine := NewEngine(p.registry)
	plan := engine.CreatePlan(rule.name, rule.description)

	for _, s := range rule.steps {
		params := make(map[string]interface{})
		for _, pe := range s.paramExtractors {
			v := pe.extractFn(tokens)
			if v == "" {
				v = pe.defaultValue
			}
			if v != "" {
				params[pe.paramKey] = v
			}
		}
		if err := engine.AddStep(plan, s.skillName, s.description, params); err != nil {
			// Skill may not be registered in this registry instance — surface clearly.
			return nil, fmt.Errorf("intent rule %q, step %q: %w", rule.name, s.skillName, err)
		}
	}

	return plan, nil
}

// ─── Rule loading ──────────────────────────────────────────────────────────

func (p *KeywordIntentParser) loadRules() {
	p.rules = []intentRule{
		// ── Deploy ──────────────────────────────────────────────────────────
		{
			triggers:    []string{"deploy", "release", "rollout", "update"},
			name:        "Deploy Service",
			description: "Deploy a new version of a service to Kubernetes with policy and health checks",
			steps: []intentStep{
				{
					skillName:   "github.pr.change_risk",
					description: "Assess change risk before deploying",
					paramExtractors: []paramExtractor{
						{paramKey: "repo", defaultValue: "org/platform", extractFn: extractAfterKeyword("repo")},
					},
				},
				{
					skillName:   "k8s.deploy",
					description: "Apply the deployment to Kubernetes",
					paramExtractors: []paramExtractor{
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
						{paramKey: "deployment", defaultValue: "app", extractFn: extractServiceName},
						{paramKey: "image", defaultValue: "", extractFn: extractImageTag},
					},
				},
				{
					skillName:   "datadog.slo.burnrate",
					description: "Verify SLO burn rate after deployment",
					paramExtractors: []paramExtractor{
						{paramKey: "window", defaultValue: "15m", extractFn: extractAfterKeyword("window")},
					},
				},
			},
		},

		// ── Rollback ────────────────────────────────────────────────────────
		{
			triggers:    []string{"rollback", "revert", "undo"},
			name:        "Deployment Rollback",
			description: "Roll back a Kubernetes deployment to the previous revision",
			steps: []intentStep{
				{
					skillName:   "aws.cloudwatch.query",
					description: "Capture current error signals before rollback",
					paramExtractors: []paramExtractor{
						{paramKey: "log_group", defaultValue: "/platform/app", extractFn: extractLogGroup},
						{paramKey: "query", defaultValue: "fields @timestamp,@message | sort @timestamp desc | limit 20", extractFn: nil},
					},
				},
				{
					skillName:   "argocd.rollout.rollback",
					description: "Execute rollback via ArgoCD",
					paramExtractors: []paramExtractor{
						{paramKey: "app", defaultValue: "app", extractFn: extractServiceName},
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
					},
				},
				{
					skillName:   "datadog.slo.burnrate",
					description: "Confirm burn rate recovers after rollback",
					paramExtractors: []paramExtractor{
						{paramKey: "window", defaultValue: "10m", extractFn: nil},
					},
				},
			},
		},

		// ── Scale ───────────────────────────────────────────────────────────
		{
			triggers:    []string{"scale", "resize", "capacity"},
			name:        "Scale Service",
			description: "Scale a Kubernetes deployment or node group",
			steps: []intentStep{
				{
					skillName:   "k8s.node.list",
					description: "List current node capacity",
				},
				{
					skillName:   "k8s.hpa.status",
					description: "Check HPA utilisation before scaling",
					paramExtractors: []paramExtractor{
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
						{paramKey: "deployment", defaultValue: "app", extractFn: extractServiceName},
					},
				},
				{
					skillName:   "k8s.deploy",
					description: "Apply replica count change",
					paramExtractors: []paramExtractor{
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
						{paramKey: "deployment", defaultValue: "app", extractFn: extractServiceName},
						{paramKey: "replicas", defaultValue: "3", extractFn: extractAfterKeyword("replicas")},
					},
				},
			},
		},

		// ── Restart ─────────────────────────────────────────────────────────
		{
			triggers:    []string{"restart", "bounce", "recycle"},
			name:        "Restart Service Pods",
			description: "Rolling restart of a Kubernetes deployment",
			steps: []intentStep{
				{
					skillName:   "k8s.pod.list",
					description: "List current pods before restart",
					paramExtractors: []paramExtractor{
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
					},
				},
				{
					skillName:   "k8s.pod.restart",
					description: "Trigger rolling restart",
					paramExtractors: []paramExtractor{
						{paramKey: "namespace", defaultValue: "default", extractFn: extractAfterKeyword("namespace")},
						{paramKey: "deployment", defaultValue: "app", extractFn: extractServiceName},
					},
				},
			},
		},

		// ── Security Scan ───────────────────────────────────────────────────
		{
			triggers:    []string{"scan", "security", "vulnerability", "cve"},
			name:        "Security Vulnerability Scan",
			description: "Run Trivy and secrets scan across infrastructure",
			steps: []intentStep{
				{
					skillName:   "trivy.scan",
					description: "Scan container images for CVEs",
					paramExtractors: []paramExtractor{
						{paramKey: "image", defaultValue: "app:latest", extractFn: extractImageTag},
					},
				},
				{
					skillName:   "security.secrets.exposure.scan",
					description: "Detect any exposed secrets in the environment",
					paramExtractors: []paramExtractor{
						{paramKey: "scope", defaultValue: "cluster", extractFn: nil},
					},
				},
				{
					skillName:   "security.iam.least_privilege.diff",
					description: "Report on IAM role over-permission",
					paramExtractors: []paramExtractor{
						{paramKey: "environment", defaultValue: "production", extractFn: extractEnv},
					},
				},
			},
		},

		// ── Compliance Audit ────────────────────────────────────────────────
		{
			triggers:    []string{"audit", "compliance", "cis", "soc2", "hipaa"},
			name:        "Compliance Audit",
			description: "Run compliance audit checks across the platform",
			steps: []intentStep{
				{
					skillName:   "compliance.evidence.collect",
					description: "Collect compliance evidence artefacts",
					paramExtractors: []paramExtractor{
						{paramKey: "framework", defaultValue: "CIS", extractFn: extractFramework},
					},
				},
				{
					skillName:   "terraform.cloud.policy.check",
					description: "Check Terraform Cloud policies for compliance drift",
				},
			},
		},

		// ── Incident Response ───────────────────────────────────────────────
		{
			triggers:    []string{"incident", "p0", "outage", "alert", "fire"},
			name:        "Incident Response",
			description: "Execute the P0 incident response workflow",
			steps: []intentStep{
				{
					skillName:   "datadog.alert.list",
					description: "List all active Datadog alerts",
				},
				{
					skillName:   "aws.cloudwatch.query",
					description: "Query CloudWatch logs for error signals",
					paramExtractors: []paramExtractor{
						{paramKey: "log_group", defaultValue: "/platform/app", extractFn: extractLogGroup},
						{paramKey: "query", defaultValue: "fields @timestamp,@message | filter @message like /ERROR/ | sort @timestamp desc | limit 50", extractFn: nil},
					},
				},
				{
					skillName:   "datadog.slo.burnrate",
					description: "Check SLO burn rate and error budget",
					paramExtractors: []paramExtractor{
						{paramKey: "window", defaultValue: "1h", extractFn: nil},
					},
				},
				{
					skillName:   "sre.incident.timeline.generate",
					description: "Generate incident timeline artefact",
					paramExtractors: []paramExtractor{
						{paramKey: "service", defaultValue: "platform", extractFn: extractServiceName},
					},
				},
			},
		},

		// ── Drift Detection ─────────────────────────────────────────────────
		{
			triggers:    []string{"drift", "detect", "terraform", "iac"},
			name:        "Infrastructure Drift Detection",
			description: "Detect configuration drift between Terraform state and live infrastructure",
			steps: []intentStep{
				{
					skillName:   "terraform.plan",
					description: "Run Terraform plan to detect drift",
					paramExtractors: []paramExtractor{
						{paramKey: "working_dir", defaultValue: "./infra", extractFn: extractAfterKeyword("dir")},
					},
				},
				{
					skillName:   "cloudtrail.event.search",
					description: "Search CloudTrail for manual changes",
					paramExtractors: []paramExtractor{
						{paramKey: "hours_back", defaultValue: "24", extractFn: nil},
					},
				},
			},
		},

		// ── Cost Analysis ───────────────────────────────────────────────────
		{
			triggers:    []string{"cost", "spend", "budget", "infracost"},
			name:        "Cloud Cost Analysis",
			description: "Analyse cloud spend and generate cost optimisation recommendations",
			steps: []intentStep{
				{
					skillName:   "aws.cost.explorer",
					description: "Pull AWS cost breakdown by service",
					paramExtractors: []paramExtractor{
						{paramKey: "period", defaultValue: "30d", extractFn: extractAfterKeyword("period")},
					},
				},
				{
					skillName:   "infracost.estimate",
					description: "Estimate cost impact of proposed infrastructure changes",
				},
			},
		},
	}
}

// ─── Matching ──────────────────────────────────────────────────────────────

// matchRule finds the first rule whose triggers appear in the token list.
func (p *KeywordIntentParser) matchRule(tokens []string) *intentRule {
	tokenSet := make(map[string]bool, len(tokens))
	for _, t := range tokens {
		tokenSet[t] = true
	}
	for i := range p.rules {
		for _, trigger := range p.rules[i].triggers {
			if tokenSet[trigger] {
				return &p.rules[i]
			}
		}
	}
	return nil
}

// ─── Parameter extractors ──────────────────────────────────────────────────

// tokenise lowercases and splits text into words.
func tokenise(text string) []string {
	text = strings.ToLower(text)
	// Remove common punctuation
	for _, ch := range []string{",", ".", "!", "?", "'", "\"", "(", ")"} {
		text = strings.ReplaceAll(text, ch, " ")
	}
	raw := strings.Fields(text)
	tokens := make([]string, 0, len(raw))
	for _, t := range raw {
		if t != "" {
			tokens = append(tokens, t)
		}
	}
	return tokens
}

// extractAfterKeyword returns a function that finds the word immediately after 'keyword'.
func extractAfterKeyword(keyword string) func([]string) string {
	return func(tokens []string) string {
		for i, t := range tokens {
			if t == keyword && i+1 < len(tokens) {
				return tokens[i+1]
			}
		}
		return ""
	}
}

// extractServiceName looks for known service-indicator keywords and returns the next token.
func extractServiceName(tokens []string) string {
	serviceKeywords := []string{"service", "deployment", "app", "application", "for", "of"}
	for _, kw := range serviceKeywords {
		for i, t := range tokens {
			if t == kw && i+1 < len(tokens) {
				return tokens[i+1]
			}
		}
	}
	return ""
}

// extractImageTag looks for version patterns like v1.2.3 or tokens containing ':'.
func extractImageTag(tokens []string) string {
	for _, t := range tokens {
		if strings.Contains(t, ":") {
			return t // e.g. "api:v2.5.0"
		}
		if len(t) > 1 && t[0] == 'v' {
			return t // e.g. "v2.5.0"
		}
	}
	return ""
}

// extractEnv looks for environment indicators.
func extractEnv(tokens []string) string {
	envKeywords := map[string]bool{
		"production": true, "prod": true, "staging": true,
		"development": true, "dev": true, "qa": true,
	}
	for _, t := range tokens {
		if envKeywords[t] {
			return t
		}
	}
	return ""
}

// extractFramework looks for compliance framework names.
func extractFramework(tokens []string) string {
	frameworks := map[string]string{
		"cis": "CIS", "soc2": "SOC2", "hipaa": "HIPAA",
	}
	for _, t := range tokens {
		if f, ok := frameworks[t]; ok {
			return f
		}
	}
	return "CIS"
}

// extractLogGroup builds a log group path from service tokens.
func extractLogGroup(tokens []string) string {
	svc := extractServiceName(tokens)
	if svc != "" {
		return "/platform/" + svc
	}
	return ""
}
