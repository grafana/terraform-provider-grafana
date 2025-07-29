package k6_test

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccDataSourceK6Schedules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var loadTest k6.LoadTestApiModel

	projectName := "Terraform Schedules Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Test Load Test for Schedules " + acctest.RandString(8)

	checkLoadTestIDMatch := func(value string) error {
		if value != strconv.Itoa(int(loadTest.GetId())) {
			return fmt.Errorf("load_test_id does not match the expected value: %s", value)
		}
		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_schedules/data-source.tf", map[string]string{
					"Terraform Schedules Test Project":       projectName,
					"Terraform Test Load Test for Schedules": loadTestName,
				}),
				Check: resource.ComposeTestCheckFunc(
					loadTestCheckExists.exists("grafana_k6_load_test.schedules_load_test", &loadTest),
					// Data source attributes
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "id"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedules.from_load_test_id", "load_test_id", checkLoadTestIDMatch),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedules.from_load_test_id", "id", checkLoadTestIDMatch),
					// Should have at least 2 schedules
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.#", "2"),
					// First schedule attributes
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.0.id"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedules.from_load_test_id", "schedules.0.load_test_id", checkLoadTestIDMatch),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.0.starts", "2024-12-25T10:00:00Z"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.0.recurrence_rule.frequency", "DAILY"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.0.recurrence_rule.interval", "1"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.0.recurrence_rule.count", "5"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.0.deactivated"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.0.created_by"),
					// Second schedule attributes
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.1.id"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedules.from_load_test_id", "schedules.1.load_test_id", checkLoadTestIDMatch),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.1.starts", "2024-12-26T14:00:00Z"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.1.recurrence_rule.frequency", "WEEKLY"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.1.recurrence_rule.interval", "2"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedules.from_load_test_id", "schedules.1.recurrence_rule.until", "2025-01-31T23:59:59Z"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.1.deactivated"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "schedules.1.created_by"),
				),
			},
		},
	})
}
