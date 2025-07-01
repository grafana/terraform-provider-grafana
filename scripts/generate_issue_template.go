package main

// This file generates GitHub issue templates with terraform resource/data source dropdowns.
// It sources component names from the provider schema (definitive source of truth).

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
)

type ProviderSchema struct {
	ProviderSchemas map[string]struct {
		ResourceSchemas   map[string]interface{} `json:"resource_schemas"`
		DataSourceSchemas map[string]interface{} `json:"data_source_schemas"`
	} `json:"provider_schemas"`
}

func main() {
	updateSchema := len(os.Args) > 1 && (os.Args[1] == "--update-schema" || os.Args[1] == "-u")

	// Load or generate schema
	schema := loadSchema(updateSchema)

	// Extract resources
	var resources []string
	grafanaProvider := schema.ProviderSchemas["registry.terraform.io/grafana/grafana"]

	for name := range grafanaProvider.ResourceSchemas {
		resources = append(resources, name+" (resource)")
	}
	for name := range grafanaProvider.DataSourceSchemas {
		resources = append(resources, name+" (data source)")
	}

	// Add "Other" option for cases not covered
	resources = append(resources, "Other (please describe in the issue)")

	sort.Strings(resources)

	// Generate template
	generateTemplate(resources)
	fmt.Printf("Generated issue template with %d resources\n", len(resources))
}

func loadSchema(update bool) ProviderSchema {
	schemaFile := "provider_schema.json"

	// Use existing schema unless updating
	if !update {
		if data, err := os.ReadFile(schemaFile); err == nil {
			var schema ProviderSchema
			if json.Unmarshal(data, &schema) == nil {
				fmt.Println("Using existing schema")
				return schema
			}
		}
	}

	// Generate new schema
	fmt.Println("Generating schema...")
	cmd := exec.Command("./scripts/generate_schema.sh")
	data, err := cmd.Output()
	if err != nil {
		panic(fmt.Sprintf("Failed to generate schema: %v", err))
	}

	var schema ProviderSchema
	if err := json.Unmarshal(data, &schema); err != nil {
		panic(fmt.Sprintf("Invalid schema JSON: %v", err))
	}

	// Save schema
	os.WriteFile(schemaFile, data, 0600)
	fmt.Println("Schema updated")
	return schema
}

func generateTemplate(resources []string) {
	os.MkdirAll(".github/ISSUE_TEMPLATE", 0755)

	template := `# NOTE: this template is automatically generated
name: Bug Report (Enhanced)
description: File a bug report with resource dropdown
type: "bug"
projects: ["grafana/513"]
body:
  - type: input
    id: terraform_version
    attributes:
      label: Terraform Version
  - type: input
    id: terraform_grafana_provider_version
    attributes:
      label: Terraform Grafana Provider Version
  - type: input
    id: grafana_version
    attributes:
      label: Grafana Version
  - type: dropdown
    id: affected_resources
    attributes:
      label: Affected Resource(s)
      description: |
        Select the terraform resources or data sources this issue relates to.
        💡 Tip: Click the dropdown and start typing to quickly find items
      multiple: true
      options:
` + "        - " + strings.Join(resources, "\n        - ") + `
  - type: textarea
    id: terraform_configuration_files
    attributes:
      label: Terraform Configuration Files
      placeholder: |
        ` + "```hcl\n        # Copy-paste your Terraform configurations here\n        ```" + `
  - type: textarea
    id: expected_behavior
    attributes:
      label: Expected Behavior
      placeholder: What should have happened?
  - type: textarea
    id: actual_behavior
    attributes:
      label: Actual Behavior
      placeholder: What actually happened?
  - type: textarea
    id: steps_to_reproduce
    attributes:
      label: Steps to Reproduce
      placeholder: |
        Please list the steps required to reproduce the issue, for example:
        1. ` + "`terraform apply`" + `
  - type: textarea
    id: important_factoids
    attributes:
      label: Important Factoids
      placeholder: |
        Are there anything atypical about your accounts that we should know?
  - type: textarea
    id: references
    attributes:
      label: References
      placeholder: |
        - GH-1234`

	os.WriteFile(".github/ISSUE_TEMPLATE/3-bug-report-enhanced.yml", []byte(template), 0600)
}
