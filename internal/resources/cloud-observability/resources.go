package cloudobservability

import (
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var DatasourcesMap = map[string]*schema.Resource{
	"grafana_cloud_observability_aws_account":  datasourceAWSAccount(),
	"grafana_cloud_observability_aws_accounts": datasourceAWSAccounts(),
}

var Resources = []*common.Resource{
	resourceAWSAccount(),
}
