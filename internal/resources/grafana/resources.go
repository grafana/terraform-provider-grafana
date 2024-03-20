package grafana

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func grafanaClientResourceValidation(d *schema.ResourceData, m interface{}) error {
	if m.(*common.Client).GrafanaOAPI == nil {
		return fmt.Errorf("the Grafana client is required for this resource. Set the auth and url provider attributes")
	}
	return nil
}

func grafanaOrgIDResourceValidation(d *schema.ResourceData, m interface{}) error {
	orgID, ok := d.GetOk("org_id")
	orgIDStr, orgIDOk := orgID.(string)
	if ok && orgIDOk && orgIDStr != "" && orgIDStr != "0" && m.(*common.Client).GrafanaAPIConfig.APIKey != "" {
		return fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
	}
	return nil
}

func addValidation(resources map[string]*schema.Resource) map[string]*schema.Resource {
	for _, r := range resources {
		createFn := r.CreateContext
		readFn := r.ReadContext
		updateFn := r.UpdateContext
		deleteFn := r.DeleteContext

		r.ReadContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
			if err := grafanaClientResourceValidation(d, m); err != nil {
				return diag.FromErr(err)
			}
			return readFn(ctx, d, m)
		}
		if updateFn != nil {
			r.UpdateContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := grafanaClientResourceValidation(d, m); err != nil {
					return diag.FromErr(err)
				}
				return updateFn(ctx, d, m)
			}
		}
		if deleteFn != nil {
			r.DeleteContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := grafanaClientResourceValidation(d, m); err != nil {
					return diag.FromErr(err)
				}
				return deleteFn(ctx, d, m)
			}
		}
		if createFn != nil {
			r.CreateContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := grafanaClientResourceValidation(d, m); err != nil {
					return diag.FromErr(err)
				}
				// Only check `org_id` on create. Some resources will have it set by the API on reads, even in a token (org-scoped) context
				if err := grafanaOrgIDResourceValidation(d, m); err != nil {
					return diag.FromErr(err)
				}
				return createFn(ctx, d, m)
			}
		}
	}

	return resources
}

var DatasourcesMap = addValidation(map[string]*schema.Resource{
	"grafana_dashboard":                datasourceDashboard(),
	"grafana_dashboards":               datasourceDashboards(),
	"grafana_data_source":              datasourceDatasource(),
	"grafana_folder":                   datasourceFolder(),
	"grafana_folders":                  datasourceFolders(),
	"grafana_library_panel":            datasourceLibraryPanel(),
	"grafana_user":                     datasourceUser(),
	"grafana_users":                    datasourceUsers(),
	"grafana_role":                     datasourceRole(),
	"grafana_service_account":          datasourceServiceAccount(),
	"grafana_team":                     datasourceTeam(),
	"grafana_organization":             datasourceOrganization(),
	"grafana_organization_preferences": datasourceOrganizationPreferences(),
})

var ResourcesMap = addValidation(map[string]*schema.Resource{
	"grafana_annotation":                 resourceAnnotation(),
	"grafana_api_key":                    resourceAPIKey(),
	"grafana_contact_point":              resourceContactPoint(),
	"grafana_dashboard":                  resourceDashboard(),
	"grafana_dashboard_public":           resourcePublicDashboard(),
	"grafana_dashboard_permission":       resourceDashboardPermission(),
	"grafana_data_source":                resourceDataSource(),
	"grafana_data_source_permission":     resourceDatasourcePermission(),
	"grafana_folder":                     resourceFolder(),
	"grafana_folder_permission":          resourceFolderPermission(),
	"grafana_library_panel":              resourceLibraryPanel(),
	"grafana_message_template":           resourceMessageTemplate(),
	"grafana_mute_timing":                resourceMuteTiming(),
	"grafana_notification_policy":        resourceNotificationPolicy(),
	"grafana_organization":               resourceOrganization(),
	"grafana_organization_preferences":   resourceOrganizationPreferences(),
	"grafana_playlist":                   resourcePlaylist(),
	"grafana_report":                     resourceReport(),
	"grafana_role":                       resourceRole(),
	"grafana_role_assignment":            resourceRoleAssignment(),
	"grafana_rule_group":                 resourceRuleGroup(),
	"grafana_team":                       resourceTeam(),
	"grafana_team_external_group":        resourceTeamExternalGroup(),
	"grafana_service_account_token":      resourceServiceAccountToken(),
	"grafana_service_account":            resourceServiceAccount(),
	"grafana_service_account_permission": resourceServiceAccountPermission(),
	"grafana_sso_settings":               resourceSSOSettings(),
	"grafana_user":                       resourceUser(),
})
