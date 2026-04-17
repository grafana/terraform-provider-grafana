package slo

import (
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var DataSources = []*common.DataSource{
	makeDatasourceSlo(),
}

var Resources = []*common.Resource{
	makeResourceSlo(),
}
