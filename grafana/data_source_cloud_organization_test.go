package grafana

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceCloudOrganization_Basic(t *testing.T) {
	t.Parallel()
	CheckCloudAPITestsEnabled(t)

	config := fmt.Sprintf(`
	data "grafana_cloud_organization" "test" {
	  	slug = "%s"
	}
	`, os.Getenv("GRAFANA_CLOUD_ORG"))

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestMatchResourceAttr("data.grafana_cloud_organization.test", "id", idRegexp),
					resource.TestCheckResourceAttr("data.grafana_cloud_organization.test", "slug", os.Getenv("GRAFANA_CLOUD_ORG")),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "created_at"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_organization.test", "updated_at"),
				),
			},
		},
	})
}
