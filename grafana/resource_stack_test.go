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
	CheckCloudStackTestsEnabled(t)
	var stack gapi.Stack
	stackName := "grafanacloudstack-test"
	stackSlug := "grafanacloudstack-test"
	stackDescription := "This is a test stack"

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheckCloudStack(t) },
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfigBasic(stackName, stackSlug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", idRegexp),
					resource.TestCheckResourceAttrSet("grafanacloud_stack.test", "id"),
					resource.TestCheckResourceAttr("grafanacloud_stack.test", "name", stackName),
					resource.TestCheckResourceAttr("grafanacloud_stack.test", "slug", stackSlug),
				),
			},
			{
				Config: testAccStackConfigUpdate(stackName, stackSlug, stackDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", stackName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "sllug", stackSlug),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "description", stackDescription),
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
		stackID, err := client.StackByID(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = stackID

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

func testAccStackConfigBasic(name string, slug string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "test" {
		name  = "%s"
		slug  = "%s"
	  }
	`, name, slug)
}

func testAccStackConfigUpdate(name string, slug string, description string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "test" {
		name        = "%s"
		slug        = "%s"
		description = "%s"
	  }
	`, name, slug, description)
}
