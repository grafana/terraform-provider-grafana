package fleetmanagement_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"connectrpc.com/connect"
	pipelinev1 "github.com/grafana/fleet-management-api/api/gen/proto/go/pipeline/v1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	pipelineResourceAlloyRequiredConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
}
`

	pipelineResourceAlloyOptionalConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
	matchers = [
		"collector.os=\"linux\"",
		"owner=\"TEAM-A\"",
	]
	enabled     = false
	config_type = "ALLOY"
}
`

	pipelineResourceAlloySemanticallyEqualContentsConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }\n"
}
`

	pipelineResourceAlloySemanticallyEqualMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
	matchers = [
		"collector.os=linux",
		"owner=TEAM-A",
	]
}
`

	pipelineResourceAlloyVariableMatchersConfig = `
variable "os" {
	type    = string
	default = "linux"
}

resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
	matchers = [
		"collector.os=\"${var.os}\"",
		"owner=\"TEAM-A\"",
	]
}
`

	pipelineResourceAlloyUnorderedMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
	matchers = [
		"owner=\"TEAM-A\"",
		"collector.os=\"linux\"",
	]
}
`

	pipelineResourceAlloyEmptyMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = "prometheus.exporter.self \"alloy\" { }"
	matchers = []
}
`

	pipelineResourceOtelRequiredConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name        = "%s"
	contents    = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	config_type = "OTEL"
}
`

	pipelineResourceOtelOptionalConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	matchers = [
		"collector.os=\"linux\"",
		"owner=\"TEAM-A\"",
	]
	enabled     = false
	config_type = "OTEL"
}
`

	pipelineResourceOtelSemanticallyEqualContentsConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name        = "%s"
	contents    = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	config_type = "OTEL"
}
`

	pipelineResourceOtelSemanticallyEqualMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	matchers = [
		"collector.os=linux",
		"owner=TEAM-A",
	]
	config_type = "OTEL"
}
`

	pipelineResourceOtelVariableMatchersConfig = `
variable "os" {
	type    = string
	default = "linux"
}

resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	matchers = [
		"collector.os=\"${var.os}\"",
		"owner=\"TEAM-A\"",
	]
	config_type = "OTEL"
}
`

	pipelineResourceOtelUnorderedMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name     = "%s"
	contents = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	matchers = [
		"owner=\"TEAM-A\"",
		"collector.os=\"linux\"",
	]
	config_type = "OTEL"
}
`

	pipelineResourceOtelEmptyMatchersConfig = `
resource "grafana_fleet_management_pipeline" "test" {
	name        = "%s"
	contents    = <<-EOT
exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
EOT
	matchers    = []
	config_type = "OTEL"
}
`
)

func TestAccPipelineResourceAlloy(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_fleet_management_pipeline.test"
	pipelineName := fmt.Sprintf("testacc_%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields
			{
				Config: fmt.Sprintf(pipelineResourceAlloyRequiredConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import state with only required fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     pipelineName,
				ImportStateVerify: true,
			},
			// Update with all optional fields
			{
				Config: fmt.Sprintf(pipelineResourceAlloyOptionalConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import state with all optional fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     pipelineName,
				ImportStateVerify: true,
			},
			// Update with semantically equal contents field
			{
				Config: fmt.Sprintf(pipelineResourceAlloySemanticallyEqualContentsConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }\n"),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with semantically equal matchers field
			{
				Config: fmt.Sprintf(pipelineResourceAlloySemanticallyEqualMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=linux"),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=TEAM-A"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with matcher value from variable
			{
				Config: fmt.Sprintf(pipelineResourceAlloyVariableMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with unordered matchers field
			{
				Config: fmt.Sprintf(pipelineResourceAlloyUnorderedMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with empty matchers field
			{
				Config: fmt.Sprintf(pipelineResourceAlloyEmptyMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with only required fields
			{
				Config: fmt.Sprintf(pipelineResourceAlloyRequiredConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", "prometheus.exporter.self \"alloy\" { }"),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "ALLOY"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
		// Delete
		CheckDestroy: testAccPipelineResourceCheckDestroy(ctx, pipelineName),
	})
}

func TestAccPipelineResourceOtel(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	ctx := context.Background()
	resourceName := "grafana_fleet_management_pipeline.test"
	pipelineName := fmt.Sprintf("testacc_%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create with only required fields
			{
				Config: fmt.Sprintf(pipelineResourceOtelRequiredConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import state with only required fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     pipelineName,
				ImportStateVerify: true,
			},
			// Update with all optional fields
			{
				Config: fmt.Sprintf(pipelineResourceOtelOptionalConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "false"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Import state with all optional fields
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateId:     pipelineName,
				ImportStateVerify: true,
			},
			// Update with semantically equal contents field
			{
				Config: fmt.Sprintf(pipelineResourceOtelSemanticallyEqualContentsConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with semantically equal matchers field
			{
				Config: fmt.Sprintf(pipelineResourceOtelSemanticallyEqualMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=linux"),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=TEAM-A"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with matcher value from variable
			{
				Config: fmt.Sprintf(pipelineResourceOtelVariableMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with unordered matchers field
			{
				Config: fmt.Sprintf(pipelineResourceOtelUnorderedMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "2"),
					resource.TestCheckResourceAttr(resourceName, "matchers.0", "owner=\"TEAM-A\""),
					resource.TestCheckResourceAttr(resourceName, "matchers.1", "collector.os=\"linux\""),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with empty matchers field
			{
				Config: fmt.Sprintf(pipelineResourceOtelEmptyMatchersConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
			// Update with only required fields
			{
				Config: fmt.Sprintf(pipelineResourceOtelRequiredConfig, pipelineName),
				Check: resource.ComposeTestCheckFunc(
					testAccPipelineResourceExists(ctx, resourceName),
					resource.TestCheckResourceAttr(resourceName, "name", pipelineName),
					resource.TestCheckResourceAttr(resourceName, "contents", `exporters:
  debug:
    verbosity: detailed
receivers:
  otlp:
    protocols:
      grpc:
        endpoint: 0.0.0.0:4317
service:
  pipelines:
    traces:
      exporters:
        - debug
      receivers:
        - otlp
`),
					resource.TestCheckResourceAttrSet(resourceName, "matchers.#"),
					resource.TestCheckResourceAttr(resourceName, "matchers.#", "0"),
					resource.TestCheckResourceAttr(resourceName, "enabled", "true"),
					resource.TestCheckResourceAttr(resourceName, "config_type", "OTEL"),
					resource.TestCheckResourceAttrSet(resourceName, "id"),
				),
			},
		},
		// Delete
		CheckDestroy: testAccPipelineResourceCheckDestroy(ctx, pipelineName),
	})
}

func testAccPipelineResourceExists(ctx context.Context, resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceState, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", resourceName, s.RootModule().Resources)
		}

		pipelineID, ok := resourceState.Primary.Attributes["id"]
		if !ok {
			return fmt.Errorf("pipeline ID not set")
		}

		client := testutils.Provider.Meta().(*common.Client).FleetManagementClient.PipelineServiceClient

		getReq := &pipelinev1.GetPipelineRequest{
			Id: pipelineID,
		}
		_, err := client.GetPipeline(ctx, connect.NewRequest(getReq))
		if err != nil {
			return fmt.Errorf("error getting pipeline: %v", err)
		}

		return nil
	}
}

func testAccPipelineResourceCheckDestroy(ctx context.Context, pipelineName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).FleetManagementClient.PipelineServiceClient

		getIDReq := &pipelinev1.GetPipelineIDRequest{
			Name: pipelineName,
		}
		_, err := client.GetPipelineID(ctx, connect.NewRequest(getIDReq))
		if err == nil {
			return errors.New("pipeline still exists")
		}
		if connect.CodeOf(err) != connect.CodeNotFound {
			return fmt.Errorf("unexpected error retrieving pipeline: %s", err)
		}

		return nil
	}
}
