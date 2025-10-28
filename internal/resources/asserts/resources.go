package asserts

import (
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var DataSources = []*common.DataSource{}

var Resources = []*common.Resource{
	makeResourceAlertConfig(),
	makeResourceDisabledAlertConfig(),
	makeResourceCustomModelRules(),
	makeResourceLogConfig(),
	makeResourceThresholds(),
}

func GetResources() []*common.Resource {
	return Resources
}
