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
	"grafana_agento11y_evaluation_rule.evaluator_ids=grafana_agento11y_evaluator.evaluator_id",
	"grafana_agento11y_rule_action.collection_ids=grafana_agento11y_collection.id",
	"grafana_agento11y_rule_action.rule_id=grafana_agento11y_evaluation_rule.rule_id",
	"grafana_annotation.dashboard_uid=grafana_dashboard.uid",
	"grafana_annotation.org_id=grafana_organization.id",
	"grafana_apps_dashboard_dashboard_v1alpha1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_dashboard_dashboard_v1beta1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_dashboard_dashboard_v2beta1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_playlist_playlist_v0alpha1.spec.value=grafana_apps_dashboard_dashboard_v1alpha1.metadata.uid",
	"grafana_apps_provisioning_repository_v0alpha1.metadata.org_id=grafana_organization.org_id",
	"grafana_apps_provisioning_repository_v0alpha1.spec.connection.name=grafana_apps_provisioning_connection_v0alpha1.metadata.uid",
	"grafana_apps_rules_alertrule_v0alpha1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_alertrule_v0alpha1.spec.datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_alertrule_v0alpha1.spec.notification_settings.simplified_routing.contact_point=grafana_contact_point.name",
	"grafana_apps_rules_alertrule_v0alpha1.spec.uid=grafana_data_source.uid",
	"grafana_apps_rules_alertrule_v1beta1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_alertrule_v1beta1.spec.datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_alertrule_v1beta1.spec.notification_settings.simplified_routing.contact_point=grafana_contact_point.name",
	"grafana_apps_rules_alertrule_v1beta1.spec.uid=grafana_data_source.uid",
	"grafana_apps_rules_recordingrule_v0alpha1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_recordingrule_v0alpha1.spec.datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_recordingrule_v0alpha1.spec.target_datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_recordingrule_v1beta1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_recordingrule_v1beta1.spec.datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_recordingrule_v1beta1.spec.target_datasource_uid=grafana_data_source.uid",
	"grafana_apps_rules_rulesequence_v0alpha1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_rulesequence_v0alpha1.spec.name=grafana_apps_rules_alertrule_v0alpha1.metadata",
	"grafana_apps_rules_rulesequence_v0alpha1.spec.name=grafana_apps_rules_alertrule_v0alpha1.metadata.uid",
	"grafana_apps_rules_rulesequence_v0alpha1.spec.name=grafana_apps_rules_recordingrule_v0alpha1.metadata",
	"grafana_apps_rules_rulesequence_v0alpha1.spec.name=grafana_apps_rules_recordingrule_v0alpha1.metadata.uid",
	"grafana_apps_rules_rulesequence_v1beta1.metadata.folder_uid=grafana_folder.uid",
	"grafana_apps_rules_rulesequence_v1beta1.spec.name=grafana_apps_rules_alertrule_v1beta1.metadata",
	"grafana_apps_rules_rulesequence_v1beta1.spec.name=grafana_apps_rules_recordingrule_v1beta1.metadata",
	"grafana_apps_secret_keeper_activation_v1beta1.metadata.org_id=grafana_organization.org_id",
	"grafana_apps_secret_keeper_activation_v1beta1.metadata.uid=grafana_apps_secret_keeper_v1beta1.metadata.uid",
	"grafana_apps_secret_keeper_v1beta1.metadata.org_id=grafana_organization.org_id",
	"grafana_apps_servicemodel_component_v1alpha1.spec.depends_on_refs.name=grafana_apps_servicemodel_component_v1alpha1.metadata.uid",
	"grafana_apps_servicemodel_component_v1alpha1.spec.owner_ref.name=grafana_team.team_uid",
	"grafana_asserts_stack.cloud_access_policy_token=grafana_cloud_access_policy_token.token",
	"grafana_asserts_stack.grafana_token=grafana_cloud_stack_service_account_token.key",
	"grafana_cloud_access_policy.realm.identifier=grafana_cloud_stack.id",
	"grafana_cloud_access_policy.region=grafana_cloud_stack.region_slug",
	"grafana_cloud_access_policy_rotating_token.access_policy_id=grafana_cloud_access_policy.policy_id",
	"grafana_cloud_access_policy_token.access_policy_id=grafana_cloud_access_policy.policy_id",
	"grafana_cloud_access_policy_token.region=grafana_cloud_access_policy.region",
	"grafana_cloud_plugin_installation.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_private_data_source_connect_network_token.pdc_network_id=grafana_cloud_private_data_source_connect_network.pdc_network_id",
	"grafana_cloud_private_data_source_connect_network_token.region=grafana_cloud_private_data_source_connect_network.region",
	"grafana_cloud_provider_aws_cloudwatch_scrape_job.aws_account_resource_id=grafana_cloud_provider_aws_account.resource_id",
	"grafana_cloud_provider_aws_resource_metadata_scrape_job.aws_account_resource_id=grafana_cloud_provider_aws_account.resource_id",
	"grafana_cloud_stack_service_account.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_stack_service_account_rotating_token.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_cloud_stack_service_account_rotating_token.stack_slug=grafana_cloud_stack.slug",
	"grafana_cloud_stack_service_account_token.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_cloud_stack_service_account_token.stack_slug=grafana_cloud_stack.slug",
	"grafana_contact_point.org_id=grafana_organization.id",
	"grafana_dashboard.folder=grafana_folder.id",
	"grafana_dashboard.folder=grafana_folder.uid",
	"grafana_dashboard.name=grafana_library_panel.name",
	"grafana_dashboard.org_id=grafana_organization.id",
	"grafana_dashboard.org_id=grafana_organization.org_id",
	"grafana_dashboard.uid=grafana_library_panel.uid",
	"grafana_dashboard_permission.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_permission.permissions.team_id=grafana_team.id",
	"grafana_dashboard_permission.permissions.user_id=grafana_user.id",
	"grafana_dashboard_permission_item.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_permission_item.team=grafana_team.id",
	"grafana_dashboard_permission_item.user=grafana_service_account.id",
	"grafana_dashboard_permission_item.user=grafana_user.id",
	"grafana_dashboard_public.dashboard_uid=grafana_dashboard.uid",
	"grafana_dashboard_public.org_id=grafana_organization.org_id",
	"grafana_data_source.datasourceUid=grafana_data_source.uid",
	"grafana_data_source.org_id=grafana_organization.id",
	"grafana_data_source_cache_config.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_config.datasourceUid=grafana_data_source.uid",
	"grafana_data_source_config.uid=grafana_data_source.uid",
	"grafana_data_source_config_lbac_rules.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission.permissions.team_id=grafana_team.id",
	"grafana_data_source_permission.permissions.user_id=grafana_service_account.id",
	"grafana_data_source_permission.permissions.user_id=grafana_user.id",
	"grafana_data_source_permission_item.datasource_uid=grafana_data_source.uid",
	"grafana_data_source_permission_item.team=grafana_team.id",
	"grafana_data_source_permission_item.user=grafana_service_account.id",
	"grafana_data_source_permission_item.user=grafana_user.id",
	"grafana_folder.org_id=grafana_organization.id",
	"grafana_folder.org_id=grafana_organization.org_id",
	"grafana_folder.parent_folder_uid=grafana_folder.uid",
	"grafana_folder_permission.folder_uid=grafana_folder.uid",
	"grafana_folder_permission.permissions.team_id=grafana_team.id",
	"grafana_folder_permission.permissions.user_id=grafana_service_account.id",
	"grafana_folder_permission.permissions.user_id=grafana_user.id",
	"grafana_folder_permission_item.folder_uid=grafana_folder.uid",
	"grafana_folder_permission_item.team=grafana_team.id",
	"grafana_folder_permission_item.user=grafana_service_account.id",
	"grafana_folder_permission_item.user=grafana_user.id",
	"grafana_k6_installation.grafana_sa_token=grafana_cloud_stack_service_account_token.key",
	"grafana_k6_installation.stack_id=grafana_cloud_stack.id",
	"grafana_k6_load_test.project_id=grafana_k6_project.id",
	"grafana_k6_project_allowed_load_zones.project_id=grafana_k6_project.id",
	"grafana_k6_project_limits.project_id=grafana_k6_project.id",
	"grafana_k6_schedule.load_test_id=grafana_k6_load_test.id",
	"grafana_library_panel.folder_uid=grafana_folder.uid",
	"grafana_library_panel.org_id=grafana_organization.id",
	"grafana_machine_learning_alert.job_id=grafana_machine_learning_job.id",
	"grafana_machine_learning_alert.outlier_id=grafana_machine_learning_outlier_detector.id",
	"grafana_machine_learning_job.datasource_uid=grafana_data_source.uid",
	"grafana_message_template.org_id=grafana_organization.id",
	"grafana_mute_timing.org_id=grafana_organization.id",
	"grafana_notification_policy.contact_point=grafana_contact_point.name",
	"grafana_notification_policy.org_id=grafana_organization.id",
	"grafana_notification_policy.policy.active_timings=grafana_mute_timing.name",
	"grafana_notification_policy.policy.contact_point=grafana_contact_point.name",
	"grafana_notification_policy.policy.mute_timings=grafana_mute_timing.name",
	"grafana_notification_policy.policy.policy.contact_point=grafana_contact_point.name",
	"grafana_oncall_escalation.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_escalation.notify_on_call_from_schedule=grafana_oncall_schedule.id",
	"grafana_oncall_integration.default_route.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_route.escalation_chain_id=grafana_oncall_escalation_chain.id",
	"grafana_oncall_route.integration_id=grafana_oncall_integration.id",
	"grafana_organization_preferences.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_organization_preferences.org_id=grafana_organization.id",
	"grafana_playlist.org_id=grafana_organization.id",
	"grafana_report.dashboards.uid=grafana_dashboard.uid",
	"grafana_report.org_id=grafana_organization.id",
	"grafana_role.org_id=grafana_organization.id",
	"grafana_role_assignment.org_id=grafana_organization.id",
	"grafana_role_assignment.role_uid=grafana_role.uid",
	"grafana_role_assignment.service_accounts=grafana_cloud_stack_service_account.id",
	"grafana_role_assignment.service_accounts=grafana_service_account.id",
	"grafana_role_assignment.teams=grafana_team.id",
	"grafana_role_assignment.users=grafana_user.id",
	"grafana_role_assignment_item.role_uid=grafana_role.uid",
	"grafana_role_assignment_item.service_account_id=grafana_service_account.id",
	"grafana_role_assignment_item.team_id=grafana_team.id",
	"grafana_role_assignment_item.user_id=grafana_user.id",
	"grafana_rule_group.folder_uid=grafana_folder.uid",
	"grafana_rule_group.org_id=grafana_organization.id",
	"grafana_rule_group.rule.notification_settings.contact_point=grafana_contact_point.name",
	"grafana_service_account.org_id=grafana_organization.id",
	"grafana_service_account_permission.org_id=grafana_organization.id",
	"grafana_service_account_permission.permissions.team_id=grafana_team.id",
	"grafana_service_account_permission.permissions.user_id=grafana_user.id",
	"grafana_service_account_permission.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_service_account_permission.service_account_id=grafana_service_account.id",
	"grafana_service_account_permission_item.org_id=grafana_organization.id",
	"grafana_service_account_permission_item.service_account_id=grafana_cloud_stack_service_account.id",
	"grafana_service_account_permission_item.service_account_id=grafana_service_account.id",
	"grafana_service_account_permission_item.team=grafana_team.id",
	"grafana_service_account_permission_item.user=grafana_user.id",
	"grafana_service_account_rotating_token.service_account_id=grafana_service_account.id",
	"grafana_service_account_token.service_account_id=grafana_service_account.id",
	"grafana_slo.folder_uid=grafana_folder.uid",
	"grafana_slo.query.freeform.source_datasource_uid=grafana_data_source.uid",
	"grafana_synthetic_monitoring_check_alerts.check_id=grafana_synthetic_monitoring_check.id",
	"grafana_synthetic_monitoring_installation.metrics_publisher_key=grafana_cloud_access_policy_token.token",
	"grafana_synthetic_monitoring_installation.stack_id=grafana_cloud_stack.id",
	"grafana_team.org_id=grafana_organization.id",
	"grafana_team_external_group.team_id=grafana_team.id",
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

			replaceReferencesInBody(block.Body(), block.Labels()[0], "", blockResource.AttributeValues, knownReferences, plannedResources)
		}
		return nil
	})
}

// replaceReferencesInBody replaces the attributes of body (at attrPrefix within the resource) that match a known reference,
// then recurses into nested blocks so that references like `rule.notification_settings.contact_point` are found too.
func replaceReferencesInBody(body *hclwrite.Body, resourceType, attrPrefix string, values map[string]any, knownReferences []string, plannedResources []*tfjson.StateResource) {
	for attrName := range body.Attributes() {
		attrPath := attrPrefix + attrName
		attrValue := values[attrName]
		attrReplaced := false

		// Check the field name. If it has a possible reference, we have to search for it in the resources
		for _, ref := range knownReferences {
			if attrReplaced {
				break
			}

			refFrom := strings.Split(ref, "=")[0]
			refTo := strings.Split(ref, "=")[1]
			hasPossibleReference := refFrom == fmt.Sprintf("%s.%s", resourceType, attrPath) || refFrom == "*."+attrPath
			if !hasPossibleReference {
				continue
			}

			refToResource, refToAttr, _ := strings.Cut(refTo, ".")
			refToAttrPath := strings.Split(refToAttr, ".")

			for _, plannedResource := range plannedResources {
				if plannedResource.Type != refToResource {
					continue
				}

				valueFromRef := plannedValue(plannedResource.AttributeValues, refToAttrPath)
				// If the value from the first block matches the value from the second block, we have a reference
				if attrValue == valueFromRef {
					// Replace the value with the reference
					body.SetAttributeTraversal(attrName, traversal(plannedResource.Type, append([]string{plannedResource.Name}, refToAttrPath...)...))
					attrReplaced = true
					break
				}
			}
		}
	}

	// Nested blocks are planned as a list of objects (one per block, in order), or as a single object.
	blockIndex := map[string]int{}
	for _, block := range body.Blocks() {
		blockType := block.Type()
		i := blockIndex[blockType]
		blockIndex[blockType]++

		var blockValues map[string]any
		switch v := values[blockType].(type) {
		case map[string]any:
			blockValues = v
		case []any:
			if i < len(v) {
				blockValues, _ = v[i].(map[string]any)
			}
		}
		if blockValues == nil {
			continue
		}
		replaceReferencesInBody(block.Body(), resourceType, attrPrefix+blockType+".", blockValues, knownReferences, plannedResources)
	}
}

// plannedValue returns the value at path in a planned resource's attributes, stepping into nested blocks.
func plannedValue(values map[string]any, path []string) any {
	var v any = values
	for _, key := range path {
		if list, ok := v.([]any); ok && len(list) == 1 {
			v = list[0]
		}
		m, ok := v.(map[string]any)
		if !ok {
			return nil
		}
		v = m[key]
	}
	return v
}
