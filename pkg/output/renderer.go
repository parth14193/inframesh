// Package output provides structured rendering for InfraCore output,
// including query reports, mutation diffs, execution plans, and tables.
package output

import (
	"fmt"
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

// Renderer produces formatted output for the InfraCore agent.
type Renderer struct{}

// NewRenderer creates a new Renderer.
func NewRenderer() *Renderer {
	return &Renderer{}
}

const separator = "─────────────────────────────────────────"

// RenderQuery formats a query/report result.
func (r *Renderer) RenderQuery(skillName, environment, provider, region string, results string, durationMs int64, resourceCount int) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🔍 SKILL: %s\n", skillName))
	b.WriteString(fmt.Sprintf("📍 TARGET: %s / %s / %s\n", environment, provider, region))
	b.WriteString(separator + "\n")
	b.WriteString(results)
	b.WriteString("\n" + separator + "\n")
	b.WriteString(fmt.Sprintf("✅ Completed in %dms | %d resources scanned\n", durationMs, resourceCount))

	return b.String()
}

// RenderMutation formats a mutation (write) operation output.
func (r *Renderer) RenderMutation(actionSummary, environment, provider, region string, blastRadius int, before, after string, riskLevel core.RiskLevel, rollbackProcedure string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("⚡ PLAN: %s\n", actionSummary))
	b.WriteString(fmt.Sprintf("📍 TARGET: %s / %s / %s\n", environment, provider, region))
	b.WriteString(fmt.Sprintf("💥 BLAST RADIUS: %d resources affected\n", blastRadius))
	b.WriteString(separator + "\n")
	b.WriteString("BEFORE:\n")
	b.WriteString(before + "\n\n")
	b.WriteString("AFTER:\n")
	b.WriteString(after + "\n")
	b.WriteString(separator + "\n")
	b.WriteString(fmt.Sprintf("⚠️  Risk Level: %s\n", riskLevel))
	b.WriteString(fmt.Sprintf("🔄 Rollback: %s\n", rollbackProcedure))
	b.WriteString("\n")

	switch riskLevel {
	case core.RiskMedium:
		b.WriteString(`> Type "yes" to proceed or "cancel" to abort` + "\n")
	case core.RiskHigh:
		b.WriteString(`> Type "yes, apply" to proceed or "cancel" to abort` + "\n")
	case core.RiskCritical:
		b.WriteString(`> Type "CONFIRM PRODUCTION" to proceed or "cancel" to abort` + "\n")
	}

	return b.String()
}

// RenderPlan formats a multi-step execution plan.
func (r *Renderer) RenderPlan(plan *core.Plan) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("📋 EXECUTION PLAN (%d steps)\n", len(plan.Steps)))
	b.WriteString(separator + "\n")

	for _, step := range plan.Steps {
		riskTag := fmt.Sprintf("[%s]", step.RiskLevel)
		padding := strings.Repeat(" ", 10-len(riskTag))

		if step.SkillName == "CONDITIONAL" {
			b.WriteString(fmt.Sprintf("Step %d %s%s → CONDITIONAL: %s\n", step.StepNumber, riskTag, padding, step.Description))
			if step.OnTrue != nil {
				b.WriteString(fmt.Sprintf("   ├─ IF TRUE  → %s: %s\n", step.OnTrue.SkillName, step.OnTrue.Description))
			}
			if step.OnFalse != nil {
				b.WriteString(fmt.Sprintf("   └─ IF FALSE → %s: %s\n", step.OnFalse.SkillName, step.OnFalse.Description))
			}
		} else {
			marker := ""
			if step.RiskLevel >= core.RiskHigh {
				marker = "  ← Requires confirmation"
			}
			b.WriteString(fmt.Sprintf("Step %d %s%s → %s: %s%s\n", step.StepNumber, riskTag, padding, step.SkillName, step.Description, marker))
		}
	}

	b.WriteString(separator + "\n")
	b.WriteString(fmt.Sprintf("Estimated duration: ~%s\n", plan.EstimatedTime))
	b.WriteString(fmt.Sprintf("Overall risk: %s\n", plan.OverallRisk))
	b.WriteString("\n")
	b.WriteString(`> Confirm to begin, or say "skip step N" to modify the plan` + "\n")

	return b.String()
}

// RenderTable renders an ASCII table with headers and rows.
func (r *Renderer) RenderTable(headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}

	// Calculate column widths
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	var b strings.Builder

	// Top border
	b.WriteString("┌")
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			b.WriteString("┬")
		}
	}
	b.WriteString("┐\n")

	// Header row
	b.WriteString("│")
	for i, h := range headers {
		b.WriteString(fmt.Sprintf(" %-*s │", widths[i], h))
	}
	b.WriteString("\n")

	// Header separator
	b.WriteString("├")
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			b.WriteString("┼")
		}
	}
	b.WriteString("┤\n")

	// Data rows
	for _, row := range rows {
		b.WriteString("│")
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			b.WriteString(fmt.Sprintf(" %-*s │", widths[i], cell))
		}
		b.WriteString("\n")
	}

	// Bottom border
	b.WriteString("└")
	for i, w := range widths {
		b.WriteString(strings.Repeat("─", w+2))
		if i < len(widths)-1 {
			b.WriteString("┴")
		}
	}
	b.WriteString("┘\n")

	return b.String()
}

// RenderSafetyReport formats a safety evaluation report.
func (r *Renderer) RenderSafetyReport(report *core.SafetyReport) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("🛡️  SAFETY REPORT: %s\n", report.SkillName))
	b.WriteString(separator + "\n")
	b.WriteString(fmt.Sprintf("⚠️  Risk Level:        %s\n", report.RiskLevel))
	b.WriteString(fmt.Sprintf("💥 Blast Radius:       %d resources\n", report.BlastRadius))
	b.WriteString(fmt.Sprintf("✋ Requires Confirm:   %t\n", report.RequiresConfirmation))
	b.WriteString(fmt.Sprintf("🔄 Rollback Available: %t\n", report.RollbackAvailable))

	if report.RollbackAvailable {
		b.WriteString(fmt.Sprintf("   Procedure: %s\n", report.RollbackProcedure))
	}

	if report.DryRunRecommended {
		b.WriteString("🧪 Dry Run:            Recommended\n")
	}

	if report.EnvironmentWarning != "" {
		b.WriteString(fmt.Sprintf("\n%s\n", report.EnvironmentWarning))
	}

	if report.ConfirmationPrompt != "" {
		b.WriteString(fmt.Sprintf("\n> %s\n", report.ConfirmationPrompt))
	}

	return b.String()
}

// RenderSkillInfo formats detailed information about a skill.
func (r *Renderer) RenderSkillInfo(skill *core.Skill) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("📦 SKILL: %s\n", skill.Name))
	b.WriteString(separator + "\n")
	b.WriteString(fmt.Sprintf("Description:  %s\n", skill.Description))
	b.WriteString(fmt.Sprintf("Provider:     %s\n", skill.Provider))
	b.WriteString(fmt.Sprintf("Category:     %s\n", skill.Category))
	b.WriteString(fmt.Sprintf("Risk Level:   %s\n", skill.RiskLevel))
	b.WriteString(fmt.Sprintf("Confirmation: %t\n", skill.RequiresConfirmation))
	b.WriteString(fmt.Sprintf("Execution:    %s → %s\n", skill.Execution.Type, skill.Execution.Command))

	if len(skill.Inputs) > 0 {
		b.WriteString("\n📥 INPUTS:\n")
		for _, in := range skill.Inputs {
			req := "optional"
			if in.Required {
				req = "required"
			}
			def := ""
			if in.Default != "" {
				def = fmt.Sprintf(" [default: %s]", in.Default)
			}
			b.WriteString(fmt.Sprintf("  • %s (%s, %s)%s — %s\n", in.Name, in.Type, req, def, in.Description))
		}
	}

	if len(skill.Outputs) > 0 {
		b.WriteString("\n📤 OUTPUTS:\n")
		for _, out := range skill.Outputs {
			b.WriteString(fmt.Sprintf("  • %s (%s) — %s\n", out.Name, out.Type, out.Description))
		}
	}

	b.WriteString(fmt.Sprintf("\n🔄 Rollback: supported=%t", skill.Rollback.Supported))
	if skill.Rollback.Procedure != "" {
		b.WriteString(fmt.Sprintf("\n   Procedure: %s", skill.Rollback.Procedure))
	}
	b.WriteString("\n")

	return b.String()
}

// RenderSuccess formats a success message.
func (r *Renderer) RenderSuccess(msg string) string {
	return fmt.Sprintf("✅ %s\n", msg)
}

// RenderError formats an error message.
func (r *Renderer) RenderError(err error) string {
	return fmt.Sprintf("❌ ERROR: %s\n", err.Error())
}

// RenderWarning formats a warning message.
func (r *Renderer) RenderWarning(msg string) string {
	return fmt.Sprintf("⚠️  WARNING: %s\n", msg)
}

// RenderSessionState formats the current session state.
func (r *Renderer) RenderSessionState(state *core.SessionState) string {
	var b strings.Builder

	b.WriteString("📊 SESSION STATE\n")
	b.WriteString(separator + "\n")
	b.WriteString(fmt.Sprintf("Session ID:   %s\n", state.SessionID))
	b.WriteString(fmt.Sprintf("Environment:  %s\n", state.ActiveEnvironment))
	b.WriteString(fmt.Sprintf("Provider:     %s\n", state.ActiveProvider))
	b.WriteString(fmt.Sprintf("Region:       %s\n", state.ActiveRegion))

	if state.ResourceContext.Cluster != "" {
		b.WriteString(fmt.Sprintf("Cluster:      %s\n", state.ResourceContext.Cluster))
	}
	if state.ResourceContext.Namespace != "" {
		b.WriteString(fmt.Sprintf("Namespace:    %s\n", state.ResourceContext.Namespace))
	}
	if state.ResourceContext.LastDeployment != "" {
		b.WriteString(fmt.Sprintf("Last Deploy:  %s\n", state.ResourceContext.LastDeployment))
	}

	b.WriteString(fmt.Sprintf("Loaded Skills: %d\n", len(state.LoadedSkills)))
	b.WriteString(fmt.Sprintf("Audit Entries: %d\n", len(state.AuditLog)))

	if len(state.PendingConfirmations) > 0 {
		b.WriteString(fmt.Sprintf("⏳ Pending Confirmations: %d\n", len(state.PendingConfirmations)))
	}

	return b.String()
}
