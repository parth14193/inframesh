// Package health — gate.go implements HealthGate, a blocking probe that
// pauses runbook/plan execution until a target becomes healthy or a deadline
// is reached. This allows deployment-rollback and similar runbooks to
// automatically verify health after each step before proceeding.
package health

import (
	"context"
	"fmt"
	"time"
)

// GateOutcome describes the result of a HealthGate wait.
type GateOutcome string

const (
	GateOutcomePassed  GateOutcome = "PASSED"  // all required probes became healthy
	GateOutcomeFailed  GateOutcome = "FAILED"  // deadline exceeded without reaching healthy
	GateOutcomeSkipped GateOutcome = "SKIPPED" // gate was configured with RequiredStatus = UNKNOWN
)

// GateConfig configures a single health gate.
type GateConfig struct {
	// Name is a human-readable identifier for log output.
	Name string

	// Probes to run on each poll tick. At least one must be provided.
	Probes []*Probe

	// RequiredStatus is the probe status that must be achieved.
	// Defaults to StatusHealthy.
	RequiredStatus ProbeStatus

	// Timeout is the maximum time to wait before declaring failure.
	// Defaults to 5 minutes.
	Timeout time.Duration

	// PollInterval is the frequency of probe re-evaluation.
	// Defaults to 15 seconds.
	PollInterval time.Duration

	// MinSuccessiveHealthy is the number of consecutive healthy polls
	// required before the gate passes. Defaults to 1.
	// Set to 2+ for additional confirmation on flaky endpoints.
	MinSuccessiveHealthy int

	// AllowDegraded, if true, accepts StatusDegraded as passing.
	AllowDegraded bool
}

// GateResult is returned after waiting on a HealthGate.
type GateResult struct {
	GateName          string        `json:"gate_name"`
	Outcome           GateOutcome   `json:"outcome"`
	Attempts          int           `json:"attempts"`
	SuccessiveHealthy int           `json:"successive_healthy"`
	WaitDuration      time.Duration `json:"wait_duration"`
	LastReport        *HealthReport `json:"last_report,omitempty"`
	Message           string        `json:"message"`
}

// Gate is a blocking health gate that polls probes until healthy or timed out.
type Gate struct {
	cfg     GateConfig
	checker *Checker
}

// NewGate creates a health gate with the given configuration.
// Defaults are applied for zero-value fields.
func NewGate(cfg GateConfig) *Gate {
	if cfg.RequiredStatus == "" {
		cfg.RequiredStatus = StatusHealthy
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Minute
	}
	if cfg.PollInterval == 0 {
		cfg.PollInterval = 15 * time.Second
	}
	if cfg.MinSuccessiveHealthy < 1 {
		cfg.MinSuccessiveHealthy = 1
	}

	checker := NewChecker()
	for _, p := range cfg.Probes {
		checker.AddProbe(p)
	}

	return &Gate{cfg: cfg, checker: checker}
}

// Wait blocks until all probes achieve the required status or the deadline passes.
// It respects ctx cancellation — cancellation returns a FAILED outcome.
func (g *Gate) Wait(ctx context.Context) GateResult {
	start := time.Now()
	result := GateResult{GateName: g.cfg.Name}

	if len(g.cfg.Probes) == 0 {
		result.Outcome = GateOutcomeSkipped
		result.Message = "no probes configured — gate skipped"
		return result
	}

	deadline := time.Now().Add(g.cfg.Timeout)
	ticker := time.NewTicker(g.cfg.PollInterval)
	defer ticker.Stop()

	successive := 0

	evaluate := func() *HealthReport {
		result.Attempts++
		report := g.checker.RunAll()
		result.LastReport = report
		return report
	}

	// First poll immediately (don't wait for the first tick).
	if report := evaluate(); g.isPassing(report) {
		successive++
	} else {
		successive = 0
	}

	for {
		if successive >= g.cfg.MinSuccessiveHealthy {
			result.Outcome = GateOutcomePassed
			result.SuccessiveHealthy = successive
			result.WaitDuration = time.Since(start)
			result.Message = fmt.Sprintf(
				"gate %q passed after %d attempt(s) in %s",
				g.cfg.Name, result.Attempts, result.WaitDuration.Round(time.Second),
			)
			return result
		}

		if time.Now().After(deadline) {
			result.Outcome = GateOutcomeFailed
			result.WaitDuration = time.Since(start)
			result.Message = fmt.Sprintf(
				"gate %q timed out after %d attempt(s) (%s) — last status: %s",
				g.cfg.Name, result.Attempts, result.WaitDuration.Round(time.Second),
				g.lastStatus(result.LastReport),
			)
			return result
		}

		select {
		case <-ctx.Done():
			result.Outcome = GateOutcomeFailed
			result.WaitDuration = time.Since(start)
			result.Message = fmt.Sprintf("gate %q cancelled by context", g.cfg.Name)
			return result

		case <-ticker.C:
			if report := evaluate(); g.isPassing(report) {
				successive++
			} else {
				successive = 0
			}
		}
	}
}

// isPassing returns true when the report meets the gate's success criteria.
func (g *Gate) isPassing(report *HealthReport) bool {
	if report == nil {
		return false
	}
	switch report.Overall {
	case StatusHealthy:
		return true
	case StatusDegraded:
		return g.cfg.AllowDegraded
	default:
		return false
	}
}

// lastStatus extracts the overall status string from the most recent report.
func (g *Gate) lastStatus(report *HealthReport) string {
	if report == nil {
		return "unknown"
	}
	return string(report.Overall)
}

// ─── Convenience constructors ──────────────────────────────────────────────

// HTTPGate creates a HealthGate that polls an HTTP endpoint.
func HTTPGate(name, url string, timeout, pollInterval time.Duration) *Gate {
	return NewGate(GateConfig{
		Name: name,
		Probes: []*Probe{{
			Name:           name,
			Type:           ProbeHTTP,
			Target:         url,
			Timeout:        10 * time.Second,
			ExpectedStatus: 200,
		}},
		Timeout:      timeout,
		PollInterval: pollInterval,
	})
}

// TCPGate creates a HealthGate that polls a TCP endpoint.
func TCPGate(name, address string, timeout, pollInterval time.Duration) *Gate {
	return NewGate(GateConfig{
		Name: name,
		Probes: []*Probe{{
			Name:    name,
			Type:    ProbeTCP,
			Target:  address,
			Timeout: 5 * time.Second,
		}},
		Timeout:      timeout,
		PollInterval: pollInterval,
	})
}

// Render formats a GateResult for terminal output.
func (r GateResult) Render() string {
	icon := "✅"
	if r.Outcome == GateOutcomeFailed {
		icon = "❌"
	} else if r.Outcome == GateOutcomeSkipped {
		icon = "⏭️"
	}
	return fmt.Sprintf("%s HEALTH GATE [%s] %s — %s\n", icon, r.GateName, r.Outcome, r.Message)
}
