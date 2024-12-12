package fleetmanagement

import "github.com/grafana/terraform-provider-grafana/v3/internal/common"

var Resources = []*common.Resource{
	newCollectorResource(),
	newPipelineResource(),
}
