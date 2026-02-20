// Package skills provides the skill registry and built-in skill definitions.
package skills

import (
	"fmt"
	"strings"
	"sync"

	"github.com/parth14193/ownbot/pkg/core"
)

// Registry manages the registration and lookup of skills.
type Registry struct {
	mu     sync.RWMutex
	skills map[string]*core.Skill
}

// NewRegistry creates a new empty skill registry.
func NewRegistry() *Registry {
	return &Registry{
		skills: make(map[string]*core.Skill),
	}
}

// Register adds a skill to the registry. Returns an error if a skill
// with the same name is already registered.
func (r *Registry) Register(skill *core.Skill) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.skills[skill.Name]; exists {
		return fmt.Errorf("skill already registered: %s", skill.Name)
	}
	r.skills[skill.Name] = skill
	return nil
}

// Get retrieves a skill by its fully qualified name.
func (r *Registry) Get(name string) (*core.Skill, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	skill, exists := r.skills[name]
	if !exists {
		return nil, fmt.Errorf("skill not found: %s", name)
	}
	return skill, nil
}

// List returns all registered skills.
func (r *Registry) List() []*core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*core.Skill, 0, len(r.skills))
	for _, skill := range r.skills {
		result = append(result, skill)
	}
	return result
}

// Search finds skills matching a provider, category, or name substring.
func (r *Registry) Search(query string) []*core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(query)
	var results []*core.Skill

	for _, skill := range r.skills {
		if strings.Contains(strings.ToLower(skill.Name), query) ||
			strings.Contains(strings.ToLower(string(skill.Provider)), query) ||
			strings.Contains(strings.ToLower(string(skill.Category)), query) ||
			strings.Contains(strings.ToLower(skill.Description), query) {
			results = append(results, skill)
		}
	}
	return results
}

// ListByProvider returns all skills for a specific provider.
func (r *Registry) ListByProvider(provider core.Provider) []*core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*core.Skill
	for _, skill := range r.skills {
		if skill.Provider == provider {
			results = append(results, skill)
		}
	}
	return results
}

// ListByCategory returns all skills in a specific category.
func (r *Registry) ListByCategory(category core.SkillCategory) []*core.Skill {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []*core.Skill
	for _, skill := range r.skills {
		if skill.Category == category {
			results = append(results, skill)
		}
	}
	return results
}

// Count returns the total number of registered skills.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.skills)
}

// LoadBuiltins registers all built-in skill definitions.
func (r *Registry) LoadBuiltins() error {
	loaders := []func() []*core.Skill{
		AWSSkills,
		KubernetesSkills,
		IaCSkills,
		GCPSkills,
		AzureSkills,
		ObservabilitySkills,
		CICDSkills,
		SecuritySkills,
		NetworkingSkills,
		CostSkills,
	}

	for _, loader := range loaders {
		for _, skill := range loader() {
			if err := r.Register(skill); err != nil {
				return fmt.Errorf("failed to load builtin skill: %w", err)
			}
		}
	}
	return nil
}
