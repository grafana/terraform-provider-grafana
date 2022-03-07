package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const testAccGrafanaAuthKeyBasicConfig = `
resource "grafana_api_key" "foo" {
	name = "foo-name"
	role = "Admin"
}
`

const testAccGrafanaAuthKeyExpandedConfig = `
resource "grafana_api_key" "bar" {
	name 			= "bar-name"
	role 			= "Viewer"
	seconds_to_live = 300
}
`

func TestAccGrafanaAuthKey(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccGrafanaAuthKeyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.foo", "foo-name", "Admin", false),
				),
			},
			{
				Config: testAccGrafanaAuthKeyExpandedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.bar", "bar-name", "Viewer", true),
				),
			},
		},
	})
}

func testAccGrafanaAuthKeyCheckDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*client).gapi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_api_key" {
			continue
		}

		idStr := rs.Primary.ID
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		keys, err := c.GetAPIKeys(false)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.ID == id {
				return fmt.Errorf("API key still exists")
			}
		}
	}

	return nil
}

func testAccGrafanaAuthKeyCheckFields(n string, name string, role string, expires bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["key"] == "" {
			return fmt.Errorf("no API key is set")
		}

		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("incorrect name field found: %s", rs.Primary.Attributes["name"])
		}

		if rs.Primary.Attributes["role"] != role {
			return fmt.Errorf("incorrect role field found: %s", rs.Primary.Attributes["role"])
		}

		expiration := rs.Primary.Attributes["expiration"]
		if expires && expiration == "" {
			return fmt.Errorf("no expiration date set")
		}

		if !expires && expiration != "" {
			return fmt.Errorf("expiration date set")
		}

		return nil
	}
}
