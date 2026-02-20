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
	}
}
