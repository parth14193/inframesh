// Package approval provides structured async approval workflows
// for high-risk production operations, with HMAC-SHA256 token authorization.
package approval

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// Status represents the approval request state.
type Status string

const (
	StatusPending  Status = "PENDING"
	StatusApproved Status = "APPROVED"
	StatusRejected Status = "REJECTED"
	StatusExpired  Status = "EXPIRED"
)

// Request represents an approval request.
type Request struct {
	ID           string                 `json:"id"`
	SkillName    string                 `json:"skill_name"`
	Environment  string                 `json:"environment"`
	Requester    string                 `json:"requester"`
	Approver     string                 `json:"approver,omitempty"`
	Status       Status                 `json:"status"`
	RiskLevel    core.RiskLevel         `json:"risk_level"`
	Params       map[string]interface{} `json:"params,omitempty"`
	Reason       string                 `json:"reason,omitempty"`
	RejectReason string                 `json:"reject_reason,omitempty"`
	Token        string                 `json:"token,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	ResolvedAt   time.Time              `json:"resolved_at,omitempty"`
	ExpiresAt    time.Time              `json:"expires_at"`
}

// Manager manages approval requests with file-backed persistence.
type Manager struct {
	mu       sync.Mutex
	requests map[string]*Request
	dir      string
	secret   []byte
	loaded   bool
}

// NewManager creates a new approval manager.
func NewManager() *Manager {
	return &Manager{
		requests: make(map[string]*Request),
		dir:      defaultApprovalDir(),
		secret:   []byte("infracore-approval-key-v1"),
	}
}

// NewManagerAt creates a manager with a custom directory.
func NewManagerAt(dir string) *Manager {
	return &Manager{
		requests: make(map[string]*Request),
		dir:      dir,
		secret:   []byte("infracore-approval-key-v1"),
	}
}

// CreateRequest creates a new pending approval request.
func (m *Manager) CreateRequest(skillName, env, requester, reason string, riskLevel core.RiskLevel, params map[string]interface{}) (*Request, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	req := &Request{
		ID:          fmt.Sprintf("apr-%d", time.Now().UnixNano()),
		SkillName:   skillName,
		Environment: env,
		Requester:   requester,
		Status:      StatusPending,
		RiskLevel:   riskLevel,
		Params:      params,
		Reason:      reason,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(24 * time.Hour),
	}

	// Generate approval token
	req.Token = m.generateToken(req)

	m.requests[req.ID] = req
	if err := m.persist(); err != nil {
		return nil, fmt.Errorf("failed to persist approval request: %w", err)
	}

	return req, nil
}

// Approve approves a pending request.
func (m *Manager) Approve(id, approver string) (*Request, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != StatusPending {
		return nil, fmt.Errorf("request %s is already %s", id, req.Status)
	}

	if time.Now().After(req.ExpiresAt) {
		req.Status = StatusExpired
		_ = m.persist()
		return nil, fmt.Errorf("request %s has expired", id)
	}

	req.Status = StatusApproved
	req.Approver = approver
	req.ResolvedAt = time.Now()

	if err := m.persist(); err != nil {
		return nil, fmt.Errorf("failed to persist approval: %w", err)
	}

	return req, nil
}

// Reject rejects a pending request.
func (m *Manager) Reject(id, reason string) (*Request, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}

	if req.Status != StatusPending {
		return nil, fmt.Errorf("request %s is already %s", id, req.Status)
	}

	req.Status = StatusRejected
	req.RejectReason = reason
	req.ResolvedAt = time.Now()

	if err := m.persist(); err != nil {
		return nil, fmt.Errorf("failed to persist rejection: %w", err)
	}

	return req, nil
}

// Get retrieves a request by ID.
func (m *Manager) Get(id string) (*Request, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	req, ok := m.requests[id]
	if !ok {
		return nil, fmt.Errorf("approval request not found: %s", id)
	}
	return req, nil
}

// ListPending returns all pending requests.
func (m *Manager) ListPending() []*Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	var result []*Request
	now := time.Now()
	for _, req := range m.requests {
		if req.Status == StatusPending {
			if now.After(req.ExpiresAt) {
				req.Status = StatusExpired
				continue
			}
			result = append(result, req)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// ListAll returns all requests sorted by creation time.
func (m *Manager) ListAll() []*Request {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	result := make([]*Request, 0, len(m.requests))
	for _, req := range m.requests {
		result = append(result, req)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	return result
}

// VerifyToken validates an approval token against a request.
func (m *Manager) VerifyToken(req *Request, token string) bool {
	expected := m.generateToken(req)
	return hmac.Equal([]byte(expected), []byte(token))
}

// IsApproved checks if a specific skill+env combination has an active approval.
func (m *Manager) IsApproved(skillName, env string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	for _, req := range m.requests {
		if req.SkillName == skillName &&
			strings.EqualFold(req.Environment, env) &&
			req.Status == StatusApproved &&
			time.Now().Before(req.ExpiresAt) {
			return true
		}
	}
	return false
}

// Count returns counts by status.
func (m *Manager) Count() (pending, approved, rejected int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ensureLoaded()

	for _, req := range m.requests {
		switch req.Status {
		case StatusPending:
			pending++
		case StatusApproved:
			approved++
		case StatusRejected:
			rejected++
		}
	}
	return
}

func (m *Manager) generateToken(req *Request) string {
	data := fmt.Sprintf("%s:%s:%s:%s:%d", req.ID, req.SkillName, req.Environment, req.Requester, req.CreatedAt.UnixNano())
	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))[:32]
}

func (m *Manager) ensureLoaded() {
	if m.loaded {
		return
	}
	m.loaded = true
	m.loadFromFile()
}

func (m *Manager) persist() error {
	if err := os.MkdirAll(m.dir, 0o755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(m.requests, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(m.dir, "approvals.json"), data, 0o644)
}

func (m *Manager) loadFromFile() {
	path := filepath.Join(m.dir, "approvals.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(data, &m.requests)
}

func defaultApprovalDir() string {
	home := os.Getenv("HOME")
	if runtime.GOOS == "windows" {
		home = os.Getenv("USERPROFILE")
	}
	return filepath.Join(home, ".infracore", "approvals")
}

// RenderRequests formats approval requests for display.
func RenderRequests(requests []*Request) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔐 APPROVAL REQUESTS (%d)\n", len(requests)))
	b.WriteString("─────────────────────────────────────────\n")

	if len(requests) == 0 {
		b.WriteString("  No approval requests.\n")
		return b.String()
	}

	for _, req := range requests {
		icon := approvalIcon(req.Status)
		b.WriteString(fmt.Sprintf("  %s %-12s  %-30s [%s] %s/%s\n",
			icon, req.Status, req.SkillName, req.RiskLevel,
			req.Environment, req.Requester))
		b.WriteString(fmt.Sprintf("    ID: %s\n", req.ID))
		if req.Reason != "" {
			b.WriteString(fmt.Sprintf("    Reason: %s\n", req.Reason))
		}
		if req.Approver != "" {
			b.WriteString(fmt.Sprintf("    Approver: %s\n", req.Approver))
		}
		if req.RejectReason != "" {
			b.WriteString(fmt.Sprintf("    Reject reason: %s\n", req.RejectReason))
		}
		ttl := time.Until(req.ExpiresAt).Round(time.Minute)
		if req.Status == StatusPending && ttl > 0 {
			b.WriteString(fmt.Sprintf("    Expires in: %s\n", ttl))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func approvalIcon(status Status) string {
	switch status {
	case StatusPending:
		return "⏳"
	case StatusApproved:
		return "✅"
	case StatusRejected:
		return "❌"
	case StatusExpired:
		return "⌛"
	default:
		return "❓"
	}
}
