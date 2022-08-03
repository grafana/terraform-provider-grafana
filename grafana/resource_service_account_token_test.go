package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccServiceAccountToken_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=8.0.0")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountTokenCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountTokenCheckFields("grafana_service_account_token.foo", "foo-name", false),
				),
			},
			{
				Config: testAccServiceAccountTokenExpandedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountTokenCheckFields("grafana_service_account_token.bar", "bar-name", true),
				),
			},
		},
	})
}

func testAccServiceAccountTokenCheckDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*client).gapi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_service_account_token" {
			continue
		}

		idStr := rs.Primary.ID
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		keys, err := c.GetServiceAccountTokens(1)
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

func testAccServiceAccountTokenCheckFields(n string, name string, expires bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["key"] == "" {
			return fmt.Errorf("no key is set")
		}

		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("incorrect name field found: %s", rs.Primary.Attributes["name"])
		}

		_, err := strconv.ParseInt(rs.Primary.Attributes["service_account_id"], 10, 64)
		if err != nil {
			return err
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

const testAccServiceAccountOne = `
resource "grafana_service_account" "sa_one" {
  name     = "SA One"
  role     = "Editor"
}
`

const testAccServiceAccountTwo = `
resource "grafana_service_account" "sa_two" {
  name     = "SA Two"
  role     = "Viewer"
}
`

const testAccServiceAccountTokenBasicConfig = testAccServiceAccountOne + `
resource "grafana_service_account_token" "foo" {
	name = "foo-name"
	service_account_id = grafana_service_account.sa_one.id
}
`

const testAccServiceAccountTokenExpandedConfig = testAccServiceAccountTwo + `
resource "grafana_service_account_token" "bar" {
	name 			= "bar-name"
	service_account_id = grafana_service_account.sa_two.id
	seconds_to_live = 300
}
`
