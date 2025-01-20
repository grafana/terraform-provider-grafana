package postprocessing

import (
	"fmt"
	"strings"

	"github.com/hashicorp/hcl/v2/hclwrite"
	tfjson "github.com/hashicorp/terraform-json"
)

// knownReferences is a map of all resource fields that can be referenced from another resource.
// For example, the `folder` field of a `grafana_dashboard` resource can be a `grafana_folder` reference.
//
//go:generate go run ./genreferences --file=$GOFILE --walk-dir=../../..
var knownReferences = []string{
	"grafana_annotation.dashboard_uid=grafana_dashboard.uid",
	"grafana_annotation.org_id=grafana_organization.id",
	"grafana_cloud_access_policy.identifier=grafana_cloud_stack.id",
	"grafana_cloud_access_policy_token.access_policy_id=grafana_cloud_access_policy.policy_id",
	"grafana_cloud_plugin_installation.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_private_data_source_connect_network.stack_identifier=grafana_cloud_stack.id",
	"grafana_cloud_private_data_source_connect_network_token.pdc_network_id=grafana_cloud_private_data_source_connect_network.pdc_network_id",
	"grafana_cloud_private_data_source_connect_network_token.region=grafana_cloud_private_data_source_connect_network.region",
	"grafana_cloud_provider_aws_cloudwatch_scrape_job.aws_account_resource_id=grafana_cloud_provider_aws_account.resource_id",
	"grafana_cloud_stack_service_account.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_stack_service_account_token.auth=grafana_cloud_stack_service_account_token.key",
	"grafana_cloud_stack_service_account_token.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_cloud_stack_service_account_token.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_stack_service_account_token.url=grafana_cloud_stack.url",
	"grafana_contact_point.org_id=grafana_organization.id",
	"grafana_dashboard.folder=grafana_folder.id",
	"grafana_dashboard.folder=grafana_folder.uid",
	"grafana_dashboard.name=grafana_library_panel.name",
	"grafana_dashboard.org_id=grafana_organization.id",
	"grafana_dashboard.org_id=grafana_organization.org_id",
	"grafana_dashboard.uid=grafana_library_panel.uid",
	"grafana_dashboard_permission.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_permission.team_id=grafana_team.id",
	"grafana_dashboard_permission.user_id=grafana_user.id",
	"grafana_dashboard_permission_item.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_permission_item.team=grafana_team.id",
	"grafana_dashboard_permission_item.user=grafana_service_account.id",
	"grafana_dashboard_permission_item.user=grafana_user.id",
	"grafana_dashboard_public.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_public.org_id=grafana_organization.org_id",
	"grafana_data_source.datasourceUid=grafana_data_source.uid",
	"grafana_data_source.org_id=grafana_organization.id",
	"grafana_data_source_config.datasourceUid=grafana_data_source.uid",
	"grafana_data_source_config.uid=grafana_data_source.uid",
	"grafana_data_source_config_lbac_rules.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission.team_id=grafana_team.id",
	"grafana_data_source_permission.user_id=grafana_service_account.id",
	"grafana_data_source_permission.user_id=grafana_user.id",
	"grafana_data_source_permission_item.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission_item.team=grafana_team.id",
	"grafana_data_source_permission_item.user=grafana_service_account.id",
	"grafana_data_source_permission_item.user=grafana_user.id",
	"grafana_folder.org_id=grafana_organization.id",
	"grafana_folder.org_id=grafana_organization.org_id",
	"grafana_folder.parent_folder_uid=grafana_folder.uid",
	"grafana_folder_permission.folder_uid=grafana_folder.uid",
	"grafana_folder_permission.team_id=grafana_team.id",
	"grafana_folder_permission.user_id=grafana_service_account.id",
	"grafana_folder_permission.user_id=grafana_user.id",
	"grafana_folder_permission_item.folder_uid=grafana_folder.uid",
	"grafana_folder_permission_item.team=grafana_team.id",
	"grafana_folder_permission_item.user=grafana_service_account.id",
	"grafana_folder_permission_item.user=grafana_user.id",
	"grafana_library_panel.folder_uid=grafana_folder.uid",
	"grafana_library_panel.org_id=grafana_organization.id",
	"grafana_machine_learning_alert.job_id=grafana_machine_learning_job.id",
	"grafana_machine_learning_alert.outlier_id=grafana_machine_learning_outlier_detector.id",
	"grafana_machine_learning_job.datasource_uid=grafana_data_source.uid",
	"grafana_message_template.org_id=grafana_organization.id",
	"grafana_mute_timing.org_id=grafana_organization.id",
	"grafana_notification_policy.contact_point=grafana_contact_point.name",
	"grafana_notification_policy.mute_timings=grafana_mute_timing.name",
	"grafana_notification_policy.org_id=grafana_organization.id",
	"grafana_oncall_escalation.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_integration.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_route.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_route.integration_id=grafana_oncall_integration.id",
	"grafana_organization.org_id=grafana_organization.id",
	"grafana_organization_preferences.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_organization_preferences.org_id=grafana_organization.id",
	"grafana_playlist.org_id=grafana_organization.id",
	"grafana_report.org_id=grafana_organization.id",
	"grafana_report.uid=grafana_dashboard.uid",
	"grafana_role.org_id=grafana_organization.id",
	"grafana_role_assignment.auth=grafana_cloud_stack_service_account_token.key",
	"grafana_role_assignment.org_id=grafana_organization.id",
	"grafana_role_assignment.role_uid=grafana_role.uid",
	"grafana_role_assignment.service_accounts=grafana_cloud_stack_service_account.id",
	"grafana_role_assignment.service_accounts=grafana_service_account.id",
	"grafana_role_assignment.teams=grafana_team.id",
	"grafana_role_assignment.url=grafana_cloud_stack.url",
	"grafana_role_assignment.users=grafana_user.id",
	"grafana_role_assignment_item.role_uid=grafana_role.uid",
	"grafana_role_assignment_item.service_account_id=grafana_service_account.id",
	"grafana_role_assignment_item.team_id=grafana_team.id",
	"grafana_role_assignment_item.user_id=grafana_user.id",
	"grafana_rule_group.contact_point=grafana_contact_point.name",
	"grafana_rule_group.folder_uid=grafana_folder.uid",
	"grafana_rule_group.org_id=grafana_organization.id",
	"grafana_service_account.org_id=grafana_organization.id",
	"grafana_service_account.role_uid=grafana_role.uid",
	"grafana_service_account.service_account_id=grafana_service_account.id",
	"grafana_service_account.team_id=grafana_team.id",
	"grafana_service_account.user_id=grafana_user.id",
	"grafana_service_account_permission.org_id=grafana_organization.id",
	"grafana_service_account_permission.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_service_account_permission.service_account_id=grafana_service_account.id",
	"grafana_service_account_permission.team_id=grafana_team.id",
	"grafana_service_account_permission.user_id=grafana_user.id",
	"grafana_service_account_permission_item.auth=grafana_cloud_stack_service_account_token.key",
	"grafana_service_account_permission_item.org_id=grafana_organization.id",
	"grafana_service_account_permission_item.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_service_account_permission_item.service_account_id=grafana_service_account.id",
	"grafana_service_account_permission_item.team=grafana_team.id",
	"grafana_service_account_permission_item.url=grafana_cloud_stack.url",
	"grafana_service_account_permission_item.user=grafana_user.id",
	"grafana_service_account_token.service_account_id=grafana_service_account.id",
	"grafana_slo.folder_uid=grafana_folder.uid",
	"grafana_synthetic_monitoring_installation.metrics_publisher_key=grafana_cloud_access_policy_token.token",
	"grafana_synthetic_monitoring_installation.sm_access_token=grafana_synthetic_monitoring_installation.sm_access_token",
	"grafana_synthetic_monitoring_installation.sm_url=grafana_synthetic_monitoring_installation.stack_sm_api_url",
	"grafana_synthetic_monitoring_installation.stack_id=grafana_cloud_stack.id",
	"grafana_team.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_team.org_id=grafana_organization.id",
	"grafana_team_external_group.team_id=grafana_team.id",
	"grafana_team_preferences.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_team_preferences.team_id=grafana_team.id",
}

func ReplaceReferences(fpath string, plannedState *tfjson.Plan, extraKnownReferences []string) error {
	return postprocessFile(fpath, func(file *hclwrite.File) error {
		knownReferences := knownReferences
		knownReferences = append(knownReferences, extraKnownReferences...)

		plannedResources := plannedState.PlannedValues.RootModule.Resources

		for _, block := range file.Body().Blocks() {
			var blockResource *tfjson.StateResource
			for _, plannedResource := range plannedResources {
				if plannedResource.Type == block.Labels()[0] && plannedResource.Name == block.Labels()[1] {
					blockResource = plannedResource
					break
				}
			}
			if blockResource == nil {
				return fmt.Errorf("resource %s.%s not found in planned state", block.Labels()[0], block.Labels()[1])
			}

			for attrName := range block.Body().Attributes() {
				attrValue := blockResource.AttributeValues[attrName]
				attrReplaced := false

				// Check the field name. If it has a possible reference, we have to search for it in the resources
				for _, ref := range knownReferences {
					if attrReplaced {
						break
					}

					refFrom := strings.Split(ref, "=")[0]
					refTo := strings.Split(ref, "=")[1]
					hasPossibleReference := refFrom == fmt.Sprintf("%s.%s", block.Labels()[0], attrName) || (strings.HasPrefix(refFrom, "*.") && strings.HasSuffix(refFrom, fmt.Sprintf(".%s", attrName)))
					if !hasPossibleReference {
						continue
					}

					refToResource := strings.Split(refTo, ".")[0]
					refToAttr := strings.Split(refTo, ".")[1]

					for _, plannedResource := range plannedResources {
						if plannedResource.Type != refToResource {
							continue
						}

						valueFromRef := plannedResource.AttributeValues[refToAttr]
						// If the value from the first block matches the value from the second block, we have a reference
						if attrValue == valueFromRef {
							// Replace the value with the reference
							block.Body().SetAttributeTraversal(attrName, traversal(plannedResource.Type, plannedResource.Name, refToAttr))
							attrReplaced = true
							break
						}
					}
				}
			}
		}
		return nil
	})
}
