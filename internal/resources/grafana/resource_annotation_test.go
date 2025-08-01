package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

var (
	testAccAnnotationInitialText string = "basic text"
	testAccAnnotationUpdatedText string = "basic text updated"
)

func TestAccAnnotation_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Annotations don't work right in OSS Grafana < 9.0.0

	var annotation models.Annotation

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             annotationsCheckExists.destroyed(&annotation, nil),
		Steps: []resource.TestStep{
			{
				// Test basic resource creation.
				Config: testAnnotationConfig(testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationInitialText),
					testutils.CheckLister("grafana_annotation.test"),
				),
			},
			{
				// Updates text in basic resource.
				Config: testAnnotationConfig(testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationUpdatedText),
				),
			},
			{
				// Importing matches the state of the previous step.
				ResourceName:      "grafana_annotation.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAnnotation_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Annotations don't work right in OSS Grafana < 9.0.0

	var annotation models.Annotation
	var org models.OrgDetailsDTO

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             annotationsCheckExists.destroyed(&annotation, &org),
		Steps: []resource.TestStep{
			{
				// Test basic resource creation.
				Config: testAnnotationConfigInOrg(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in basic resource.
				Config: testAnnotationConfigInOrg(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
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
				// Test resource creation with declared panel_id.
				Config: testAnnotationConfigInOrgWithPanelID(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared panel_id.
				Config: testAnnotationConfigInOrgWithPanelID(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test_with_panel_id", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
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
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0")

	var annotation models.Annotation
	var org models.OrgDetailsDTO

	orgName := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             annotationsCheckExists.destroyed(&annotation, &org),
		Steps: []resource.TestStep{
			{
				// Test resource creation with declared dashboard_uid.
				Config: testAnnotationConfigInOrgWithDashboardUID(orgName, testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test_with_dashboard_uid", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_uid", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared dashboard_id.
				Config: testAnnotationConfigInOrgWithDashboardUID(orgName, testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					annotationsCheckExists.exists("grafana_annotation.test_with_dashboard_uid", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_uid", "text", testAccAnnotationUpdatedText),

					// Check that the annotation is in the correct organization
					resource.TestMatchResourceAttr("grafana_annotation.test_with_dashboard_uid", "id", nonDefaultOrgIDRegexp),
					orgCheckExists.exists("grafana_organization.test", &org),
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

func testAnnotationConfig(text string) string {
	return fmt.Sprintf(`
	resource "grafana_annotation" "test" {
		text = "%s"
	}`, text)
}

func testAnnotationConfigInOrg(orgName, text string) string {
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

func testAnnotationConfigInOrgWithDashboardUID(orgName, text string) string {
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

func testAnnotationConfigInOrgWithPanelID(orgName, text string) string {
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
