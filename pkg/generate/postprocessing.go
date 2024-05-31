package generate

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
	"github.com/zclconf/go-cty/cty"
)

// knownReferences is a map of all resource fields that can be referenced from another resource.
// For example, the `folder` field of a `grafana_dashboard` resource can be a `grafana_folder` reference.
//
//go:generate go run ./genreferences --file=$GOFILE --walk-dir=../..
var knownReferences = []string{
	"grafana_annotation.dashboard_uid=grafana_dashboard.uid",
	"grafana_annotation.org_id=grafana_organization.id",
	"grafana_api_key.auth=grafana_api_key.key",
	"grafana_cloud_access_policy.identifier=grafana_cloud_stack.id",
	"grafana_cloud_access_policy_token.access_policy_id=grafana_cloud_access_policy.policy_id",
	"grafana_cloud_plugin_installation.stack_slug=grafana_cloud_stack.slug",
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
	"grafana_machine_learning_job.datasource_uid=grafana_data_source.uid",
	"grafana_message_template.org_id=grafana_organization.id",
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
	"grafana_report.dashboard_id=grafana_dashboard.dashboard_id",
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
	"grafana_service_account_token.auth=grafana_service_account_token.key",
	"grafana_service_account_token.service_account_id=grafana_service_account.id",
	"grafana_slo.folder_uid=grafana_folder.uid",
	"grafana_synthetic_monitoring_installation.logs_instance_id=grafana_cloud_stack.logs_user_id",
	"grafana_synthetic_monitoring_installation.metrics_instance_id=grafana_cloud_stack.prometheus_user_id",
	"grafana_synthetic_monitoring_installation.metrics_publisher_key=grafana_cloud_access_policy_token.token",
	"grafana_synthetic_monitoring_installation.metrics_publisher_key=grafana_cloud_api_key.key",
	"grafana_synthetic_monitoring_installation.sm_access_token=grafana_synthetic_monitoring_installation.sm_access_token",
	"grafana_synthetic_monitoring_installation.sm_url=grafana_synthetic_monitoring_installation.stack_sm_api_url",
	"grafana_synthetic_monitoring_installation.stack_id=grafana_cloud_stack.id",
	"grafana_team.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_team.org_id=grafana_organization.id",
	"grafana_team_external_group.team_id=grafana_team.id",
	"grafana_team_preferences.home_dashboard_uid=grafana_dashboard.uid",
	"grafana_team_preferences.team_id=grafana_team.id",
}

// TODO: Also find references from the state (computed fields, like ID)
func replaceReferences(fpath string, extraKnownReferences []string) error {
	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false

	knownReferences := knownReferences
	knownReferences = append(knownReferences, extraKnownReferences...)
	// Find all resources. This map will be used to search for references
	resourcesBlocks := map[string]*hclwrite.Block{}
	for _, block := range file.Body().Blocks() {
		if block.Type() == "resource" {
			resourcesBlocks[block.Labels()[0]+"."+block.Labels()[1]] = block
		}
	}

	for _, block := range file.Body().Blocks() {
		for attrName, attr := range block.Body().Attributes() {
			attrValue := string(attr.Expr().BuildTokens(nil).Bytes())
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

				for possibleResourceRefName, possibleResourceRef := range resourcesBlocks {
					if strings.HasPrefix(possibleResourceRefName, refToResource+".") {
						valueFromRef := ""
						if possibleResourceRef.Body().GetAttribute(refToAttr) != nil {
							valueFromRef = string(possibleResourceRef.Body().GetAttribute(refToAttr).Expr().BuildTokens(nil).Bytes())
						}
						// If the value from the first block matches the value from the second block, we have a reference
						if attrValue == valueFromRef {
							// Replace the value with the reference
							block.Body().SetAttributeTraversal(attrName, traversal(possibleResourceRefName, refToAttr))
							hasChanges = true
							attrReplaced = true
							break
						}
					}
				}
			}
		}
	}

	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}

	return nil
}

func stripDefaults(fpath string, extraFieldsToRemove map[string]any) error {
	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	for _, block := range file.Body().Blocks() {
		if s := stripDefaultsFromBlock(block, extraFieldsToRemove); s {
			hasChanges = true
		}
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func wrapJSONFieldsInFunction(fpath string) error {
	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	// Find json attributes and use jsonencode
	for _, block := range file.Body().Blocks() {
		for key, attr := range block.Body().Attributes() {
			asMap, err := attributeToMap(attr)
			if err != nil || asMap == nil {
				continue
			}
			tokens := hclwrite.TokensForValue(HCL2ValueFromConfigValue(asMap))
			block.Body().SetAttributeRaw(key, hclwrite.TokensForFunctionCall("jsonencode", tokens))
			hasChanges = true
		}
	}

	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func abstractDashboards(fpath string) error {
	fDir := filepath.Dir(fpath)
	outPath := filepath.Join(fDir, "files")

	file, err := readHCLFile(fpath)
	if err != nil {
		return err
	}

	hasChanges := false
	dashboardJsons := map[string][]byte{}
	for _, block := range file.Body().Blocks() {
		labels := block.Labels()
		if len(labels) == 0 || labels[0] != "grafana_dashboard" {
			continue
		}

		dashboard, err := attributeToJSON(block.Body().GetAttribute("config_json"))
		if err != nil {
			return err
		}

		if dashboard == nil {
			continue
		}

		writeTo := filepath.Join(outPath, fmt.Sprintf("%s.json", block.Labels()[1]))

		// Replace $${ with ${ in the json. No need to escape in the json file
		dashboard = []byte(strings.ReplaceAll(string(dashboard), "$${", "${"))
		dashboardJsons[writeTo] = dashboard

		// Hacky relative path with interpolation
		relativePath := strings.ReplaceAll(writeTo, fDir, "")
		pathWithInterpolation := hclwrite.Tokens{
			{Type: hclsyntax.TokenOQuote, Bytes: []byte(`"`)},
			{Type: hclsyntax.TokenTemplateInterp, Bytes: []byte(`${`)},
			{Type: hclsyntax.TokenIdent, Bytes: []byte(`path.module`)},
			{Type: hclsyntax.TokenTemplateSeqEnd, Bytes: []byte(`}`)},
			{Type: hclsyntax.TokenQuotedLit, Bytes: []byte(relativePath)},
			{Type: hclsyntax.TokenCQuote, Bytes: []byte(`"`)},
		}

		block.Body().SetAttributeRaw(
			"config_json",
			hclwrite.TokensForFunctionCall("file", pathWithInterpolation),
		)

		hasChanges = true
	}
	if hasChanges {
		log.Printf("Updating file: %s\n", fpath)
		os.Mkdir(outPath, 0755)
		for writeTo, dashboard := range dashboardJsons {
			err := os.WriteFile(writeTo, dashboard, 0600)
			if err != nil {
				panic(err)
			}
		}
		return os.WriteFile(fpath, file.Bytes(), 0600)
	}
	return nil
}

func attributeToMap(attr *hclwrite.Attribute) (map[string]interface{}, error) {
	var err error

	// Convert jsonencode to raw json
	s := strings.TrimPrefix(string(attr.Expr().BuildTokens(nil).Bytes()), " ")

	if strings.HasPrefix(s, "jsonencode(") {
		return nil, nil // Figure out how to handle those
	}

	if !strings.HasPrefix(s, "\"") {
		// if expr is not a string, assume it's already converted, return (idempotency
		return nil, nil
	}
	s, err = strconv.Unquote(s)
	if err != nil {
		return nil, err
	}
	s = strings.ReplaceAll(s, "$${", "${") // These are escaped interpolations

	var dashboardMap map[string]interface{}
	err = json.Unmarshal([]byte(s), &dashboardMap)
	if err != nil {
		return nil, err
	}

	return dashboardMap, nil
}

func attributeToJSON(attr *hclwrite.Attribute) ([]byte, error) {
	jsonMap, err := attributeToMap(attr)
	if err != nil || jsonMap == nil {
		return nil, err
	}

	jsonMarshalled, err := json.MarshalIndent(jsonMap, "", "\t")
	if err != nil {
		return nil, err
	}

	return jsonMarshalled, nil
}

func readHCLFile(fpath string) (*hclwrite.File, error) {
	src, err := os.ReadFile(fpath)
	if err != nil {
		return nil, err
	}

	file, diags := hclwrite.ParseConfig(src, fpath, hcl.Pos{Line: 1, Column: 1})
	if diags.HasErrors() {
		return nil, errors.New(diags.Error())
	}

	return file, nil
}

func stripDefaultsFromBlock(block *hclwrite.Block, extraFieldsToRemove map[string]any) bool {
	hasChanges := false
	for _, innblock := range block.Body().Blocks() {
		if s := stripDefaultsFromBlock(innblock, extraFieldsToRemove); s {
			hasChanges = true
		}
		if len(innblock.Body().Attributes()) == 0 && len(innblock.Body().Blocks()) == 0 {
			if rm := block.Body().RemoveBlock(innblock); rm {
				hasChanges = true
			}
		}
	}
	for name, attribute := range block.Body().Attributes() {
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " null" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " {}" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		if string(attribute.Expr().BuildTokens(nil).Bytes()) == " []" {
			if rm := block.Body().RemoveAttribute(name); rm != nil {
				hasChanges = true
			}
		}
		for key, valueToRemove := range extraFieldsToRemove {
			if name == key {
				toRemove := false
				fieldValue := strings.TrimSpace(string(attribute.Expr().BuildTokens(nil).Bytes()))
				fieldValue, err := extractJSONEncode(fieldValue)
				if err != nil {
					continue
				}

				if v, ok := valueToRemove.(bool); ok && v {
					toRemove = true
				} else if v, ok := valueToRemove.(string); ok && v == fieldValue {
					toRemove = true
				}
				if toRemove {
					if rm := block.Body().RemoveAttribute(name); rm != nil {
						hasChanges = true
					}
				}
			}
		}
	}
	return hasChanges
}

// BELOW IS FROM https://github.com/hashicorp/terraform/blob/main/internal/configs/hcl2shim/values.go

// UnknownVariableValue is a sentinel value that can be used
// to denote that the value of a variable is unknown at this time.
// RawConfig uses this information to build up data about
// unknown keys.
const UnknownVariableValue = "74D93920-ED26-11E3-AC10-0800200C9A66"

// HCL2ValueFromConfigValue is the opposite of configValueFromHCL2: it takes
// a value as would be returned from the old interpolator and turns it into
// a cty.Value so it can be used within, for example, an HCL2 EvalContext.
func HCL2ValueFromConfigValue(v interface{}) cty.Value {
	if v == nil {
		return cty.NullVal(cty.DynamicPseudoType)
	}
	if v == UnknownVariableValue {
		return cty.DynamicVal
	}

	switch tv := v.(type) {
	case bool:
		return cty.BoolVal(tv)
	case string:
		return cty.StringVal(tv)
	case int:
		return cty.NumberIntVal(int64(tv))
	case float64:
		return cty.NumberFloatVal(tv)
	case []interface{}:
		vals := make([]cty.Value, len(tv))
		for i, ev := range tv {
			vals[i] = HCL2ValueFromConfigValue(ev)
		}
		return cty.TupleVal(vals)
	case map[string]interface{}:
		vals := map[string]cty.Value{}
		for k, ev := range tv {
			vals[k] = HCL2ValueFromConfigValue(ev)
		}
		return cty.ObjectVal(vals)
	default:
		// HCL/HIL should never generate anything that isn't caught by
		// the above, so if we get here something has gone very wrong.
		panic(fmt.Errorf("can't convert %#v to cty.Value", v))
	}
}

func extractJSONEncode(value string) (string, error) {
	if !strings.HasPrefix(value, "jsonencode(") {
		return "", nil
	}
	value = strings.TrimPrefix(value, "jsonencode(")
	value = strings.TrimSuffix(value, ")")

	b, err := json.MarshalIndent(value, "", "  ")
	return string(b), err
}
