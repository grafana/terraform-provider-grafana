package provider

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceOrganizationPreferences_WithDashboardID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=8.0.0")
	testAccResourceOrganizationPreferences(t, false)
}

func TestAccResourceOrganizationPreferences_WithDashboardUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0") // UID support was added in 9.0.0
	testAccResourceOrganizationPreferences(t, true)
}

func testAccResourceOrganizationPreferences(t *testing.T, withUID bool) {
	t.Helper()

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

	// In versions < 9.0.0, the home dashboard UID is not returned by the API
	dashboardCheck := resource.TestMatchResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", idRegexp)
	if withUID {
		dashboardCheck = resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", testRandName)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccOrganizationPreferencesCheckDestroy(),
		Steps: []resource.TestStep{
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, prefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", prefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", prefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", prefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", prefs.WeekStart),
					dashboardCheck,
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, updatedPrefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", updatedPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", updatedPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", updatedPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", updatedPrefs.WeekStart),
					dashboardCheck,
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, finalPrefs),
				Check: resource.ComposeTestCheckFunc(
					testAccOrganizationPreferencesCheckExists("grafana_organization_preferences.test", finalPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", idRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", finalPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", finalPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", finalPrefs.WeekStart),
					dashboardCheck,
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
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI.WithOrgID(id)
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
		client := testAccProvider.Meta().(*common.Client).GrafanaAPI
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

func testOrganizationPreferencesConfig(orgName string, withUID bool, prefs gapi.Preferences) string {
	dashboardBlock := ""
	if withUID {
		dashboardBlock = "home_dashboard_uid = grafana_dashboard.test.uid"
	} else {
		dashboardBlock = "home_dashboard_id = grafana_dashboard.test.dashboard_id"
	}

	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = "%[1]s"
}

resource "grafana_dashboard" "test" {
	org_id = grafana_organization.test.id
	config_json = jsonencode({
	  title = "test-org-%[1]s"
	  uid   = "%[1]s"
	})
}

resource "grafana_organization_preferences" "test" {
  org_id     = grafana_organization.test.id
  theme      = "%[2]s"
  timezone   = "%[3]s"
  week_start = "%[4]s"
  %[5]s
}
`, orgName, prefs.Theme, prefs.Timezone, prefs.WeekStart, dashboardBlock)
}
