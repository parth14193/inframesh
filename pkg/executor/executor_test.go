package executor

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/parth14193/inframesh/pkg/core"
	"github.com/parth14193/inframesh/pkg/safety"
)

func testEchoCommand() string {
	if runtime.GOOS == "windows" {
		return "echo forced-exec"
	}
	return "echo forced-exec"
}

func TestCLIExecutorHighRiskDefaultsToDryRun(t *testing.T) {
	exec := NewCLIExecutor(nil, false)
	skill := &core.Skill{
		Name:      "k8s.deploy",
		RiskLevel: core.RiskHigh,
		Execution: core.ExecutionConfig{
			Type:    core.ExecCLI,
			Command: testEchoCommand(),
			Timeout: 2 * time.Second,
		},
	}

	result := exec.Execute(context.Background(), skill, map[string]interface{}{}, "staging")
	if result.Status != core.StatusDryRun {
		t.Fatalf("expected dry_run, got %s", result.Status)
	}
}

func TestCLIExecutorForceRunsCommand(t *testing.T) {
	exec := NewCLIExecutor(nil, false)
	skill := &core.Skill{
		Name:      "k8s.deploy",
		RiskLevel: core.RiskHigh,
		Execution: core.ExecutionConfig{
			Type:    core.ExecCLI,
			Command: testEchoCommand(),
			Timeout: 2 * time.Second,
		},
	}

	result := exec.Execute(context.Background(), skill, map[string]interface{}{"_force": true}, "staging")
	if result.Status != core.StatusSuccess {
		t.Fatalf("expected success, got %s (%s)", result.Status, result.Message)
	}

	stdout, _ := result.Output["stdout"].(string)
	if !strings.Contains(stdout, "forced-exec") {
		t.Fatalf("expected stdout to contain forced-exec, got: %q", stdout)
	}
}

func TestCLIExecutorRequiresConfirmationWithoutConfirmedFlag(t *testing.T) {
	exec := NewCLIExecutor(safety.NewLayer(), false)
	skill := &core.Skill{
		Name:      "aws.ec2.list",
		RiskLevel: core.RiskLow,
		Execution: core.ExecutionConfig{
			Type:    core.ExecCLI,
			Command: testEchoCommand(),
			Timeout: 2 * time.Second,
		},
	}

	result := exec.Execute(context.Background(), skill, map[string]interface{}{}, "production")
	if result.Status != core.StatusPending {
		t.Fatalf("expected pending, got %s", result.Status)
	}
}
