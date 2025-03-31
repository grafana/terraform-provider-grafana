package fleetmanagement_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	collectorDataSourceConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	remote_attributes = {
		"env"   = "PROD",
		"owner" = "TEAM-A"
	}
	enabled = false
}

data "grafana_fleet_management_collector" "test" {
	id = grafana_fleet_management_collector.test.id
}
`
)

func TestAccCollectorDataSource(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	dataSourceName := "data.grafana_fleet_management_collector.test"
	collectorID := fmt.Sprintf("testacc_%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(collectorDataSourceConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(dataSourceName, "id", collectorID),
					resource.TestCheckResourceAttr(dataSourceName, "remote_attributes.%", "2"),
					resource.TestCheckResourceAttr(dataSourceName, "remote_attributes.env", "PROD"),
					resource.TestCheckResourceAttr(dataSourceName, "remote_attributes.owner", "TEAM-A"),
					resource.TestCheckResourceAttrSet(dataSourceName, "local_attributes.%"),
					resource.TestCheckResourceAttr(dataSourceName, "local_attributes.%", "0"),
					resource.TestCheckResourceAttr(dataSourceName, "enabled", "false"),
				),
			},
		},
	})
}
