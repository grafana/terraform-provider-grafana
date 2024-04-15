package grafana

import (
	"context"
	"fmt"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func grafanaClientResourceValidation(d *schema.ResourceData, m interface{}) error {
	if m.(*common.Client).GrafanaAPI == nil {
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

func addValidationToSchema(r *schema.Resource) {
	if r == nil {
		return
	}
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

func addValidationToMap(resources map[string]*schema.Resource) map[string]*schema.Resource {
	for _, r := range resources {
		addValidationToSchema(r)
	}
	return resources
}

func addValidationToList(resources []*common.Resource) []*common.Resource {
	for _, r := range resources {
		addValidationToSchema(r.Schema)
	}
	return resources
}

var DatasourcesMap = addValidationToMap(map[string]*schema.Resource{
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

var Resources = addValidationToList([]*common.Resource{
	makeResourceFolderPermissionItem(),
	makeResourceDashboardPermissionItem(),
	makeResourceDatasourcePermissionItem(),
	makeResourceRoleAssignmentItem(),
	makeResourceServiceAccountPermissionItem(),
	resourceAnnotation(),
	resourceAPIKey(),
	resourceContactPoint(),
	resourceDashboard(),
	resourcePublicDashboard(),
	resourceDashboardPermission(),
	resourceDataSource(),
	resourceDataSourceConfig(),
	resourceDatasourcePermission(),
	resourceFolder(),
	resourceFolderPermission(),
	resourceLibraryPanel(),
	resourceMessageTemplate(),
	resourceMuteTiming(),
	resourceNotificationPolicy(),
	resourceOrganization(),
	resourceOrganizationPreferences(),
	resourcePlaylist(),
	resourceReport(),
	resourceRole(),
	resourceRoleAssignment(),
	resourceRuleGroup(),
	resourceTeam(),
	resourceTeamExternalGroup(),
	resourceServiceAccountToken(),
	resourceServiceAccount(),
	resourceServiceAccountPermission(),
	resourceSSOSettings(),
	resourceUser(),
})
