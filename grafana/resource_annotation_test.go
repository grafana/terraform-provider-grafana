package grafana

import (
	"errors"
	"fmt"
	"net/url"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	testAccAnnotationInitialText string = "basic text"
	testAccAnnotationUpdatedText string = "basic text updated"
)

func TestAccAnnotation_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	var annotation gapi.Annotation

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccAnnotationCheckDestroy(&annotation),
		Steps: []resource.TestStep{
			{
				// Test basic resource creation.
				Config: testAnnotationConfig(testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in basic resource.
				Config: testAnnotationConfig(testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test", "text", testAccAnnotationUpdatedText),
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
				Config: testAnnotationConfigWithDashboardID(testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_id", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared dashboard_id.
				Config: testAnnotationConfigWithDashboardID(testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_dashboard_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_dashboard_id", "text", testAccAnnotationUpdatedText),
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
				Config: testAnnotationConfigWithPanelID(testAccAnnotationInitialText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationInitialText),
				),
			},
			{
				// Updates text in resource with declared panel_id.
				Config: testAnnotationConfigWithPanelID(testAccAnnotationUpdatedText),
				Check: resource.ComposeTestCheckFunc(
					testAccAnnotationCheckExists("grafana_annotation.test_with_panel_id", &annotation),
					resource.TestCheckResourceAttr("grafana_annotation.test_with_panel_id", "text", testAccAnnotationUpdatedText),
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

func testAccAnnotationCheckExists(rn string, annotation *gapi.Annotation) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testAccProvider.Meta().(*client).gapi
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
		client := testAccProvider.Meta().(*client).gapi
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

func testAnnotationConfig(text string) string {
	return fmt.Sprintf(`
resource "grafana_annotation" "test" {
    text = "%s"
}
`, text)
}

func testAnnotationConfigWithDashboardID(text string) string {
	return fmt.Sprintf(`
resource "grafana_dashboard" "test_with_dashboard_id" {
  config_json = <<EOD
{
  "title": "%s"
}
EOD
}

resource "grafana_annotation" "test_with_dashboard_id" {
    text         = "%s"
		dashboard_id = grafana_dashboard.test_with_dashboard_id.dashboard_id
}
`, text, text)
}

func testAnnotationConfigWithPanelID(text string) string {
	return fmt.Sprintf(`
resource "grafana_dashboard" "test_with_panel_id" {
  config_json = <<EOD
{
  "title": "%s",
	"panels": [{
		"name": "%s"
	}]
}
EOD
}

data "grafana_dashboard" "test_with_panel_id" {
	dashboard_id = grafana_dashboard.test_with_panel_id.dashboard_id
}

locals {
  dashboard_json = jsondecode(data.grafana_dashboard.test_with_panel_id.config_json)
  panel_id       = local.dashboard_json.dashboard.panels[0].id
}

resource "grafana_annotation" "test_with_panel_id" {
    text     = "%s"
		panel_id = local.panel_id
}
`, text, text, text)
}
