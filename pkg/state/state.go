// Package state provides session-level state management, tracking
// the active environment, provider, resource context, and audit log.
package state

import (
	"fmt"
	"sync"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// Manager manages session-level state for the InfraCore agent.
type Manager struct {
	mu    sync.RWMutex
	state *core.SessionState
}

// NewManager creates a new StateManager with a fresh session.
func NewManager(sessionID string) *Manager {
	return &Manager{
		state: &core.SessionState{
			SessionID:           sessionID,
			ActiveEnvironment:   "staging",
			ActiveProvider:      core.ProviderAWS,
			ActiveRegion:        "us-east-1",
			LoadedSkills:        []string{},
			ResourceContext:     core.ResourceContext{},
			PendingConfirmations: []string{},
			AuditLog:            []AuditEntry{},
			CustomData:          make(map[string]interface{}),
		},
	}
}

// AuditEntry is the state package's local audit entry for convenience.
type AuditEntry = core.AuditEntry

// GetState returns a copy of the current session state.
func (m *Manager) GetState() core.SessionState {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return *m.state
}

// SetEnvironment updates the active environment.
func (m *Manager) SetEnvironment(env string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.ActiveEnvironment = env
}

// SetProvider updates the active provider.
func (m *Manager) SetProvider(provider core.Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.ActiveProvider = provider
}

// SetRegion updates the active region.
func (m *Manager) SetRegion(region string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.ActiveRegion = region
}

// GetEnvironment returns the current active environment.
func (m *Manager) GetEnvironment() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.ActiveEnvironment
}

// GetProvider returns the current active provider.
func (m *Manager) GetProvider() core.Provider {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.ActiveProvider
}

// GetRegion returns the current active region.
func (m *Manager) GetRegion() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.ActiveRegion
}

// AddToAuditLog appends an entry to the session audit trail.
func (m *Manager) AddToAuditLog(skillName, action, target string, status core.ExecutionStatus, riskLevel core.RiskLevel, details string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	entry := core.AuditEntry{
		Timestamp: time.Now(),
		SkillName: skillName,
		Action:    action,
		Target:    target,
		Status:    status,
		RiskLevel: riskLevel,
		Details:   details,
	}
	m.state.AuditLog = append(m.state.AuditLog, entry)
}

// GetAuditLog returns a copy of the audit log.
func (m *Manager) GetAuditLog() []core.AuditEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	log := make([]core.AuditEntry, len(m.state.AuditLog))
	copy(log, m.state.AuditLog)
	return log
}

// GetContext returns the current resource context.
func (m *Manager) GetContext() core.ResourceContext {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.ResourceContext
}

// UpdateResourceContext updates a field in the resource context.
func (m *Manager) UpdateResourceContext(key, value string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	switch key {
	case "cluster":
		m.state.ResourceContext.Cluster = value
	case "namespace":
		m.state.ResourceContext.Namespace = value
	case "last_deployment":
		m.state.ResourceContext.LastDeployment = value
	default:
		return fmt.Errorf("unknown resource context key: %s", key)
	}
	return nil
}

// LoadSkill marks a skill as loaded in the session.
func (m *Manager) LoadSkill(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Avoid duplicates
	for _, s := range m.state.LoadedSkills {
		if s == name {
			return
		}
	}
	m.state.LoadedSkills = append(m.state.LoadedSkills, name)
}

// GetLoadedSkills returns the list of loaded skill names.
func (m *Manager) GetLoadedSkills() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]string, len(m.state.LoadedSkills))
	copy(result, m.state.LoadedSkills)
	return result
}

// AddPendingConfirmation adds a pending confirmation to the queue.
func (m *Manager) AddPendingConfirmation(confirmation string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.PendingConfirmations = append(m.state.PendingConfirmations, confirmation)
}

// ClearPendingConfirmations removes all pending confirmations.
func (m *Manager) ClearPendingConfirmations() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.PendingConfirmations = []string{}
}

// SetCustomData sets a key-value pair in custom session data.
func (m *Manager) SetCustomData(key string, value interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.state.CustomData[key] = value
}

// GetCustomData retrieves a value from custom session data.
func (m *Manager) GetCustomData(key string) (interface{}, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	val, ok := m.state.CustomData[key]
	return val, ok
}
