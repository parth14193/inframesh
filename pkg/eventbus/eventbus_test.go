package eventbus

import (
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

func TestPublishSubscribe(t *testing.T) {
	bus := New()
	var received int32

	bus.Subscribe(EventSkillExecuted, func(event *Event) error {
		atomic.AddInt32(&received, 1)
		return nil
	})

	event := NewEvent(EventSkillExecuted, "test", map[string]interface{}{"skill": "aws.ec2.list"})
	errs := bus.Publish(event)
	if len(errs) > 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if atomic.LoadInt32(&received) != 1 {
		t.Fatalf("expected 1 event received, got %d", received)
	}
}

func TestWildcardSubscription(t *testing.T) {
	bus := New()
	var count int32

	bus.Subscribe(EventAll, func(event *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(NewEvent(EventSkillExecuted, "test", nil))
	bus.Publish(NewEvent(EventPolicyViolation, "test", nil))
	bus.Publish(NewEvent(EventDriftDetected, "test", nil))

	if atomic.LoadInt32(&count) != 3 {
		t.Fatalf("expected 3 events, got %d", count)
	}
}

func TestAsyncSubscription(t *testing.T) {
	bus := New()
	done := make(chan struct{})

	bus.SubscribeAsync(EventSkillExecuted, func(event *Event) error {
		close(done)
		return nil
	})

	bus.Publish(NewEvent(EventSkillExecuted, "test", nil))

	select {
	case <-done:
		// success
	case <-time.After(2 * time.Second):
		t.Fatal("async handler did not complete in time")
	}
}

func TestUnsubscribe(t *testing.T) {
	bus := New()
	var count int32

	id := bus.Subscribe(EventSkillExecuted, func(event *Event) error {
		atomic.AddInt32(&count, 1)
		return nil
	})

	bus.Publish(NewEvent(EventSkillExecuted, "test", nil))
	bus.Unsubscribe(id)
	bus.Publish(NewEvent(EventSkillExecuted, "test", nil))

	if atomic.LoadInt32(&count) != 1 {
		t.Fatalf("expected 1 after unsubscribe, got %d", count)
	}
}

func TestHandlerError(t *testing.T) {
	bus := New()
	bus.Subscribe(EventSkillExecuted, func(event *Event) error {
		return errors.New("handler failed")
	})

	errs := bus.Publish(NewEvent(EventSkillExecuted, "test", nil))
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
}

func TestHistory(t *testing.T) {
	bus := New()
	bus.Publish(NewEvent(EventSkillExecuted, "a", nil))
	bus.Publish(NewEvent(EventDriftDetected, "b", nil))
	bus.Publish(NewEvent(EventPolicyViolation, "c", nil))

	all := bus.History(0)
	if len(all) != 3 {
		t.Fatalf("expected 3 events in history, got %d", len(all))
	}

	last2 := bus.History(2)
	if len(last2) != 2 {
		t.Fatalf("expected 2 events, got %d", len(last2))
	}
	if last2[0].Source != "b" {
		t.Fatalf("expected source 'b', got %s", last2[0].Source)
	}
}

func TestHistoryByType(t *testing.T) {
	bus := New()
	bus.Publish(NewEvent(EventSkillExecuted, "a", nil))
	bus.Publish(NewEvent(EventDriftDetected, "b", nil))
	bus.Publish(NewEvent(EventSkillExecuted, "c", nil))

	filtered := bus.HistoryByType(EventSkillExecuted)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 skill events, got %d", len(filtered))
	}
}

func TestEventCorrelation(t *testing.T) {
	event := NewEvent(EventSkillExecuted, "test", nil).
		WithCorrelation("corr-123").
		WithMeta("user", "admin")

	if event.CorrelationID != "corr-123" {
		t.Fatalf("expected correlation ID 'corr-123', got %s", event.CorrelationID)
	}
	if event.Metadata["user"] != "admin" {
		t.Fatal("expected metadata user=admin")
	}
}

func TestSubscriberCount(t *testing.T) {
	bus := New()
	if bus.SubscriberCount() != 0 {
		t.Fatal("expected 0 subscribers")
	}
	id := bus.Subscribe(EventAll, func(e *Event) error { return nil })
	if bus.SubscriberCount() != 1 {
		t.Fatal("expected 1 subscriber")
	}
	bus.Unsubscribe(id)
	if bus.SubscriberCount() != 0 {
		t.Fatal("expected 0 after unsubscribe")
	}
}
