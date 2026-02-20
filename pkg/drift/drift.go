// Package drift detects when live infrastructure diverges from
// the desired state defined in Infrastructure as Code.
package drift

import (
	"fmt"
	"strings"
	"time"
)

// DriftSeverity classifies how serious the drift is.
type DriftSeverity string

const (
	DriftInfo     DriftSeverity = "INFO"
	DriftWarning  DriftSeverity = "WARNING"
	DriftCritical DriftSeverity = "CRITICAL"
)

// DriftStatus represents whether a resource has drifted.
type DriftStatus string

const (
	DriftStatusInSync  DriftStatus = "IN_SYNC"
	DriftStatusDrifted DriftStatus = "DRIFTED"
	DriftStatusNew     DriftStatus = "NEW"      // exists in live but not in IaC
	DriftStatusDeleted DriftStatus = "DELETED"  // exists in IaC but not live
	DriftStatusUnknown DriftStatus = "UNKNOWN"
)

// ResourceDrift describes a single resource that has drifted.
type ResourceDrift struct {
	ResourceID   string        `json:"resource_id"`
	ResourceType string        `json:"resource_type"`
	Provider     string        `json:"provider"`
	Status       DriftStatus   `json:"status"`
	Severity     DriftSeverity `json:"severity"`
	FieldDrifts  []FieldDrift  `json:"field_drifts"`
	DetectedAt   time.Time     `json:"detected_at"`
}

// FieldDrift describes a single field that differs between live and declared.
type FieldDrift struct {
	FieldPath     string `json:"field_path"`
	ExpectedValue string `json:"expected_value"`
	ActualValue   string `json:"actual_value"`
}

// DriftReport aggregates drift detection results.
type DriftReport struct {
	Provider    string          `json:"provider"`
	Region      string          `json:"region"`
	Environment string          `json:"environment"`
	Timestamp   time.Time       `json:"timestamp"`
	Resources   []ResourceDrift `json:"resources"`
	InSync      int             `json:"in_sync"`
	Drifted     int             `json:"drifted"`
	New         int             `json:"new"`
	Deleted     int             `json:"deleted"`
}

// Detector analyses infrastructure drift.
type Detector struct {
	parsers map[string]OutputParser
}

// OutputParser parses IaC tool output into resource drift information.
type OutputParser interface {
	ParseDrift(output string) ([]ResourceDrift, error)
}

// NewDetector creates a new DriftDetector.
func NewDetector() *Detector {
	return &Detector{
		parsers: make(map[string]OutputParser),
	}
}

// RegisterParser adds a parser for a specific IaC tool output format.
func (d *Detector) RegisterParser(tool string, parser OutputParser) {
	d.parsers[tool] = parser
}

// AnalyzeTerraformPlan parses terraform plan output to detect drift.
func (d *Detector) AnalyzeTerraformPlan(planOutput string) *DriftReport {
	report := &DriftReport{
		Provider:  "terraform",
		Timestamp: time.Now(),
	}

	lines := strings.Split(planOutput, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		switch {
		case strings.Contains(trimmed, "# ") && strings.Contains(trimmed, " will be created"):
			resource := extractResourceName(trimmed, "will be created")
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   resource,
				ResourceType: extractResourceType(resource),
				Status:       DriftStatusNew,
				Severity:     DriftWarning,
				DetectedAt:   time.Now(),
			})
			report.New++

		case strings.Contains(trimmed, "# ") && strings.Contains(trimmed, " will be destroyed"):
			resource := extractResourceName(trimmed, "will be destroyed")
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   resource,
				ResourceType: extractResourceType(resource),
				Status:       DriftStatusDeleted,
				Severity:     DriftCritical,
				DetectedAt:   time.Now(),
			})
			report.Deleted++

		case strings.Contains(trimmed, "# ") && strings.Contains(trimmed, " will be updated in-place"):
			resource := extractResourceName(trimmed, "will be updated in-place")
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   resource,
				ResourceType: extractResourceType(resource),
				Status:       DriftStatusDrifted,
				Severity:     DriftWarning,
				DetectedAt:   time.Now(),
			})
			report.Drifted++

		case strings.Contains(trimmed, "# ") && strings.Contains(trimmed, "must be replaced"):
			resource := extractResourceName(trimmed, "must be replaced")
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   resource,
				ResourceType: extractResourceType(resource),
				Status:       DriftStatusDrifted,
				Severity:     DriftCritical,
				DetectedAt:   time.Now(),
			})
			report.Drifted++

		case strings.HasPrefix(trimmed, "~ ") && strings.Contains(trimmed, "->"):
			// Field-level drift: ~ field_name = "old" -> "new"
			parts := strings.SplitN(trimmed[2:], "=", 2)
			if len(parts) == 2 && len(report.Resources) > 0 {
				field := strings.TrimSpace(parts[0])
				values := strings.SplitN(parts[1], "->", 2)
				if len(values) == 2 {
					last := &report.Resources[len(report.Resources)-1]
					last.FieldDrifts = append(last.FieldDrifts, FieldDrift{
						FieldPath:     field,
						ExpectedValue: strings.TrimSpace(values[0]),
						ActualValue:   strings.TrimSpace(values[1]),
					})
				}
			}
		}
	}

	return report
}

// DetectManualChanges simulates detection of manual changes (not managed by IaC).
func (d *Detector) DetectManualChanges(provider, resourceType string, liveResources, declaredResources []string) *DriftReport {
	report := &DriftReport{
		Provider:  provider,
		Timestamp: time.Now(),
	}

	declaredSet := make(map[string]bool)
	for _, r := range declaredResources {
		declaredSet[r] = true
	}

	liveSet := make(map[string]bool)
	for _, r := range liveResources {
		liveSet[r] = true
	}

	// Resources in live but not in IaC (manual creation)
	for _, r := range liveResources {
		if !declaredSet[r] {
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   r,
				ResourceType: resourceType,
				Provider:     provider,
				Status:       DriftStatusNew,
				Severity:     DriftWarning,
				DetectedAt:   time.Now(),
			})
			report.New++
		} else {
			report.InSync++
		}
	}

	// Resources in IaC but not live (manual deletion)
	for _, r := range declaredResources {
		if !liveSet[r] {
			report.Resources = append(report.Resources, ResourceDrift{
				ResourceID:   r,
				ResourceType: resourceType,
				Provider:     provider,
				Status:       DriftStatusDeleted,
				Severity:     DriftCritical,
				DetectedAt:   time.Now(),
			})
			report.Deleted++
		}
	}

	return report
}

// Render formats a drift report for display.
func (r *DriftReport) Render() string {
	var b strings.Builder

	b.WriteString("🔍 DRIFT DETECTION REPORT\n")
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Provider: %s | Region: %s | Env: %s\n", r.Provider, r.Region, r.Environment))
	b.WriteString(fmt.Sprintf("Detected: %s\n\n", r.Timestamp.Format(time.RFC3339)))

	total := r.InSync + r.Drifted + r.New + r.Deleted
	b.WriteString(fmt.Sprintf("📊 SUMMARY (%d resources)\n", total))
	b.WriteString(fmt.Sprintf("  ✅ In Sync:  %d\n", r.InSync))
	b.WriteString(fmt.Sprintf("  ⚠️  Drifted:  %d\n", r.Drifted))
	b.WriteString(fmt.Sprintf("  🆕 New:      %d (not in IaC)\n", r.New))
	b.WriteString(fmt.Sprintf("  🗑️  Deleted:  %d (in IaC, not live)\n", r.Deleted))

	if len(r.Resources) > 0 {
		b.WriteString("\n📋 RESOURCES:\n")
		for _, res := range r.Resources {
			icon := driftIcon(res.Status)
			b.WriteString(fmt.Sprintf("  %s [%s] %s (%s)\n", icon, res.Severity, res.ResourceID, res.Status))
			for _, fd := range res.FieldDrifts {
				b.WriteString(fmt.Sprintf("       %s: %s → %s\n", fd.FieldPath, fd.ExpectedValue, fd.ActualValue))
			}
		}
	}

	return b.String()
}

func driftIcon(status DriftStatus) string {
	switch status {
	case DriftStatusInSync:
		return "✅"
	case DriftStatusDrifted:
		return "⚠️ "
	case DriftStatusNew:
		return "🆕"
	case DriftStatusDeleted:
		return "🗑️ "
	default:
		return "❓"
	}
}

func extractResourceName(line, action string) string {
	parts := strings.Split(line, action)
	if len(parts) > 0 {
		name := strings.TrimPrefix(parts[0], "# ")
		name = strings.TrimSpace(name)
		return name
	}
	return "unknown"
}

func extractResourceType(resourceName string) string {
	parts := strings.Split(resourceName, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return resourceName
}
