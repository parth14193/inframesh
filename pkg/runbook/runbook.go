// Package runbook provides codified operational procedures and
// incident response workflows that chain skills together.
package runbook

import (
	"fmt"
	"strings"
	"time"
)

// StepType defines what kind of action a runbook step performs.
type StepType string

const (
	StepSkill        StepType = "skill"        // Execute a registered skill
	StepManual       StepType = "manual"       // Pause for manual intervention
	StepWait         StepType = "wait"         // Wait for a duration
	StepNotification StepType = "notification" // Send a notification
	StepCondition    StepType = "condition"    // Conditional branching
	StepHealthGate   StepType = "health_gate"  // Pause until health probes pass
)

// TriggerType defines what can trigger a runbook.
type TriggerType string

const (
	TriggerManual   TriggerType = "manual"   // Operator starts it
	TriggerAlert    TriggerType = "alert"    // Triggered by monitoring alert
	TriggerSchedule TriggerType = "schedule" // Cron-scheduled
	TriggerWebhook  TriggerType = "webhook"  // External system webhook
)

// Step represents a single step in a runbook.
type Step struct {
	Name         string                 `json:"name" yaml:"name"`
	Type         StepType               `json:"type" yaml:"type"`
	SkillName    string                 `json:"skill_name,omitempty" yaml:"skill_name,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty" yaml:"params,omitempty"`
	Description  string                 `json:"description" yaml:"description"`
	Timeout      time.Duration          `json:"timeout,omitempty" yaml:"timeout,omitempty"`
	WaitDuration time.Duration          `json:"wait_duration,omitempty" yaml:"wait_duration,omitempty"`
	Condition    string                 `json:"condition,omitempty" yaml:"condition,omitempty"`
	OnFailure    string                 `json:"on_failure,omitempty" yaml:"on_failure,omitempty"` // skip, abort, retry, goto:<step>
	MaxRetries   int                    `json:"max_retries,omitempty" yaml:"max_retries,omitempty"`
	Notification string                 `json:"notification,omitempty" yaml:"notification,omitempty"`
}

// Runbook is a complete operational procedure.
type Runbook struct {
	Name        string      `json:"name" yaml:"name"`
	Description string      `json:"description" yaml:"description"`
	Trigger     TriggerType `json:"trigger" yaml:"trigger"`
	Tags        []string    `json:"tags,omitempty" yaml:"tags,omitempty"`
	Steps       []Step      `json:"steps" yaml:"steps"`
	Escalation  *Escalation `json:"escalation,omitempty" yaml:"escalation,omitempty"`
	CreatedAt   time.Time   `json:"created_at" yaml:"created_at"`
}

// Escalation defines what happens if a runbook fails.
type Escalation struct {
	NotifyChannel string        `json:"notify_channel" yaml:"notify_channel"`
	WaitBefore    time.Duration `json:"wait_before" yaml:"wait_before"`
	Message       string        `json:"message" yaml:"message"`
}

// ExecutionLog records the result of running a runbook.
type ExecutionLog struct {
	RunbookName string       `json:"runbook_name"`
	StartedAt   time.Time    `json:"started_at"`
	CompletedAt time.Time    `json:"completed_at"`
	Status      string       `json:"status"` // completed, failed, aborted
	StepResults []StepResult `json:"step_results"`
}

// StepResult records the result of a single runbook step.
type StepResult struct {
	StepName string        `json:"step_name"`
	Status   string        `json:"status"`
	Duration time.Duration `json:"duration"`
	Output   string        `json:"output"`
	Error    string        `json:"error,omitempty"`
}

// Engine manages and executes runbooks.
type Engine struct {
	runbooks map[string]*Runbook
}

// NewEngine creates a new RunbookEngine.
func NewEngine() *Engine {
	return &Engine{
		runbooks: make(map[string]*Runbook),
	}
}

// Register adds a runbook to the engine.
func (e *Engine) Register(rb *Runbook) error {
	if rb.Name == "" {
		return fmt.Errorf("runbook name cannot be empty")
	}
	if _, exists := e.runbooks[rb.Name]; exists {
		return fmt.Errorf("runbook '%s' already registered", rb.Name)
	}
	rb.CreatedAt = time.Now()
	e.runbooks[rb.Name] = rb
	return nil
}

// Get retrieves a runbook by name.
func (e *Engine) Get(name string) (*Runbook, error) {
	rb, ok := e.runbooks[name]
	if !ok {
		return nil, fmt.Errorf("runbook not found: %s", name)
	}
	return rb, nil
}

// List returns all registered runbooks.
func (e *Engine) List() []*Runbook {
	result := make([]*Runbook, 0, len(e.runbooks))
	for _, rb := range e.runbooks {
		result = append(result, rb)
	}
	return result
}

// Validate checks a runbook for structural correctness.
func (e *Engine) Validate(rb *Runbook) []error {
	var errs []error

	if rb.Name == "" {
		errs = append(errs, fmt.Errorf("runbook name is required"))
	}
	if len(rb.Steps) == 0 {
		errs = append(errs, fmt.Errorf("runbook must have at least one step"))
	}

	for i, step := range rb.Steps {
		if step.Name == "" {
			errs = append(errs, fmt.Errorf("step %d: name is required", i+1))
		}
		if step.Type == StepSkill && step.SkillName == "" {
			errs = append(errs, fmt.Errorf("step %d: skill_name is required for skill steps", i+1))
		}
		if step.Type == StepWait && step.WaitDuration == 0 {
			errs = append(errs, fmt.Errorf("step %d: wait_duration is required for wait steps", i+1))
		}
	}

	return errs
}

// SimulateRun does a dry-run of a runbook, returning the planned execution.
func (e *Engine) SimulateRun(rb *Runbook) *ExecutionLog {
	log := &ExecutionLog{
		RunbookName: rb.Name,
		StartedAt:   time.Now(),
		Status:      "simulated",
	}

	for _, step := range rb.Steps {
		result := StepResult{
			StepName: step.Name,
			Status:   "would_execute",
		}

		switch step.Type {
		case StepSkill:
			result.Output = fmt.Sprintf("Would execute skill: %s with params: %v", step.SkillName, step.Params)
		case StepManual:
			result.Output = fmt.Sprintf("Would pause for manual step: %s", step.Description)
		case StepWait:
			result.Output = fmt.Sprintf("Would wait for %s", step.WaitDuration)
		case StepNotification:
			result.Output = fmt.Sprintf("Would send notification: %s", step.Notification)
		case StepCondition:
			result.Output = fmt.Sprintf("Would evaluate condition: %s", step.Condition)
		case StepHealthGate:
			result.Output = fmt.Sprintf("Would wait for health gate: %s", step.Name)
		}

		log.StepResults = append(log.StepResults, result)
	}

	log.CompletedAt = time.Now()
	return log
}

// LoadBuiltins registers all built-in operational runbooks.
func (e *Engine) LoadBuiltins() {
	builtins := []*Runbook{
		highCPURunbook(),
		diskFullRunbook(),
		deploymentRollbackRunbook(),
		secretRotationRunbook(),
		incidentResponseRunbook(),
		p0IncidentResponseRunbook(),
		reliabilityGuardrailRunbook(),
		securityComplianceGovernanceRunbook(),
	}
	for _, rb := range builtins {
		_ = e.Register(rb)
	}
}

// ── Built-in Runbooks ──────────────────────────────────────────

func highCPURunbook() *Runbook {
	return &Runbook{
		Name:        "high-cpu-response",
		Description: "Respond to sustained high CPU usage on compute instances",
		Trigger:     TriggerAlert,
		Tags:        []string{"incident", "compute", "performance"},
		Steps: []Step{
			{Name: "identify", Type: StepSkill, SkillName: "aws.ec2.list", Description: "Identify the affected instance", Params: map[string]interface{}{"filter": "high-cpu"}},
			{Name: "metrics", Type: StepSkill, SkillName: "aws.cloudwatch.query", Description: "Get CPU metrics for the last hour", Params: map[string]interface{}{"metric": "CPUUtilization", "period": "3600"}},
			{Name: "notify-team", Type: StepNotification, Description: "Alert the on-call team", Notification: "High CPU detected on instance — investigating"},
			{Name: "check-processes", Type: StepManual, Description: "SSH into instance and identify top CPU consumers (top, htop)", Timeout: 10 * time.Minute},
			{Name: "evaluate-scaling", Type: StepCondition, Description: "If CPU > 90% for > 15min, scale up", Condition: "cpu_avg > 90"},
			{Name: "scale-if-needed", Type: StepSkill, SkillName: "aws.ec2.scale", Description: "Scale the ASG if needed", OnFailure: "skip"},
			{Name: "wait-stabilize", Type: StepWait, Description: "Wait for metrics to stabilize", WaitDuration: 5 * time.Minute},
			{Name: "verify", Type: StepSkill, SkillName: "aws.cloudwatch.query", Description: "Verify CPU has decreased", Params: map[string]interface{}{"metric": "CPUUtilization", "period": "300"}},
		},
		Escalation: &Escalation{
			NotifyChannel: "slack-oncall",
			WaitBefore:    15 * time.Minute,
			Message:       "High CPU runbook did not resolve — escalating to senior on-call",
		},
	}
}

func diskFullRunbook() *Runbook {
	return &Runbook{
		Name:        "disk-full-response",
		Description: "Respond to disk space warnings (>85% utilization)",
		Trigger:     TriggerAlert,
		Tags:        []string{"incident", "storage"},
		Steps: []Step{
			{Name: "identify-volume", Type: StepSkill, SkillName: "aws.ec2.list", Description: "Identify the instance with disk issue"},
			{Name: "notify", Type: StepNotification, Description: "Alert ops team", Notification: "Disk space warning triggered"},
			{Name: "cleanup-logs", Type: StepManual, Description: "Identify and clean up old log files (journalctl --vacuum-size=500M)", Timeout: 15 * time.Minute},
			{Name: "cleanup-docker", Type: StepManual, Description: "Run docker system prune if applicable", Timeout: 5 * time.Minute},
			{Name: "snapshot", Type: StepSkill, SkillName: "gcp.gce.snapshot", Description: "Create snapshot before volume resize", OnFailure: "abort"},
			{Name: "evaluate-resize", Type: StepCondition, Description: "If still > 85%, resize volume", Condition: "disk_usage > 85"},
			{Name: "verify", Type: StepManual, Description: "Verify disk usage is now below threshold"},
		},
	}
}

func deploymentRollbackRunbook() *Runbook {
	return &Runbook{
		Name:        "deployment-rollback",
		Description: "Roll back a failed Kubernetes deployment and verify health",
		Trigger:     TriggerManual,
		Tags:        []string{"deployment", "k8s", "rollback"},
		Steps: []Step{
			{Name: "check-status", Type: StepSkill, SkillName: "k8s.rollout.status", Description: "Check current rollout status"},
			{Name: "notify-team", Type: StepNotification, Description: "Alert team about rollback", Notification: "Initiating deployment rollback"},
			{Name: "rollback", Type: StepSkill, SkillName: "k8s.rollback", Description: "Execute rollback to previous revision", OnFailure: "abort", MaxRetries: 2},
			{Name: "wait-rollout", Type: StepWait, Description: "Wait for rollback container startup", WaitDuration: 30 * time.Second},
			{Name: "health-gate", Type: StepHealthGate, Description: "Verify application endpoint is healthy before finishing", Timeout: 5 * time.Minute},
			{Name: "verify-status", Type: StepSkill, SkillName: "k8s.rollout.status", Description: "Verify k8s rollout succeeded"},
			{Name: "confirm", Type: StepNotification, Description: "Confirm rollback completion", Notification: "Rollback completed and health verified"},
		},
	}
}

func secretRotationRunbook() *Runbook {
	return &Runbook{
		Name:        "secret-rotation",
		Description: "Rotate application secrets and API keys",
		Trigger:     TriggerSchedule,
		Tags:        []string{"security", "secrets"},
		Steps: []Step{
			{Name: "audit-secrets", Type: StepSkill, SkillName: "aws.secrets.rotate", Description: "Identify secrets due for rotation"},
			{Name: "notify", Type: StepNotification, Description: "Alert security team", Notification: "Starting scheduled secret rotation"},
			{Name: "backup-current", Type: StepManual, Description: "Back up current secret values in Vault"},
			{Name: "rotate", Type: StepSkill, SkillName: "aws.secrets.rotate", Description: "Rotate secrets in SecretsManager", OnFailure: "abort"},
			{Name: "update-vault", Type: StepSkill, SkillName: "vault.policy.update", Description: "Update Vault references"},
			{Name: "restart-services", Type: StepManual, Description: "Restart dependent services to pick up new secrets"},
			{Name: "verify", Type: StepManual, Description: "Verify services are healthy with new secrets"},
			{Name: "report", Type: StepNotification, Description: "Report rotation outcome", Notification: "Secret rotation completed"},
		},
	}
}

func incidentResponseRunbook() *Runbook {
	return &Runbook{
		Name:        "incident-response",
		Description: "General incident response procedure for production outages",
		Trigger:     TriggerAlert,
		Tags:        []string{"incident", "production", "outage"},
		Steps: []Step{
			{Name: "acknowledge", Type: StepSkill, SkillName: "pagerduty.incident.status", Description: "Acknowledge the incident in PagerDuty"},
			{Name: "open-bridge", Type: StepNotification, Description: "Open incident bridge", Notification: "🚨 INCIDENT — opening communications bridge"},
			{Name: "assess-impact", Type: StepManual, Description: "Assess user impact: error rates, latency, affected regions", Timeout: 5 * time.Minute},
			{Name: "check-dashboards", Type: StepSkill, SkillName: "datadog.alert.list", Description: "Review active alerts and dashboards"},
			{Name: "identify-cause", Type: StepManual, Description: "Identify root cause from logs, metrics, and recent changes", Timeout: 15 * time.Minute},
			{Name: "mitigate", Type: StepManual, Description: "Apply mitigation (rollback, scale, failover, etc.)", Timeout: 30 * time.Minute},
			{Name: "verify-recovery", Type: StepManual, Description: "Verify service recovery and user impact resolved"},
			{Name: "post-mortem", Type: StepNotification, Description: "Schedule post-mortem", Notification: "Incident resolved — scheduling post-mortem within 48 hours"},
		},
		Escalation: &Escalation{
			NotifyChannel: "slack-engineering",
			WaitBefore:    30 * time.Minute,
			Message:       "Incident not resolved after 30 minutes — escalating to engineering leads",
		},
	}
}

func p0IncidentResponseRunbook() *Runbook {
	return &Runbook{
		Name:        "p0-incident-response",
		Description: "P0 incident workflow: immediate acknowledgement, blast-radius analysis, and guided mitigation",
		Trigger:     TriggerAlert,
		Tags:        []string{"incident", "p0", "production", "runbook"},
		Steps: []Step{
			{Name: "acknowledge-within-60s", Type: StepSkill, SkillName: "pagerduty.incident.status", Description: "Acknowledge and verify active P0 incident metadata"},
			{Name: "page-oncall", Type: StepNotification, Description: "Page primary/secondary on-call", Notification: "P0 declared: paging on-call responders now"},
			{Name: "open-incident-channel", Type: StepNotification, Description: "Open dedicated comms channel", Notification: "Opening #inc-p0 bridge and posting live status thread"},
			{Name: "assess-blast-radius", Type: StepManual, Description: "Identify impacted services, regions, and customer-facing surface area", Timeout: 10 * time.Minute},
			{Name: "pull-runbook", Type: StepManual, Description: "Load closest service runbook from ~/.infrabot/runbooks/", Timeout: 5 * time.Minute},
			{Name: "collect-signals", Type: StepSkill, SkillName: "datadog.alert.list", Description: "Gather active alerts and correlated events"},
			{Name: "mitigate", Type: StepManual, Description: "Execute safest reversible mitigation (rollback, failover, or controlled scale-up)", Timeout: 30 * time.Minute},
			{Name: "verify-recovery", Type: StepManual, Description: "Confirm SLO and user impact recovery", Timeout: 10 * time.Minute},
			{Name: "draft-postmortem", Type: StepNotification, Description: "Create postmortem draft", Notification: "Generating timeline and postmortem draft for review"},
		},
		Escalation: &Escalation{
			NotifyChannel: "slack-incidents",
			WaitBefore:    15 * time.Minute,
			Message:       "P0 still unresolved after 15 minutes - escalate to incident commander and platform lead",
		},
	}
}

func reliabilityGuardrailRunbook() *Runbook {
	return &Runbook{
		Name:        "reliability-guardrail-loop",
		Description: "Continuous SRE loop for SLO burn-rate checks and bounded auto-remediation",
		Trigger:     TriggerSchedule,
		Tags:        []string{"sre", "reliability", "automanage"},
		Steps: []Step{
			{Name: "collect-alerts", Type: StepSkill, SkillName: "datadog.alert.list", Description: "Collect current alert surface"},
			{Name: "evaluate-slo", Type: StepSkill, SkillName: "datadog.slo.burnrate", Description: "Evaluate error budget consumption", Params: map[string]interface{}{"slo_id": "core-api", "window": "1h"}},
			{Name: "capacity-forecast", Type: StepSkill, SkillName: "platform.capacity.forecast", Description: "Forecast saturation risk", Params: map[string]interface{}{"service": "core-api", "horizon": "7d"}},
			{Name: "gated-autoremediate", Type: StepManual, Description: "If thresholds are breached, approve bounded remediation with _confirmed and rollback metadata", Timeout: 10 * time.Minute},
			{Name: "execute-autoremediate", Type: StepSkill, SkillName: "platform.reliability.autoremediate", Description: "Execute approved remediation playbook", Params: map[string]interface{}{"playbook": "latency-spike-v1", "target": "core-api", "environment": "staging"}},
			{Name: "verify-post-action", Type: StepSkill, SkillName: "prometheus.query", Description: "Verify latency and error-rate recovery", Params: map[string]interface{}{"query": "histogram_quantile(0.99, rate(http_request_duration_seconds_bucket[5m]))", "window": "15m"}},
			{Name: "timeline", Type: StepSkill, SkillName: "sre.incident.timeline.generate", Description: "Persist operational timeline artifact", Params: map[string]interface{}{"incident_id": "auto-remediation", "start_time": "now-30m"}},
		},
		Escalation: &Escalation{
			NotifyChannel: "slack-sre",
			WaitBefore:    20 * time.Minute,
			Message:       "Reliability guardrail loop needs human intervention after failed remediation",
		},
	}
}

func securityComplianceGovernanceRunbook() *Runbook {
	return &Runbook{
		Name:        "security-compliance-governance-cycle",
		Description: "Daily governance cycle for security posture, compliance evidence, and policy drift",
		Trigger:     TriggerSchedule,
		Tags:        []string{"security", "compliance", "governance", "automanage"},
		Steps: []Step{
			{Name: "cloudtrail-audit", Type: StepSkill, SkillName: "cloudtrail.event.search", Description: "Search privileged API calls", Params: map[string]interface{}{"lookup_attribute": "EventName", "value": "AuthorizeSecurityGroupIngress"}},
			{Name: "least-privilege-diff", Type: StepSkill, SkillName: "security.iam.least_privilege.diff", Description: "Detect IAM drift from baseline", Params: map[string]interface{}{"principal_arn": "arn:aws:iam::123456789012:role/platform-admin", "policy_baseline": "policies/platform-admin.json"}},
			{Name: "secret-exposure-scan", Type: StepSkill, SkillName: "security.secrets.exposure.scan", Description: "Scan repo and logs for leaked credentials", Params: map[string]interface{}{"scope": "both", "target": "platform"}},
			{Name: "collect-evidence", Type: StepSkill, SkillName: "compliance.evidence.collect", Description: "Collect SOC2 evidence package", Params: map[string]interface{}{"framework": "SOC2", "period": "30d"}},
			{Name: "risk-gate", Type: StepSkill, SkillName: "github.pr.change_risk", Description: "Assess open infra PR risk", Params: map[string]interface{}{"repo": "org/platform", "pr_number": 1}},
			{Name: "operator-approval", Type: StepManual, Description: "For production changes, validate _change_ticket and rollback references", Timeout: 10 * time.Minute},
			{Name: "record-governance-approval", Type: StepSkill, SkillName: "governance.change.approve", Description: "Record approved governance decision", Params: map[string]interface{}{"change_ticket": "CHG-0000", "approver": "ops-lead", "environment": "production"}},
		},
		Escalation: &Escalation{
			NotifyChannel: "slack-security",
			WaitBefore:    12 * time.Hour,
			Message:       "Governance cycle found unresolved critical controls",
		},
	}
}

// Render formats a runbook for display.
func (rb *Runbook) Render() string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("📖 RUNBOOK: %s\n", rb.Name))
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Description: %s\n", rb.Description))
	b.WriteString(fmt.Sprintf("Trigger:     %s\n", rb.Trigger))
	if len(rb.Tags) > 0 {
		b.WriteString(fmt.Sprintf("Tags:        %s\n", strings.Join(rb.Tags, ", ")))
	}
	b.WriteString(fmt.Sprintf("\n📋 STEPS (%d):\n", len(rb.Steps)))

	for i, step := range rb.Steps {
		icon := stepIcon(step.Type)
		b.WriteString(fmt.Sprintf("  %d. %s [%s] %s\n", i+1, icon, step.Type, step.Name))
		b.WriteString(fmt.Sprintf("     %s\n", step.Description))
		if step.SkillName != "" {
			b.WriteString(fmt.Sprintf("     → skill: %s\n", step.SkillName))
		}
		if step.OnFailure != "" {
			b.WriteString(fmt.Sprintf("     → on_failure: %s\n", step.OnFailure))
		}
	}

	if rb.Escalation != nil {
		b.WriteString(fmt.Sprintf("\n🚨 ESCALATION: %s (after %s)\n", rb.Escalation.Message, rb.Escalation.WaitBefore))
	}

	return b.String()
}

func stepIcon(t StepType) string {
	switch t {
	case StepSkill:
		return "⚡"
	case StepManual:
		return "👤"
	case StepWait:
		return "⏳"
	case StepNotification:
		return "🔔"
	case StepCondition:
		return "🔀"
	case StepHealthGate:
		return "❤️"
	default:
		return "📋"
	}
}

// RenderExecutionLog formats a runbook execution log.
func (log *ExecutionLog) Render() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📖 RUNBOOK EXECUTION: %s\n", log.RunbookName))
	b.WriteString(fmt.Sprintf("Status: %s | Duration: %s\n\n", log.Status, log.CompletedAt.Sub(log.StartedAt).Round(time.Millisecond)))

	for _, r := range log.StepResults {
		b.WriteString(fmt.Sprintf("  [%s] %s: %s\n", r.Status, r.StepName, r.Output))
	}
	return b.String()
}
