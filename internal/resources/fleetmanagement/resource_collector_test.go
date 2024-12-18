package fleetmanagement_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	collectorResourceRequiredConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
}
`

	collectorResourceOptionalConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	attribute_overrides = {
		"env"   = "PROD",
		"owner" = "TEAM-A"
	}
	enabled = false
}
`

	collectorResourceEmptyAttributeOverridesConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	attribute_overrides = {}
}
`
)

func TestAccCollectorResource(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_fleet_management_collector.test"
	collectorID := fmt.Sprintf("testacc_%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields
			{
				Config: fmt.Sprintf(collectorResourceRequiredConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "attribute_overrides.%"),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			// Import state with only required fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     collectorID,
				ImportStateVerify: true,
			},
			// Update with all optional fields
			{
				Config: fmt.Sprintf(collectorResourceOptionalConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.env", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.owner", "TEAM-A"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
				),
			},
			// Import state with all optional fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     collectorID,
				ImportStateVerify: true,
			},
			// Update with empty attribute_overrides field
			{
				Config: fmt.Sprintf(collectorResourceEmptyAttributeOverridesConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "attribute_overrides.%"),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
			// Update with only required fields
			{
				Config: fmt.Sprintf(collectorResourceRequiredConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "attribute_overrides.%"),
					resource.TestCheckResourceAttr(resourceName, "attribute_overrides.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
				),
			},
		},
		// Delete
		CheckDestroy: testAccCollectorResourceCheckDestroy(ctx, collectorID),
	})
}

func testAccCollectorResourceExists(ctx context.Context, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", resourceName, s.RootModule().Resources)
		}

		collectorID, ok := resourceState.Primary.Attributes["id"]
		if !ok {
			return fmt.Errorf("collector ID not set")
		}

		client := testutils.Provider.Meta().(*common.Client).FleetManagementClient.CollectorServiceClient

		getReq := &collectorv1.GetCollectorRequest{
			Id: collectorID,
		}
		_, err := client.GetCollector(ctx, connect.NewRequest(getReq))
		if err != nil {
			return fmt.Errorf("error getting collector: %v", err)
		}

		return nil
	}
}

func testAccCollectorResourceCheckDestroy(ctx context.Context, collectorID string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).FleetManagementClient.CollectorServiceClient

		getReq := &collectorv1.GetCollectorRequest{
			Id: collectorID,
		}
		_, err := client.GetCollector(ctx, connect.NewRequest(getReq))
		if err == nil {
			return errors.New("collector still exists")
		}
		if connect.CodeOf(err) != connect.CodeNotFound {
			return fmt.Errorf("unexpected error retrieving collector: %s", err)
		}

		return nil
	}
}
