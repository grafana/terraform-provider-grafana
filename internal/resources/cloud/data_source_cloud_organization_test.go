package cloud_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOrganization_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	config := fmt.Sprintf(`
	data "grafana_cloud_organization" "test" {
	  	slug = "%s"
	}
	`, os.Getenv("GRAFANA_CLOUD_ORG"))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.grafana_cloud_organization.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("data.grafana_cloud_organization.test", "slug", os.Getenv("GRAFANA_CLOUD_ORG")),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "updated_at"),
				),
			},
		},
	})
}
