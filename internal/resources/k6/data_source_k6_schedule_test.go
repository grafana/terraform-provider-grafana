package k6_test

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccDataSourceK6Schedule_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var schedule k6.ScheduleApiModel

	projectName := "Terraform Schedule Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Test Load Test for Schedule " + acctest.RandString(8)

	checkScheduleIDMatch := func(value string) error {
		if value != strconv.Itoa(int(schedule.GetId())) {
			return fmt.Errorf("schedule id does not match the expected value: %s", value)
		}
		return nil
	}

	checkLoadTestIDMatch := func(value string) error {
		if value != strconv.Itoa(int(schedule.GetLoadTestId())) {
			return fmt.Errorf("load_test_id does not match the expected value: %s", value)
		}
		return nil
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_schedule/data-source.tf", map[string]string{
					"Terraform Schedule Test Project":       projectName,
					"Terraform Test Load Test for Schedule": loadTestName,
				}),
				Check: resource.ComposeTestCheckFunc(
					scheduleCheckExists.exists("grafana_k6_schedule.test_schedule", &schedule),
					// Basic attributes
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedule.from_load_test", "id"),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedule.from_load_test", "id", checkScheduleIDMatch),
					resource.TestCheckResourceAttrWith("data.grafana_k6_schedule.from_load_test", "load_test_id", checkLoadTestIDMatch),
					// Schedule configuration attributes
					resource.TestCheckResourceAttr("data.grafana_k6_schedule.from_load_test", "starts", "2024-12-25T10:00:00Z"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedule.from_load_test", "recurrence_rule.frequency", "MONTHLY"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedule.from_load_test", "recurrence_rule.interval", "12"),
					resource.TestCheckResourceAttr("data.grafana_k6_schedule.from_load_test", "recurrence_rule.count", "100"),
					// Optional attributes that should be set
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedule.from_load_test", "deactivated"),
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedule.from_load_test", "created_by"),
					// until and next_run are optional and may be null
				),
			},
		},
	})
}

func TestAccDataSourceK6Schedule_nonexistent(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "grafana_k6_schedule" "nonexistent" {
  load_test_id = "999999"
}
`,
				ExpectError: regexp.MustCompile(`Error reading k6 schedule`),
			},
		},
	})
}

func TestAccDataSourceK6Schedule_invalidID(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `
data "grafana_k6_schedule" "invalid" {
  load_test_id = "not-a-number"
}
`,
				ExpectError: regexp.MustCompile(`Error parsing load test ID`),
			},
		},
	})
}
