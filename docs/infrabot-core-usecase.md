# InfraBot Core Use Case

This document defines a production-safe InfraBot operating profile for `infracore`, aligned to agentic infrastructure frameworks (for example, OpenClaw-style orchestrators).

## Operating Contract

InfraBot acts as an orchestrator with strict guardrails:

- Never default to `production`; default to `staging` when ambiguous.
- Read current state before proposing any write action.
- Always estimate blast radius before mutation.
- Prefer reversible mitigations over irreversible actions.
- Require explicit confirmation for production mutations.
- Keep an append-only audit trail for every evaluation and action.

## Multi-Agent Topology

InfraBot is structured as coordinated agents, not a single CLI routine:

- `Controller`: central orchestrator and approval gatekeeper.
- `SRE Agent`: incident analysis, SLO burn-rate checks, bounded remediation.
- `Platform Agent`: deployment risk and governance checks.
- `Infra Agent`: infrastructure diagnostics and node/cloud actions.

Each agent emits proposals; controller selects observe-first actions, then bounded remediation when required by urgency.

## Confirmation and Safety Model

Production write actions require:

- Safety report (`infracore run <skill> --env=production`)
- Policy pass (including production guardrails)
- Explicit confirmation flag: `--param _confirmed=true`
- Approved change ticket: `--param _change_ticket=CHG-1234`
- Rollback metadata: `--param _rollback_plan="rollback procedure"`

Optional projected capacity gate:

- `--param _capacity_impact_pct=<n>` is denied if `n > 20` in production.

## P0 Incident Workflow

Use runbook:

```bash
infracore runbook info p0-incident-response
infracore runbook run p0-incident-response
infracore runbook run reliability-guardrail-loop
infracore runbook run security-compliance-governance-cycle
```

Expected flow:

1. Acknowledge incident immediately.
2. Page on-call and open incident comms channel.
3. Determine blast radius and impacted services.
4. Pull service-specific runbook.
5. Execute safest reversible mitigation.
6. Verify recovery.
7. Draft postmortem timeline.

## Example Commands

```bash
# Evaluate a production scale action (requires explicit confirmation policy)
infracore run aws.ec2.scale --env=production --param asg_name=payments --param desired_capacity=12 --param _confirmed=true --param _capacity_impact_pct=15

# Simulate incident workflow
infracore runbook run p0-incident-response
```
