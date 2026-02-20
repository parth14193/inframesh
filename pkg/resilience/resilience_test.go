package resilience_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/parth14193/ownbot/pkg/resilience"
)

func TestRetrySuccess(t *testing.T) {
	policy := &resilience.RetryPolicy{MaxRetries: 3, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 10 * time.Millisecond, BackoffFactor: 2.0}
	attempts := 0
	result := resilience.WithRetry(policy, func() error {
		attempts++
		if attempts < 3 {
			return fmt.Errorf("timeout")
		}
		return nil
	})
	if !result.Succeeded {
		t.Error("should succeed on 3rd attempt")
	}
	if result.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", result.Attempts)
	}
}

func TestRetryExhausted(t *testing.T) {
	policy := &resilience.RetryPolicy{MaxRetries: 2, InitialBackoff: 1 * time.Millisecond, MaxBackoff: 5 * time.Millisecond, BackoffFactor: 2.0, RetryableErrs: []string{"timeout"}}
	result := resilience.WithRetry(policy, func() error {
		return fmt.Errorf("timeout error")
	})
	if result.Succeeded {
		t.Error("should fail after exhausting retries")
	}
	if result.Attempts != 3 {
		t.Errorf("expected 3 total attempts (1 + 2 retries), got %d", result.Attempts)
	}
}

func TestRetryNonRetryableError(t *testing.T) {
	policy := &resilience.RetryPolicy{MaxRetries: 3, InitialBackoff: 1 * time.Millisecond, BackoffFactor: 2.0, RetryableErrs: []string{"timeout"}}
	result := resilience.WithRetry(policy, func() error {
		return fmt.Errorf("permission denied")
	})
	if result.Succeeded {
		t.Error("should fail immediately on non-retryable error")
	}
	if result.Attempts != 1 {
		t.Errorf("expected 1 attempt for non-retryable, got %d", result.Attempts)
	}
}

func TestCircuitBreakerNormal(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test", 3, 1*time.Second)
	err := cb.Execute(func() error { return nil })
	if err != nil {
		t.Errorf("should succeed: %v", err)
	}
	if cb.State() != resilience.StateClosed {
		t.Errorf("should be CLOSED, got %s", cb.State())
	}
}

func TestCircuitBreakerTrips(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test", 3, 100*time.Millisecond)

	for i := 0; i < 3; i++ {
		_ = cb.Execute(func() error { return fmt.Errorf("fail") })
	}

	if cb.State() != resilience.StateOpen {
		t.Errorf("should be OPEN after 3 failures, got %s", cb.State())
	}

	err := cb.Execute(func() error { return nil })
	if err == nil {
		t.Error("should reject requests when OPEN")
	}
}

func TestCircuitBreakerRecovery(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test", 2, 50*time.Millisecond)

	_ = cb.Execute(func() error { return fmt.Errorf("fail") })
	_ = cb.Execute(func() error { return fmt.Errorf("fail") })

	if cb.State() != resilience.StateOpen {
		t.Fatalf("expected OPEN, got %s", cb.State())
	}

	time.Sleep(60 * time.Millisecond) // wait for reset timeout

	// Should transition to half-open
	_ = cb.Execute(func() error { return nil })
	_ = cb.Execute(func() error { return nil })

	if cb.State() != resilience.StateClosed {
		t.Errorf("should be CLOSED after recovery, got %s", cb.State())
	}
}

func TestCircuitBreakerReset(t *testing.T) {
	cb := resilience.NewCircuitBreaker("test", 1, 10*time.Second)
	_ = cb.Execute(func() error { return fmt.Errorf("fail") })
	if cb.State() != resilience.StateOpen {
		t.Fatal("expected OPEN")
	}
	cb.Reset()
	if cb.State() != resilience.StateClosed {
		t.Error("should be CLOSED after reset")
	}
}
