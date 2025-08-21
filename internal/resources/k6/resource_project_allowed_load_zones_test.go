package k6_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// Private load zone cannot be created via terraform
// We can only test for sending empty allowed_load_zones
func TestAccProjectAllowedLoadZones_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var project k6.ProjectApiModel

	projectName := "Terraform Project Test Allowed Load Zones " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(`
resource "grafana_k6_project" "test_project_allowed_load_zones" {
  name = "%s"
}

resource "grafana_k6_project_allowed_load_zones" "test_allowed_zones" {
  project_id         = grafana_k6_project.test_project_allowed_load_zones.id
  allowed_load_zones = []
}
`, projectName),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test_project_allowed_load_zones", &project),
					resource.TestCheckResourceAttr("grafana_k6_project_allowed_load_zones.test_allowed_zones", "allowed_load_zones.#", "0"),
				),
			},
			// Import test
			{
				ResourceName:      "grafana_k6_project_allowed_load_zones.test_allowed_zones",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					return strconv.Itoa(int(project.GetId())), nil
				},
			},
		},
	})
}
