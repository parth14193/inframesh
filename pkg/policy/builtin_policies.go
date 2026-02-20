package policy

import (
	"fmt"
	"strings"
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// BuiltinPolicies returns all built-in infrastructure guardrail policies.
func BuiltinPolicies() []*Policy {
	return []*Policy{
		noPublicS3Policy(),
		requireTagsPolicy(),
		noWideOpenSGPolicy(),
		productionDeployWindowPolicy(),
		requirePeerReviewPolicy(),
		maxBlastRadiusPolicy(),
		noDirectProdAccess(),
		enforceEncryptionPolicy(),
	}
}

// ── Policy Implementations ─────────────────────────────────────

func noPublicS3Policy() *Policy {
	return &Policy{
		Name:        "no_public_s3",
		Description: "Deny S3 operations that could expose buckets publicly",
		Enforcement: EnforcementDeny,
		Severity:    SeverityCritical,
		AppliesTo:   []string{"aws.s3.*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil {
				return false, ""
			}
			if acl, ok := params["acl"]; ok {
				aclStr := fmt.Sprintf("%v", acl)
				if aclStr == "public-read" || aclStr == "public-read-write" {
					return true, fmt.Sprintf("S3 bucket cannot use public ACL '%s' — use private ACL with CloudFront for public access", aclStr)
				}
			}
			return false, ""
		},
	}
}

func requireTagsPolicy() *Policy {
	requiredTags := []string{"team", "env", "service"}
	return &Policy{
		Name:        "require_tags",
		Description: "Resources must have required tags (team, env, service)",
		Enforcement: EnforcementWarn,
		Severity:    SeverityWarning,
		AppliesTo:   []string{"aws.ec2.*", "aws.lambda.*", "gcp.gce.*", "azure.vm.*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil {
				return true, fmt.Sprintf("No tags provided — required tags: %s", strings.Join(requiredTags, ", "))
			}

			tags, ok := params["tags"]
			if !ok {
				return true, fmt.Sprintf("Missing required tags: %s", strings.Join(requiredTags, ", "))
			}

			tagsMap, ok := tags.(map[string]interface{})
			if !ok {
				return true, "Tags must be a key-value map"
			}

			var missing []string
			for _, tag := range requiredTags {
				if _, exists := tagsMap[tag]; !exists {
					missing = append(missing, tag)
				}
			}

			if len(missing) > 0 {
				return true, fmt.Sprintf("Missing required tags: %s", strings.Join(missing, ", "))
			}
			return false, ""
		},
	}
}

func noWideOpenSGPolicy() *Policy {
	return &Policy{
		Name:        "no_wide_open_sg",
		Description: "Deny security group rules allowing 0.0.0.0/0 on sensitive ports",
		Enforcement: EnforcementDeny,
		Severity:    SeverityCritical,
		AppliesTo:   []string{"aws.sg.*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil {
				return false, ""
			}
			if cidr, ok := params["cidr"]; ok {
				cidrStr := fmt.Sprintf("%v", cidr)
				if cidrStr == "0.0.0.0/0" || cidrStr == "::/0" {
					port := params["port"]
					if port != nil {
						portStr := fmt.Sprintf("%v", port)
						sensitivePort := portStr == "22" || portStr == "3389" || portStr == "3306" || portStr == "5432"
						if sensitivePort {
							return true, fmt.Sprintf("Cannot open port %s to %s — use VPN or bastion host", portStr, cidrStr)
						}
					}
					return true, fmt.Sprintf("Inbound rule for %s is too permissive — restrict to specific CIDR ranges", cidrStr)
				}
			}
			return false, ""
		},
	}
}

func productionDeployWindowPolicy() *Policy {
	return &Policy{
		Name:         "production_deploy_window",
		Description:  "Production deployments only allowed during business hours (09:00-17:00 UTC, Mon-Fri)",
		Enforcement:  EnforcementWarn,
		Severity:     SeverityWarning,
		AppliesTo:    []string{"k8s.deploy", "helm.upgrade", "terraform.apply", "argocd.sync"},
		Environments: []string{"production", "prod"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			now := time.Now().UTC()
			hour := now.Hour()
			weekday := now.Weekday()

			if weekday == time.Saturday || weekday == time.Sunday {
				return true, fmt.Sprintf("Production deploys not recommended on weekends (%s)", weekday)
			}
			if hour < 9 || hour > 17 {
				return true, fmt.Sprintf("Production deploys not recommended outside business hours (current: %02d:00 UTC, window: 09:00-17:00)", hour)
			}
			return false, ""
		},
	}
}

func requirePeerReviewPolicy() *Policy {
	return &Policy{
		Name:        "require_peer_review",
		Description: "CRITICAL actions require a peer reviewer confirmation",
		Enforcement: EnforcementDeny,
		Severity:    SeverityCritical,
		AppliesTo:   []string{"terraform.apply", "k8s.deploy", "aws.secrets.rotate"},
		Environments: []string{"production", "prod"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if skill.RiskLevel >= core.RiskCritical {
				if params == nil {
					return true, "CRITICAL action in production requires peer review — set _peer_reviewer param"
				}
				if _, ok := params["_peer_reviewer"]; !ok {
					return true, "CRITICAL action in production requires peer review — set _peer_reviewer param"
				}
			}
			return false, ""
		},
	}
}

func maxBlastRadiusPolicy() *Policy {
	return &Policy{
		Name:        "max_blast_radius",
		Description: "Deny operations affecting more than 50 resources at once",
		Enforcement: EnforcementDeny,
		Severity:    SeverityCritical,
		AppliesTo:   []string{"*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil {
				return false, ""
			}
			if count, ok := params["_resource_count"]; ok {
				if c, ok := count.(int); ok && c > 50 {
					return true, fmt.Sprintf("Operation affects %d resources (max: 50) — break into smaller batches", c)
				}
			}
			return false, ""
		},
	}
}

func noDirectProdAccess() *Policy {
	return &Policy{
		Name:         "no_direct_prod_access",
		Description:  "Deny direct mutation of production resources without going through IaC",
		Enforcement:  EnforcementWarn,
		Severity:     SeverityWarning,
		AppliesTo:    []string{"aws.ec2.scale", "azure.vm.resize", "aws.sg.*"},
		Environments: []string{"production", "prod"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil || params["_iac_managed"] == nil {
				return true, "Direct production mutations should go through Terraform/Pulumi — set _iac_managed=true to override"
			}
			return false, ""
		},
	}
}

func enforceEncryptionPolicy() *Policy {
	return &Policy{
		Name:        "enforce_encryption",
		Description: "Storage resources must have encryption enabled",
		Enforcement: EnforcementDeny,
		Severity:    SeverityCritical,
		AppliesTo:   []string{"aws.s3.*", "gcp.gcs.*", "azure.blob.*"},
		CheckFunc: func(skill *core.Skill, params map[string]interface{}, env string) (bool, string) {
			if params == nil {
				return false, "" // Can't check without params
			}
			if encryption, ok := params["encryption"]; ok {
				if fmt.Sprintf("%v", encryption) == "none" || fmt.Sprintf("%v", encryption) == "false" {
					return true, "Storage resources must have encryption enabled — use AES256 or aws:kms"
				}
			}
			return false, ""
		},
	}
}
