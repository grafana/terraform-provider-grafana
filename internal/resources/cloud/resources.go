package cloud

import (
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_cloud_ips":          datasourceIPs(),
	"grafana_cloud_organization": datasourceOrganization(),
	"grafana_cloud_stack":        datasourceStack(),
}

var ResourcesMap = map[string]*schema.Resource{
	"grafana_cloud_access_policy":               resourceAccessPolicy(),
	"grafana_cloud_access_policy_token":         resourceAccessPolicyToken(),
	"grafana_cloud_api_key":                     resourceAPIKey(),
	"grafana_cloud_plugin_installation":         resourcePluginInstallation(),
	"grafana_cloud_stack":                       resourceStack(),
	"grafana_cloud_stack_api_key":               resourceStackAPIKey(),
	"grafana_cloud_stack_service_account":       resourceStackServiceAccount(),
	"grafana_cloud_stack_service_account_token": resourceStackServiceAccountToken(),
	"grafana_synthetic_monitoring_installation": resourceSyntheticMonitoringInstallation(),
}
