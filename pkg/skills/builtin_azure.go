package skills

import (
	"time"

	"github.com/parth14193/ownbot/pkg/core"
)

// AzureSkills returns all built-in Azure skill definitions.
func AzureSkills() []*core.Skill {
	return []*core.Skill{
		{
			Name:        "azure.vm.resize",
			Description: "Resize Azure Virtual Machines",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryCompute,
			Inputs: []core.SkillInput{
				{Name: "resource_group", Type: "string", Required: true, Description: "Azure resource group"},
				{Name: "vm_name", Type: "string", Required: true, Description: "Virtual machine name"},
				{Name: "new_size", Type: "string", Required: true, Description: "Target VM size (e.g., Standard_D4s_v3)"},
			},
			Outputs: []core.SkillOutput{
				{Name: "previous_size", Type: "string", Description: "Previous VM size"},
				{Name: "new_size", Type: "string", Description: "New VM size"},
				{Name: "status", Type: "string", Description: "Resize operation status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "az vm resize --resource-group {rg} --name {vm} --size {size}",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: true,
				Procedure: "az vm resize --resource-group {rg} --name {vm} --size {previous_size}",
			},
		},
		{
			Name:        "azure.blob.migrate",
			Description: "Migrate Azure Blob storage between accounts",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryStorage,
			Inputs: []core.SkillInput{
				{Name: "source_account", Type: "string", Required: true, Description: "Source storage account name"},
				{Name: "source_container", Type: "string", Required: true, Description: "Source container name"},
				{Name: "dest_account", Type: "string", Required: true, Description: "Destination storage account name"},
				{Name: "dest_container", Type: "string", Required: true, Description: "Destination container name"},
			},
			Outputs: []core.SkillOutput{
				{Name: "blobs_migrated", Type: "int", Description: "Number of blobs migrated"},
				{Name: "bytes_transferred", Type: "int", Description: "Total bytes transferred"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "azcopy copy 'https://{src}.blob.core.windows.net/{container}' 'https://{dst}.blob.core.windows.net/{container}' --recursive",
				Timeout: 1800 * time.Second,
			},
			Rollback: core.RollbackConfig{
				Supported: false,
				Procedure: "Manual cleanup of destination container required",
			},
		},
		{
			Name:        "azure.aks.deploy",
			Description: "Deploy an application to Azure Kubernetes Service",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "resource_group", Type: "string", Required: true, Description: "Azure resource group"},
				{Name: "cluster_name", Type: "string", Required: true, Description: "AKS cluster name"},
				{Name: "namespace", Type: "string", Required: false, Description: "Kubernetes namespace", Default: "default"},
				{Name: "deployment", Type: "string", Required: false, Description: "Deployment name", Default: "app"},
				{Name: "image", Type: "string", Required: false, Description: "Container image", Default: "app:latest"},
			},
			Outputs: []core.SkillOutput{
				{Name: "rollout_status", Type: "string", Description: "Deployment rollout status"},
				{Name: "cluster_fqdn", Type: "string", Description: "AKS API endpoint"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "az aks get-credentials --resource-group {resource_group} --name {cluster_name} && kubectl apply -f deployment.yaml && kubectl rollout status",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "kubectl rollout undo deployment/{name} -n {namespace}"},
		},
		{
			Name:        "azure.vm.deploy.cpu_optimized",
			Description: "Launch or update Azure VM workloads using CPU-optimized VM sizes",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "resource_group", Type: "string", Required: true, Description: "Azure resource group"},
				{Name: "vm_name", Type: "string", Required: true, Description: "Virtual machine name"},
				{Name: "vm_size", Type: "string", Required: false, Description: "CPU-optimized VM size", Default: "Standard_F4s_v2"},
				{Name: "image", Type: "string", Required: false, Description: "Marketplace image", Default: "Ubuntu2204"},
				{Name: "location", Type: "string", Required: false, Description: "Azure region", Default: "eastus"},
			},
			Outputs: []core.SkillOutput{
				{Name: "vm_id", Type: "string", Description: "Created VM resource ID"},
				{Name: "vm_size", Type: "string", Description: "Resolved CPU-optimized VM size"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "az vm create --resource-group {resource_group} --name {vm_name} --size {vm_size}",
				Timeout: 240 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "az vm delete --resource-group {resource_group} --name {vm_name} --yes"},
		},
		{
			Name:        "azure.sql.deploy.secure",
			Description: "Launch Azure SQL with private networking, backup retention, and security defaults",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "resource_group", Type: "string", Required: true, Description: "Azure resource group"},
				{Name: "server_name", Type: "string", Required: true, Description: "Azure SQL server name"},
				{Name: "database_name", Type: "string", Required: true, Description: "Azure SQL database name"},
				{Name: "service_objective", Type: "string", Required: false, Description: "Azure SQL performance tier", Default: "S2"},
				{Name: "private_endpoint", Type: "bool", Required: false, Description: "Enable private endpoint", Default: "true"},
				{Name: "zone_redundant", Type: "bool", Required: false, Description: "Enable zone redundancy", Default: "true"},
				{Name: "backup_retention_days", Type: "int", Required: false, Description: "Long-term backup retention in days", Default: "7"},
			},
			Outputs: []core.SkillOutput{
				{Name: "server_fqdn", Type: "string", Description: "Azure SQL server FQDN"},
				{Name: "database_id", Type: "string", Description: "Azure SQL database resource ID"},
			},
			RiskLevel:            core.RiskHigh,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "az sql server create && az sql db create --resource-group {resource_group} --server {server_name} --name {database_name}",
				Timeout: 600 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "az sql db delete --resource-group {resource_group} --server {server_name} --name {database_name}"},
		},
		{
			Name:        "azure.service.deploy",
			Description: "Deploy an Azure service workload using a service name and deployment profile",
			Provider:    core.ProviderAzure,
			Category:    core.CategoryDeployment,
			Inputs: []core.SkillInput{
				{Name: "service", Type: "string", Required: true, Description: "Azure service name (aks, vm, sql, appservice, functions, etc.)"},
				{Name: "profile", Type: "string", Required: false, Description: "Deployment profile such as secure, cpu_optimized, or standard", Default: "standard"},
				{Name: "environment", Type: "string", Required: false, Description: "Target environment", Default: "staging"},
				{Name: "location", Type: "string", Required: false, Description: "Azure region", Default: "eastus"},
			},
			Outputs: []core.SkillOutput{
				{Name: "service", Type: "string", Description: "Resolved Azure service"},
				{Name: "status", Type: "string", Description: "Deployment status"},
			},
			RiskLevel:            core.RiskMedium,
			RequiresConfirmation: true,
			Execution: core.ExecutionConfig{
				Type:    core.ExecCLI,
				Command: "az <service> <deploy-command>",
				Timeout: 300 * time.Second,
			},
			Rollback: core.RollbackConfig{Supported: true, Procedure: "Run service-specific rollback or previous deployment version"},
		},
	}
}
