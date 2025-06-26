package main

// This file generates GitHub issue templates with dynamic terraform resource/data source dropdowns.
// It sources component names from Backstage catalog files or falls back to a hardcoded list.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

func main() {
	// Try methods in priority order:
	// 1. Catalog files from PR #2228 (when available)
	// 2. Reliable fallback list
	resources, err := loadResourcesFromCatalogFiles()
	if err != nil || len(resources) == 0 {
		fmt.Printf("No catalog files found (%v), using fallback list...\n", err)
		resources = getFallbackResources()
		fmt.Printf("Using %d fallback resources and data sources\n", len(resources))
	} else {
		fmt.Printf("Loaded %d resources from catalog files (PR #2228)\n", len(resources))
	}

	// Sort for consistent output
	sort.Strings(resources)

	// Generate the issue template
	err = generateIssueTemplate(resources)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating issue template: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated issue template with %d terraform resources and data sources\n", len(resources))
}

// loadResourcesFromCatalogFiles loads resources from PR #2228 catalog files (primary method)
func loadResourcesFromCatalogFiles() ([]string, error) {
	var allItems []string

	// Process resources
	resourceMatches, err := filepath.Glob("internal/resources/*/catalog-resource.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to find resource catalog files: %w", err)
	}

	resourceSet := make(map[string]bool)
	for _, catalogFile := range resourceMatches {
		fileResources, err := parseResourcesFromCatalogFile(catalogFile)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", catalogFile, err)
			continue
		}
		for _, resource := range fileResources {
			resourceSet[resource] = true
		}
	}

	// Add resources with (resource) suffix
	for resource := range resourceSet {
		allItems = append(allItems, fmt.Sprintf("%s (resource)", resource))
	}

	// Process data sources
	dataSourceMatches, err := filepath.Glob("internal/resources/*/catalog-data-source.yaml")
	if err != nil {
		return nil, fmt.Errorf("failed to find data source catalog files: %w", err)
	}

	dataSourceSet := make(map[string]bool)
	for _, catalogFile := range dataSourceMatches {
		fileResources, err := parseResourcesFromCatalogFile(catalogFile)
		if err != nil {
			fmt.Printf("Warning: failed to parse %s: %v\n", catalogFile, err)
			continue
		}
		for _, resource := range fileResources {
			dataSourceSet[resource] = true
		}
	}

	// Add data sources with (data source) suffix, excluding duplicates
	for dataSource := range dataSourceSet {
		if !resourceSet[dataSource] {
			allItems = append(allItems, fmt.Sprintf("%s (data source)", dataSource))
		}
	}

	if len(allItems) == 0 {
		return nil, fmt.Errorf("no catalog files found")
	}

	return allItems, nil
}

// parseResourcesFromCatalogFile extracts terraform resource names from catalog files
func parseResourcesFromCatalogFile(filePath string) ([]string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var resources []string

	// Look for component names that start with "grafana_"
	// Example: name: grafana_dashboard
	nameRegex := regexp.MustCompile(`name:\s+(grafana_\w+)`)
	matches := nameRegex.FindAllStringSubmatch(string(content), -1)

	for _, match := range matches {
		if len(match) > 1 {
			resources = append(resources, match[1])
		}
	}

	return resources, nil
}

// getFallbackResources provides comprehensive list when catalog files unavailable
func getFallbackResources() []string {
	var allItems []string

	// Core Grafana resources
	resources := []string{
		"grafana_annotation",
		"grafana_contact_point",
		"grafana_dashboard",
		"grafana_dashboard_permission",
		"grafana_dashboard_permission_item",
		"grafana_dashboard_public",
		"grafana_data_source",
		"grafana_data_source_config",
		"grafana_data_source_config_lbac_rules",
		"grafana_data_source_permission",
		"grafana_data_source_permission_item",
		"grafana_folder",
		"grafana_folder_permission",
		"grafana_folder_permission_item",
		"grafana_library_panel",
		"grafana_message_template",
		"grafana_mute_timing",
		"grafana_notification_policy",
		"grafana_organization",
		"grafana_organization_preferences",
		"grafana_playlist",
		"grafana_report",
		"grafana_role",
		"grafana_role_assignment",
		"grafana_role_assignment_item",
		"grafana_rule_group",
		"grafana_service_account",
		"grafana_service_account_permission",
		"grafana_service_account_permission_item",
		"grafana_service_account_token",
		"grafana_sso_settings",
		"grafana_team",
		"grafana_team_external_group",
		"grafana_user",
		"grafana_cloud_access_policy",
		"grafana_cloud_access_policy_token",
		"grafana_cloud_org_member",
		"grafana_cloud_plugin_installation",
		"grafana_cloud_stack",
		"grafana_cloud_stack_service_account",
		"grafana_cloud_stack_service_account_token",
		"grafana_cloud_private_data_source_connect_network",
		"grafana_cloud_private_data_source_connect_network_token",
		"grafana_cloud_provider_aws_account",
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		"grafana_cloud_provider_aws_resource_metadata_scrape_job",
		"grafana_cloud_provider_azure_credential",
		"grafana_cloud_synthetic_monitoring_installation",
		"grafana_cloud_k6_installation",
		"grafana_connections_metrics_endpoint_scrape_job",
		"grafana_fleet_management_collector",
		"grafana_fleet_management_pipeline",
		"grafana_frontend_o11y_app",
		"grafana_k6_load_test",
		"grafana_k6_project",
		"grafana_k6_project_limits",
		"grafana_machine_learning_alert",
		"grafana_machine_learning_holiday",
		"grafana_machine_learning_job",
		"grafana_machine_learning_outlier_detector",
		"grafana_oncall_escalation",
		"grafana_oncall_escalation_chain",
		"grafana_oncall_integration",
		"grafana_oncall_on_call_shift",
		"grafana_oncall_outgoing_webhook",
		"grafana_oncall_route",
		"grafana_oncall_schedule",
		"grafana_oncall_user_notification_rule",
		"grafana_slo",
		"grafana_synthetic_monitoring_check",
		"grafana_synthetic_monitoring_check_alerts",
		"grafana_synthetic_monitoring_probe",
	}

	// Data sources
	dataSources := []string{
		"grafana_dashboard",
		"grafana_dashboards",
		"grafana_data_source",
		"grafana_folder",
		"grafana_folders",
		"grafana_library_panel",
		"grafana_library_panels",
		"grafana_organization",
		"grafana_organization_preferences",
		"grafana_organization_user",
		"grafana_role",
		"grafana_service_account",
		"grafana_team",
		"grafana_user",
		"grafana_users",
		"grafana_cloud_access_policies",
		"grafana_cloud_ips",
		"grafana_cloud_organization",
		"grafana_cloud_stack",
		"grafana_cloud_private_data_source_connect_networks",
		"grafana_cloud_provider_aws_account",
		"grafana_cloud_provider_aws_cloudwatch_scrape_job",
		"grafana_cloud_provider_aws_cloudwatch_scrape_jobs",
		"grafana_cloud_provider_azure_credential",
		"grafana_connections_metrics_endpoint_scrape_job",
		"grafana_fleet_management_collector",
		"grafana_fleet_management_collectors",
		"grafana_frontend_o11y_app",
		"grafana_k6_load_test",
		"grafana_k6_load_tests",
		"grafana_k6_project",
		"grafana_k6_project_limits",
		"grafana_k6_projects",
		"grafana_oncall_escalation_chain",
		"grafana_oncall_integration",
		"grafana_oncall_label",
		"grafana_oncall_outgoing_webhook",
		"grafana_oncall_schedule",
		"grafana_oncall_slack_channel",
		"grafana_oncall_team",
		"grafana_oncall_user",
		"grafana_oncall_user_group",
		"grafana_oncall_users",
		"grafana_slos",
		"grafana_synthetic_monitoring_probe",
		"grafana_synthetic_monitoring_probes",
	}

	// Add resources with (resource) suffix
	for _, resource := range resources {
		allItems = append(allItems, fmt.Sprintf("%s (resource)", resource))
	}

	// Add data sources with (data source) suffix
	for _, dataSource := range dataSources {
		allItems = append(allItems, fmt.Sprintf("%s (data source)", dataSource))
	}

	return allItems
}

func generateIssueTemplate(resources []string) error {
	templatePath := ".github/ISSUE_TEMPLATE/3-bug-report-enhanced.yml"

	// Ensure directory exists
	err := os.MkdirAll(filepath.Dir(templatePath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create template directory: %w", err)
	}

	file, err := os.Create(templatePath)
	if err != nil {
		return fmt.Errorf("failed to create bug report template: %w", err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	// Template header
	template := `name: Bug Report (Enhanced)
description: File a bug report with resource dropdown
type: "bug"
projects: ["grafana/513"] # Platform Monitoring
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
        💡 Tip: Click the dropdown and start typing to quickly find items (e.g., type "dash" for dashboard resources)
      multiple: true
      options:`

	fmt.Fprint(writer, template)

	// Add all resources as options
	for _, resource := range resources {
		fmt.Fprintf(writer, "\n        - %s", resource)
	}

	// Write the rest matching the existing template structure
	footer := `
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
      placeholder: |
        What should have happened?
  - type: textarea
    id: actual_behavior
    attributes:
      label: Actual Behavior
      placeholder: |
        What actually happened?
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
        Are there anything atypical about your accounts that we should know? For example: Running in EC2 Classic? Custom version of OpenStack? Tight ACLs?
  - type: textarea
    id: references
    attributes:
      label: References
      placeholder: |
        - GH-1234`

	fmt.Fprint(writer, footer)
	return nil
}
