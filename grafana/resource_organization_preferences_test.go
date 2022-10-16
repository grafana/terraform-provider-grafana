package grafana

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceOrganizationPreferences(t *testing.T) {
	CheckOSSTestsEnabled(t)

	prefs := gapi.Preferences{
		Theme:     "light",
		Timezone:  "utc",
		WeekStart: "Monday",
	}
	updatedPrefs := gapi.Preferences{
		Theme:     "dark",
		Timezone:  "utc",
		WeekStart: "Tuesday",
	}

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationPreferencesCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testOrganizationPreferencesConfig(prefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", prefs),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "id", "organization_preferences"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", prefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "0"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", ""),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", prefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", prefs.WeekStart),
				),
			},
			{
				Config: testOrganizationPreferencesConfig(updatedPrefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", updatedPrefs),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "id", "organization_preferences"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", updatedPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "0"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", ""),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", updatedPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", updatedPrefs.WeekStart),
				),
			},
		},
	})
}

func testAccOrganizationPreferencesCheckExists(rn string, prefs gapi.Preferences) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
		p, err := client.OrgPreferences()
		if err != nil {
			return fmt.Errorf("error getting organization preferences: %s", err)
		}

		errs := []string{}
		if p.Theme != prefs.Theme {
			errs = append(errs, fmt.Sprintf("expected organization preferences theme '%s'; got '%s'", prefs.Theme, p.Theme))
		}
		if p.Timezone != prefs.Timezone {
			errs = append(errs, fmt.Sprintf("expected organization preferences timezone '%s'; got '%s'", prefs.Timezone, p.Timezone))
		}
		if p.WeekStart != prefs.WeekStart {
			errs = append(errs, fmt.Sprintf("expected organization preferences week start '%s'; got '%s'", prefs.WeekStart, p.WeekStart))
		}

		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "\n"))
		}

		return nil
	}
}

func testAccOrganizationPreferencesCheckDestroy() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		prefs, err := client.OrgPreferences()
		if err != nil {
			return err
		}

		if prefs.Theme != "" || prefs.Timezone != "" || prefs.WeekStart != "" {
			return fmt.Errorf("customized organization preferences still exist")
		}
		return nil
	}
}

func testOrganizationPreferencesConfig(prefs gapi.Preferences) string {
	return fmt.Sprintf(`
resource "grafana_organization_preferences" "test" {
  theme      = "%s"
  timezone   = "%s"
  week_start = "%s"
}
`, prefs.Theme, prefs.Timezone, prefs.WeekStart)
}
