package slo_test

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	"github.com/stretchr/testify/require"
)

func TestAccResourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	randomName := acctest.RandomWithPrefix("SLO Terraform Testing")

	var slo slo.SloV00Slo
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// Implicitly tests destroy
		CheckDestroy: testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			{
				// Tests Create
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.window", "30d"),
					resource.TestCheckNoResourceAttr("grafana_slo.test", "folder_uid"),
					testutils.CheckLister("grafana_slo.test"),
				),
			},
			{
				// Tests Update
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource_update.tf", map[string]string{
					"Terraform Testing": randomName,
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", "Updated - "+randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Updated - Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.value", "0.9995"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.window", "7d"),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "folder_uid"),
				),
			},
			{
				// Tests that No Alerting Rules are Generated when No Alerting Field is defined on the Terraform State File
				Config: noAlert(randomName + " - No Alerting Check"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.no_alert", &slo),
					testAlertingExists(false, "grafana_slo.no_alert", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.no_alert", "id"),
					resource.TestCheckResourceAttr("grafana_slo.no_alert", "name", randomName+" - No Alerting Check"),
				),
			},
			{
				// Tests that Alerting Rules are Generated when an Empty Alerting Field is defined on the Terraform State File
				Config: emptyAlert(randomName + " - Empty Alerting Check"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.empty_alert", &slo),
					testAlertingExists(true, "grafana_slo.empty_alert", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.empty_alert", "id"),
					resource.TestCheckResourceAttr("grafana_slo.empty_alert", "name", randomName+" - Empty Alerting Check"),
				),
			},
			{
				// Tests Create
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource_ratio.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.ratio", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.ratio", "id"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "name", "Terraform Testing - Ratio Query"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "description", "Terraform Description - Ratio Query"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.type", "ratio"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.success_metric", "kubelet_http_requests_total{status!~\"5..\"}"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.total_metric", "kubelet_http_requests_total"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.group_by_labels.0", "job"),
					resource.TestCheckResourceAttr("grafana_slo.ratio", "query.0.ratio.0.group_by_labels.1", "instance"),
				),
			},
			{
				// Import test (this tests that all fields are read correctly)
				ResourceName:      "grafana_slo.ratio",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Tests Advanced Options
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource_ratio_advanced_options.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.ratio_options", &slo),
					testAlertingExists(true, "grafana_slo.ratio_options", &slo),
					testAdvancedOptionsExists(true, "grafana_slo.ratio_options", &slo),
					resource.TestCheckResourceAttr("grafana_slo.ratio_options", "alerting.0.advanced_options.0.min_failures", "10"),
				),
			},
			{
				// Import test (this tests that all fields are read correctly)
				ResourceName:      "grafana_slo.ratio_options",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Tests the Search Expression
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource_search_expression.tf"),
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.search_expression", &slo),
					resource.TestCheckResourceAttr("grafana_slo.search_expression", "search_expression", "Entity Search for RCA Workbench"),
				),
			},
			{
				// Import test (this tests that all fields are read correctly)
				ResourceName:      "grafana_slo.search_expression",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

// Tests that recreating an out-of-band deleted SLO works without error.
func TestAccSLO_recreate(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	var slo slo.SloV00Slo
	randomName := acctest.RandomWithPrefix("SLO Terraform Testing")
	config := testutils.TestAccExampleWithReplace(t, "resources/grafana_slo/resource.tf", map[string]string{
		"Terraform Testing": randomName,
	})
	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,

		// Implicitly tests destroy
		CheckDestroy: testAccSloCheckDestroy(&slo),
		Steps: []resource.TestStep{
			// Create
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.window", "30d"),
					resource.TestCheckNoResourceAttr("grafana_slo.test", "folder_uid"),
					testutils.CheckLister("grafana_slo.test"),
				),
			},
			// Delete out-of-band
			{
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client).SLOClient
					req := client.DefaultAPI.V1SloIdDelete(context.Background(), slo.Uuid)
					_, err := req.Execute()
					require.NoError(t, err)
					// A short delay while background tasks clean up the SLO. After this, the plan should be non-empty again.
					time.Sleep(5 * time.Second)
				},
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
			// Re-create
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccSloCheckExists("grafana_slo.test", &slo),
					resource.TestCheckResourceAttrSet("grafana_slo.test", "id"),
					resource.TestCheckResourceAttr("grafana_slo.test", "name", randomName),
					resource.TestCheckResourceAttr("grafana_slo.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.type", "freeform"),
					resource.TestCheckResourceAttr("grafana_slo.test", "query.0.freeform.0.query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo.test", "objectives.0.window", "30d"),
					resource.TestCheckNoResourceAttr("grafana_slo.test", "folder_uid"),
					testutils.CheckLister("grafana_slo.test"),
				),
			},
		},
	})
}

func testAccSloCheckExists(rn string, slo *slo.SloV00Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID)
		gotSlo, _, err := req.Execute()
		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}

		if *gotSlo.ReadOnly.Provenance != "terraform" {
			return fmt.Errorf("provenance header missing - verify within the Grafana Terraform Provider that the 'Grafana-Terraform-Provider' request header is set to 'true'")
		}

		*slo = *gotSlo

		return nil
	}
}

func testAlertingExists(expectation bool, rn string, slo *slo.SloV00Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[rn]
		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID)
		gotSlo, _, err := req.Execute()

		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}
		*slo = *gotSlo

		if slo.Alerting == nil && expectation == false {
			return nil
		}

		if slo.Alerting != nil && expectation == true {
			return nil
		}

		return fmt.Errorf("SLO Alerting expectation mismatch")
	}
}

func testAdvancedOptionsExists(expectation bool, rn string, slo *slo.SloV00Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[rn]
		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID)
		gotSlo, _, err := req.Execute()

		if err != nil {
			return fmt.Errorf("error getting SLO: %s", err)
		}
		*slo = *gotSlo

		if slo.Alerting.AdvancedOptions == nil && expectation == false {
			return nil
		}

		if slo.Alerting.AdvancedOptions != nil && expectation == true {
			return nil
		}

		return fmt.Errorf("SLO Advanced Options expectation mismatch")
	}
}

func testAccSloCheckDestroy(sloObj *slo.SloV00Slo) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).SLOClient
		req := client.DefaultAPI.V1SloIdGet(context.Background(), sloObj.Uuid)
		gotSlo, resp, err := req.Execute()
		if err != nil {
			var oapiErr slo.GenericOpenAPIError
			if errors.As(err, &oapiErr) && strings.Contains(oapiErr.Error(), "404 Not Found") {
				return nil
			}

			return fmt.Errorf("error getting SLO: %s", err)
		}

		if resp.StatusCode == 404 {
			return nil
		}

		sloType := gotSlo.ReadOnly.Status.Type
		message := gotSlo.ReadOnly.Status.GetMessage()

		if sloType == "deleting" {
			return nil
		}

		// Rule group is limited in the instance, and sometimes it makes Cloud tests to fail...
		if sloType == "error" && strings.Contains(message, "rule group limit exceeded") {
			return nil
		}

		return fmt.Errorf("grafana SLO still exists: %+v, status type: %+v, status message: %s", gotSlo, gotSlo.ReadOnly.Status.GetType(), gotSlo.ReadOnly.Status.GetMessage())
	}
}

const sloObjectivesInvalid = `
resource  "grafana_slo" "invalid" {
  name            = "Test SLO"
  description     = "Description Test SLO"
  query {
	freeform {
		query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
	}
    type = "freeform"
  }
  destination_datasource {
	uid = "grafanacloud-prom"
  }
  objectives {
	value  = 1.5
    window = "1m"
  }
}
`

const sloMissingDestinationDatasource = `
resource  "grafana_slo" "invalid" {
  name            = "Test SLO"
  description     = "Description Test SLO"
  query {
	freeform {
		query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
	}
    type = "freeform"
  }
  objectives {
	value  = 1.5
    window = "1m"
  }
}
`

func emptyAlert(name string) string {
	return fmt.Sprintf(`
	resource "grafana_slo" "empty_alert" {
	  description = "%[1]s"
	  name        = "%[1]s"
	  objectives {
		value  = 0.995
		window = "28d"
	  }
	  destination_datasource {
		uid = "grafanacloud-prom"
	  }
	  query {
		type = "freeform"
		freeform {
		  query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
		}
	  }
	  alerting {}
	}
	`, name)
}

func noAlert(name string) string {
	return fmt.Sprintf(`
resource "grafana_slo" "no_alert" {
	description = "%[1]s"
	name        = "%[1]s"
  objectives {
    value  = 0.995
    window = "28d"
  }
  destination_datasource {
	uid = "grafanacloud-prom"
  }
  query {
    type = "freeform"
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
  }
}
`, name)
}

func TestAccResourceInvalidSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      sloObjectivesInvalid,
				ExpectError: regexp.MustCompile("Error:"),
			},
			{
				Config:      sloMissingDestinationDatasource,
				ExpectError: regexp.MustCompile("Error: Insufficient destination_datasource blocks"),
			},
		},
	})
}
