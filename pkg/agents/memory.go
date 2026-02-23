package agents

import "sync"

// MemoryStore keeps lightweight incident and decision history for coordination.
type MemoryStore struct {
	mu        sync.RWMutex
	decisions []Decision
}

// NewMemoryStore creates an in-memory coordination memory.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		decisions: []Decision{},
	}
}

// RecordDecision appends a controller decision to memory.
func (m *MemoryStore) RecordDecision(d Decision) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.decisions = append(m.decisions, d)
}

// LastDecision returns the latest decision if available.
func (m *MemoryStore) LastDecision() (Decision, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if len(m.decisions) == 0 {
		return Decision{}, false
	}
	return m.decisions[len(m.decisions)-1], true
}
