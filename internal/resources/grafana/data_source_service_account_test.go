package grafana_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceServiceAccount_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var sa models.ServiceAccountDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountCheckExists.destroyed(&sa, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountDatasourceConfig(name),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("data.grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("data.grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("data.grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("data.grafana_service_account.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("data.grafana_service_account.test", "id", defaultOrgIDRegexp),
				),
			},
		},
	})
}

func testServiceAccountDatasourceConfig(name string) string {
	return fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name        = "%[1]s"
	role        = "Editor"
	is_disabled = false
}

data "grafana_service_account" "test" {
	name = grafana_service_account.test.name
}`, name)
}
