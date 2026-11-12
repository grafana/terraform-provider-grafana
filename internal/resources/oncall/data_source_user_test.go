package oncall_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceUser_Basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	username := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccDataSourceUserConfig(username),
				ExpectError: regexp.MustCompile(`couldn't find a user`),
			},
		},
	})
}

// TestAccDataSourceUser_Teams looks a user up by the username the plural data
// source reports, so the two stay consistent about what a user is.
func TestAccDataSourceUser_Teams(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "grafana_oncall_users" "test-acc-all" {}

data "grafana_oncall_user" "test-acc-first" {
	username = data.grafana_oncall_users.test-acc-all.users[0].username
}
`,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair(
						"data.grafana_oncall_user.test-acc-first", "id",
						"data.grafana_oncall_users.test-acc-all", "users.0.id",
					),
					// Empty rather than null for a user on no team. A null list is
					// left out of state entirely, so a missing count is a regression.
					resource.TestCheckResourceAttrSet("data.grafana_oncall_user.test-acc-first", "teams.#"),
				),
			},
		},
	})
}

func testAccDataSourceUserConfig(username string) string {
	return fmt.Sprintf(`
data "grafana_oncall_user" "test-acc-user" {
	username = "%s"
}
`, username)
}
