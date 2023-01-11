package grafana

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceOrganizationPreferences(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=8.0.0")

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
	finalPrefs := gapi.Preferences{
		Theme:     "",
		Timezone:  "browser",
		WeekStart: "Monday",
	}

	testRandName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationPreferencesCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testOrganizationPreferencesConfig(testRandName, prefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", prefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", prefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "0"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", ""),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", prefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", prefs.WeekStart),
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, updatedPrefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", updatedPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", updatedPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "0"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", ""),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", updatedPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", updatedPrefs.WeekStart),
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, finalPrefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", finalPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", finalPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", "0"),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", ""),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", finalPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", finalPrefs.WeekStart),
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

		id, err := strconv.ParseInt(rs.Primary.Attributes["org_id"], 10, 64)
		if err != nil {
			return fmt.Errorf("error parsing org_id: %s", err)
		}
		client := testAccProvider.Meta().(*client).gapi.WithOrgID(id)
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

func testOrganizationPreferencesConfig(orgName string, prefs gapi.Preferences) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_organization_preferences" "test" {
  org_id     = grafana_organization.test.id
  theme      = "%[2]s"
  timezone   = "%[3]s"
  week_start = "%[4]s"
}
`, orgName, prefs.Theme, prefs.Timezone, prefs.WeekStart)
}
