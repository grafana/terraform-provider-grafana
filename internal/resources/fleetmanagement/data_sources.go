package fleetmanagement

import "github.com/grafana/terraform-provider-grafana/v4/internal/common"

var DataSources = []*common.DataSource{
	newCollectorDataSource(),
	newCollectorsDataSource(),
}
