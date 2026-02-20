// Package config provides YAML-based configuration for the InfraCore agent,
// including environment profiles, provider credentials, and default settings.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/parth14193/ownbot/pkg/core"
)

// Config is the top-level InfraCore configuration.
type Config struct {
	Version      string                `yaml:"version" json:"version"`
	DefaultEnv   string                `yaml:"default_environment" json:"default_environment"`
	DefaultRegion string               `yaml:"default_region" json:"default_region"`
	Profiles     map[string]*Profile   `yaml:"profiles" json:"profiles"`
	Credentials  map[string]*Credential `yaml:"credentials" json:"credentials"`
	Notifications *NotificationConfig  `yaml:"notifications,omitempty" json:"notifications,omitempty"`
	Policies     *PolicyConfig         `yaml:"policies,omitempty" json:"policies,omitempty"`
	RBAC         *RBACConfig           `yaml:"rbac,omitempty" json:"rbac,omitempty"`
}

// Profile represents an environment profile (dev, staging, production).
type Profile struct {
	Name        string        `yaml:"name" json:"name"`
	Environment string        `yaml:"environment" json:"environment"`
	Provider    core.Provider `yaml:"provider" json:"provider"`
	Region      string        `yaml:"region" json:"region"`
	Credential  string        `yaml:"credential" json:"credential"` // references Credentials map key
	Cluster     string        `yaml:"cluster,omitempty" json:"cluster,omitempty"`
	Namespace   string        `yaml:"namespace,omitempty" json:"namespace,omitempty"`
	Tags        map[string]string `yaml:"tags,omitempty" json:"tags,omitempty"`
}

// Credential holds authentication details for a cloud provider.
type Credential struct {
	Provider    core.Provider `yaml:"provider" json:"provider"`
	Type        string        `yaml:"type" json:"type"` // access_key, service_account, kubeconfig, profile
	AccessKey   string        `yaml:"access_key,omitempty" json:"access_key,omitempty"`
	SecretKey   string        `yaml:"secret_key,omitempty" json:"secret_key,omitempty"`
	Profile     string        `yaml:"profile,omitempty" json:"profile,omitempty"`
	RoleARN     string        `yaml:"role_arn,omitempty" json:"role_arn,omitempty"`
	KeyFile     string        `yaml:"key_file,omitempty" json:"key_file,omitempty"`
	Kubeconfig  string        `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty"`
	Context     string        `yaml:"context,omitempty" json:"context,omitempty"`
}

// NotificationConfig holds notification channel settings.
type NotificationConfig struct {
	Enabled  bool                    `yaml:"enabled" json:"enabled"`
	Channels map[string]*ChannelConfig `yaml:"channels" json:"channels"`
}

// ChannelConfig defines a notification channel.
type ChannelConfig struct {
	Type       string `yaml:"type" json:"type"` // slack, webhook, console
	WebhookURL string `yaml:"webhook_url,omitempty" json:"webhook_url,omitempty"`
	Channel    string `yaml:"channel,omitempty" json:"channel,omitempty"`
	OnSuccess  bool   `yaml:"on_success" json:"on_success"`
	OnFailure  bool   `yaml:"on_failure" json:"on_failure"`
	OnHighRisk bool   `yaml:"on_high_risk" json:"on_high_risk"`
}

// PolicyConfig holds policy engine settings.
type PolicyConfig struct {
	Enabled         bool     `yaml:"enabled" json:"enabled"`
	EnforcementMode string   `yaml:"enforcement_mode" json:"enforcement_mode"` // warn, deny
	EnabledPolicies []string `yaml:"enabled_policies" json:"enabled_policies"`
}

// RBACConfig holds access control settings.
type RBACConfig struct {
	Enabled bool              `yaml:"enabled" json:"enabled"`
	Users   map[string]string `yaml:"users" json:"users"` // username -> role
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Version:       "1.0",
		DefaultEnv:    "staging",
		DefaultRegion: "us-east-1",
		Profiles: map[string]*Profile{
			"staging": {
				Name:        "staging",
				Environment: "staging",
				Provider:    core.ProviderAWS,
				Region:      "us-east-1",
				Credential:  "default",
			},
			"production": {
				Name:        "production",
				Environment: "production",
				Provider:    core.ProviderAWS,
				Region:      "us-east-1",
				Credential:  "default",
			},
		},
		Credentials: map[string]*Credential{
			"default": {
				Provider: core.ProviderAWS,
				Type:     "profile",
				Profile:  "default",
			},
		},
		Notifications: &NotificationConfig{
			Enabled: true,
			Channels: map[string]*ChannelConfig{
				"console": {
					Type:      "console",
					OnSuccess: true,
					OnFailure: true,
				},
			},
		},
		Policies: &PolicyConfig{
			Enabled:         true,
			EnforcementMode: "warn",
			EnabledPolicies: []string{"no_public_s3", "require_tags", "no_wide_open_sg"},
		},
		RBAC: &RBACConfig{
			Enabled: false,
			Users:   map[string]string{},
		},
	}
}

// GetProfile returns a profile by name, falling back to default environment.
func (c *Config) GetProfile(name string) (*Profile, error) {
	if p, ok := c.Profiles[name]; ok {
		return p, nil
	}
	return nil, fmt.Errorf("profile not found: %s", name)
}

// GetCredential returns a credential by name.
func (c *Config) GetCredential(name string) (*Credential, error) {
	if cred, ok := c.Credentials[name]; ok {
		return cred, nil
	}
	return nil, fmt.Errorf("credential not found: %s", name)
}

// DefaultConfigPath returns the default config file path.
func DefaultConfigPath() string {
	home := homeDir()
	return filepath.Join(home, ".infracore", "config.yaml")
}

// GenerateConfigYAML produces a sample YAML configuration string.
func GenerateConfigYAML() string {
	return `# InfraCore Configuration
version: "1.0"
default_environment: staging
default_region: us-east-1

profiles:
  staging:
    name: staging
    environment: staging
    provider: aws
    region: us-east-1
    credential: aws-staging
    tags:
      team: platform
      env: staging

  production:
    name: production
    environment: production
    provider: aws
    region: us-east-1
    credential: aws-prod
    cluster: eks-prod-us-east-1
    namespace: default
    tags:
      team: platform
      env: production

credentials:
  aws-staging:
    provider: aws
    type: profile
    profile: staging

  aws-prod:
    provider: aws
    type: profile
    profile: production
    role_arn: arn:aws:iam::123456789012:role/InfraCoreProd

  k8s-prod:
    provider: k8s
    type: kubeconfig
    kubeconfig: ~/.kube/config
    context: eks-prod

notifications:
  enabled: true
  channels:
    slack-ops:
      type: slack
      webhook_url: https://hooks.slack.com/services/xxx/yyy/zzz
      channel: "#ops-alerts"
      on_success: false
      on_failure: true
      on_high_risk: true
    console:
      type: console
      on_success: true
      on_failure: true

policies:
  enabled: true
  enforcement_mode: deny  # warn | deny
  enabled_policies:
    - no_public_s3
    - require_tags
    - no_wide_open_sg
    - production_deploy_window
    - max_blast_radius

rbac:
  enabled: true
  users:
    admin: superadmin
    deployer: operator
    viewer: viewer
`
}

// Validate checks the configuration for required fields and consistency.
func (c *Config) Validate() []error {
	var errs []error

	if c.DefaultEnv == "" {
		errs = append(errs, fmt.Errorf("default_environment is required"))
	}

	for name, profile := range c.Profiles {
		if profile.Environment == "" {
			errs = append(errs, fmt.Errorf("profile '%s': environment is required", name))
		}
		if profile.Credential != "" {
			if _, ok := c.Credentials[profile.Credential]; !ok {
				errs = append(errs, fmt.Errorf("profile '%s': credential '%s' not found", name, profile.Credential))
			}
		}
	}

	return errs
}

// Render returns a human-readable string of the configuration.
func (c *Config) Render() string {
	var b strings.Builder
	b.WriteString("⚙️  INFRACORE CONFIGURATION\n")
	b.WriteString("─────────────────────────────────────────\n")
	b.WriteString(fmt.Sprintf("Version:      %s\n", c.Version))
	b.WriteString(fmt.Sprintf("Default Env:  %s\n", c.DefaultEnv))
	b.WriteString(fmt.Sprintf("Default Rgn:  %s\n", c.DefaultRegion))

	b.WriteString(fmt.Sprintf("\n📂 PROFILES (%d):\n", len(c.Profiles)))
	for name, p := range c.Profiles {
		b.WriteString(fmt.Sprintf("  • %s → %s / %s / %s (cred: %s)\n", name, p.Environment, p.Provider, p.Region, p.Credential))
	}

	b.WriteString(fmt.Sprintf("\n🔑 CREDENTIALS (%d):\n", len(c.Credentials)))
	for name, cred := range c.Credentials {
		b.WriteString(fmt.Sprintf("  • %s → %s / %s\n", name, cred.Provider, cred.Type))
	}

	if c.Notifications != nil {
		b.WriteString(fmt.Sprintf("\n🔔 NOTIFICATIONS: enabled=%t (%d channels)\n", c.Notifications.Enabled, len(c.Notifications.Channels)))
	}
	if c.Policies != nil {
		b.WriteString(fmt.Sprintf("🛡️  POLICIES: enabled=%t mode=%s (%d active)\n", c.Policies.Enabled, c.Policies.EnforcementMode, len(c.Policies.EnabledPolicies)))
	}
	if c.RBAC != nil {
		b.WriteString(fmt.Sprintf("🔐 RBAC: enabled=%t (%d users)\n", c.RBAC.Enabled, len(c.RBAC.Users)))
	}

	return b.String()
}

func homeDir() string {
	if runtime.GOOS == "windows" {
		return os.Getenv("USERPROFILE")
	}
	return os.Getenv("HOME")
}
