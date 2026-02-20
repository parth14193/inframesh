// Package policy provides an infrastructure guardrail engine that enforces
// policies before any mutation executes, preventing dangerous misconfigurations.
package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// EnforcementLevel determines what happens when a policy is violated.
type EnforcementLevel string

const (
	EnforcementWarn EnforcementLevel = "warn" // Log warning but allow execution
	EnforcementDeny EnforcementLevel = "deny" // Block execution
)

// Severity classifies the impact of a policy violation.
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityWarning  Severity = "WARNING"
	SeverityCritical Severity = "CRITICAL"
)

// Policy defines an infrastructure guardrail rule.
type Policy struct {
	Name            string           `json:"name" yaml:"name"`
	Description     string           `json:"description" yaml:"description"`
	Enforcement     EnforcementLevel `json:"enforcement" yaml:"enforcement"`
	Severity        Severity         `json:"severity" yaml:"severity"`
	AppliesTo       []string         `json:"applies_to" yaml:"applies_to"`       // skill name patterns
	Environments    []string         `json:"environments" yaml:"environments"`   // which envs this applies to
	CheckFunc       PolicyCheckFunc  `json:"-" yaml:"-"`                         // the actual check function
}

// PolicyCheckFunc evaluates whether a policy is satisfied.
// Returns (violated bool, reason string).
type PolicyCheckFunc func(skill *core.Skill, params map[string]interface{}, env string) (bool, string)

// Violation represents a detected policy violation.
type Violation struct {
	PolicyName  string           `json:"policy_name"`
	Description string           `json:"description"`
	Severity    Severity         `json:"severity"`
	Enforcement EnforcementLevel `json:"enforcement"`
	Reason      string           `json:"reason"`
	SkillName   string           `json:"skill_name"`
	Environment string           `json:"environment"`
	Timestamp   time.Time        `json:"timestamp"`
}

// EvaluationResult is the outcome of all policy checks for a single action.
type EvaluationResult struct {
	Passed     bool        `json:"passed"`
	Violations []Violation `json:"violations"`
	Warnings   []Violation `json:"warnings"`
	Denied     bool        `json:"denied"`
}

// Engine evaluates policies against skill executions.
type Engine struct {
	policies        []*Policy
	enforcementMode EnforcementLevel
}

// NewEngine creates a new PolicyEngine.
func NewEngine(enforcementMode EnforcementLevel) *Engine {
	return &Engine{
		policies:        []*Policy{},
		enforcementMode: enforcementMode,
	}
}

// Register adds a policy to the engine.
func (e *Engine) Register(policy *Policy) {
	e.policies = append(e.policies, policy)
}

// LoadBuiltins registers all built-in policies.
func (e *Engine) LoadBuiltins() {
	for _, p := range BuiltinPolicies() {
		e.Register(p)
	}
}

// Evaluate checks all applicable policies against a skill execution.
func (e *Engine) Evaluate(skill *core.Skill, params map[string]interface{}, env string) *EvaluationResult {
	result := &EvaluationResult{
		Passed:     true,
		Violations: []Violation{},
		Warnings:   []Violation{},
	}

	for _, policy := range e.policies {
		if !e.policyApplies(policy, skill, env) {
			continue
		}

		violated, reason := policy.CheckFunc(skill, params, env)
		if !violated {
			continue
		}

		v := Violation{
			PolicyName:  policy.Name,
			Description: policy.Description,
			Severity:    policy.Severity,
			Enforcement: policy.Enforcement,
			Reason:      reason,
			SkillName:   skill.Name,
			Environment: env,
			Timestamp:   time.Now(),
		}

		// Effective enforcement = stricter of (global, policy-level)
		// Global warn ⇒ always warn. Global deny ⇒ always deny.
		effectiveEnforcement := policy.Enforcement
		if e.enforcementMode == EnforcementDeny {
			effectiveEnforcement = EnforcementDeny
		} else if e.enforcementMode == EnforcementWarn {
			effectiveEnforcement = EnforcementWarn
		}

		if effectiveEnforcement == EnforcementDeny {
			result.Violations = append(result.Violations, v)
			result.Passed = false
			result.Denied = true
		} else {
			result.Warnings = append(result.Warnings, v)
		}
	}

	return result
}

// ListPolicies returns all registered policies.
func (e *Engine) ListPolicies() []*Policy {
	return e.policies
}

// policyApplies checks if a policy is relevant for the given skill and environment.
func (e *Engine) policyApplies(policy *Policy, skill *core.Skill, env string) bool {
	// Check if skill matches
	if len(policy.AppliesTo) > 0 {
		matched := false
		for _, pattern := range policy.AppliesTo {
			if matchSkillPattern(skill.Name, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check if environment matches
	if len(policy.Environments) > 0 {
		envMatched := false
		for _, e := range policy.Environments {
			if strings.EqualFold(e, env) {
				envMatched = true
				break
			}
		}
		if !envMatched {
			return false
		}
	}

	return true
}

// matchSkillPattern supports simple wildcard matching for skill names.
func matchSkillPattern(skillName, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, ".*") {
		prefix := strings.TrimSuffix(pattern, ".*")
		return strings.HasPrefix(skillName, prefix+".")
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(skillName, prefix)
	}
	return skillName == pattern
}

// Render formats an EvaluationResult for display.
func (r *EvaluationResult) Render() string {
	var b strings.Builder

	if r.Passed && len(r.Warnings) == 0 {
		b.WriteString("✅ All policies passed\n")
		return b.String()
	}

	if len(r.Warnings) > 0 {
		b.WriteString(fmt.Sprintf("⚠️  POLICY WARNINGS (%d)\n", len(r.Warnings)))
		for _, w := range r.Warnings {
			b.WriteString(fmt.Sprintf("  ⚠️  [%s] %s: %s\n", w.Severity, w.PolicyName, w.Reason))
		}
	}

	if len(r.Violations) > 0 {
		b.WriteString(fmt.Sprintf("\n🚫 POLICY VIOLATIONS (%d) — execution BLOCKED\n", len(r.Violations)))
		for _, v := range r.Violations {
			b.WriteString(fmt.Sprintf("  ❌ [%s] %s: %s\n", v.Severity, v.PolicyName, v.Reason))
		}
	}

	return b.String()
}
