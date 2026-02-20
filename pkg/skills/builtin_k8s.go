package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// KubernetesSkills returns all built-in Kubernetes skill definitions.
func KubernetesSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "k8s.deploy",
			Description: "Deploy or rollout Kubernetes workloads",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "namespace", Type: "string", Required: true, Description: "Target Kubernetes namespace"},
				{Name: "deployment", Type: "string", Required: true, Description: "Deployment name"},
				{Name: "image", Type: "string", Required: true, Description: "Container image with tag"},
				{Name: "replicas", Type: "int", Required: false, Description: "Number of replicas"},
				{Name: "context", Type: "string", Required: false, Description: "kubectl context to use"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Rollout status"},
				{Name: "revision", Type: "int", Description: "Deployment revision number"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl set image deployment/{name} {container}={image} -n {namespace}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "kubectl rollout undo deployment/{name} -n {namespace}",
			},
		},
		{
			Name:        "k8s.rollback",
			Description: "Rollback a Kubernetes deployment to a previous revision",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "namespace", Type: "string", Required: true, Description: "Target Kubernetes namespace"},
				{Name: "deployment", Type: "string", Required: true, Description: "Deployment name"},
				{Name: "revision", Type: "int", Required: false, Description: "Target revision (previous if omitted)"},
				{Name: "context", Type: "string", Required: false, Description: "kubectl context to use"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Rollback status"},
				{Name: "rolled_back_to", Type: "int", Description: "Revision rolled back to"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl rollout undo deployment/{name} --to-revision={revision} -n {namespace}",
				Timeout: 120 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "Re-deploy the image that was active before the rollback",
			},
		},
		{
			Name:        "k8s.rollout.status",
			Description: "Watch rollout status of a Kubernetes deployment",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "namespace", Type: "string", Required: true, Description: "Target Kubernetes namespace"},
				{Name: "deployment", Type: "string", Required: true, Description: "Deployment name"},
				{Name: "timeout", Type: "int", Required: false, Description: "Timeout in seconds", Default: "300"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Current rollout status"},
				{Name: "ready_replicas", Type: "int", Description: "Number of ready replicas"},
			},
			RiskLevel:            core.RiskLow,
			RequiresConfirmation: false,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl rollout status deployment/{name} -n {namespace} --timeout={timeout}s",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: false, Procedure: "Read-only operation"},
		},
		{
			Name:        "k8s.ingress.update",
			Description: "Update Kubernetes Ingress rules",
			Provider:    core.ProviderKubernetes,
			Category:    core.CategoryNetworking,
			Inputs: []core.SkillInput{
				{Name: "namespace", Type: "string", Required: true, Description: "Target namespace"},
				{Name: "ingress_name", Type: "string", Required: true, Description: "Ingress resource name"},
				{Name: "host", Type: "string", Required: true, Description: "Host rule to update"},
				{Name: "path", Type: "string", Required: false, Description: "Path rule", Default: "/"},
				{Name: "service", Type: "string", Required: true, Description: "Backend service name"},
				{Name: "port", Type: "int", Required: true, Description: "Backend service port"},
			},
			Outputs: []core.SkillOutput{
				{Name: "status", Type: "string", Description: "Apply status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "kubectl apply -f ingress.yaml -n {namespace}",
				Timeout: 30 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "kubectl apply -f previous-ingress.yaml -n {namespace}",
			},
		},
	}
}
