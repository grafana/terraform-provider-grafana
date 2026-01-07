package fleetmanagement_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	collectorv1 "github.com/grafana/fleet-management-api/api/gen/proto/go/collector/v1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	collectorResourceAlloyDefaultConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
}
`

	collectorResourceAlloyOptionalConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	remote_attributes = {
		"env"   = "PROD",
		"owner" = "TEAM-A"
	}
	enabled        = false
	collector_type = "ALLOY"
}
`

	collectorResourceAlloyEmptyRemoteAttributesConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	remote_attributes = {}
}
`

	collectorResourceOtelRequiredConfig = `
resource "grafana_fleet_management_collector" "test" {
	id             = "%s"
	collector_type = "OTEL"
}
`

	collectorResourceOtelOptionalConfig = `
resource "grafana_fleet_management_collector" "test" {
	id = "%s"
	remote_attributes = {
		"env"   = "PROD",
		"owner" = "TEAM-A"
	}
	enabled        = false
	collector_type = "OTEL"
}
`

	collectorResourceOtelEmptyRemoteAttributesConfig = `
resource "grafana_fleet_management_collector" "test" {
	id             = "%s"
	remote_attributes = {}
	collector_type = "OTEL"
}
`
)

func TestAccCollectorResourceAlloy(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_fleet_management_collector.test"
	collectorID := fmt.Sprintf("testacc_%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields
			{
				Config: fmt.Sprintf(collectorResourceAlloyDefaultConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "ALLOY"),
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
				Config: fmt.Sprintf(collectorResourceAlloyOptionalConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.env", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.owner", "TEAM-A"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "ALLOY"),
				),
			},
			// Import state with all optional fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     collectorID,
				ImportStateVerify: true,
			},
			// Update with empty remote_attributes field
			{
				Config: fmt.Sprintf(collectorResourceAlloyEmptyRemoteAttributesConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "ALLOY"),
				),
			},
			// Update with only required fields
			{
				Config: fmt.Sprintf(collectorResourceAlloyDefaultConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "ALLOY"),
				),
			},
		},
		// Delete
		CheckDestroy: testAccCollectorResourceCheckDestroy(ctx, collectorID),
	})
}

func TestAccCollectorResourceOtel(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_fleet_management_collector.test"
	collectorID := uuid.NewString()

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields
			{
				Config: fmt.Sprintf(collectorResourceOtelRequiredConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "OTEL"),
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
				Config: fmt.Sprintf(collectorResourceOtelOptionalConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "2"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.env", "PROD"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.owner", "TEAM-A"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "OTEL"),
				),
			},
			// Import state with all optional fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     collectorID,
				ImportStateVerify: true,
			},
			// Update with empty remote_attributes field
			{
				Config: fmt.Sprintf(collectorResourceOtelEmptyRemoteAttributesConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "OTEL"),
				),
			},
			// Update with only required fields
			{
				Config: fmt.Sprintf(collectorResourceOtelRequiredConfig, collectorID),
				Check: resource.ComposeTestCheckFunc(
					testAccCollectorResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "id", collectorID),
					resource.TestCheckResourceAttrSet(resourceName, "remote_attributes.%"),
					resource.TestCheckResourceAttr(resourceName, "remote_attributes.%", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "collector_type", "OTEL"),
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
