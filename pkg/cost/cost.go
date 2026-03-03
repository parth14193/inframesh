// Package cost provides pre-execution cost impact analysis with
// a built-in pricing catalog for common cloud resources.
package cost

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/parth14193/inframesh/pkg/core"
)

// Estimate represents the estimated cost impact of an operation.
type Estimate struct {
	SkillName    string  `json:"skill_name"`
	ResourceType string  `json:"resource_type"`
	Provider     string  `json:"provider"`
	MonthlyCost  float64 `json:"monthly_cost"`
	HourlyCost   float64 `json:"hourly_cost"`
	PricingTier  string  `json:"pricing_tier"`
	Confidence   string  `json:"confidence"` // high, medium, low
	Notes        string  `json:"notes,omitempty"`
}

// PlanCostReport aggregates cost estimates across a plan.
type PlanCostReport struct {
	PlanName     string     `json:"plan_name"`
	Estimates    []Estimate `json:"estimates"`
	TotalMonthly float64    `json:"total_monthly"`
	TotalHourly  float64    `json:"total_hourly"`
	TotalAnnual  float64    `json:"total_annual"`
	HighestItem  string     `json:"highest_item"`
}

// PricingEntry defines hourly pricing for a resource type.
type PricingEntry struct {
	Provider     string
	ResourceType string
	InstanceType string
	HourlyUSD    float64
	Notes        string
}

// Estimator evaluates cost impact of infrastructure operations.
type Estimator struct {
	catalog map[string][]PricingEntry
}

// NewEstimator creates a cost estimator with built-in pricing catalog.
func NewEstimator() *Estimator {
	e := &Estimator{
		catalog: make(map[string][]PricingEntry),
	}
	e.loadBuiltinCatalog()
	return e
}

// EstimateSkill calculates the cost impact of a single skill execution.
func (e *Estimator) EstimateSkill(skill *core.Skill, params map[string]interface{}) *Estimate {
	est := &Estimate{
		SkillName:    skill.Name,
		Provider:     string(skill.Provider),
		ResourceType: string(skill.Category),
		Confidence:   "medium",
	}

	// Try to match instance type from params
	instanceType := extractInstanceType(params)
	if instanceType != "" {
		est.PricingTier = instanceType
	}

	// Look up pricing
	key := catalogKey(string(skill.Provider), string(skill.Category))
	entries, exists := e.catalog[key]
	if !exists {
		// Try generic provider match
		key = catalogKey(string(skill.Provider), "")
		entries, exists = e.catalog[key]
	}

	if exists {
		entry := findBestMatch(entries, instanceType)
		if entry != nil {
			est.HourlyCost = entry.HourlyUSD
			est.MonthlyCost = entry.HourlyUSD * 730 // avg hours/month
			est.PricingTier = entry.InstanceType
			est.Confidence = "high"
			est.Notes = entry.Notes
			est.ResourceType = entry.ResourceType
		}
	}

	// Apply multiplier for scaling operations
	if multiplier := extractMultiplier(params); multiplier > 1 {
		est.HourlyCost *= float64(multiplier)
		est.MonthlyCost *= float64(multiplier)
		est.Notes += fmt.Sprintf(" (x%d instances)", multiplier)
	}

	if est.MonthlyCost == 0 {
		est.Confidence = "low"
		est.Notes = "No pricing data available for this resource type"
	}

	return est
}

// EstimatePlan calculates the total cost impact of a multi-step plan.
func (e *Estimator) EstimatePlan(planName string, skills []*core.Skill, paramSets []map[string]interface{}) *PlanCostReport {
	report := &PlanCostReport{PlanName: planName}

	var highestCost float64
	for i, skill := range skills {
		var params map[string]interface{}
		if i < len(paramSets) {
			params = paramSets[i]
		}
		est := e.EstimateSkill(skill, params)
		if est.MonthlyCost > 0 {
			report.Estimates = append(report.Estimates, *est)
			report.TotalMonthly += est.MonthlyCost
			report.TotalHourly += est.HourlyCost
			if est.MonthlyCost > highestCost {
				highestCost = est.MonthlyCost
				report.HighestItem = est.SkillName
			}
		}
	}

	report.TotalAnnual = report.TotalMonthly * 12

	return report
}

// loadBuiltinCatalog populates the pricing catalog with common resources.
func (e *Estimator) loadBuiltinCatalog() {
	entries := []PricingEntry{
		// AWS EC2
		{Provider: "aws", ResourceType: "compute", InstanceType: "t3.micro", HourlyUSD: 0.0104, Notes: "General purpose, burstable"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "t3.small", HourlyUSD: 0.0208, Notes: "General purpose, burstable"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "t3.medium", HourlyUSD: 0.0416, Notes: "General purpose, burstable"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "t3.large", HourlyUSD: 0.0832, Notes: "General purpose, burstable"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "m6i.large", HourlyUSD: 0.096, Notes: "General purpose, latest gen"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "m6i.xlarge", HourlyUSD: 0.192, Notes: "General purpose, latest gen"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "c7i.large", HourlyUSD: 0.089, Notes: "Compute optimized"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "c7i.xlarge", HourlyUSD: 0.178, Notes: "Compute optimized"},
		{Provider: "aws", ResourceType: "compute", InstanceType: "r6i.large", HourlyUSD: 0.126, Notes: "Memory optimized"},
		// AWS RDS
		{Provider: "aws", ResourceType: "storage", InstanceType: "db.t3.micro", HourlyUSD: 0.017, Notes: "RDS burstable"},
		{Provider: "aws", ResourceType: "storage", InstanceType: "db.t3.medium", HourlyUSD: 0.068, Notes: "RDS burstable"},
		{Provider: "aws", ResourceType: "storage", InstanceType: "db.m6i.large", HourlyUSD: 0.171, Notes: "RDS general purpose"},
		{Provider: "aws", ResourceType: "storage", InstanceType: "db.r6i.large", HourlyUSD: 0.25, Notes: "RDS memory optimized"},
		// AWS EKS
		{Provider: "aws", ResourceType: "deployment", InstanceType: "eks-cluster", HourlyUSD: 0.10, Notes: "EKS control plane"},
		// AWS Lambda
		{Provider: "aws", ResourceType: "compute", InstanceType: "lambda-128mb", HourlyUSD: 0.0000002083, Notes: "Lambda per-invocation (128MB, 1M req/mo)"},
		// GCP
		{Provider: "gcp", ResourceType: "compute", InstanceType: "e2-standard-2", HourlyUSD: 0.067, Notes: "GCE general purpose"},
		{Provider: "gcp", ResourceType: "compute", InstanceType: "e2-standard-4", HourlyUSD: 0.134, Notes: "GCE general purpose"},
		{Provider: "gcp", ResourceType: "compute", InstanceType: "c3-standard-4", HourlyUSD: 0.166, Notes: "GCE compute optimized"},
		{Provider: "gcp", ResourceType: "deployment", InstanceType: "gke-cluster", HourlyUSD: 0.10, Notes: "GKE control plane"},
		{Provider: "gcp", ResourceType: "storage", InstanceType: "db-custom-2-7680", HourlyUSD: 0.129, Notes: "Cloud SQL"},
		// Azure
		{Provider: "azure", ResourceType: "compute", InstanceType: "Standard_B2s", HourlyUSD: 0.0416, Notes: "Azure burstable"},
		{Provider: "azure", ResourceType: "compute", InstanceType: "Standard_D2s_v5", HourlyUSD: 0.096, Notes: "Azure general purpose"},
		{Provider: "azure", ResourceType: "compute", InstanceType: "Standard_F4s_v2", HourlyUSD: 0.169, Notes: "Azure compute optimized"},
		{Provider: "azure", ResourceType: "deployment", InstanceType: "aks-cluster", HourlyUSD: 0.10, Notes: "AKS control plane (standard tier)"},
	}

	for _, entry := range entries {
		key := catalogKey(entry.Provider, entry.ResourceType)
		e.catalog[key] = append(e.catalog[key], entry)
	}
}

func catalogKey(provider, resourceType string) string {
	return strings.ToLower(provider + "/" + resourceType)
}

func findBestMatch(entries []PricingEntry, instanceType string) *PricingEntry {
	if instanceType != "" {
		lower := strings.ToLower(instanceType)
		for i, e := range entries {
			if strings.ToLower(e.InstanceType) == lower {
				return &entries[i]
			}
		}
	}
	// Return median price entry as default
	if len(entries) > 0 {
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].HourlyUSD < entries[j].HourlyUSD
		})
		mid := len(entries) / 2
		return &entries[mid]
	}
	return nil
}

func extractInstanceType(params map[string]interface{}) string {
	if params == nil {
		return ""
	}
	keys := []string{"instance_type", "machine_type", "vm_size", "instance_class", "tier", "node_instance_type"}
	for _, key := range keys {
		if val, ok := params[key]; ok {
			return fmt.Sprintf("%v", val)
		}
	}
	return ""
}

func extractMultiplier(params map[string]interface{}) int {
	if params == nil {
		return 1
	}
	keys := []string{"desired_capacity", "min_nodes", "replicas", "count"}
	for _, key := range keys {
		if val, ok := params[key]; ok {
			switch v := val.(type) {
			case int:
				if v > 1 {
					return v
				}
			case string:
				if n := parseInt(v); n > 1 {
					return n
				}
			}
		}
	}
	return 1
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// RenderEstimate formats a cost estimate for display.
func RenderEstimate(est *Estimate) string {
	var b strings.Builder
	b.WriteString("💰 COST ESTIMATE\n")
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Skill:        %s\n", est.SkillName))
	b.WriteString(fmt.Sprintf("  Provider:     %s\n", est.Provider))
	b.WriteString(fmt.Sprintf("  Resource:     %s\n", est.ResourceType))
	if est.PricingTier != "" {
		b.WriteString(fmt.Sprintf("  Pricing Tier: %s\n", est.PricingTier))
	}
	b.WriteString(fmt.Sprintf("  Hourly:       $%.4f/hr\n", est.HourlyCost))
	b.WriteString(fmt.Sprintf("  Monthly:      $%.2f/mo\n", est.MonthlyCost))
	b.WriteString(fmt.Sprintf("  Annual:       $%.2f/yr\n", est.MonthlyCost*12))
	b.WriteString(fmt.Sprintf("  Confidence:   %s\n", confidenceIcon(est.Confidence)))
	if est.Notes != "" {
		b.WriteString(fmt.Sprintf("  Notes:        %s\n", est.Notes))
	}
	return b.String()
}

// RenderPlanReport formats a plan cost report for display.
func RenderPlanReport(report *PlanCostReport) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("💰 PLAN COST REPORT: %s\n", report.PlanName))
	b.WriteString("─────────────────────────────────────────\n")

	if len(report.Estimates) == 0 {
		b.WriteString("  No cost data available for plan steps.\n")
		return b.String()
	}

	for _, est := range report.Estimates {
		b.WriteString(fmt.Sprintf("  %-35s $%.2f/mo  [%s] %s\n",
			est.SkillName, est.MonthlyCost, est.Confidence, est.PricingTier))
	}

	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  TOTAL MONTHLY:  $%.2f\n", report.TotalMonthly))
	b.WriteString(fmt.Sprintf("  TOTAL ANNUAL:   $%.2f\n", report.TotalAnnual))
	b.WriteString(fmt.Sprintf("  TOTAL HOURLY:   $%.4f\n", report.TotalHourly))

	if report.HighestItem != "" {
		b.WriteString(fmt.Sprintf("\n  💡 Highest cost: %s ($%.2f/mo)\n", report.HighestItem,
			highestCost(report.Estimates)))
	}

	return b.String()
}

func highestCost(estimates []Estimate) float64 {
	max := 0.0
	for _, e := range estimates {
		if e.MonthlyCost > max {
			max = e.MonthlyCost
		}
	}
	return math.Round(max*100) / 100
}

func confidenceIcon(confidence string) string {
	switch confidence {
	case "high":
		return "🟢 High"
	case "medium":
		return "🟡 Medium"
	case "low":
		return "🔴 Low"
	default:
		return "⚪ Unknown"
	}
}
