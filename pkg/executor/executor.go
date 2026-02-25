// Package executor provides the execution engine for running skills
// against cloud APIs and CLI tools with dry-run support.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/parth14193/inframesh/pkg/core"
	"github.com/parth14193/inframesh/pkg/rbac"
	"github.com/parth14193/inframesh/pkg/resilience"
	"github.com/parth14193/inframesh/pkg/safety"
)

// Executor defines the interface for skill execution.
type Executor interface {
	Execute(ctx context.Context, skill *core.Skill, params map[string]interface{}, env string) *core.ExecutionResult
}

// RBACChecker is the subset of rbac.Engine used by the executor.
// Keeping it as an interface allows tests to substitute a mock.
type RBACChecker interface {
	CanExecute(username string, skill *core.Skill, env string) (bool, string)
}

// Ensure rbac.Engine satisfies the interface at compile time.
var _ RBACChecker = (*rbac.Engine)(nil)

// CLIExecutor runs skills by shelling out to cloud CLI tools.
type CLIExecutor struct {
	safetyLayer *safety.Layer
	dryRun      bool
	workDir     string
	rbacEngine  RBACChecker
	activeUser  string
	retryPolicy *resilience.RetryPolicy
}

// NewCLIExecutor creates a new CLIExecutor.
func NewCLIExecutor(safetyLayer *safety.Layer, dryRun bool) *CLIExecutor {
	return &CLIExecutor{
		safetyLayer: safetyLayer,
		dryRun:      dryRun,
	}
}

// WithRBAC attaches an RBAC engine and the calling operator's username.
// When set, every Execute call is gated by role-based access control before
// any safety or dry-run checks are applied.
func (e *CLIExecutor) WithRBAC(engine *rbac.Engine, username string) *CLIExecutor {
	e.rbacEngine = engine
	e.activeUser = username
	return e
}

// WithRetryPolicy attaches a resilience retry policy used to retry transient command failures.
// Use resilience.DefaultRetryPolicy() for sensible defaults.
func (e *CLIExecutor) WithRetryPolicy(policy *resilience.RetryPolicy) *CLIExecutor {
	e.retryPolicy = policy
	return e
}

// SetWorkDir sets the working directory for command execution.
func (e *CLIExecutor) SetWorkDir(dir string) {
	e.workDir = dir
}

// Execute runs a skill's command, interpolating parameters and capturing output.
func (e *CLIExecutor) Execute(ctx context.Context, skill *core.Skill, params map[string]interface{}, env string) *core.ExecutionResult {
	start := time.Now()
	result := &core.ExecutionResult{
		SkillName: skill.Name,
		Timestamp: start,
		Output:    make(map[string]interface{}),
	}

	// ── 1. RBAC check — first gate, cheapest to evaluate. ────────────────
	if e.rbacEngine != nil {
		if allowed, reason := e.rbacEngine.CanExecute(e.activeUser, skill, env); !allowed {
			result.Status = core.StatusCancelled
			result.Error = "rbac_denied"
			result.Message = fmt.Sprintf("🚫 RBAC: %s", reason)
			result.Duration = time.Since(start)
			return result
		}
	}

	// ── 2. Safety check ──────────────────────────────────────────────────
	if e.safetyLayer != nil {
		report := e.safetyLayer.Evaluate(skill, params, env)
		if report.RequiresConfirmation && !e.hasConfirmation(params) {
			result.Status = core.StatusPending
			result.Message = fmt.Sprintf("Action requires confirmation: %s", report.ConfirmationPrompt)
			result.Duration = time.Since(start)
			return result
		}
	}

	// ── 3. Dry run mode ──────────────────────────────────────────────────
	if e.dryRun || e.shouldDryRun(skill, params) {
		result.Status = core.StatusDryRun
		result.Message = fmt.Sprintf("[DRY RUN] Would execute: %s", e.interpolateCommand(skill.Execution.Command, params))
		result.Output["command"] = e.interpolateCommand(skill.Execution.Command, params)
		result.Output["params"] = params
		result.Duration = time.Since(start)
		return result
	}

	// ── 4. Build and execute the command (with optional retry) ───────────
	command := e.interpolateCommand(skill.Execution.Command, params)
	timeout := skill.Execution.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var stdout, stderr string
	var exitCode int
	var err error

	if e.retryPolicy != nil {
		// Wrap the command in a retryable function.
		result2 := resilience.WithRetry(e.retryPolicy, func() error {
			stdout, stderr, exitCode, err = e.runCommand(cmdCtx, command)
			// Only signal retry for transient errors; permanent failures stop.
			if err != nil && isTransientError(exitCode, stderr) {
				return err
			}
			return nil
		})
		if !result2.Succeeded && err == nil {
			err = fmt.Errorf("command failed after %d attempts: %s", result2.Attempts, result2.LastError)
		}
	} else {
		stdout, stderr, exitCode, err = e.runCommand(cmdCtx, command)
	}

	result.Duration = time.Since(start)
	result.Output["stdout"] = stdout
	result.Output["stderr"] = stderr
	result.Output["exit_code"] = exitCode
	result.Output["command"] = command

	if err != nil {
		result.Status = core.StatusFailed
		result.Error = err.Error()
		result.Message = fmt.Sprintf("Command failed (exit %d): %s", exitCode, truncate(stderr, 200))
	} else {
		result.Status = core.StatusSuccess
		result.Message = fmt.Sprintf("Completed successfully in %s", result.Duration.Round(time.Millisecond))
	}

	return result
}

// interpolateCommand replaces {param} placeholders in the command template.
func (e *CLIExecutor) interpolateCommand(template string, params map[string]interface{}) string {
	result := template
	for key, val := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", val))
	}
	return result
}

// runCommand executes a shell command and returns stdout, stderr, and exit code.
func (e *CLIExecutor) runCommand(ctx context.Context, command string) (string, string, int, error) {
	var cmd *exec.Cmd

	// Use appropriate shell
	if isWindows() {
		cmd = exec.CommandContext(ctx, "cmd", "/C", command)
	} else {
		cmd = exec.CommandContext(ctx, "sh", "-c", command)
	}

	if e.workDir != "" {
		cmd.Dir = e.workDir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = -1
		}
	}

	return stdout.String(), stderr.String(), exitCode, err
}

// hasConfirmation checks if the params include a confirmation flag.
func (e *CLIExecutor) hasConfirmation(params map[string]interface{}) bool {
	if params == nil {
		return false
	}
	if confirm, ok := params["_confirmed"]; ok {
		if b, ok := confirm.(bool); ok {
			return b
		}
	}
	return false
}

// shouldDryRun checks if this skill type defaults to dry-run.
func (e *CLIExecutor) shouldDryRun(skill *core.Skill, params map[string]interface{}) bool {
	return skill.RiskLevel >= core.RiskHigh && !e.hasForce(params)
}

// hasForce checks if force flag is set.
func (e *CLIExecutor) hasForce(params map[string]interface{}) bool {
	if params == nil {
		return false
	}
	if force, ok := params["_force"]; ok {
		if b, ok := force.(bool); ok {
			return b
		}
	}
	return false
}

// isTransientError returns true for exit codes that suggest temporary infra issues.
func isTransientError(exitCode int, stderr string) bool {
	if exitCode == -1 {
		return true // execution error (e.g. binary not found) — don't retry
	}
	s := strings.ToLower(stderr)
	return strings.Contains(s, "timeout") ||
		strings.Contains(s, "connection refused") ||
		strings.Contains(s, "network") ||
		strings.Contains(s, "throttl") ||
		strings.Contains(s, "rate limit")
}

func isWindows() bool {
	return runtime.GOOS == "windows"
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// ── DryRunExecutor ─────────────────────────────────────────────

// DryRunExecutor always simulates execution without side effects.
type DryRunExecutor struct{}

// NewDryRunExecutor creates a new DryRunExecutor.
func NewDryRunExecutor() *DryRunExecutor {
	return &DryRunExecutor{}
}

// Execute simulates skill execution and returns a dry-run result.
func (e *DryRunExecutor) Execute(_ context.Context, skill *core.Skill, params map[string]interface{}, env string) *core.ExecutionResult {
	start := time.Now()

	cmd := skill.Execution.Command
	for key, val := range params {
		cmd = strings.ReplaceAll(cmd, fmt.Sprintf("{%s}", key), fmt.Sprintf("%v", val))
	}

	return &core.ExecutionResult{
		SkillName: skill.Name,
		Status:    core.StatusDryRun,
		Output: map[string]interface{}{
			"command":     cmd,
			"environment": env,
			"params":      params,
			"risk_level":  skill.RiskLevel.String(),
		},
		Message:   fmt.Sprintf("[DRY RUN] Would execute: %s (env=%s, risk=%s)", cmd, env, skill.RiskLevel),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// ── CompositeExecutor ──────────────────────────────────────────

// CompositeExecutor chains multiple executors with pre/post hooks.
type CompositeExecutor struct {
	primary   Executor
	preHooks  []ExecutionHook
	postHooks []ExecutionHook
}

// ExecutionHook is called before or after skill execution.
type ExecutionHook func(skill *core.Skill, params map[string]interface{}, result *core.ExecutionResult)

// NewCompositeExecutor creates an executor with hook support.
func NewCompositeExecutor(primary Executor) *CompositeExecutor {
	return &CompositeExecutor{
		primary:   primary,
		preHooks:  []ExecutionHook{},
		postHooks: []ExecutionHook{},
	}
}

// AddPreHook adds a hook to run before execution.
func (e *CompositeExecutor) AddPreHook(hook ExecutionHook) {
	e.preHooks = append(e.preHooks, hook)
}

// AddPostHook adds a hook to run after execution.
func (e *CompositeExecutor) AddPostHook(hook ExecutionHook) {
	e.postHooks = append(e.postHooks, hook)
}

// Execute runs all pre-hooks, executes the skill, then runs post-hooks.
func (e *CompositeExecutor) Execute(ctx context.Context, skill *core.Skill, params map[string]interface{}, env string) *core.ExecutionResult {
	// Pre-hooks
	for _, hook := range e.preHooks {
		hook(skill, params, nil)
	}

	// Execute
	result := e.primary.Execute(ctx, skill, params, env)

	// Post-hooks
	for _, hook := range e.postHooks {
		hook(skill, params, result)
	}

	return result
}
