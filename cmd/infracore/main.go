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
	"strconv"
	"strings"
	"time"

	"github.com/parth14193/ownbot/pkg/agents"
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
	case "agent":
		handleAgent(os.Args[2:], renderer)
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
  agent simulate   Run multi-agent controller simulation

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
  infracore health check
  infracore agent simulate --urgency=P0 --service=checkout-api --signal=latency --symptoms=5xx,timeout`)
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
	if !populatePlanFromDescription(planEngine, plan, description) {
		fmt.Printf("📋 Plan requested: %s\n\nCould not auto-generate. Use 'infracore skills list'.\n", description)
		return
	}
	fmt.Print(renderer.RenderPlan(plan))
}

func populatePlanFromDescription(planEngine *planner.Engine, plan *core.Plan, description string) bool {
	dl := strings.ToLower(description)
	containsAny := func(values ...string) bool {
		for _, value := range values {
			if strings.Contains(dl, value) {
				return true
			}
		}
		return false
	}
	isDeployIntent := containsAny("deploy", "launch", "provision", "create", "rollout")
	profile := "standard"
	if containsAny("secure", "hardened", "encrypted", "private") {
		profile = "secure"
	}
	if containsAny("cpu optimized", "cpu optimised", "compute optimized", "compute optimised", "high cpu") {
		profile = "cpu_optimized"
	}
	service := "generic"
	switch {
	case containsAny("eks"):
		service = "eks"
	case containsAny("ec2"):
		service = "ec2"
	case containsAny("rds"):
		service = "rds"
	case containsAny("lambda"):
		service = "lambda"
	case containsAny("ecs"):
		service = "ecs"
	case containsAny("gke"):
		service = "gke"
	case containsAny("gce", "compute engine", "google compute"):
		service = "compute"
	case containsAny("cloud sql", "sql"):
		service = "sql"
	case containsAny("aks"):
		service = "aks"
	case containsAny("azure vm", " vm "):
		service = "vm"
	}

	switch {
	case containsAny("deploy eks", "launch eks") || (strings.Contains(dl, "eks") && isDeployIntent):
		_ = planEngine.AddStep(plan, "aws.eks.deploy", "Deploy workload on EKS", map[string]interface{}{
			"cluster_name":       "main-cluster",
			"namespace":          "default",
			"deployment":         "app",
			"image":              "app:latest",
			"node_instance_type": "m6i.large",
			"min_nodes":          "2",
			"max_nodes":          "6",
		})
		_ = planEngine.AddStep(plan, "aws.sg.audit", "Validate security groups for EKS networking", nil)
	case strings.Contains(dl, "ec2") && containsAny("cpu optimized", "cpu optimised", "compute optimized", "compute optimised", "high cpu"):
		_ = planEngine.AddStep(plan, "aws.ec2.deploy.cpu_optimized", "Deploy EC2 workload with CPU-optimized profile", map[string]interface{}{
			"name":          "cpu-app",
			"instance_type": "c7i.large",
		})
		_ = planEngine.AddStep(plan, "aws.cloudwatch.query", "Query CPU metrics and logs post-deploy", map[string]interface{}{
			"log_group": "/aws/ec2/cpu-app",
			"query":     "fields @timestamp, @message | sort @timestamp desc | limit 20",
		})
	case strings.Contains(dl, "rds") && containsAny("secure", "hardened", "private", "encrypted", "launch", "deploy", "provision", "create"):
		_ = planEngine.AddStep(plan, "aws.rds.launch.secure", "Launch secure RDS instance", map[string]interface{}{
			"db_identifier":         "app-db",
			"engine":                "postgres",
			"instance_class":        "db.m6i.large",
			"storage_encrypted":     "true",
			"publicly_accessible":   "false",
			"deletion_protection":   "true",
			"backup_retention_days": "7",
		})
		_ = planEngine.AddStep(plan, "aws.sg.audit", "Audit DB security groups for restricted ingress", nil)
	case strings.Contains(dl, "gke") && isDeployIntent:
		_ = planEngine.AddStep(plan, "gcp.gke.deploy", "Deploy workload on GKE", map[string]interface{}{
			"cluster_name": "main-gke",
			"location":     "us-central1",
			"namespace":    "default",
			"deployment":   "app",
			"image":        "gcr.io/project/app:latest",
		})
	case containsAny("gce", "compute engine", "google compute") && containsAny("cpu optimized", "cpu optimised", "compute optimized", "compute optimised", "high cpu"):
		_ = planEngine.AddStep(plan, "gcp.gce.deploy.cpu_optimized", "Deploy GCE workload with CPU-optimized profile", map[string]interface{}{
			"instance":     "cpu-app",
			"machine_type": "c3-standard-4",
			"zone":         "us-central1-a",
		})
	case containsAny("cloud sql", "gcp sql") && containsAny("secure", "hardened", "private", "encrypted", "launch", "deploy", "provision", "create"):
		_ = planEngine.AddStep(plan, "gcp.sql.deploy.secure", "Launch secure Cloud SQL instance", map[string]interface{}{
			"instance":         "app-db",
			"database_version": "POSTGRES_15",
			"tier":             "db-custom-2-7680",
			"region":           "us-central1",
			"private_ip":       "true",
			"backup_enabled":   "true",
		})
	case strings.Contains(dl, "aks") && isDeployIntent:
		_ = planEngine.AddStep(plan, "azure.aks.deploy", "Deploy workload on AKS", map[string]interface{}{
			"resource_group": "rg-main",
			"cluster_name":   "aks-main",
			"namespace":      "default",
			"deployment":     "app",
			"image":          "app:latest",
		})
	case containsAny("azure vm", "vm") && containsAny("cpu optimized", "cpu optimised", "compute optimized", "compute optimised", "high cpu"):
		_ = planEngine.AddStep(plan, "azure.vm.deploy.cpu_optimized", "Deploy Azure VM with CPU-optimized profile", map[string]interface{}{
			"resource_group": "rg-main",
			"vm_name":        "cpu-app",
			"vm_size":        "Standard_F4s_v2",
		})
	case containsAny("azure sql", "sql database", "sql db") && containsAny("secure", "hardened", "private", "encrypted", "launch", "deploy", "provision", "create"):
		_ = planEngine.AddStep(plan, "azure.sql.deploy.secure", "Launch secure Azure SQL database", map[string]interface{}{
			"resource_group":        "rg-main",
			"server_name":           "sql-main",
			"database_name":         "appdb",
			"service_objective":     "S2",
			"private_endpoint":      "true",
			"backup_retention_days": "7",
		})
	case strings.Contains(dl, "aws") && isDeployIntent:
		_ = planEngine.AddStep(plan, "aws.service.deploy", "Deploy AWS service from natural-language request", map[string]interface{}{
			"service":     service,
			"profile":     profile,
			"environment": "staging",
		})
	case containsAny("gcp", "google cloud") && isDeployIntent:
		_ = planEngine.AddStep(plan, "gcp.service.deploy", "Deploy GCP service from natural-language request", map[string]interface{}{
			"service":     service,
			"profile":     profile,
			"environment": "staging",
		})
	case strings.Contains(dl, "azure") && isDeployIntent:
		_ = planEngine.AddStep(plan, "azure.service.deploy", "Deploy Azure service from natural-language request", map[string]interface{}{
			"service":     service,
			"profile":     profile,
			"environment": "staging",
		})
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
		return false
	}

	return true
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

// --- Agent --------------------------------------------------

func handleAgent(args []string, renderer *output.Renderer) {
	if len(args) == 0 {
		fmt.Println("Usage: infracore agent simulate [--env=<env>] [--urgency=<P0|P1|P2>] [--service=<name>] [--signal=<type>] [--symptoms=a,b]")
		return
	}
	switch args[0] {
	case "simulate":
		handleAgentSimulate(args[1:], renderer)
	default:
		fmt.Fprintf(os.Stderr, "Unknown agent subcommand: %s\n", args[0])
	}
}

func handleAgentSimulate(args []string, renderer *output.Renderer) {
	env := extractFlag(args, "--env")
	if env == "" {
		env = "staging"
	}
	urgency := extractFlag(args, "--urgency")
	if urgency == "" {
		urgency = "P2"
	}
	service := extractFlag(args, "--service")
	if service == "" {
		service = "platform-api"
	}
	signal := extractFlag(args, "--signal")
	if signal == "" {
		signal = "latency"
	}
	symptomRaw := extractFlag(args, "--symptoms")
	var symptoms []string
	if symptomRaw != "" {
		for _, value := range strings.Split(symptomRaw, ",") {
			v := strings.TrimSpace(value)
			if v != "" {
				symptoms = append(symptoms, v)
			}
		}
	}

	controller := agents.NewController(nil)
	ctx := agents.EvaluationContext{
		Environment: env,
		Urgency:     urgency,
		Service:     service,
		SignalType:  signal,
		Symptoms:    symptoms,
		Timestamp:   time.Now(),
	}
	decision := controller.Decide(ctx)

	fmt.Printf("MULTI-AGENT DECISION (%s)\n", decision.Controller)
	headers := []string{"Agent", "Summary", "Risk", "Skills", "NeedsApproval"}
	rows := make([][]string, 0, len(decision.SelectedProposals))
	for _, p := range decision.SelectedProposals {
		rows = append(rows, []string{
			string(p.Agent),
			p.Summary,
			p.RiskLevel.String(),
			strings.Join(p.Skills, ","),
			fmt.Sprintf("%t", p.RequiresConfirmation),
		})
	}
	if len(rows) > 0 {
		fmt.Print(renderer.RenderTable(headers, rows))
	} else {
		fmt.Println("No proposals selected.")
	}
	fmt.Printf("Requires human approval: %t\n", decision.RequiresHumanApproval)
	fmt.Println("Reasoning:")
	for _, r := range decision.Reasoning {
		fmt.Printf("  - %s\n", r)
	}
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
				params[parts[0]] = parseParamValue(parts[1])
			}
		}
	}
	for i, arg := range args {
		if arg == "--param" && i+1 < len(args) {
			parts := strings.SplitN(args[i+1], "=", 2)
			if len(parts) == 2 {
				params[parts[0]] = parseParamValue(parts[1])
			}
		}
	}
	return params
}

func parseParamValue(value string) interface{} {
	v := strings.TrimSpace(value)
	if strings.EqualFold(v, "true") {
		return true
	}
	if strings.EqualFold(v, "false") {
		return false
	}
	if i, err := strconv.Atoi(v); err == nil {
		return i
	}
	return value
}
