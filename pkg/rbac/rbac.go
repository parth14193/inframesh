// Package rbac provides role-based access control for skills and environments.
package rbac

import (
	"fmt"
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

// Role represents a user role with specific permissions.
type Role string

const (
	RoleViewer     Role = "viewer"
	RoleOperator   Role = "operator"
	RoleAdmin      Role = "admin"
	RoleSuperAdmin Role = "superadmin"
)

// Permission defines what a role can do.
type Permission struct {
	AllowedRiskLevels  []core.RiskLevel `json:"allowed_risk_levels"`
	AllowedCategories  []string         `json:"allowed_categories"`
	AllowedEnvironments []string        `json:"allowed_environments"`
	AllowedSkillPatterns []string       `json:"allowed_skill_patterns"`
	DenySkillPatterns   []string        `json:"deny_skill_patterns"`
	CanApprove         bool             `json:"can_approve"`
	CanManagePolicies  bool             `json:"can_manage_policies"`
	CanManageUsers     bool             `json:"can_manage_users"`
}

// User represents a user with a role.
type User struct {
	Username string `json:"username"`
	Role     Role   `json:"role"`
	Teams    []string `json:"teams,omitempty"`
}

// Engine evaluates access control decisions.
type Engine struct {
	users       map[string]*User
	permissions map[Role]*Permission
	enabled     bool
}

// NewEngine creates a new RBAC engine with default role permissions.
func NewEngine() *Engine {
	e := &Engine{
		users:       make(map[string]*User),
		permissions: make(map[Role]*Permission),
		enabled:     true,
	}
	e.loadDefaultPermissions()
	return e
}

// SetEnabled enables or disables RBAC enforcement.
func (e *Engine) SetEnabled(enabled bool) { e.enabled = enabled }

// IsEnabled returns whether RBAC is active.
func (e *Engine) IsEnabled() bool { return e.enabled }

// AddUser registers a user with a role.
func (e *Engine) AddUser(username string, role Role, teams []string) {
	e.users[username] = &User{Username: username, Role: role, Teams: teams}
}

// GetUser retrieves a user by username.
func (e *Engine) GetUser(username string) (*User, error) {
	u, ok := e.users[username]
	if !ok {
		return nil, fmt.Errorf("user not found: %s", username)
	}
	return u, nil
}

// CanExecute checks if a user can execute a specific skill in an environment.
func (e *Engine) CanExecute(username string, skill *core.Skill, env string) (bool, string) {
	if !e.enabled {
		return true, ""
	}

	user, err := e.GetUser(username)
	if err != nil {
		return false, fmt.Sprintf("Access denied: %s", err)
	}

	perm, ok := e.permissions[user.Role]
	if !ok {
		return false, fmt.Sprintf("No permissions defined for role: %s", user.Role)
	}

	// Check risk level
	if !containsRisk(perm.AllowedRiskLevels, skill.RiskLevel) {
		return false, fmt.Sprintf("Role '%s' cannot execute %s-risk operations", user.Role, skill.RiskLevel)
	}

	// Check environment
	if len(perm.AllowedEnvironments) > 0 && !containsStr(perm.AllowedEnvironments, env) {
		return false, fmt.Sprintf("Role '%s' cannot access environment '%s'", user.Role, env)
	}

	// Check deny patterns
	for _, pattern := range perm.DenySkillPatterns {
		if matchPattern(skill.Name, pattern) {
			return false, fmt.Sprintf("Skill '%s' is denied for role '%s'", skill.Name, user.Role)
		}
	}

	return true, ""
}

// CanApprove checks if a user can approve high-risk operations.
func (e *Engine) CanApprove(username string) bool {
	if !e.enabled {
		return true
	}
	user, err := e.GetUser(username)
	if err != nil {
		return false
	}
	perm, ok := e.permissions[user.Role]
	if !ok {
		return false
	}
	return perm.CanApprove
}

// ListUsers returns all registered users.
func (e *Engine) ListUsers() []*User {
	result := make([]*User, 0, len(e.users))
	for _, u := range e.users {
		result = append(result, u)
	}
	return result
}

func (e *Engine) loadDefaultPermissions() {
	e.permissions[RoleViewer] = &Permission{
		AllowedRiskLevels:   []core.RiskLevel{core.RiskLow},
		AllowedEnvironments: []string{"staging", "dev"},
		CanApprove:          false,
		CanManagePolicies:   false,
		CanManageUsers:      false,
	}
	e.permissions[RoleOperator] = &Permission{
		AllowedRiskLevels:   []core.RiskLevel{core.RiskLow, core.RiskMedium},
		AllowedEnvironments: []string{"staging", "dev", "qa"},
		CanApprove:          false,
		CanManagePolicies:   false,
		CanManageUsers:      false,
	}
	e.permissions[RoleAdmin] = &Permission{
		AllowedRiskLevels:   []core.RiskLevel{core.RiskLow, core.RiskMedium, core.RiskHigh},
		AllowedEnvironments: []string{"staging", "dev", "qa", "production"},
		CanApprove:          true,
		CanManagePolicies:   true,
		CanManageUsers:      false,
	}
	e.permissions[RoleSuperAdmin] = &Permission{
		AllowedRiskLevels:   []core.RiskLevel{core.RiskLow, core.RiskMedium, core.RiskHigh, core.RiskCritical},
		AllowedEnvironments: []string{}, // empty = all
		CanApprove:          true,
		CanManagePolicies:   true,
		CanManageUsers:      true,
	}
}

func containsRisk(levels []core.RiskLevel, target core.RiskLevel) bool {
	for _, l := range levels {
		if l == target {
			return true
		}
	}
	return false
}

func containsStr(items []string, target string) bool {
	for _, i := range items {
		if strings.EqualFold(i, target) {
			return true
		}
	}
	return false
}

func matchPattern(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(name, strings.TrimSuffix(pattern, "*"))
	}
	return name == pattern
}

// Render formats RBAC state for display.
func (e *Engine) Render() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("🔐 RBAC — enabled=%t\n", e.enabled))
	b.WriteString("─────────────────────────────────────────\n")
	for _, u := range e.ListUsers() {
		b.WriteString(fmt.Sprintf("  👤 %-15s role=%-12s teams=%v\n", u.Username, u.Role, u.Teams))
	}
	return b.String()
}
