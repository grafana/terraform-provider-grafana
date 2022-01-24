package grafana

import (
	"fmt"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceStack_Basic(t *testing.T) {
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
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", idRegexp),
					resource.TestCheckResourceAttrSet("grafanacloud_stack.test", "id"),
					resource.TestCheckResourceAttr("grafanacloud_stack.test", "name", "terraform-acc-test"),
					resource.TestCheckResourceAttr("grafanacloud_stack.test", "slug", "terraform-acc-test"),
				),
			},
			{
				Config: testAccStackConfig_updateName,
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", "terraform-acc-test-update"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "sllug", "terraform-acc-test-slug"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "description", "test description"),
				),
			},
		},
	})
}

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

func testAccStackCheckDestroy(a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		stack, err := client.StackByID(a.ID)
		if err == nil {
			return err
		}

		if stack.Name != "" || stack.Status != "deleted" {
			return fmt.Errorf("stack `%s` with ID `%d` still exists after destroy", stack.Name, stack.ID)
		}

		return nil
	}
}

const testAccStackConfig_basic = `
resource "grafana_cloud_stack" "test" {
  name  = "terraform-acc-test"
  slug = "terraform-acc-test
}
`
const testAccStackConfig_updateName = `
resource "grafana_cloud_stack" "test" {
  name    = "terraform-acc-test-update"
  slug = "terraform-acc-test-slug"
  description = "test description"
}
`
