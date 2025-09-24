package asserts

import (
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var DataSources = []*common.DataSource{}

var Resources = []*common.Resource{
	makeResourceAlertConfig(),
	makeResourceDisabledAlertConfig(),
	makeResourceCustomModelRules(),
	makeResourceThresholds(),
}

func GetResources() []*common.Resource {
	return Resources
}
