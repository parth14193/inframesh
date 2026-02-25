// drift_watch.go — scheduled drift detection watcher.
// Implements `infracore drift watch --interval=15m` behaviour.
package drift

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// WatchResult wraps a DriftReport with its detection timestamp and any error.
type WatchResult struct {
	Report    *DriftReport
	RunAt     time.Time
	Iteration int
	Err       error
}

// WatchConfig configures the drift watcher.
type WatchConfig struct {
	// Interval between drift detection runs. Minimum is 1 minute.
	Interval time.Duration

	// Provider, Region, Environment to tag each report with.
	Provider    string
	Region      string
	Environment string

	// TerraformPlanFn, when set, is called each tick to produce Terraform plan
	// output for AnalyzeTerraformPlan. If nil, only ManualChangesFn is called.
	TerraformPlanFn func(ctx context.Context) (string, error)

	// ManualChangesFn, when set, is called each tick and returns (live, declared)
	// resource ID slices for DetectManualChanges.
	ManualChangesFn func(ctx context.Context) (live, declared []string, resourceType string, err error)

	// OnResult is called synchronously after each drift detection run.
	// Use it to log, notify, or persist the result.
	OnResult func(result WatchResult)
}

// Watcher performs periodic drift detection.
type Watcher struct {
	mu       sync.Mutex
	detector *Detector
	cfg      WatchConfig
	running  bool
	cancel   context.CancelFunc
	history  []WatchResult
}

// NewWatcher creates a drift watcher for the given Detector and config.
// If cfg.Interval < 1 minute it is clamped to 1 minute.
func NewWatcher(detector *Detector, cfg WatchConfig) *Watcher {
	if cfg.Interval < time.Minute {
		cfg.Interval = time.Minute
	}
	return &Watcher{
		detector: detector,
		cfg:      cfg,
	}
}

// Start begins the watch loop in a background goroutine.
// Call Stop() to halt it. Calling Start() on an already-running watcher is a no-op.
func (w *Watcher) Start(ctx context.Context) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return fmt.Errorf("drift watcher is already running")
	}

	watchCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.running = true

	go w.loop(watchCtx)
	return nil
}

// Stop halts the watch loop. Safe to call multiple times.
func (w *Watcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.cancel != nil {
		w.cancel()
	}
	w.running = false
}

// IsRunning returns true if the watcher goroutine is active.
func (w *Watcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// History returns a snapshot of all past drift detection results.
func (w *Watcher) History() []WatchResult {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make([]WatchResult, len(w.history))
	copy(out, w.history)
	return out
}

// RunOnce executes a single drift detection pass synchronously and returns the result.
func (w *Watcher) RunOnce(ctx context.Context) WatchResult {
	iteration := 0
	w.mu.Lock()
	iteration = len(w.history) + 1
	w.mu.Unlock()

	result := w.detect(ctx, iteration)

	w.mu.Lock()
	w.history = append(w.history, result)
	w.mu.Unlock()

	return result
}

// loop is the internal goroutine that fires a detection on every tick.
func (w *Watcher) loop(ctx context.Context) {
	defer func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
	}()

	// Run immediately on start, then on each interval.
	iteration := 0
	runAndRecord := func() {
		iteration++
		result := w.detect(ctx, iteration)

		w.mu.Lock()
		w.history = append(w.history, result)
		w.mu.Unlock()

		if w.cfg.OnResult != nil {
			w.cfg.OnResult(result)
		}
	}

	runAndRecord()

	ticker := time.NewTicker(w.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runAndRecord()
		}
	}
}

// detect executes all configured detection strategies and merges results.
func (w *Watcher) detect(ctx context.Context, iteration int) WatchResult {
	result := WatchResult{
		RunAt:     time.Now(),
		Iteration: iteration,
		Report: &DriftReport{
			Provider:    w.cfg.Provider,
			Region:      w.cfg.Region,
			Environment: w.cfg.Environment,
			Timestamp:   time.Now(),
		},
	}

	// Terraform plan drift
	if w.cfg.TerraformPlanFn != nil {
		planOutput, err := w.cfg.TerraformPlanFn(ctx)
		if err != nil {
			result.Err = fmt.Errorf("terraform plan: %w", err)
			return result
		}
		tfReport := w.detector.AnalyzeTerraformPlan(planOutput)
		result.Report.Resources = append(result.Report.Resources, tfReport.Resources...)
		result.Report.New += tfReport.New
		result.Report.Drifted += tfReport.Drifted
		result.Report.Deleted += tfReport.Deleted
	}

	// Manual changes drift
	if w.cfg.ManualChangesFn != nil {
		live, declared, resourceType, err := w.cfg.ManualChangesFn(ctx)
		if err != nil {
			result.Err = fmt.Errorf("manual changes: %w", err)
			return result
		}
		manualReport := w.detector.DetectManualChanges(w.cfg.Provider, resourceType, live, declared)
		result.Report.Resources = append(result.Report.Resources, manualReport.Resources...)
		result.Report.InSync += manualReport.InSync
		result.Report.New += manualReport.New
		result.Report.Deleted += manualReport.Deleted
	}

	return result
}

// HasDrift returns true if the most recent run found any non-in-sync resources.
func (w *Watcher) HasDrift() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.history) == 0 {
		return false
	}
	last := w.history[len(w.history)-1]
	if last.Report == nil {
		return false
	}
	return last.Report.Drifted > 0 || last.Report.New > 0 || last.Report.Deleted > 0
}
