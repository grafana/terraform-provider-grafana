package grafana_test

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccAnnotationInitialText string = "basic text"
	testAccAnnotationUpdatedText string = "basic text updated"
)

func TestAccAnnotation_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var annotation gapi.Annotation
	var org gapi.Org

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccAnnotationCheckDestroy(&annotation),
		Steps: []resource.TestStep{
			{
				// Test basic resource creation.
				Config: testAnnotationConfig(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in basic resource.
				Config: testAnnotationConfig(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_annotation.test", "grafana_organization.test"),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_annotation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test resource creation with declared dashboard_id.
				Config: testAnnotationConfigWithDashboardID(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_id", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared dashboard_id.
				Config: testAnnotationConfigWithDashboardID(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_id", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test_with_dashboard_id", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_annotation.test_with_dashboard_id", "grafana_organization.test"),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_annotation.test_with_dashboard_id",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test resource creation with declared panel_id.
				Config: testAnnotationConfigWithPanelID(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared panel_id.
				Config: testAnnotationConfigWithPanelID(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test_with_panel_id", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_annotation.test_with_panel_id", "grafana_organization.test"),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_annotation.test_with_panel_id",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAnnotation_dashboardUID(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)
	testutils.CheckOSSTestsSemver(t, ">=9.0.0")

	var annotation gapi.Annotation
	var org gapi.Org

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccAnnotationCheckDestroy(&annotation),
		Steps: []resource.TestStep{
			{
				// Test resource creation with declared dashboard_uid.
				Config: testAnnotationConfigWithDashboardUID(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_uid", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_uid", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared dashboard_id.
				Config: testAnnotationConfigWithDashboardUID(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_uid", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_uid", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test_with_dashboard_uid", "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg("grafana_annotation.test_with_dashboard_uid", "grafana_organization.test"),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_annotation.test_with_dashboard_uid",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccAnnotationCheckExists(rn string, annotation *gapi.Annotation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		orgID, _ := grafana.SplitOrgResourceID(rs.Primary.ID)

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		// If the org ID is set, check that the annotation doesn't exist in the default org
		if orgID > 0 {
			annotations, err := client.Annotations(url.Values{})
			if err != nil {
				return fmt.Errorf("error getting annotations: %s", err)
			}
			if len(annotations) > 0 {
				return fmt.Errorf("annotation exists in the default org")
			}
			client = client.WithOrgID(orgID)
		}

		annotations, err := client.Annotations(url.Values{})
		if err != nil {
			return fmt.Errorf("error getting annotation: %s", err)
		}

		if len(annotations) < 1 {
			return errors.New("Grafana API returned no annotations")
		}

		*annotation = annotations[0]

		return nil
	}
}

func testAccAnnotationCheckDestroy(annotation *gapi.Annotation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		annotations, err := client.Annotations(url.Values{})
		if err != nil {
			return err
		}

		if len(annotations) > 0 {
			return errors.New("annotation still exists")
		}

		return nil
	}
}

func testAnnotationConfig(orgName, text string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
    name = "%[1]s"
}

resource "grafana_annotation" "test" {
    org_id = grafana_organization.test.id
    text = "%s"
}
`, orgName, text)
}

func testAnnotationConfigWithDashboardID(orgName, text string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	  name = "%[1]s"
}

resource "grafana_dashboard" "test_with_dashboard_id" {
    org_id = grafana_organization.test.id
    config_json = <<EOD
{
  "title": "%[2]s"
}
EOD
}

resource "grafana_annotation" "test_with_dashboard_id" {
    org_id = grafana_organization.test.id
    text         = "%[2]s"
    dashboard_id = grafana_dashboard.test_with_dashboard_id.dashboard_id
}
`, orgName, text)
}

func testAnnotationConfigWithDashboardUID(orgName, text string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	  name = "%[1]s"
}

resource "grafana_dashboard" "test_with_dashboard_uid" {
    org_id = grafana_organization.test.id
    config_json = <<EOD
{
  "title": "%[2]s"
}
EOD
}

resource "grafana_annotation" "test_with_dashboard_uid" {
    org_id = grafana_organization.test.id
    text         = "%[2]s"
    dashboard_uid = grafana_dashboard.test_with_dashboard_uid.uid
}
`, orgName, text)
}

func testAnnotationConfigWithPanelID(orgName, text string) string {
	panelID := 123

	return fmt.Sprintf(`
resource "grafana_organization" "test" {
    name = "%[1]s"
}

resource "grafana_dashboard" "test_with_panel_id" {
    org_id = grafana_organization.test.id
	config_json = <<EOD
{
  "title": "%[2]s",
	"panels": [{
		"name": "%[2]s",
		"id": %[3]d
	}]
}
EOD
}

resource "grafana_annotation" "test_with_panel_id" {
    org_id = grafana_organization.test.id
    text     = "%[2]s"
    panel_id = %[3]d
}
`, orgName, text, panelID)
}
