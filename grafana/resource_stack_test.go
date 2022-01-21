package grafana

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceStack_basic(t *testing.T) {
	CheckCloudTestsEnabled(t)
	var stack gapi.Stack

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_stack.test", &stack),
					resource.TestCheckResourceAttr(
						"grafana_stack.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_stack.test", "email", "stackEmail@example.com",
					),
					resource.TestMatchResourceAttr(
						"grafana_stack.test", "id", idRegexp,
					),
				),
			},
			{
				Config: testAccStackConfig_updateName,
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_stack.test", &stack),
					resource.TestCheckResourceAttr(
						"grafana_stack.test", "name", "terraform-acc-test-update",
					),
					resource.TestCheckResourceAttr(
						"grafana_stack.test", "slug", "",
					),
				),
			},
		},
	})
}

func testAccStackCheckDestroy(a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		stack, err := client.StackByID(a.ID)
		if err == nil && stack.Name != "" && stack.Status != "deleted" {
			return fmt.Errorf("stack still exists")
		}
		return nil
	}
}

//nolint:unparam // `rn` always receives `"grafana_stack.test"`
func testAccStackCheckExists(rn string, a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*client).gapi
		stackId, err := client.StackByID(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = stackId

		return nil
	}
}

const testAccStackConfig_basic = `
resource "grafana_stack" "test" {
  name  = "terraform-acc-test"
  email = "stackEmail@example.com"
}
`
const testAccStackConfig_updateName = `
resource "grafana_stack" "test" {
  name    = "terraform-acc-test-update"
  email   = "stackEmailUpdate@example.com"
}
`
