package slo_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccResourceSlo(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// var job mlapi.Job - TBD
	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		// CheckDestroy:      testAccMLJobCheckDestroy(&job), - TBD
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_slo/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					// testAccMLJobCheckExists("grafana_machine_learning_job.test_job", &job), - TBD
					resource.TestCheckResourceAttrSet("grafana_slo_resource.test", "id"),
					resource.TestCheckResourceAttrSet("grafana_slo_resource.test", "dashboard_uid"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "name", "Terraform Testing"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "description", "Terraform Description"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "service", "serviceA"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "query", "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "objectives.0.objective_value", "0.995"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "objectives.0.objective_window", "30d"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "labels.0.key", "custom"),
					resource.TestCheckResourceAttr("grafana_slo_resource.test", "labels.0.value", "value"),
				),
			},
		},
	})
}
