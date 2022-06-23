package grafana

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaSAT(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccServiceAccountTokenCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccServiceAccountTokenBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountTokenCheckFields("grafana_service_account_token.foo", "foo-name", 4, false),
				),
			},
			{
				Config: testAccServiceAccountTokenExpandedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccServiceAccountTokenCheckFields("grafana_service_account_token.bar", "bar-name", 1, true),
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

func testAccServiceAccountTokenCheckFields(n string, name string, serviceAccountID int64, expires bool) resource.TestCheckFunc {
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

		saID, err := strconv.ParseInt(rs.Primary.Attributes["service_account_id"], 10, 64)
		if err != nil {
			return err
		}
		if saID != serviceAccountID {
			return fmt.Errorf("incorrect service account id field found: %s", rs.Primary.Attributes["service_account_id"])
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

const testAccServiceAccountTokenBasicConfig = `
resource "grafana_service_account_token" "foo" {
	name = "foo-name"
	service_account_id = 4
}
`

const testAccServiceAccountTokenExpandedConfig = `
resource "grafana_service_account_token" "bar" {
	name 			= "bar-name"
	service_account_id = 1
	seconds_to_live = 300
}
`
