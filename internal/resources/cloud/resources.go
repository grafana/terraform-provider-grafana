package cloud

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

var DataSources = []*common.DataSource{
	datasourceAccessPolicies(),
	datasourceIPs(),
	datasourceOrganization(),
	datasourceStack(),
	datasourcePrivateDataSourceConnectNetworks(),
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
	resourcePDCNetwork(),
	resourcePDCNetworkToken(),
}
