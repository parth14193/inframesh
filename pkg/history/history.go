// Package history provides persistent execution history with
// file-based JSON-lines storage, search, replay, and statistics.
package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/parth14193/inframesh/pkg/core"
)

// Record captures a complete execution record.
type Record struct {
	ID          string                 `json:"id"`
	SkillName   string                 `json:"skill_name"`
	Action      string                 `json:"action"` // run, plan-execute, simulate
	Environment string                 `json:"environment"`
	Provider    string                 `json:"provider"`
	Region      string                 `json:"region"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Status      core.ExecutionStatus   `json:"status"`
	RiskLevel   core.RiskLevel         `json:"risk_level"`
	Duration    time.Duration          `json:"duration"`
	Message     string                 `json:"message,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	User        string                 `json:"user,omitempty"`
}

// Stats aggregates execution statistics.
type Stats struct {
	TotalExecutions int            `json:"total_executions"`
	Successes       int            `json:"successes"`
	Failures        int            `json:"failures"`
	DryRuns         int            `json:"dry_runs"`
	AvgDuration     time.Duration  `json:"avg_duration"`
	TopSkills       []SkillCount   `json:"top_skills"`
	ByProvider      map[string]int `json:"by_provider"`
	ByEnvironment   map[string]int `json:"by_environment"`
	FailureRate     float64        `json:"failure_rate"`
}

// SkillCount tracks execution frequency.
type SkillCount struct {
	Skill string `json:"skill"`
	Count int    `json:"count"`
}

// SearchQuery defines filters for searching history.
type SearchQuery struct {
	Skill  string
	Status string
	Env    string
	Last   int
	Since  time.Time
}

// Store manages persistent execution history.
type Store struct {
	mu      sync.Mutex
	dir     string
	records []Record
	loaded  bool
}

// NewStore creates a history store at the default location.
func NewStore() *Store {
	return &Store{
		dir:     defaultHistoryDir(),
		records: []Record{},
	}
}

// NewStoreAt creates a history store at the given directory.
func NewStoreAt(dir string) *Store {
	return &Store{
		dir:     dir,
		records: []Record{},
	}
}

// Append adds a new execution record and persists it.
func (s *Store) Append(record Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ensureLoaded()

	if record.ID == "" {
		record.ID = fmt.Sprintf("exec-%d", time.Now().UnixNano())
	}
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	s.records = append(s.records, record)

	// Persist to file
	return s.appendToFile(record)
}

// Search returns records matching the query.
func (s *Store) Search(query SearchQuery) []Record {
	s.mu.Lock()
	s.ensureLoaded()
	records := make([]Record, len(s.records))
	copy(records, s.records)
	s.mu.Unlock()

	var results []Record
	for _, r := range records {
		if query.Skill != "" && !strings.Contains(strings.ToLower(r.SkillName), strings.ToLower(query.Skill)) {
			continue
		}
		if query.Status != "" && string(r.Status) != query.Status {
			continue
		}
		if query.Env != "" && !strings.EqualFold(r.Environment, query.Env) {
			continue
		}
		if !query.Since.IsZero() && r.Timestamp.Before(query.Since) {
			continue
		}
		results = append(results, r)
	}

	// Sort by timestamp descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	if query.Last > 0 && len(results) > query.Last {
		results = results[:query.Last]
	}

	return results
}

// GetByID retrieves a specific record.
func (s *Store) GetByID(id string) (*Record, error) {
	s.mu.Lock()
	s.ensureLoaded()
	s.mu.Unlock()

	for _, r := range s.records {
		if r.ID == id {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("execution record not found: %s", id)
}

// GetStats returns aggregated execution statistics.
func (s *Store) GetStats() Stats {
	s.mu.Lock()
	s.ensureLoaded()
	records := make([]Record, len(s.records))
	copy(records, s.records)
	s.mu.Unlock()

	stats := Stats{
		TotalExecutions: len(records),
		ByProvider:      make(map[string]int),
		ByEnvironment:   make(map[string]int),
	}

	skillCounts := make(map[string]int)
	var totalDuration time.Duration

	for _, r := range records {
		switch r.Status {
		case core.StatusSuccess:
			stats.Successes++
		case core.StatusFailed:
			stats.Failures++
		case core.StatusDryRun:
			stats.DryRuns++
		}
		totalDuration += r.Duration
		skillCounts[r.SkillName]++
		stats.ByProvider[r.Provider]++
		stats.ByEnvironment[r.Environment]++
	}

	if stats.TotalExecutions > 0 {
		stats.AvgDuration = totalDuration / time.Duration(stats.TotalExecutions)
		stats.FailureRate = float64(stats.Failures) / float64(stats.TotalExecutions) * 100
	}

	// Top skills
	for skill, count := range skillCounts {
		stats.TopSkills = append(stats.TopSkills, SkillCount{Skill: skill, Count: count})
	}
	sort.Slice(stats.TopSkills, func(i, j int) bool {
		return stats.TopSkills[i].Count > stats.TopSkills[j].Count
	})
	if len(stats.TopSkills) > 10 {
		stats.TopSkills = stats.TopSkills[:10]
	}

	return stats
}

// Count returns the total number of records.
func (s *Store) Count() int {
	s.mu.Lock()
	s.ensureLoaded()
	s.mu.Unlock()
	return len(s.records)
}

// ensureLoaded loads records from disk if not already loaded.
func (s *Store) ensureLoaded() {
	if s.loaded {
		return
	}
	s.loaded = true
	s.loadFromFile()
}

func (s *Store) appendToFile(record Record) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return fmt.Errorf("failed to create history dir: %w", err)
	}

	f, err := os.OpenFile(filepath.Join(s.dir, "executions.jsonl"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open history file: %w", err)
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal record: %w", err)
	}

	_, err = f.Write(append(data, '\n'))
	return err
}

func (s *Store) loadFromFile() {
	path := filepath.Join(s.dir, "executions.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		return // File doesn't exist yet, that's fine
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record Record
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			continue
		}
		s.records = append(s.records, record)
	}
}

func defaultHistoryDir() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".infracore", "history")
}

// RenderRecords formats execution records for display.
func RenderRecords(records []Record) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("📜 EXECUTION HISTORY (%d records)\n", len(records)))
	b.WriteString("─────────────────────────────────────────\n")

	if len(records) == 0 {
		b.WriteString("  No executions recorded yet.\n")
		return b.String()
	}

	for _, r := range records {
		icon := statusIcon(r.Status)
		b.WriteString(fmt.Sprintf("  %s %-30s [%s] %s/%s %s\n",
			icon, r.SkillName, r.Status, r.Environment, r.Provider,
			r.Timestamp.Format("2006-01-02 15:04:05")))
		if r.Message != "" && len(r.Message) > 70 {
			b.WriteString(fmt.Sprintf("    → %s...\n", r.Message[:67]))
		} else if r.Message != "" {
			b.WriteString(fmt.Sprintf("    → %s\n", r.Message))
		}
	}
	return b.String()
}

// RenderStats formats execution statistics for display.
func RenderStats(stats Stats) string {
	var b strings.Builder
	b.WriteString("📊 EXECUTION STATISTICS\n")
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("  Total Executions: %d\n", stats.TotalExecutions))
	b.WriteString(fmt.Sprintf("  ✅ Successes:     %d\n", stats.Successes))
	b.WriteString(fmt.Sprintf("  ❌ Failures:      %d (%.1f%%)\n", stats.Failures, stats.FailureRate))
	b.WriteString(fmt.Sprintf("  🧪 Dry Runs:      %d\n", stats.DryRuns))
	b.WriteString(fmt.Sprintf("  ⏱️  Avg Duration:  %s\n", stats.AvgDuration.Round(time.Millisecond)))

	if len(stats.TopSkills) > 0 {
		b.WriteString("\n  🏆 TOP SKILLS:\n")
		for i, sc := range stats.TopSkills {
			if i >= 5 {
				break
			}
			b.WriteString(fmt.Sprintf("    %d. %-30s (%d executions)\n", i+1, sc.Skill, sc.Count))
		}
	}

	if len(stats.ByProvider) > 0 {
		b.WriteString("\n  ☁️  BY PROVIDER:\n")
		for provider, count := range stats.ByProvider {
			b.WriteString(fmt.Sprintf("    • %-15s %d\n", provider, count))
		}
	}

	if len(stats.ByEnvironment) > 0 {
		b.WriteString("\n  🌍 BY ENVIRONMENT:\n")
		for env, count := range stats.ByEnvironment {
			b.WriteString(fmt.Sprintf("    • %-15s %d\n", env, count))
		}
	}

	return b.String()
}

func statusIcon(status core.ExecutionStatus) string {
	switch status {
	case core.StatusSuccess:
		return "✅"
	case core.StatusFailed:
		return "❌"
	case core.StatusDryRun:
		return "🧪"
	case core.StatusCancelled:
		return "🚫"
	case core.StatusPending:
		return "⏳"
	default:
		return "📋"
	}
}
