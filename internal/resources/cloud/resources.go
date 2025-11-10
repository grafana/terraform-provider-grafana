package cloud

import (
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
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
	resourceAccessPolicyRotatingToken(),
	resourceOrgMember(),
	resourcePluginInstallation(),
	resourceStack(),
	resourceStackServiceAccount(),
	resourceStackServiceAccountToken(),
	resourceK6Installation(),
	resourceSyntheticMonitoringInstallation(),
	resourcePDCNetwork(),
	resourcePDCNetworkToken(),
}
