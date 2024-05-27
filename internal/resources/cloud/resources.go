package cloud

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
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
	resourceOrgMember(),
	resourcePluginInstallation(),
	resourceStack(),
	resourceStackServiceAccount(),
	resourceStackServiceAccountToken(),
	resourceSyntheticMonitoringInstallation(),
}
