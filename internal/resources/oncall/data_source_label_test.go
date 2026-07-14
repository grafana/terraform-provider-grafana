package oncall_test

import (
	"fmt"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceLabel_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	key := fmt.Sprintf("Key%sOne", acctest.RandString(8))
	value := fmt.Sprintf("Value%sOne", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckOnCallIntegrationLabelDatasourceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLabelConfig(key, value),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_label.test-acc-label", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_label.test-acc-label", "key", key),
					resource.TestCheckResourceAttr("data.grafana_oncall_label.test-acc-label", "value", value),
				),
			},
		},
	})
}

func testAccDataSourceLabelConfig(key string, value string) string {
	return fmt.Sprintf(`
data "grafana_oncall_label" "test-acc-label" {
	key = "%[1]s"
	value = "%[2]s"
}
`, key, value)
}

func testAccCheckOnCallIntegrationLabelDatasourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).OnCallClient
	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_oncall_label" {
			continue
		}

		if _, _, err := client.Integrations.GetIntegration(r.Primary.ID, &onCallAPI.GetIntegrationOptions{}); err == nil {
			return fmt.Errorf("integration label still exists")
		}
	}
	return nil
}
