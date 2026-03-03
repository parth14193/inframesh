// Package topology provides infrastructure topology visualization
// with provider/category grouping, risk heat maps, and Mermaid diagram export.
package topology

import (
	"fmt"
	"sort"
	"strings"

	"github.com/parth14193/inframesh/pkg/core"
	"github.com/parth14193/inframesh/pkg/skills"
)

// Node represents a skill in the topology graph.
type Node struct {
	Name        string             `json:"name"`
	Provider    core.Provider      `json:"provider"`
	Category    core.SkillCategory `json:"category"`
	RiskLevel   core.RiskLevel     `json:"risk_level"`
	Description string             `json:"description"`
	Mutating    bool               `json:"mutating"`
}

// ProviderGroup aggregates skills by provider.
type ProviderGroup struct {
	Provider    core.Provider                  `json:"provider"`
	Categories  map[core.SkillCategory][]*Node `json:"categories"`
	TotalSkills int                            `json:"total_skills"`
}

// RiskDistribution shows risk level counts.
type RiskDistribution struct {
	Low      int `json:"low"`
	Medium   int `json:"medium"`
	High     int `json:"high"`
	Critical int `json:"critical"`
}

// TopologyStats captures overall topology metrics.
type TopologyStats struct {
	TotalSkills      int                        `json:"total_skills"`
	TotalProviders   int                        `json:"total_providers"`
	TotalCategories  int                        `json:"total_categories"`
	MutatingSkills   int                        `json:"mutating_skills"`
	ReadOnlySkills   int                        `json:"read_only_skills"`
	RiskDistribution RiskDistribution           `json:"risk_distribution"`
	ByProvider       map[core.Provider]int      `json:"by_provider"`
	ByCategory       map[core.SkillCategory]int `json:"by_category"`
}

// Graph represents the complete skill topology.
type Graph struct {
	nodes  []*Node
	groups map[core.Provider]*ProviderGroup
}

// BuildFromRegistry constructs a topology graph from the skill registry.
func BuildFromRegistry(registry *skills.Registry) *Graph {
	g := &Graph{
		nodes:  []*Node{},
		groups: make(map[core.Provider]*ProviderGroup),
	}

	allSkills := registry.List()
	for _, skill := range allSkills {
		node := &Node{
			Name:        skill.Name,
			Provider:    skill.Provider,
			Category:    skill.Category,
			RiskLevel:   skill.RiskLevel,
			Description: skill.Description,
			Mutating:    isMutating(skill.Name),
		}
		g.nodes = append(g.nodes, node)

		group, exists := g.groups[skill.Provider]
		if !exists {
			group = &ProviderGroup{
				Provider:   skill.Provider,
				Categories: make(map[core.SkillCategory][]*Node),
			}
			g.groups[skill.Provider] = group
		}
		group.Categories[skill.Category] = append(group.Categories[skill.Category], node)
		group.TotalSkills++
	}

	return g
}

// FilterByProvider returns a filtered graph for a single provider.
func (g *Graph) FilterByProvider(provider core.Provider) *Graph {
	filtered := &Graph{
		nodes:  []*Node{},
		groups: make(map[core.Provider]*ProviderGroup),
	}
	for _, node := range g.nodes {
		if node.Provider == provider {
			filtered.nodes = append(filtered.nodes, node)
		}
	}
	if group, ok := g.groups[provider]; ok {
		filtered.groups[provider] = group
	}
	return filtered
}

// GetStats calculates topology statistics.
func (g *Graph) GetStats() TopologyStats {
	stats := TopologyStats{
		TotalSkills:    len(g.nodes),
		TotalProviders: len(g.groups),
		ByProvider:     make(map[core.Provider]int),
		ByCategory:     make(map[core.SkillCategory]int),
	}

	categories := make(map[core.SkillCategory]bool)
	for _, node := range g.nodes {
		stats.ByProvider[node.Provider]++
		stats.ByCategory[node.Category]++
		categories[node.Category] = true

		if node.Mutating {
			stats.MutatingSkills++
		} else {
			stats.ReadOnlySkills++
		}

		switch node.RiskLevel {
		case core.RiskLow:
			stats.RiskDistribution.Low++
		case core.RiskMedium:
			stats.RiskDistribution.Medium++
		case core.RiskHigh:
			stats.RiskDistribution.High++
		case core.RiskCritical:
			stats.RiskDistribution.Critical++
		}
	}
	stats.TotalCategories = len(categories)

	return stats
}

// RenderCLI formats the topology as an ASCII tree grouped by provider → category.
func (g *Graph) RenderCLI() string {
	var b strings.Builder
	b.WriteString("🗺️  INFRASTRUCTURE TOPOLOGY\n")
	b.WriteString("═══════════════════════════════════════════\n\n")

	// Sort providers
	providers := make([]core.Provider, 0, len(g.groups))
	for p := range g.groups {
		providers = append(providers, p)
	}
	sort.Slice(providers, func(i, j int) bool {
		return providers[i] < providers[j]
	})

	for _, provider := range providers {
		group := g.groups[provider]
		b.WriteString(fmt.Sprintf("  ☁️  %s (%d skills)\n", strings.ToUpper(string(provider)), group.TotalSkills))

		// Sort categories
		cats := make([]core.SkillCategory, 0, len(group.Categories))
		for c := range group.Categories {
			cats = append(cats, c)
		}
		sort.Slice(cats, func(i, j int) bool {
			return cats[i] < cats[j]
		})

		for ci, cat := range cats {
			nodes := group.Categories[cat]
			prefix := "├"
			if ci == len(cats)-1 {
				prefix = "└"
			}
			b.WriteString(fmt.Sprintf("  %s── 📁 %s (%d)\n", prefix, cat, len(nodes)))

			// Sort nodes
			sort.Slice(nodes, func(i, j int) bool {
				return nodes[i].Name < nodes[j].Name
			})

			for ni, node := range nodes {
				riskIcon := riskIcon(node.RiskLevel)
				mutIcon := "📖"
				if node.Mutating {
					mutIcon = "⚡"
				}
				innerPrefix := "│   ├"
				if ci == len(cats)-1 {
					innerPrefix = "    ├"
				}
				if ni == len(nodes)-1 {
					innerPrefix = strings.Replace(innerPrefix, "├", "└", 1)
				}
				desc := node.Description
				if len(desc) > 40 {
					desc = desc[:37] + "..."
				}
				b.WriteString(fmt.Sprintf("  %s── %s %s %-30s %s\n",
					innerPrefix, mutIcon, riskIcon, node.Name, desc))
			}
		}
		b.WriteString("\n")
	}

	return b.String()
}

// RenderMermaid exports the topology as a Mermaid diagram.
func (g *Graph) RenderMermaid() string {
	var b strings.Builder
	b.WriteString("```mermaid\ngraph TD\n")

	for provider, group := range g.groups {
		providerID := sanitizeID(string(provider))
		b.WriteString(fmt.Sprintf("    subgraph %s[\"%s\"]\n", providerID, strings.ToUpper(string(provider))))

		for cat, nodes := range group.Categories {
			catID := sanitizeID(string(provider) + "_" + string(cat))
			b.WriteString(fmt.Sprintf("        subgraph %s[\"%s\"]\n", catID, cat))

			for _, node := range nodes {
				nodeID := sanitizeID(node.Name)
				shape := riskShape(node.RiskLevel)
				b.WriteString(fmt.Sprintf("            %s%s\n", nodeID, shape))
			}
			b.WriteString("        end\n")
		}
		b.WriteString("    end\n\n")
	}

	// Style risk levels
	b.WriteString("    classDef low fill:#4ade80,stroke:#16a34a\n")
	b.WriteString("    classDef medium fill:#facc15,stroke:#ca8a04\n")
	b.WriteString("    classDef high fill:#fb923c,stroke:#ea580c\n")
	b.WriteString("    classDef critical fill:#f87171,stroke:#dc2626\n\n")

	// Apply classes
	for _, node := range g.nodes {
		nodeID := sanitizeID(node.Name)
		class := strings.ToLower(node.RiskLevel.String())
		b.WriteString(fmt.Sprintf("    class %s %s\n", nodeID, class))
	}

	b.WriteString("```\n")
	return b.String()
}

// RenderStats formats topology statistics for display.
func (g *Graph) RenderStats() string {
	stats := g.GetStats()

	var b strings.Builder
	b.WriteString("📊 TOPOLOGY STATISTICS\n")
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Total Skills:     %d\n", stats.TotalSkills))
	b.WriteString(fmt.Sprintf("  Total Providers:  %d\n", stats.TotalProviders))
	b.WriteString(fmt.Sprintf("  Total Categories: %d\n", stats.TotalCategories))
	b.WriteString(fmt.Sprintf("  ⚡ Mutating:      %d\n", stats.MutatingSkills))
	b.WriteString(fmt.Sprintf("  📖 Read-only:     %d\n", stats.ReadOnlySkills))

	b.WriteString("\n  🎯 RISK HEAT MAP:\n")
	b.WriteString(fmt.Sprintf("    🟢 Low:      %s (%d)\n", bar(stats.RiskDistribution.Low, stats.TotalSkills), stats.RiskDistribution.Low))
	b.WriteString(fmt.Sprintf("    🟡 Medium:   %s (%d)\n", bar(stats.RiskDistribution.Medium, stats.TotalSkills), stats.RiskDistribution.Medium))
	b.WriteString(fmt.Sprintf("    🟠 High:     %s (%d)\n", bar(stats.RiskDistribution.High, stats.TotalSkills), stats.RiskDistribution.High))
	b.WriteString(fmt.Sprintf("    🔴 Critical: %s (%d)\n", bar(stats.RiskDistribution.Critical, stats.TotalSkills), stats.RiskDistribution.Critical))

	b.WriteString("\n  ☁️  BY PROVIDER:\n")
	providerKeys := make([]core.Provider, 0, len(stats.ByProvider))
	for p := range stats.ByProvider {
		providerKeys = append(providerKeys, p)
	}
	sort.Slice(providerKeys, func(i, j int) bool {
		return stats.ByProvider[providerKeys[i]] > stats.ByProvider[providerKeys[j]]
	})
	for _, p := range providerKeys {
		count := stats.ByProvider[p]
		b.WriteString(fmt.Sprintf("    %-15s %s (%d)\n", p, bar(count, stats.TotalSkills), count))
	}

	b.WriteString("\n  📁 BY CATEGORY:\n")
	catKeys := make([]core.SkillCategory, 0, len(stats.ByCategory))
	for c := range stats.ByCategory {
		catKeys = append(catKeys, c)
	}
	sort.Slice(catKeys, func(i, j int) bool {
		return stats.ByCategory[catKeys[i]] > stats.ByCategory[catKeys[j]]
	})
	for _, c := range catKeys {
		count := stats.ByCategory[c]
		b.WriteString(fmt.Sprintf("    %-15s %s (%d)\n", c, bar(count, stats.TotalSkills), count))
	}

	return b.String()
}

func isMutating(name string) bool {
	lower := strings.ToLower(name)
	keywords := []string{".deploy", ".apply", ".scale", ".drain", ".delete", ".terminate",
		".sync", ".migrate", ".update", ".upgrade", ".rollback", ".cordon", ".resize",
		".rotate", ".restart", ".launch", ".create"}
	for _, kw := range keywords {
		if strings.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func riskIcon(level core.RiskLevel) string {
	switch level {
	case core.RiskLow:
		return "🟢"
	case core.RiskMedium:
		return "🟡"
	case core.RiskHigh:
		return "🟠"
	case core.RiskCritical:
		return "🔴"
	default:
		return "⚪"
	}
}

func riskShape(level core.RiskLevel) string {
	switch level {
	case core.RiskCritical:
		return "{{\"⚠️\"}}"
	case core.RiskHigh:
		return "([\"⚡\"])"
	default:
		return "[\" \"]"
	}
}

func sanitizeID(s string) string {
	r := strings.NewReplacer(".", "_", " ", "_", "-", "_", "/", "_")
	return r.Replace(s)
}

func bar(count, total int) string {
	if total == 0 {
		return ""
	}
	width := 20
	filled := (count * width) / total
	if filled == 0 && count > 0 {
		filled = 1
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}
