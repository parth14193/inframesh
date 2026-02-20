# InfraCore — Cloud Infrastructure Agent Framework

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Skills](https://img.shields.io/badge/Skills-42-blue)](pkg/skills/)
[![Policies](https://img.shields.io/badge/Policies-8-orange)](pkg/policy/)
[![Runbooks](https://img.shields.io/badge/Runbooks-5-purple)](pkg/runbook/)

> An intelligent, modular orchestrator for multi-cloud infrastructure tasks with built-in safety controls, policy enforcement, compliance auditing, drift detection, and operational runbooks.

---

## Architecture

```
InfraCore v2.0.0
├── cmd/infracore/              CLI entry point (16 subcommands)
├── pkg/
│   ├── core/                   Types & interfaces
│   ├── skills/                 Skill Registry (42 built-in skills)
│   ├── executor/               Tool Runner (CLI/DryRun/Composite)
│   ├── planner/                Multi-step plan engine
│   ├── safety/                 Blast radius & risk evaluation
│   ├── policy/                 Policy Engine (8 guardrails)
│   ├── compliance/             CIS / SOC2 / HIPAA auditing
│   ├── drift/                  Infrastructure drift detection
│   ├── runbook/                Operational runbooks (5 built-in)
│   ├── health/                 Health probes (HTTP/TCP/DNS)
│   ├── notify/                 Slack / Webhook / Console alerts
│   ├── rbac/                   Role-based access control
│   ├── resilience/             Retry & Circuit Breaker
│   ├── config/                 YAML profiles & credentials
│   ├── state/                  Session state & audit log
│   └── output/                 Structured ASCII rendering
└── go.mod
```

---

## Feature Matrix

| Feature | Package | Key Capabilities |
|---|---|---|
| **42 Skills** | `pkg/skills` | AWS, K8s, Terraform, GCP, Azure, Datadog, Vault, etc. |
| **Executor** | `pkg/executor` | CLI execution, dry-run, composite with pre/post hooks |
| **Policy Engine** | `pkg/policy` | 8 guardrails: no public S3, require tags, deploy windows |
| **Compliance** | `pkg/compliance` | 17 checks: CIS, SOC2, HIPAA frameworks |
| **Drift Detection** | `pkg/drift` | Terraform plan parsing, manual change detection |
| **Runbooks** | `pkg/runbook` | 5 operational procedures: incident response, rollback, etc. |
| **Health Checks** | `pkg/health` | HTTP, TCP, DNS probes with latency tracking |
| **Notifications** | `pkg/notify` | Slack, webhook, console with event filtering |
| **RBAC** | `pkg/rbac` | 4 roles: viewer → operator → admin → superadmin |
| **Resilience** | `pkg/resilience` | Exponential backoff retry + circuit breaker |
| **Safety Layer** | `pkg/safety` | Blast radius analysis, production escalation |
| **Config** | `pkg/config` | YAML profiles, credentials, environment profiles |

---

## Policy Engine (8 Built-in Guardrails)

| Policy | Enforcement | What it Prevents |
|---|---|---|
| `no_public_s3` | DENY | Public S3 bucket ACLs |
| `require_tags` | WARN | Resources missing team/env/service tags |
| `no_wide_open_sg` | DENY | Security groups open to 0.0.0.0/0 on sensitive ports |
| `production_deploy_window` | WARN | Deploys outside business hours (09-17 UTC) |
| `require_peer_review` | DENY | CRITICAL actions without peer reviewer |
| `max_blast_radius` | DENY | Operations affecting >50 resources |
| `no_direct_prod_access` | WARN | Direct prod mutations without IaC |
| `enforce_encryption` | DENY | Unencrypted storage resources |

---

## RBAC Roles

| Role | Risk Access | Environments | Approve |
|---|---|---|---|
| `viewer` | LOW only | staging, dev | ❌ |
| `operator` | LOW, MEDIUM | staging, dev, qa | ❌ |
| `admin` | LOW → HIGH | all including production | ✅ |
| `superadmin` | ALL including CRITICAL | all | ✅ |

---

## Operational Runbooks (5 Built-in)

- **`high-cpu-response`** — Auto-investigate, scale, verify (8 steps + escalation)
- **`disk-full-response`** — Cleanup, snapshot, resize (7 steps)
- **`deployment-rollback`** — Rollback K8s deployment with verification (6 steps)
- **`secret-rotation`** — Scheduled secret rotation workflow (8 steps)
- **`incident-response`** — Full incident response procedure (8 steps + escalation)

---

## Usage

```bash
# Core
infracore skills list --provider=aws
infracore run aws.ec2.list --param region=us-west-2
infracore plan "deploy v2.5.0 to production"

# Policy & Compliance
infracore policy list
infracore policy check k8s.deploy --env=production
infracore compliance audit CIS

# Operations
infracore drift detect
infracore runbook list
infracore runbook run deployment-rollback
infracore health check

# Configuration
infracore config show
infracore config init
infracore rbac show
```

---

## Building & Testing

```bash
go build -o infracore ./cmd/infracore/
go test ./... -v
```

---

**InfraCore v2.0.0** | 16 packages | 42 skills | 8 policies | 17 compliance checks | 5 runbooks