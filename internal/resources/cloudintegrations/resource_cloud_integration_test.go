package cloudintegrations_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	cloudIntegrationResourceDefaultConfig = `
resource "grafana_cloud_integration" "test" {
	slug = "%s"
}
`

	cloudIntegrationResourceAlertsDisabledConfig = `
resource "grafana_cloud_integration" "test" {
	slug           = "%s"
	alerts_enabled = false
}
`

	cloudIntegrationResourceAlertsEnabledConfig = `
resource "grafana_cloud_integration" "test" {
	slug           = "%s"
	alerts_enabled = true
}
`
)

func TestAccCloudIntegrationResource(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_cloud_integration.test"
	slug := "docker"

	// Note, we don't actually expose installed dashboards/alerts to the client,
	// so the best we can test here is success in changes
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields (alerts_enabled defaults to true)
			{
				Config: fmt.Sprintf(cloudIntegrationResourceDefaultConfig, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudIntegrationResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", slug),
					resource.TestCheckResourceAttr(resourceName, "slug", slug),
					resource.TestCheckResourceAttr(resourceName, "alerts_enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "installed_version"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "dashboard_folder"),
				),
			},
			// Import state with only required fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     slug,
				ImportStateVerify: true,
			},
			// Update with alerts disabled
			{
				Config: fmt.Sprintf(cloudIntegrationResourceAlertsDisabledConfig, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudIntegrationResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", slug),
					resource.TestCheckResourceAttr(resourceName, "slug", slug),
					resource.TestCheckResourceAttr(resourceName, "alerts_enabled", "false"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "installed_version"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "dashboard_folder"),
				),
			},
			// Import state with alerts disabled
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     slug,
				ImportStateVerify: true,
			},
			// Update back to alerts enabled
			{
				Config: fmt.Sprintf(cloudIntegrationResourceAlertsEnabledConfig, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudIntegrationResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", slug),
					resource.TestCheckResourceAttr(resourceName, "slug", slug),
					resource.TestCheckResourceAttr(resourceName, "alerts_enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "installed_version"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "dashboard_folder"),
				),
			},
			// Update with only required fields (defaults back to alerts_enabled=true)
			{
				Config: fmt.Sprintf(cloudIntegrationResourceDefaultConfig, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccCloudIntegrationResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", slug),
					resource.TestCheckResourceAttr(resourceName, "slug", slug),
					resource.TestCheckResourceAttr(resourceName, "alerts_enabled", "true"),
					resource.TestCheckResourceAttrSet(resourceName, "name"),
					resource.TestCheckResourceAttrSet(resourceName, "installed_version"),
					resource.TestCheckResourceAttrSet(resourceName, "latest_version"),
					resource.TestCheckResourceAttrSet(resourceName, "dashboard_folder"),
				),
			},
		},
		// Delete
		CheckDestroy: testAccCloudIntegrationResourceCheckDestroy(ctx, slug),
	})
}

func testAccCloudIntegrationResourceExists(ctx context.Context, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", resourceName, s.RootModule().Resources)
		}

		slug, ok := resourceState.Primary.Attributes["slug"]
		if !ok {
			return fmt.Errorf("slug not set")
		}

		client := testutils.Provider.Meta().(*common.Client).CloudIntegrationsAPIClient

		integration, err := client.GetIntegration(ctx, slug)
		if err != nil {
			return fmt.Errorf("error getting integration: %v", err)
		}

		if integration.Data.Installation == nil {
			return fmt.Errorf("integration %s is not installed", slug)
		}

		return nil
	}
}

func testAccCloudIntegrationResourceCheckDestroy(ctx context.Context, slug string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).CloudIntegrationsAPIClient

		integration, err := client.GetIntegration(ctx, slug)
		if err != nil {
			// Integration not found is acceptable for destroy
			return nil
		}

		if integration.Data.Installation != nil {
			return fmt.Errorf("integration %s is still installed", slug)
		}

		return nil
	}
}
