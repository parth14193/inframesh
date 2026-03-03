## Feature Prompt: Plan Execution Workflow

You are working in the `inframesh` Go repository. Implement a new CLI capability to execute generated plans end-to-end.

### Objective
Add `infracore plan execute <description>` so operators can generate and run a multi-step plan in one command.

### Requirements
- Reuse existing plan generation logic (`populatePlanFromDescription`).
- Add explicit execution gate: execution only occurs when `--force` is provided.
- Before execution:
  - Render the generated plan.
  - Validate the plan with `planner.Engine.Validate`.
- Execute plan steps sequentially using existing `executor`, `policy`, `safety`, and `state` components.
- For each step:
  - Resolve skill from registry.
  - Merge step params with global `--param` overrides.
  - Run policy checks and block denied steps.
  - Evaluate safety report and execute via CLI executor.
  - Append audit log entries to session state.
- Handle conditional steps safely (skip with a clear message).
- Stop on first failure by default; support `--continue-on-error`.
- Print final execution summary (`executed`, `skipped`, `failed`, `total`).

### CLI/UX Updates
- Update help/usage text and examples to include `plan execute`.

### Test Expectations
- Add/extend tests in `cmd/infracore/main_test.go` for new helper logic:
  - Description extraction excludes flags and `--param`.
  - Param merge respects overrides.
- Ensure all tests pass with `go test ./...`.

### Constraints
- Keep changes idiomatic and minimal.
- Reuse existing package responsibilities; avoid introducing unnecessary new abstractions.
