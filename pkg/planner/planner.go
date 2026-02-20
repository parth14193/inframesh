// Package planner provides the multi-step plan engine for decomposing
// complex infrastructure tasks into ordered skill executions.
package planner

import (
	"fmt"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
	"github.com/parth14193/ownbot/pkg/skills"
)

// Engine decomposes user intents into multi-step execution plans.
type Engine struct {
	registry *skills.Registry
}

// NewEngine creates a new PlanEngine with the given skill registry.
func NewEngine(registry *skills.Registry) *Engine {
	return &Engine{registry: registry}
}

// CreatePlan builds a new empty plan with a name and description.
func (e *Engine) CreatePlan(name, description string) *core.Plan {
	return &core.Plan{
		Name:        name,
		Description: description,
		Steps:       []core.PlanStep{},
		CreatedAt:   time.Now(),
	}
}

// AddStep appends a step to the plan, auto-numbering and resolving risk level.
func (e *Engine) AddStep(plan *core.Plan, skillName, description string, params map[string]interface{}) error {
	skill, err := e.registry.Get(skillName)
	if err != nil {
		return fmt.Errorf("cannot add step — %w", err)
	}

	step := core.PlanStep{
		StepNumber:  len(plan.Steps) + 1,
		SkillName:   skillName,
		Description: description,
		Params:      params,
		RiskLevel:   skill.RiskLevel,
	}

	plan.Steps = append(plan.Steps, step)
	e.recalculateOverallRisk(plan)
	return nil
}

// AddConditionalStep adds a step with conditional logic (if/else branching).
func (e *Engine) AddConditionalStep(plan *core.Plan, conditionExpr string, onTrueSkill, onTrueDesc string, onFalseSkill, onFalseDesc string) error {
	// Validate both skills exist
	trueSkill, err := e.registry.Get(onTrueSkill)
	if err != nil {
		return fmt.Errorf("on_true skill — %w", err)
	}

	var falseSkill *core.Skill
	if onFalseSkill != "" {
		falseSkill, err = e.registry.Get(onFalseSkill)
		if err != nil {
			return fmt.Errorf("on_false skill — %w", err)
		}
	}

	step := core.PlanStep{
		StepNumber:    len(plan.Steps) + 1,
		SkillName:     "CONDITIONAL",
		Description:   fmt.Sprintf("IF %s", conditionExpr),
		RiskLevel:     trueSkill.RiskLevel,
		Condition:     core.ConditionIfElse,
		ConditionExpr: conditionExpr,
		OnTrue: &core.PlanStep{
			SkillName:   onTrueSkill,
			Description: onTrueDesc,
			RiskLevel:   trueSkill.RiskLevel,
		},
	}

	if falseSkill != nil {
		step.OnFalse = &core.PlanStep{
			SkillName:   onFalseSkill,
			Description: onFalseDesc,
			RiskLevel:   falseSkill.RiskLevel,
		}
		// Take the higher risk
		if falseSkill.RiskLevel > step.RiskLevel {
			step.RiskLevel = falseSkill.RiskLevel
		}
	}

	plan.Steps = append(plan.Steps, step)
	e.recalculateOverallRisk(plan)
	return nil
}

// Validate checks that all referenced skills exist and required inputs are satisfiable.
func (e *Engine) Validate(plan *core.Plan) []error {
	var errs []error

	if len(plan.Steps) == 0 {
		errs = append(errs, fmt.Errorf("plan has no steps"))
		return errs
	}

	for _, step := range plan.Steps {
		if step.SkillName == "CONDITIONAL" {
			if step.OnTrue != nil {
				if _, err := e.registry.Get(step.OnTrue.SkillName); err != nil {
					errs = append(errs, fmt.Errorf("step %d on_true: %w", step.StepNumber, err))
				}
			}
			if step.OnFalse != nil {
				if _, err := e.registry.Get(step.OnFalse.SkillName); err != nil {
					errs = append(errs, fmt.Errorf("step %d on_false: %w", step.StepNumber, err))
				}
			}
			continue
		}

		skill, err := e.registry.Get(step.SkillName)
		if err != nil {
			errs = append(errs, fmt.Errorf("step %d: %w", step.StepNumber, err))
			continue
		}

		// Check required inputs
		for _, input := range skill.Inputs {
			if input.Required {
				if step.Params == nil {
					errs = append(errs, fmt.Errorf("step %d: missing required param '%s' for %s", step.StepNumber, input.Name, step.SkillName))
					continue
				}
				if _, ok := step.Params[input.Name]; !ok {
					errs = append(errs, fmt.Errorf("step %d: missing required param '%s' for %s", step.StepNumber, input.Name, step.SkillName))
				}
			}
		}
	}

	return errs
}

// StepsRequiringConfirmation returns the step numbers that need user confirmation.
func (e *Engine) StepsRequiringConfirmation(plan *core.Plan) []int {
	var steps []int
	for _, step := range plan.Steps {
		if step.RiskLevel >= core.RiskMedium {
			steps = append(steps, step.StepNumber)
		}
	}
	return steps
}

// EstimateDuration estimates the total plan execution time.
func (e *Engine) EstimateDuration(plan *core.Plan) time.Duration {
	var total time.Duration
	for _, step := range plan.Steps {
		if step.SkillName == "CONDITIONAL" {
			total += 30 * time.Second // estimate for conditional evaluation
			continue
		}
		skill, err := e.registry.Get(step.SkillName)
		if err != nil {
			total += 60 * time.Second // default estimate
			continue
		}
		if skill.Execution.Timeout > 0 {
			total += skill.Execution.Timeout / 2 // assume average is half of timeout
		} else {
			total += 30 * time.Second
		}
	}
	return total
}

// recalculateOverallRisk sets the plan's overall risk to the highest step risk.
func (e *Engine) recalculateOverallRisk(plan *core.Plan) {
	var maxRisk core.RiskLevel
	for _, step := range plan.Steps {
		if step.RiskLevel > maxRisk {
			maxRisk = step.RiskLevel
		}
	}
	plan.OverallRisk = maxRisk
	plan.EstimatedTime = e.EstimateDuration(plan).String()
}
