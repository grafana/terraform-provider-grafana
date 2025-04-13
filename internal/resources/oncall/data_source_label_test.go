package oncall_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceLabel_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	key := fmt.Sprintf("Key%sOne", acctest.RandString(8))
	value := fmt.Sprintf("Value%sOne", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceLabelConfig(key, value),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.grafana_oncall_label.test-acc-label", "id"),
					resource.TestCheckResourceAttr("data.grafana_oncall_label.test-acc-label", "key", "1"),
					resource.TestCheckResourceAttr("data.grafana_oncall_label.test-acc-label", "value", value),
				),
			},
		},
	})
}

func testAccDataSourceLabelConfig(key string, value string) string {
	return fmt.Sprintf(`
data "grafana_oncall_label" "test-acc-label" {
	provider = grafana.oncall
	key = "%[1]s"
	value = "%[2]s"
}
`, key, value)
}
