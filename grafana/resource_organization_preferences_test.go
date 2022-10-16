package grafana

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceOrganizationPreferences(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationPreferencesCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "resources/grafana_organization_preferences/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "id", "organization_preferences"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", "light"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "wrong"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", "wrong"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", "utc"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", "wrong"),
				),
			},
		},
	})
}

func testAccOrganizationPreferencesCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		prefs, err := client.OrgPreferences()
		if err != nil {
			return err
		}

		// TODO: beef up these assertions
		if prefs.Theme != "" {
			return fmt.Errorf("customized organization preferences still exist")
		}
		return nil
	}
}
