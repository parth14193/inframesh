package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/parth14193/inframesh/pkg/core"
)

func TestAppendAndSearch(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-history-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store := NewStoreAt(dir)

	err := store.Append(Record{
		SkillName:   "aws.ec2.list",
		Action:      "run",
		Environment: "staging",
		Provider:    "aws",
		Status:      core.StatusSuccess,
		RiskLevel:   core.RiskLow,
		Duration:    500 * time.Millisecond,
		Message:     "Listed 5 instances",
		Timestamp:   time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	err = store.Append(Record{
		SkillName:   "k8s.deploy",
		Action:      "run",
		Environment: "production",
		Provider:    "k8s",
		Status:      core.StatusFailed,
		RiskLevel:   core.RiskHigh,
		Duration:    2 * time.Second,
		Message:     "Deployment failed",
		Timestamp:   time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to append: %v", err)
	}

	results := store.Search(SearchQuery{})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	results = store.Search(SearchQuery{Skill: "ec2"})
	if len(results) != 1 {
		t.Fatalf("expected 1 result for ec2, got %d", len(results))
	}

	results = store.Search(SearchQuery{Status: "failed"})
	if len(results) != 1 {
		t.Fatalf("expected 1 failed, got %d", len(results))
	}

	results = store.Search(SearchQuery{Env: "production"})
	if len(results) != 1 {
		t.Fatalf("expected 1 production, got %d", len(results))
	}
}

func TestStats(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-history-stats-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store := NewStoreAt(dir)
	for i := 0; i < 5; i++ {
		status := core.StatusSuccess
		if i == 3 {
			status = core.StatusFailed
		}
		_ = store.Append(Record{
			SkillName: "aws.ec2.list",
			Status:    status,
			Provider:  "aws",
			Duration:  time.Duration(i+1) * time.Second,
			Timestamp: time.Now(),
		})
	}

	stats := store.GetStats()
	if stats.TotalExecutions != 5 {
		t.Fatalf("expected 5 total, got %d", stats.TotalExecutions)
	}
	if stats.Successes != 4 {
		t.Fatalf("expected 4 successes, got %d", stats.Successes)
	}
	if stats.Failures != 1 {
		t.Fatalf("expected 1 failure, got %d", stats.Failures)
	}
}

func TestPersistence(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-history-persist-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store1 := NewStoreAt(dir)
	_ = store1.Append(Record{
		SkillName: "aws.ec2.list",
		Status:    core.StatusSuccess,
		Timestamp: time.Now(),
	})

	// Create new store pointing to same dir
	store2 := NewStoreAt(dir)
	results := store2.Search(SearchQuery{})
	if len(results) != 1 {
		t.Fatalf("expected 1 persisted record, got %d", len(results))
	}
}

func TestSearchLast(t *testing.T) {
	dir := filepath.Join(os.TempDir(), "infracore-history-last-test")
	os.RemoveAll(dir)
	defer os.RemoveAll(dir)

	store := NewStoreAt(dir)
	for i := 0; i < 10; i++ {
		_ = store.Append(Record{
			SkillName: "test",
			Status:    core.StatusSuccess,
			Timestamp: time.Now(),
		})
	}

	results := store.Search(SearchQuery{Last: 3})
	if len(results) != 3 {
		t.Fatalf("expected 3 with Last, got %d", len(results))
	}
}

func TestRenderRecords(t *testing.T) {
	records := []Record{
		{SkillName: "aws.ec2.list", Status: core.StatusSuccess, Environment: "staging", Provider: "aws"},
	}
	output := RenderRecords(records)
	if output == "" {
		t.Fatal("expected non-empty render output")
	}
}
