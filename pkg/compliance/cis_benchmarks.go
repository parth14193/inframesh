package compliance

// CISBenchmarks returns CIS AWS Foundations Benchmark checks.
func CISBenchmarks() []*Check {
	return []*Check{
		{
			ID:          "CIS-1.1",
			Framework:   FrameworkCIS,
			Title:       "Avoid the use of the root account",
			Description: "The root account has unrestricted access. Verify it has not been used recently.",
			Severity:    SeverityCritical,
			Category:    "Identity and Access Management",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires AWS API access to check root account last login",
					Remediation: "Run: aws iam get-credential-report and verify root LastUsedDate > 90 days",
				}
			},
		},
		{
			ID:          "CIS-1.2",
			Framework:   FrameworkCIS,
			Title:       "Ensure MFA is enabled for all IAM users with console access",
			Description: "Multi-factor authentication adds a second layer of protection.",
			Severity:    SeverityHigh,
			Category:    "Identity and Access Management",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires AWS API access to list IAM users and MFA devices",
					Remediation: "Run: aws iam list-users + aws iam list-mfa-devices for each user",
				}
			},
		},
		{
			ID:          "CIS-1.3",
			Framework:   FrameworkCIS,
			Title:       "Ensure credentials unused for 90+ days are disabled",
			Description: "Stale credentials increase attack surface.",
			Severity:    SeverityMedium,
			Category:    "Identity and Access Management",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires credential report analysis",
					Remediation: "Run: aws iam generate-credential-report && aws iam get-credential-report",
				}
			},
		},
		{
			ID:          "CIS-2.1",
			Framework:   FrameworkCIS,
			Title:       "Ensure CloudTrail is enabled in all regions",
			Description: "CloudTrail logs all API calls for audit and forensic purposes.",
			Severity:    SeverityHigh,
			Category:    "Logging",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires AWS API: aws cloudtrail describe-trails",
					Remediation: "Enable multi-region CloudTrail: aws cloudtrail create-trail --is-multi-region-trail",
				}
			},
		},
		{
			ID:          "CIS-2.2",
			Framework:   FrameworkCIS,
			Title:       "Ensure CloudTrail log file validation is enabled",
			Description: "Log file validation ensures logs are not tampered with.",
			Severity:    SeverityMedium,
			Category:    "Logging",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires CloudTrail trail configuration",
					Remediation: "aws cloudtrail update-trail --enable-log-file-validation",
				}
			},
		},
		{
			ID:          "CIS-2.3",
			Framework:   FrameworkCIS,
			Title:       "Ensure CloudTrail logs are encrypted with KMS",
			Description: "Server-side encryption protects log data at rest.",
			Severity:    SeverityHigh,
			Category:    "Logging",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires trail encryption configuration check",
					Remediation: "aws cloudtrail update-trail --kms-key-id <KMS_KEY_ARN>",
				}
			},
		},
		{
			ID:          "CIS-3.1",
			Framework:   FrameworkCIS,
			Title:       "Ensure VPC flow logging is enabled in all VPCs",
			Description: "VPC flow logs capture IP traffic for network monitoring.",
			Severity:    SeverityMedium,
			Category:    "Networking",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires VPC and flow log enumeration",
					Remediation: "aws ec2 create-flow-logs --resource-ids <VPC_ID> --traffic-type ALL",
				}
			},
		},
		{
			ID:          "CIS-3.2",
			Framework:   FrameworkCIS,
			Title:       "Ensure default security groups restrict all traffic",
			Description: "The default security group should not allow any inbound/outbound traffic.",
			Severity:    SeverityHigh,
			Category:    "Networking",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires SG rule inspection",
					Remediation: "Remove all rules from default SGs: aws ec2 revoke-security-group-ingress",
				}
			},
		},
		{
			ID:          "CIS-4.1",
			Framework:   FrameworkCIS,
			Title:       "Ensure S3 bucket access logging is enabled",
			Description: "Access logging tracks requests made to S3 buckets.",
			Severity:    SeverityMedium,
			Category:    "Storage",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires S3 bucket logging configuration check",
					Remediation: "aws s3api put-bucket-logging --bucket <BUCKET> --bucket-logging-status ...",
				}
			},
		},
		{
			ID:          "CIS-4.2",
			Framework:   FrameworkCIS,
			Title:       "Ensure S3 buckets have server-side encryption enabled",
			Description: "Encryption at rest protects data stored in S3.",
			Severity:    SeverityHigh,
			Category:    "Storage",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Cannot verify — requires S3 encryption configuration check",
					Remediation: "aws s3api put-bucket-encryption --bucket <BUCKET> --sse AES256",
				}
			},
		},
	}
}

// SOC2Controls returns SOC2 compliance checks.
func SOC2Controls() []*Check {
	return []*Check{
		{
			ID:          "SOC2-CC6.1",
			Framework:   FrameworkSOC2,
			Title:       "Logical and physical access controls",
			Description: "Ensure access to systems is restricted and monitored.",
			Severity:    SeverityHigh,
			Category:    "Access Control",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify IAM policies follow least-privilege principle",
					Remediation: "Review IAM policies and remove overly permissive access",
				}
			},
		},
		{
			ID:          "SOC2-CC6.2",
			Framework:   FrameworkSOC2,
			Title:       "User access provisioning and deprovisioning",
			Description: "Ensure timely provisioning and removal of access.",
			Severity:    SeverityHigh,
			Category:    "Access Control",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify inactive accounts are disabled within 30 days",
					Remediation: "Implement automated deprovisioning via SSO integration",
				}
			},
		},
		{
			ID:          "SOC2-CC7.1",
			Framework:   FrameworkSOC2,
			Title:       "System monitoring",
			Description: "Ensure infrastructure monitoring and alerting is in place.",
			Severity:    SeverityMedium,
			Category:    "Monitoring",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify CloudWatch/Datadog monitoring covers all production resources",
					Remediation: "Enable comprehensive monitoring with alerting for anomalies",
				}
			},
		},
		{
			ID:          "SOC2-CC8.1",
			Framework:   FrameworkSOC2,
			Title:       "Change management process",
			Description: "Ensure all infrastructure changes go through a controlled process.",
			Severity:    SeverityHigh,
			Category:    "Change Management",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify all changes go through IaC with peer review",
					Remediation: "Enforce Terraform/Pulumi for all infra changes with PR reviews",
				}
			},
		},
	}
}

// HIPAAControls returns HIPAA compliance checks.
func HIPAAControls() []*Check {
	return []*Check{
		{
			ID:          "HIPAA-164.312a",
			Framework:   FrameworkHIPAA,
			Title:       "Access control — unique user identification",
			Description: "Each user accessing ePHI must have a unique ID.",
			Severity:    SeverityCritical,
			Category:    "Access Control",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify no shared IAM accounts for systems handling PHI",
					Remediation: "Use SSO with individual user accounts; disable shared credentials",
				}
			},
		},
		{
			ID:          "HIPAA-164.312c",
			Framework:   FrameworkHIPAA,
			Title:       "Integrity controls",
			Description: "Ensure ePHI is not improperly altered or destroyed.",
			Severity:    SeverityCritical,
			Category:    "Data Integrity",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify S3 versioning and MFA-delete are enabled for PHI buckets",
					Remediation: "Enable S3 versioning: aws s3api put-bucket-versioning --status Enabled",
				}
			},
		},
		{
			ID:          "HIPAA-164.312e",
			Framework:   FrameworkHIPAA,
			Title:       "Encryption of ePHI in transit and at rest",
			Description: "All ePHI must be encrypted in transit (TLS) and at rest (AES-256/KMS).",
			Severity:    SeverityCritical,
			Category:    "Encryption",
			CheckFunc: func() CheckResult {
				return CheckResult{
					Status:      StatusWarn,
					Details:     "Verify TLS 1.2+ enforced on all endpoints; S3/RDS/EBS encrypted",
					Remediation: "Enable encryption on all storage; enforce TLS on ALBs and CloudFront",
				}
			},
		},
	}
}
