package grafana

import (
	"fmt"
	"strings"

	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceCloudStack_Basic(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	prefix := "tfresourcetest"

	var stack gapi.Stack
	resourceName := GetRandomStackName(prefix)
	stackDescription := "This is a test stack"

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccStackConfigBasic(resourceName, resourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", idRegexp),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "id"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
				),
			},
			{
				Config: testAccStackConfigUpdate(resourceName+"new", resourceName, stackDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName+"new"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "description", stackDescription),
				),
			},
		},
	})
}

func testAccDeleteExistingStacks(t *testing.T, prefix string) {
	client := testAccProvider.Meta().(*client).gcloudapi
	resp, err := client.Stacks()
	if err != nil {
		t.Error(err)
	}

	for _, stack := range resp.Items {
		if strings.HasPrefix(stack.Name, prefix) {
			err := client.DeleteStack(stack.Slug)
			if err != nil {
				t.Error(err)
			}
		}
	}
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

		client := testAccProvider.Meta().(*client).gcloudapi
		stack, err := client.StackByID(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = stack

		return nil
	}
}

func testAccStackCheckDestroy(a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gcloudapi
		stack, err := client.StackBySlug(a.Slug)
		if err == nil && stack.Name != "" {
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
		region_slug = "eu"
	  }
	`, name, slug)
}

func testAccStackConfigUpdate(name string, slug string, description string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "test" {
		name  = "%s"
		slug  = "%s"
		region_slug = "eu"
		description = "%s"
	  }
	`, name, slug, description)
}

// Prefix a character as stack name can't start with a number
func GetRandomStackName(prefix string) string {
	return prefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}
