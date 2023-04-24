package slo_test

import (
	"fmt"
	"regexp"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccResourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var slo gapi.Slo
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo_resource.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo_resource.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_slo_resource.test", "dashboard_uid"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "name", "Terraform Testing"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "objectives.0.objective_value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "objectives.0.objective_window", "30d"),
				),
			},
			{
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource_update.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo_resource.update", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo_resource.update", "id"),
					resource.TestCheckResourceAttrSet("grafana_slo_resource.update", "dashboard_uid"),
					resource.TestCheckResourceAttr("grafana_slo_resource.update", "name", "Updated - Terraform Testing"),
					resource.TestCheckResourceAttr("grafana_slo_resource.update", "description", "Updated - Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo_resource.update", "query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo_resource.update", "objectives.0.objective_value", "0.9995"),
					resource.TestCheckResourceAttr("grafana_slo_resource.update", "objectives.0.objective_window", "7d"),
				),
			},
		},
	})
}

func testAccSloCheckExists(rn string, slo *gapi.Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		gotSlo, err := client.GetSlo(rs.Primary.ID)

		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}

		*slo = gotSlo

		return nil
	}
}

func testAccSloCheckDestroy(slo *gapi.Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI
		err := client.DeleteSlo(slo.UUID)

		if err == nil {
			return fmt.Errorf("SLO with a UUID %s still exists after destroy", slo.UUID)
		}

		return nil
	}
}

const sloQueryInvalid = `
resource  "grafana_slo_resource" "invalid" {
  name            = "Test SLO"
  description     = "Description Test SLO"
  query           = "Invalid Query"
  objectives {
	objective_value  = 0.995
    objective_window = "30d"
  }
}
`

const sloObjectivesInvalid = `
resource  "grafana_slo_resource" "invalid" {
  name            = "Test SLO"
  description     = "Description Test SLO"
  query           = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
  objectives {
	objective_value  = 1.5
    objective_window = "1m"
  }
}
`

func TestAccResourceInvalidSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      sloQueryInvalid,
				ExpectError: regexp.MustCompile("Unable to create SLO"),
			},
			{
				Config:      sloObjectivesInvalid,
				ExpectError: regexp.MustCompile("Unable to create SLO"),
			},
		},
	})
}
