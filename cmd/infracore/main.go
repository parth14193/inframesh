// InfraCore — Cloud Infrastructure Agent Framework
//
// Usage:
//
//	infracore skills list [--provider=aws] [--category=compute]
//	infracore skills search <query>
//	infracore skills info <skill_name>
//	infracore run <skill_name> [--param key=value ...]
//	infracore plan <description>
//	infracore state
//	infracore discover --provider <p> --action <a>
//	infracore policy list | infracore policy check <skill>
//	infracore compliance audit <framework>
//	infracore drift detect
//	infracore runbook list | infracore runbook run <name>
//	infracore health check
//	infracore config show
package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/parth14193/ownbot/pkg/compliance"
	"github.com/parth14193/ownbot/pkg/config"
	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/drift"
	"github.com/parth14193/ownbot/pkg/health"
	"github.com/parth14193/ownbot/pkg/output"
	"github.com/parth14193/ownbot/pkg/planner"
	"github.com/parth14193/ownbot/pkg/policy"
	"github.com/parth14193/ownbot/pkg/rbac"
	"github.com/parth14193/ownbot/pkg/runbook"
	"github.com/parth14193/ownbot/pkg/safety"
	"github.com/parth14193/ownbot/pkg/skills"
	"github.com/parth14193/ownbot/pkg/state"
)

const version = "2.0.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	// Initialize all subsystems
	registry := skills.NewRegistry()
	if err := registry.LoadBuiltins(); err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load built-in skills: %v\n", err)
		os.Exit(1)
	}

	renderer := output.NewRenderer()
	safetyLayer := safety.NewLayer()
	planEngine := planner.NewEngine(registry)
	stateManager := state.NewManager("cli-session")
	cfg := config.DefaultConfig()
	policyEngine := policy.NewEngine(policy.EnforcementWarn)
	policyEngine.LoadBuiltins()
	rbacEngine := rbac.NewEngine()
	runbookEngine := runbook.NewEngine()
	runbookEngine.LoadBuiltins()
	healthChecker := health.NewChecker()
	healthChecker.LoadBuiltins()
	auditor := compliance.NewAuditor()
	auditor.LoadAll()
	driftDetector := drift.NewDetector()

	switch os.Args[1] {
	case "skills":
		handleSkills(os.Args[2:], registry, renderer)
	case "run":
		handleRun(os.Args[2:], registry, renderer, safetyLayer, stateManager, policyEngine)
	case "plan":
		handlePlan(os.Args[2:], renderer, planEngine)
	case "state":
		handleState(renderer, stateManager)
	case "discover":
		handleDiscover(os.Args[2:], registry, renderer)
	case "policy":
		handlePolicy(os.Args[2:], policyEngine, registry, renderer)
	case "compliance":
		handleCompliance(os.Args[2:], auditor)
	case "drift":
		handleDrift(os.Args[2:], driftDetector)
	case "runbook":
		handleRunbook(os.Args[2:], runbookEngine)
	case "health":
		handleHealth(os.Args[2:], healthChecker)
	case "config":
		handleConfig(os.Args[2:], cfg)
	case "rbac":
		handleRBAC(os.Args[2:], rbacEngine)
	case "version":
		fmt.Printf("InfraCore Agent Framework v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "❌ Unknown command: %s\n\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`
╔══════════════════════════════════════════════════════════╗
║           InfraCore — Cloud Infrastructure Agent        ║
║                    Framework v2.0.0                     ║
╚══════════════════════════════════════════════════════════╝

USAGE:
  infracore <command> [options]

CORE COMMANDS:
  skills list      List all registered skills
  skills search    Search skills by query
  skills info      Show detailed skill information
  run              Execute a skill (dry-run by default)
  plan             Create a multi-step execution plan
  state            Show current session state
  discover         Enter skill discovery mode

PLATFORM COMMANDS:
  policy list      List all registered policies
  policy check     Check policies against a skill
  compliance audit Run compliance audit (CIS, SOC2, HIPAA)
  drift detect     Detect infrastructure drift
  runbook list     List operational runbooks
  runbook run      Execute or simulate a runbook
  health check     Run infrastructure health probes
  config show      Show current configuration
  config init      Generate sample config file
  rbac show        Show RBAC roles and users

OPTIONS:
  --provider=<p>      Filter by provider
  --category=<c>      Filter by category
  --param key=value   Set skill parameters
  --env=<env>         Set target environment
  --region=<r>        Set target region

EXAMPLES:
  infracore skills list --provider=aws
  infracore run aws.ec2.list --param region=us-west-2
  infracore policy check k8s.deploy --env=production
  infracore compliance audit CIS
  infracore drift detect
  infracore runbook run deployment-rollback
  infracore health check`)
}

// ─── Skills ───────────────────────────────────────────────────

func handleSkills(args []string, registry *skills.Registry, renderer *output.Renderer) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore skills <list|search|info> [options]")
		return
	}
	switch args[0] {
	case "list":
		handleSkillsList(args[1:], registry, renderer)
	case "search":
		handleSkillsSearch(args[1:], registry, renderer)
	case "info":
		handleSkillsInfo(args[1:], registry, renderer)
	default:
		fmt.Fprintf(os.Stderr, "❌ Unknown skills subcommand: %s\n", args[0])
	}
}

func handleSkillsList(args []string, registry *skills.Registry, renderer *output.Renderer) {
	pf := extractFlag(args, "--provider")
	cf := extractFlag(args, "--category")
	var allSkills []*core.Skill
	if pf != "" {
		allSkills = registry.ListByProvider(core.Provider(pf))
	} else if cf != "" {
		allSkills = registry.ListByCategory(core.SkillCategory(cf))
	} else {
		allSkills = registry.List()
	}
	sort.Slice(allSkills, func(i, j int) bool { return allSkills[i].Name < allSkills[j].Name })
	if len(allSkills) == 0 {
		fmt.Println("No skills found.")
		return
	}
	headers := []string{"Skill Name", "Provider", "Category", "Risk", "Description"}
	rows := make([][]string, len(allSkills))
	for i, s := range allSkills {
		d := s.Description
		if len(d) > 50 {
			d = d[:47] + "..."
		}
		rows[i] = []string{s.Name, string(s.Provider), string(s.Category), s.RiskLevel.String(), d}
	}
	fmt.Printf("📦 SKILL REGISTRY (%d skills)\n", len(allSkills))
	fmt.Print(renderer.RenderTable(headers, rows))
}

func handleSkillsSearch(args []string, registry *skills.Registry, renderer *output.Renderer) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore skills search <query>")
		return
	}
	query := strings.Join(args, " ")
	results := registry.Search(query)
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	if len(results) == 0 {
		fmt.Printf("No skills matching \"%s\".\n", query)
		return
	}
	headers := []string{"Skill Name", "Provider", "Risk", "Description"}
	rows := make([][]string, len(results))
	for i, s := range results {
		d := s.Description
		if len(d) > 55 {
			d = d[:52] + "..."
		}
		rows[i] = []string{s.Name, string(s.Provider), s.RiskLevel.String(), d}
	}
	fmt.Printf("🔍 SEARCH RESULTS for \"%s\" (%d matches)\n", query, len(results))
	fmt.Print(renderer.RenderTable(headers, rows))
}

func handleSkillsInfo(args []string, registry *skills.Registry, renderer *output.Renderer) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore skills info <skill_name>")
		return
	}
	skill, err := registry.Get(args[0])
	if err != nil {
		fmt.Println(renderer.RenderError(err))
		return
	}
	fmt.Print(renderer.RenderSkillInfo(skill))
}

// ─── Run ──────────────────────────────────────────────────────

func handleRun(args []string, registry *skills.Registry, renderer *output.Renderer, safetyLayer *safety.Layer, stateManager *state.Manager, pe *policy.Engine) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore run <skill_name> [--param key=value ...]")
		return
	}
	skillName := args[0]
	skill, err := registry.Get(skillName)
	if err != nil {
		fmt.Println(renderer.RenderError(err))
		return
	}
	params := parseParams(args[1:])
	env := stateManager.GetEnvironment()
	if e := extractFlag(args[1:], "--env"); e != "" {
		env = e
		stateManager.SetEnvironment(env)
	}

	// Policy check
	policyResult := pe.Evaluate(skill, params, env)
	if !policyResult.Passed {
		fmt.Print(policyResult.Render())
		return
	}
	if len(policyResult.Warnings) > 0 {
		fmt.Print(policyResult.Render())
	}

	// Safety evaluation
	report := safetyLayer.Evaluate(skill, params, env)
	fmt.Print(renderer.RenderSafetyReport(report))

	stateManager.LoadSkill(skillName)
	stateManager.AddToAuditLog(skillName, "evaluate",
		fmt.Sprintf("%s/%s/%s", env, stateManager.GetProvider(), stateManager.GetRegion()),
		core.StatusDryRun, skill.RiskLevel, "Safety evaluation completed — dry run mode")
	fmt.Println()
	fmt.Println(renderer.RenderSuccess(fmt.Sprintf("Skill '%s' evaluated in dry-run mode. Use --force to execute.", skillName)))
}

// ─── Plan ─────────────────────────────────────────────────────

func handlePlan(args []string, renderer *output.Renderer, planEngine *planner.Engine) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore plan <description>")
		return
	}
	description := strings.Join(args, " ")
	plan := planEngine.CreatePlan("Execution Plan", description)
	dl := strings.ToLower(description)
	switch {
	case strings.Contains(dl, "deploy"):
		_ = planEngine.AddStep(plan, "k8s.deploy", "Deploy workload", map[string]interface{}{"namespace": "default", "deployment": "app", "image": "app:latest"})
		_ = planEngine.AddStep(plan, "k8s.rollout.status", "Watch rollout", map[string]interface{}{"namespace": "default", "deployment": "app"})
	case strings.Contains(dl, "audit") || strings.Contains(dl, "security"):
		_ = planEngine.AddStep(plan, "aws.iam.audit", "Audit IAM", nil)
		_ = planEngine.AddStep(plan, "aws.sg.audit", "Audit SGs", nil)
		_ = planEngine.AddStep(plan, "aws.s3.audit", "Audit S3", nil)
	case strings.Contains(dl, "cost"):
		_ = planEngine.AddStep(plan, "aws.cost.report", "Cost report", map[string]interface{}{"granularity": "MONTHLY"})
		_ = planEngine.AddStep(plan, "aws.rightsizing.suggest", "Rightsizing", nil)
	default:
		fmt.Printf("📋 Plan requested: %s\n\nCould not auto-generate. Use 'infracore skills list'.\n", description)
		return
	}
	fmt.Print(renderer.RenderPlan(plan))
}

// ─── State ────────────────────────────────────────────────────

func handleState(renderer *output.Renderer, sm *state.Manager) {
	s := sm.GetState()
	fmt.Print(renderer.RenderSessionState(&s))
}

// ─── Discover ─────────────────────────────────────────────────

func handleDiscover(args []string, registry *skills.Registry, renderer *output.Renderer) {
	provider := extractFlag(args, "--provider")
	action := extractFlag(args, "--action")
	if provider == "" {
		provider = "custom"
	}
	if action == "" {
		action = "action"
	}
	discovery := skills.NewDiscovery(registry)
	template := discovery.GenerateTemplate(provider, action)
	fmt.Println("🔍 SKILL DISCOVERY MODE")
	fmt.Println(renderer.RenderWarning("No matching skill found. Generate a custom skill definition:"))
	fmt.Println()
	fmt.Println(template)
}

// ─── Policy ───────────────────────────────────────────────────

func handlePolicy(args []string, pe *policy.Engine, registry *skills.Registry, renderer *output.Renderer) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore policy <list|check> [options]")
		return
	}
	switch args[0] {
	case "list":
		policies := pe.ListPolicies()
		fmt.Printf("🛡️  POLICIES (%d registered)\n", len(policies))
		for _, p := range policies {
			fmt.Printf("  • %-25s [%s/%s] %s\n", p.Name, p.Enforcement, p.Severity, p.Description)
		}
	case "check":
		if len(args) < 2 {
			fmt.Println("Usage: infracore policy check <skill_name> [--env=<env>]")
			return
		}
		skill, err := registry.Get(args[1])
		if err != nil {
			fmt.Println(renderer.RenderError(err))
			return
		}
		env := extractFlag(args[2:], "--env")
		if env == "" {
			env = "staging"
		}
		params := parseParams(args[2:])
		result := pe.Evaluate(skill, params, env)
		fmt.Print(result.Render())
	}
}

// ─── Compliance ───────────────────────────────────────────────

func handleCompliance(args []string, auditor *compliance.Auditor) {
	if len(args) < 2 || args[0] != "audit" {
		fmt.Println("Usage: infracore compliance audit <CIS|SOC2|HIPAA>")
		return
	}
	fw := compliance.Framework(strings.ToUpper(args[1]))
	report := auditor.RunAudit(fw)
	fmt.Print(report.Render())
}

// ─── Drift ────────────────────────────────────────────────────

func handleDrift(args []string, detector *drift.Detector) {
	if len(args) == 0 || args[0] != "detect" {
		fmt.Println("Usage: infracore drift detect")
		return
	}
	// Demo drift detection with sample terraform plan output
	samplePlan := `
# aws_instance.web will be updated in-place
  ~ instance_type = "t3.micro" -> "t3.large"
# aws_s3_bucket.logs will be created
# aws_security_group.old will be destroyed
`
	report := detector.AnalyzeTerraformPlan(samplePlan)
	report.Environment = "staging"
	report.Region = "us-east-1"
	fmt.Print(report.Render())
}

// ─── Runbook ──────────────────────────────────────────────────

func handleRunbook(args []string, engine *runbook.Engine) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore runbook <list|run|info> [name]")
		return
	}
	switch args[0] {
	case "list":
		rbs := engine.List()
		fmt.Printf("📖 RUNBOOKS (%d registered)\n", len(rbs))
		for _, rb := range rbs {
			fmt.Printf("  • %-25s [%s] %d steps — %s\n", rb.Name, rb.Trigger, len(rb.Steps), rb.Description)
		}
	case "info":
		if len(args) < 2 {
			fmt.Println("Usage: infracore runbook info <name>")
			return
		}
		rb, err := engine.Get(args[1])
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
		fmt.Print(rb.Render())
	case "run":
		if len(args) < 2 {
			fmt.Println("Usage: infracore runbook run <name>")
			return
		}
		rb, err := engine.Get(args[1])
		if err != nil {
			fmt.Printf("❌ %v\n", err)
			return
		}
		log := engine.SimulateRun(rb)
		fmt.Print(log.Render())
	}
}

// ─── Health ───────────────────────────────────────────────────

func handleHealth(args []string, checker *health.Checker) {
	if len(args) == 0 || args[0] != "check" {
		fmt.Println("Usage: infracore health check [--tag=<tag>]")
		return
	}
	tag := extractFlag(args[1:], "--tag")
	var report *health.HealthReport
	if tag != "" {
		report = checker.RunByTag(tag)
	} else {
		report = checker.RunAll()
	}
	fmt.Print(report.Render())
}

// ─── Config ───────────────────────────────────────────────────

func handleConfig(args []string, cfg *config.Config) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore config <show|init>")
		return
	}
	switch args[0] {
	case "show":
		fmt.Print(cfg.Render())
	case "init":
		fmt.Println(config.GenerateConfigYAML())
	}
}

// ─── RBAC ─────────────────────────────────────────────────────

func handleRBAC(args []string, engine *rbac.Engine) {
	if len(args) == 0 || args[0] != "show" {
		fmt.Println("Usage: infracore rbac show")
		return
	}
	fmt.Print(engine.Render())
}

// ─── Helpers ──────────────────────────────────────────────────

func extractFlag(args []string, flag string) string {
	prefix := flag + "="
	for _, arg := range args {
		if strings.HasPrefix(arg, prefix) {
			return strings.TrimPrefix(arg, prefix)
		}
	}
	return ""
}

func parseParams(args []string) map[string]interface{} {
	params := make(map[string]interface{})
	for _, arg := range args {
		if strings.HasPrefix(arg, "--param") {
			continue
		}
		if strings.Contains(arg, "=") && !strings.HasPrefix(arg, "--") {
			parts := strings.SplitN(arg, "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}
	}
	for i, arg := range args {
		if arg == "--param" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parts[1]
			}
		}
	}
	return params
}
