package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceIntegration_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := fmt.Sprintf("test-name-%s", acctest.RandString(10))

	integrationPath := "grafana_oncall_integration.test_integration"
	dataSourcePath := "data.grafana_oncall_integration.test_integration_ds"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallIntegrationResourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceIntegration(randomName),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOnCallIntegrationResourceExists(integrationPath),
					resource.TestCheckResourceAttr(integrationPath, "name", randomName),
					resource.TestCheckResourceAttrSet(integrationPath, "link"),
					resource.TestCheckResourceAttrPair(
						integrationPath, "id",
						dataSourcePath, "id",
					),
				),
			},
		},
	})
}

func testAccDataSourceIntegration(randomName string) string {
	return fmt.Sprintf(`
resource "grafana_oncall_integration" "test_integration" {
	name = "%[1]s"
	type = "grafana"
	default_route {
	}
}

data "grafana_oncall_integration" "test_integration_ds" {
	id = grafana_oncall_integration.test_integration.id
}
`, randomName)
}
