package fleetmanagement

import "github.com/grafana/terraform-provider-grafana/v4/internal/common"

var Resources = []*common.Resource{
	newCollectorResource(),
	newPipelineResource(),
}
