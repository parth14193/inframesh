package skills

import (
	"fmt"

	"github.com/parth14193/ownbot/pkg/core"
)

// Discovery handles dynamic creation and registration of custom skills.
type Discovery struct {
	registry *Registry
}

// NewDiscovery creates a new SkillDiscovery instance.
func NewDiscovery(registry *Registry) *Discovery {
	return &Discovery{registry: registry}
}

// SkillDefinition holds the raw definition for a custom skill.
type SkillDefinition struct {
	Name        string                `yaml:"name"`
	Description string                `yaml:"description"`
	Provider    string                `yaml:"provider"`
	Category    string                `yaml:"category"`
	Inputs      []SkillInputDef       `yaml:"inputs"`
	Outputs     []SkillOutputDef      `yaml:"outputs"`
	RiskLevel   string                `yaml:"risk_level"`
	Execution   SkillExecutionDef     `yaml:"execution"`
	Rollback    SkillRollbackDef      `yaml:"rollback"`
}

// SkillInputDef defines a skill input in YAML format.
type SkillInputDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
	Default     string `yaml:"default,omitempty"`
}

// SkillOutputDef defines a skill output in YAML format.
type SkillOutputDef struct {
	Name        string `yaml:"name"`
	Type        string `yaml:"type"`
	Description string `yaml:"description"`
}

// SkillExecutionDef defines the execution config in YAML format.
type SkillExecutionDef struct {
	Type    string `yaml:"type"`
	Command string `yaml:"command"`
}

// SkillRollbackDef defines the rollback config in YAML format.
type SkillRollbackDef struct {
	Supported bool   `yaml:"supported"`
	Procedure string `yaml:"procedure"`
}

// CreateSkill creates a core.Skill from a SkillDefinition and registers it.
func (d *Discovery) CreateSkill(def *SkillDefinition) (*core.Skill, error) {
	riskLevel, err := core.ParseRiskLevel(def.RiskLevel)
	if err != nil {
		return nil, fmt.Errorf("invalid risk level: %w", err)
	}

	execType := core.ExecutionType(def.Execution.Type)

	inputs := make([]core.SkillInput, len(def.Inputs))
	for i, in := range def.Inputs {
		inputs[i] = core.SkillInput{
			Name:        in.Name,
			Type:        in.Type,
			Required:    in.Required,
			Description: in.Description,
			Default:     in.Default,
		}
	}

	outputs := make([]core.SkillOutput, len(def.Outputs))
	for i, out := range def.Outputs {
		outputs[i] = core.SkillOutput{
			Name:        out.Name,
			Type:        out.Type,
			Description: out.Description,
		}
	}

	skill := &core.Skill{
		Name:                 def.Name,
		Description:          def.Description,
		Provider:             core.Provider(def.Provider),
		Category:             core.SkillCategory(def.Category),
		Inputs:               inputs,
		Outputs:              outputs,
		RiskLevel:            riskLevel,
		RequiresConfirmation: riskLevel >= core.RiskHigh,
		Execution: core.ExecutionConfig{
			Type:    execType,
			Command: def.Execution.Command,
		},
		Rollback: core.RollbackConfig{
			Supported: def.Rollback.Supported,
			Procedure: def.Rollback.Procedure,
		},
	}

	if err := d.registry.Register(skill); err != nil {
		return nil, fmt.Errorf("failed to register custom skill: %w", err)
	}

	return skill, nil
}

// Validate checks a SkillDefinition for required fields.
func (d *Discovery) Validate(def *SkillDefinition) error {
	if def.Name == "" {
		return fmt.Errorf("skill name is required")
	}
	if def.Description == "" {
		return fmt.Errorf("skill description is required")
	}
	if def.Provider == "" {
		return fmt.Errorf("skill provider is required")
	}
	if def.RiskLevel == "" {
		return fmt.Errorf("skill risk_level is required")
	}
	if _, err := core.ParseRiskLevel(def.RiskLevel); err != nil {
		return err
	}
	if def.Execution.Command == "" {
		return fmt.Errorf("skill execution command is required")
	}
	return nil
}

// GenerateTemplate returns a YAML template for defining a custom skill.
func (d *Discovery) GenerateTemplate(provider, action string) string {
	return fmt.Sprintf(`skill:
  name: "custom.%s.%s"
  description: "One-line description of what this skill does"
  provider: "%s"
  category: "compute"
  inputs:
    - name: param_name
      type: string
      required: true
      description: "What this parameter controls"
  outputs:
    - name: result
      type: object
      description: "What is returned"
  risk_level: LOW
  execution:
    type: cli
    command: "command to execute"
  rollback:
    supported: false
    procedure: "How to undo this action"
`, provider, action, provider)
}
