// Package eventbus provides an internal publish/subscribe event system
// that allows all InfraCore subsystems to emit and react to events.
package eventbus

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

// EventType identifies a category of event.
type EventType string

const (
	EventSkillExecuted    EventType = "skill.executed"
	EventPolicyViolation  EventType = "policy.violation"
	EventDriftDetected    EventType = "drift.detected"
	EventRunbookCompleted EventType = "runbook.completed"
	EventApprovalRequired EventType = "approval.required"
	EventApprovalResolved EventType = "approval.resolved"
	EventHealthChanged    EventType = "health.changed"
	EventPlanCreated      EventType = "plan.created"
	EventPlanExecuted     EventType = "plan.executed"
	EventCostEstimated    EventType = "cost.estimated"
	EventAll              EventType = "*"
)

// Event is the common envelope for all events.
type Event struct {
	ID            string                 `json:"id"`
	Type          EventType              `json:"type"`
	Source        string                 `json:"source"`
	Timestamp     time.Time              `json:"timestamp"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
	Payload       map[string]interface{} `json:"payload,omitempty"`
	Metadata      map[string]string      `json:"metadata,omitempty"`
}

// NewEvent creates a new event with auto-generated ID and timestamp.
func NewEvent(eventType EventType, source string, payload map[string]interface{}) *Event {
	return &Event{
		ID:        generateID(),
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   payload,
		Metadata:  make(map[string]string),
	}
}

// WithCorrelation sets a correlation ID for event chaining.
func (e *Event) WithCorrelation(id string) *Event {
	e.CorrelationID = id
	return e
}

// WithMeta adds metadata key-value pair.
func (e *Event) WithMeta(key, value string) *Event {
	e.Metadata[key] = value
	return e
}

// Handler processes an event. Return an error to signal processing failure.
type Handler func(event *Event) error

// subscription tracks a single subscriber.
type subscription struct {
	id        string
	eventType EventType
	handler   Handler
	async     bool
}

// Bus is a thread-safe pub/sub event bus.
type Bus struct {
	mu            sync.RWMutex
	subscriptions []subscription
	history       []*Event
	maxHistory    int
	idCounter     int
}

// New creates a new EventBus.
func New() *Bus {
	return &Bus{
		subscriptions: []subscription{},
		history:       []*Event{},
		maxHistory:    1000,
	}
}

// Subscribe registers a synchronous handler for a specific event type.
// Use EventAll ("*") to subscribe to all events.
func (b *Bus) Subscribe(eventType EventType, handler Handler) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.idCounter++
	id := fmt.Sprintf("sub-%d", b.idCounter)
	b.subscriptions = append(b.subscriptions, subscription{
		id:        id,
		eventType: eventType,
		handler:   handler,
		async:     false,
	})
	return id
}

// SubscribeAsync registers an asynchronous handler.
func (b *Bus) SubscribeAsync(eventType EventType, handler Handler) string {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.idCounter++
	id := fmt.Sprintf("sub-%d", b.idCounter)
	b.subscriptions = append(b.subscriptions, subscription{
		id:        id,
		eventType: eventType,
		handler:   handler,
		async:     true,
	})
	return id
}

// Unsubscribe removes a handler by its subscription ID.
func (b *Bus) Unsubscribe(subID string) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for i, s := range b.subscriptions {
		if s.id == subID {
			b.subscriptions = append(b.subscriptions[:i], b.subscriptions[i+1:]...)
			return
		}
	}
}

// Publish emits an event to all matching subscribers.
func (b *Bus) Publish(event *Event) []error {
	b.mu.Lock()
	b.history = append(b.history, event)
	if len(b.history) > b.maxHistory {
		b.history = b.history[len(b.history)-b.maxHistory:]
	}

	// Copy subscriptions under lock
	subs := make([]subscription, len(b.subscriptions))
	copy(subs, b.subscriptions)
	b.mu.Unlock()

	var errs []error
	var wg sync.WaitGroup

	for _, sub := range subs {
		if !b.matches(sub.eventType, event.Type) {
			continue
		}
		if sub.async {
			wg.Add(1)
			go func(s subscription) {
				defer wg.Done()
				_ = s.handler(event)
			}(sub)
		} else {
			if err := sub.handler(event); err != nil {
				errs = append(errs, fmt.Errorf("handler %s: %w", sub.id, err))
			}
		}
	}

	wg.Wait()
	return errs
}

// History returns the last N events.
func (b *Bus) History(n int) []*Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	if n <= 0 || n > len(b.history) {
		n = len(b.history)
	}
	start := len(b.history) - n
	result := make([]*Event, n)
	copy(result, b.history[start:])
	return result
}

// HistoryByType returns events of a specific type.
func (b *Bus) HistoryByType(eventType EventType) []*Event {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var result []*Event
	for _, e := range b.history {
		if b.matches(eventType, e.Type) {
			result = append(result, e)
		}
	}
	return result
}

// SubscriberCount returns the number of active subscribers.
func (b *Bus) SubscriberCount() int {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return len(b.subscriptions)
}

// matches checks if a subscription pattern matches an event type.
func (b *Bus) matches(pattern, actual EventType) bool {
	if pattern == EventAll {
		return true
	}
	p := string(pattern)
	a := string(actual)
	if strings.HasSuffix(p, ".*") {
		prefix := strings.TrimSuffix(p, ".*")
		return strings.HasPrefix(a, prefix+".")
	}
	return p == a
}

// Render formats the event bus state for display.
func (b *Bus) Render() string {
	b.mu.RLock()
	defer b.mu.RUnlock()

	var sb strings.Builder
	sb.WriteString("📡 EVENT BUS\n")
	sb.WriteString("─────────────────────────────────────────\n")
	sb.WriteString(fmt.Sprintf("  Subscribers: %d\n", len(b.subscriptions)))
	sb.WriteString(fmt.Sprintf("  Events logged: %d\n", len(b.history)))

	if len(b.history) > 0 {
		sb.WriteString("\n  📋 RECENT EVENTS (last 5):\n")
		start := len(b.history) - 5
		if start < 0 {
			start = 0
		}
		for _, e := range b.history[start:] {
			sb.WriteString(fmt.Sprintf("    [%s] %s → %s\n", e.Timestamp.Format("15:04:05"), e.Type, e.Source))
		}
	}

	return sb.String()
}

var idMu sync.Mutex
var globalID int

func generateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	globalID++
	return fmt.Sprintf("evt-%d-%d", time.Now().UnixMilli(), globalID)
}
