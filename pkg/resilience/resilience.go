// Package resilience provides retry and circuit breaker patterns
// for resilient execution of cloud API calls.
package resilience

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"
)

// ── Retry ──────────────────────────────────────────────────────

// RetryPolicy configures retry behavior.
type RetryPolicy struct {
	MaxRetries     int           `json:"max_retries"`
	InitialBackoff time.Duration `json:"initial_backoff"`
	MaxBackoff     time.Duration `json:"max_backoff"`
	BackoffFactor  float64       `json:"backoff_factor"`
	Jitter         bool          `json:"jitter"`
	RetryableErrs  []string      `json:"retryable_errors,omitempty"`
}

// DefaultRetryPolicy returns a sensible default.
func DefaultRetryPolicy() *RetryPolicy {
	return &RetryPolicy{
		MaxRetries:     3,
		InitialBackoff: 1 * time.Second,
		MaxBackoff:     30 * time.Second,
		BackoffFactor:  2.0,
		Jitter:         true,
		RetryableErrs: []string{
			"timeout", "connection refused", "503",
			"429", "throttl", "rate limit",
		},
	}
}

// RetryResult captures the outcome of a retried operation.
type RetryResult struct {
	Attempts  int           `json:"attempts"`
	Succeeded bool          `json:"succeeded"`
	LastError string        `json:"last_error,omitempty"`
	Duration  time.Duration `json:"total_duration"`
	Backoffs  []time.Duration `json:"backoffs"`
}

// WithRetry executes fn with retry logic per the policy.
func WithRetry(policy *RetryPolicy, fn func() error) *RetryResult {
	start := time.Now()
	result := &RetryResult{Backoffs: []time.Duration{}}

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		result.Attempts = attempt + 1
		err := fn()
		if err == nil {
			result.Succeeded = true
			result.Duration = time.Since(start)
			return result
		}

		result.LastError = err.Error()

		if attempt == policy.MaxRetries {
			break
		}

		if !isRetryable(err.Error(), policy.RetryableErrs) {
			break
		}

		backoff := calculateBackoff(attempt, policy)
		result.Backoffs = append(result.Backoffs, backoff)
		time.Sleep(backoff)
	}

	result.Duration = time.Since(start)
	return result
}

func calculateBackoff(attempt int, policy *RetryPolicy) time.Duration {
	backoff := float64(policy.InitialBackoff) * math.Pow(policy.BackoffFactor, float64(attempt))
	if time.Duration(backoff) > policy.MaxBackoff {
		backoff = float64(policy.MaxBackoff)
	}
	if policy.Jitter {
		backoff = backoff * (0.5 + rand.Float64()*0.5)
	}
	return time.Duration(backoff)
}

func isRetryable(errMsg string, patterns []string) bool {
	if len(patterns) == 0 {
		return true
	}
	lower := strings.ToLower(errMsg)
	for _, p := range patterns {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

// ── Circuit Breaker ────────────────────────────────────────────

// CircuitState represents the circuit breaker state.
type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"    // Normal — requests pass through
	StateOpen     CircuitState = "OPEN"      // Tripped — requests rejected
	StateHalfOpen CircuitState = "HALF_OPEN" // Testing — limited requests pass
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu               sync.Mutex
	name             string
	state            CircuitState
	failureCount     int
	successCount     int
	failureThreshold int
	successThreshold int // successes needed in half-open to close
	resetTimeout     time.Duration
	lastFailureTime  time.Time
	onStateChange    func(from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(name string, failureThreshold int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		name:             name,
		state:            StateClosed,
		failureThreshold: failureThreshold,
		successThreshold: 2,
		resetTimeout:     resetTimeout,
	}
}

// OnStateChange sets a callback for state transitions.
func (cb *CircuitBreaker) OnStateChange(fn func(from, to CircuitState)) {
	cb.onStateChange = fn
}

// Execute runs fn through the circuit breaker.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	cb.mu.Lock()
	state := cb.state

	if state == StateOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.transitionTo(StateHalfOpen)
			state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return fmt.Errorf("circuit breaker '%s' is OPEN — request rejected", cb.name)
		}
	}
	cb.mu.Unlock()

	err := fn()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()
		if cb.state == StateHalfOpen || cb.failureCount >= cb.failureThreshold {
			cb.transitionTo(StateOpen)
		}
		return err
	}

	if cb.state == StateHalfOpen {
		cb.successCount++
		if cb.successCount >= cb.successThreshold {
			cb.transitionTo(StateClosed)
		}
	} else {
		cb.failureCount = 0
	}

	return nil
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

// Reset forces the circuit back to closed.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionTo(StateClosed)
}

func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	if cb.state == newState {
		return
	}
	old := cb.state
	cb.state = newState
	cb.failureCount = 0
	cb.successCount = 0
	if cb.onStateChange != nil {
		cb.onStateChange(old, newState)
	}
}

// Render formats circuit breaker status.
func (cb *CircuitBreaker) Render() string {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	icon := "✅"
	if cb.state == StateOpen {
		icon = "🔴"
	} else if cb.state == StateHalfOpen {
		icon = "🟡"
	}
	return fmt.Sprintf("%s Circuit '%s': %s (failures: %d/%d)", icon, cb.name, cb.state, cb.failureCount, cb.failureThreshold)
}
