// Package executor provides the execution engine for running skills
// against cloud APIs and CLI tools with dry-run support.
package executor

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/safety"
)

// Executor defines the interface for skill execution.
type Executor interface {
	Execute(ctx context.Context, skill *core.Skill, params map[string]interface{}, env string) *core.ExecutionResult
}

// CLIExecutor runs skills by shelling out to cloud CLI tools.
type CLIExecutor struct {
	safetyLayer *safety.Layer
	dryRun      bool
	workDir     string
}

// NewCLIExecutor creates a new CLIExecutor.
func NewCLIExecutor(safetyLayer *safety.Layer, dryRun bool) *CLIExecutor {
	return &CLIExecutor{
		safetyLayer: safetyLayer,
		dryRun:      dryRun,
	}
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

	// Safety check
	if e.safetyLayer != nil {
		report := e.safetyLayer.Evaluate(skill, params, env)
		if report.RequiresConfirmation && !e.hasConfirmation(params) {
			result.Status = core.StatusPending
			result.Message = fmt.Sprintf("Action requires confirmation: %s", report.ConfirmationPrompt)
			result.Duration = time.Since(start)
			return result
		}
	}

	// Dry run mode
	if e.dryRun || e.shouldDryRun(skill) {
		result.Status = core.StatusDryRun
		result.Message = fmt.Sprintf("[DRY RUN] Would execute: %s", e.interpolateCommand(skill.Execution.Command, params))
		result.Output["command"] = e.interpolateCommand(skill.Execution.Command, params)
		result.Output["params"] = params
		result.Duration = time.Since(start)
		return result
	}

	// Build and execute the command
	command := e.interpolateCommand(skill.Execution.Command, params)
	timeout := skill.Execution.Timeout
	if timeout == 0 {
		timeout = 60 * time.Second
	}

	cmdCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	stdout, stderr, exitCode, err := e.runCommand(cmdCtx, command)

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
func (e *CLIExecutor) shouldDryRun(skill *core.Skill) bool {
	return skill.RiskLevel >= core.RiskHigh && !e.hasForce(nil)
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

func isWindows() bool {
	return strings.Contains(strings.ToLower(fmt.Sprintf("%s", "os")), "windows")
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
			"params":     params,
			"risk_level": skill.RiskLevel.String(),
		},
		Message:   fmt.Sprintf("[DRY RUN] Would execute: %s (env=%s, risk=%s)", cmd, env, skill.RiskLevel),
		Duration:  time.Since(start),
		Timestamp: start,
	}
}

// ── CompositeExecutor ──────────────────────────────────────────

// CompositeExecutor chains multiple executors with pre/post hooks.
type CompositeExecutor struct {
	primary    Executor
	preHooks   []ExecutionHook
	postHooks  []ExecutionHook
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
