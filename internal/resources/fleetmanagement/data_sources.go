package fleetmanagement

import "github.com/grafana/terraform-provider-grafana/v3/internal/common"

var DataSources = []*common.DataSource{
	newCollectorDataSource(),
	newCollectorsDataSource(),
}
