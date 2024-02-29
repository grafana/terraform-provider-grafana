package cloud

import (
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_cloud_ips":          datasourceIPs(),
	"grafana_cloud_organization": datasourceOrganization(),
	"grafana_cloud_stack":        datasourceStack(),
}

var Resources = []*common.Resource{
	resourceAccessPolicy(),
	resourceAccessPolicyToken(),
	resourceAPIKey(),
	resourcePluginInstallation(),
	resourceStack(),
	resourceStackAPIKey(),
	resourceStackServiceAccount(),
	resourceStackServiceAccountToken(),
	resourceSyntheticMonitoringInstallation(),
}

func ResourcesMap() map[string]*schema.Resource {
	m := make(map[string]*schema.Resource)
	for _, r := range Resources {
		m[r.Name] = r.Schema
	}
	return m
}
