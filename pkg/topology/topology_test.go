package topology

import (
	"strings"
	"testing"

	"github.com/parth14193/inframesh/pkg/skills"
)

func setupRegistry(t *testing.T) *skills.Registry {
	t.Helper()
	r := skills.NewRegistry()
	if err := r.LoadBuiltins(); err != nil {
		t.Fatalf("failed to load builtins: %v", err)
	}
	return r
}

func TestBuildFromRegistry(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)

	if len(g.nodes) == 0 {
		t.Fatal("expected non-zero nodes")
	}
	if len(g.groups) == 0 {
		t.Fatal("expected non-zero provider groups")
	}
}

func TestGetStats(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)
	stats := g.GetStats()

	if stats.TotalSkills == 0 {
		t.Fatal("expected non-zero skills")
	}
	if stats.TotalProviders == 0 {
		t.Fatal("expected non-zero providers")
	}
	if stats.TotalCategories == 0 {
		t.Fatal("expected non-zero categories")
	}
	if stats.MutatingSkills+stats.ReadOnlySkills != stats.TotalSkills {
		t.Fatalf("mutating (%d) + readonly (%d) != total (%d)",
			stats.MutatingSkills, stats.ReadOnlySkills, stats.TotalSkills)
	}
	totalRisk := stats.RiskDistribution.Low + stats.RiskDistribution.Medium +
		stats.RiskDistribution.High + stats.RiskDistribution.Critical
	if totalRisk != stats.TotalSkills {
		t.Fatalf("risk distribution sum (%d) != total (%d)", totalRisk, stats.TotalSkills)
	}
}

func TestFilterByProvider(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)
	filtered := g.FilterByProvider("aws")

	for _, node := range filtered.nodes {
		if node.Provider != "aws" {
			t.Fatalf("expected provider aws, got %s", node.Provider)
		}
	}
	if len(filtered.nodes) == 0 {
		t.Fatal("expected AWS skills")
	}
}

func TestRenderCLI(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)
	output := g.RenderCLI()

	if output == "" {
		t.Fatal("expected non-empty CLI render")
	}
	if !strings.Contains(output, "TOPOLOGY") {
		t.Fatal("expected TOPOLOGY header")
	}
}

func TestRenderMermaid(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)
	output := g.RenderMermaid()

	if !strings.Contains(output, "```mermaid") {
		t.Fatal("expected mermaid code block")
	}
	if !strings.Contains(output, "graph TD") {
		t.Fatal("expected graph TD directive")
	}
}

func TestRenderStats(t *testing.T) {
	r := setupRegistry(t)
	g := BuildFromRegistry(r)
	output := g.RenderStats()

	if !strings.Contains(output, "STATISTICS") {
		t.Fatal("expected STATISTICS header")
	}
	if !strings.Contains(output, "RISK HEAT MAP") {
		t.Fatal("expected RISK HEAT MAP section")
	}
}

func TestIsMutating(t *testing.T) {
	mutating := []string{"aws.ec2.deploy", "k8s.rollback", "terraform.apply", "aws.secrets.rotate"}
	readOnly := []string{"aws.ec2.list", "aws.iam.audit", "prometheus.query", "datadog.alert.list"}

	for _, name := range mutating {
		if !isMutating(name) {
			t.Errorf("expected %s to be mutating", name)
		}
	}
	for _, name := range readOnly {
		if isMutating(name) {
			t.Errorf("expected %s to be read-only", name)
		}
	}
}
