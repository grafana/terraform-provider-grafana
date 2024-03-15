package grafana_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceOrganizationPreferences_WithDashboardID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=8.0.0")
	testAccResourceOrganizationPreferences(t, false)
}

func TestAccResourceOrganizationPreferences_WithDashboardUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // UID support was added in 9.0.0
	testAccResourceOrganizationPreferences(t, true)
}

func testAccResourceOrganizationPreferences(t *testing.T, withUID bool) {
	t.Helper()

	var org models.OrgDetailsDTO
	prefs := models.Preferences{
		Theme:     "light",
		Timezone:  "utc",
		WeekStart: "monday",
	}
	updatedPrefs := models.Preferences{
		Theme:     "dark",
		Timezone:  "utc",
		WeekStart: "sunday",
	}
	finalPrefs := models.Preferences{
		Theme:     "",
		Timezone:  "browser",
		WeekStart: "saturday",
	}
	emptyPrefs := models.Preferences{
		Theme:     "",
		Timezone:  "",
		WeekStart: "",
	}

	testRandName := acctest.RandString(10)

	// In versions < 9.0.0, the home dashboard UID is not returned by the API
	dashboardCheck := resource.TestMatchResourceAttr("grafana_organization_preferences.test", "home_dashboard_id", common.IDRegexp)
	if withUID {
		dashboardCheck = resource.TestCheckResourceAttr("grafana_organization_preferences.test", "home_dashboard_uid", testRandName)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, prefs),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					testAccCheckOrganizationPreferences(&org, prefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", prefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", prefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", prefs.WeekStart),
					dashboardCheck,
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, updatedPrefs),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					testAccCheckOrganizationPreferences(&org, updatedPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", updatedPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", updatedPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", updatedPrefs.WeekStart),
					dashboardCheck,
				),
			},
			{
				Config: testOrganizationPreferencesConfig(testRandName, withUID, finalPrefs),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					testAccCheckOrganizationPreferences(&org, finalPrefs),
					resource.TestMatchResourceAttr("grafana_organization_preferences.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "theme", finalPrefs.Theme),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "timezone", finalPrefs.Timezone),
					resource.TestCheckResourceAttr("grafana_organization_preferences.test", "week_start", finalPrefs.WeekStart),
					dashboardCheck,
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_organization_preferences.test",
				ImportStateVerify: true,
			},
			// Test removing preferences (CheckDestroy is insufficient because it removes the whole organization)
			{
				Config: testutils.WithoutResource(t, testOrganizationPreferencesConfig(testRandName, withUID, prefs), "grafana_organization_preferences.test"),
				Check: resource.ComposeTestCheckFunc(
					orgCheckExists.exists("grafana_organization.test", &org),
					testAccCheckOrganizationPreferences(&org, emptyPrefs),
				),
			},
		},
	})
}

func testAccCheckOrganizationPreferences(org *models.OrgDetailsDTO, expectedPrefs models.Preferences) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := grafana.OAPIGlobalClient(testutils.Provider.Meta()).WithOrgID(org.ID)
		resp, err := client.OrgPreferences.GetOrgPreferences()
		if err != nil {
			return fmt.Errorf("error getting organization preferences: %s", err)
		}
		gotPrefs := resp.Payload

		errs := []string{}
		if gotPrefs.Theme != expectedPrefs.Theme {
			errs = append(errs, fmt.Sprintf("expected organization preferences theme '%s'; got '%s'", expectedPrefs.Theme, gotPrefs.Theme))
		}
		if gotPrefs.Timezone != expectedPrefs.Timezone {
			errs = append(errs, fmt.Sprintf("expected organization preferences timezone '%s'; got '%s'", expectedPrefs.Timezone, gotPrefs.Timezone))
		}
		if gotPrefs.WeekStart != expectedPrefs.WeekStart {
			errs = append(errs, fmt.Sprintf("expected organization preferences week start '%s'; got '%s'", expectedPrefs.WeekStart, gotPrefs.WeekStart))
		}

		if len(errs) > 0 {
			return errors.New(strings.Join(errs, "\n"))
		}

		return nil
	}
}

func testOrganizationPreferencesConfig(orgName string, withUID bool, prefs models.Preferences) string {
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
