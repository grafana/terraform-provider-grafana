package grafana

import (
	"crypto/rand"
	"fmt"
	"math/big"

	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	// CharSetAlphaNum is the alphanumeric character set for use with
	// RandStringFromCharSet
	CharSetAlphaNum = "abcdefghijklmnopqrstuvwxyz012346789"
)

func TestResourceStack_Basic(t *testing.T) {
	CheckCloudTestsEnabled(t)
	var stack gapi.Stack
	stackName, _ := RandStringFromCharSet(10, CharSetAlphaNum)
	stackSlug := stackName
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
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", stackSlug),
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

// RandStringFromCharSet generates a random string by selecting characters from
// the charset provided
func RandStringFromCharSet(strlen int, charSet string) (string, error) {
	result := make([]byte, strlen)

	for i := 0; i < strlen; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charSet))))
		if err != nil {
			return "", err
		}
		result[i] = charSet[num.Int64()]
	}

	return string(result), nil
}
