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

func TestAccDataSourceK6Schedules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var loadTest1, loadTest2 k6.LoadTestApiModel

	projectName := "Terraform Schedules Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Test Load Test for Schedules " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "data-sources/grafana_k6_schedules/data-source.tf", map[string]string{
					"Terraform Schedules Test Project":       projectName,
					"Terraform Test Load Test for Schedules": loadTestName,
				}),
				Check: resource.ComposeTestCheckFunc(
					loadTestCheckExists.exists("grafana_k6_load_test.schedules_load_test", &loadTest1),
					loadTestCheckExists.exists("grafana_k6_load_test.schedules_load_test_2", &loadTest2),
					// Data source attributes
					resource.TestCheckResourceAttrSet("data.grafana_k6_schedules.from_load_test_id", "id"),
					// Should have at least 2 schedules (the ones we created)
					func(s *terraform.State) error {
						rs, ok := s.RootModule().Resources["data.grafana_k6_schedules.from_load_test_id"]
						if !ok {
							return fmt.Errorf("data source not found")
						}

						schedulesCount := rs.Primary.Attributes["schedules.#"]
						count, err := strconv.Atoi(schedulesCount)
						if err != nil {
							return fmt.Errorf("invalid schedules count: %s", schedulesCount)
						}

						if count < 2 {
							return fmt.Errorf("expected at least 2 schedules, got %d", count)
						}

						// Check that our created schedules are in the list
						loadTest1ID := strconv.Itoa(int(loadTest1.GetId()))
						loadTest2ID := strconv.Itoa(int(loadTest2.GetId()))

						foundLoadTest1Schedule := false
						foundLoadTest2Schedule := false

						for i := 0; i < count; i++ {
							scheduleLoadTestID := rs.Primary.Attributes[fmt.Sprintf("schedules.%d.load_test_id", i)]
							if scheduleLoadTestID == loadTest1ID {
								foundLoadTest1Schedule = true
								// Validate specific attributes for load test 1 schedule
								starts := rs.Primary.Attributes[fmt.Sprintf("schedules.%d.starts", i)]
								if starts != "2029-12-25T10:00:00Z" {
									return fmt.Errorf("expected starts to be 2029-12-25T10:00:00Z, got %s", starts)
								}
								frequency := rs.Primary.Attributes[fmt.Sprintf("schedules.%d.recurrence_rule.frequency", i)]
								if frequency != "MONTHLY" {
									return fmt.Errorf("expected frequency to be MONTHLY, got %s", frequency)
								}
							} else if scheduleLoadTestID == loadTest2ID {
								foundLoadTest2Schedule = true
								// Validate specific attributes for load test 2 schedule
								starts := rs.Primary.Attributes[fmt.Sprintf("schedules.%d.starts", i)]
								if starts != "2023-12-26T14:00:00Z" {
									return fmt.Errorf("expected starts to be 2023-12-26T14:00:00Z, got %s", starts)
								}
								frequency := rs.Primary.Attributes[fmt.Sprintf("schedules.%d.recurrence_rule.frequency", i)]
								if frequency != "WEEKLY" {
									return fmt.Errorf("expected frequency to be WEEKLY, got %s", frequency)
								}
							}
						}

						if !foundLoadTest1Schedule {
							return fmt.Errorf("schedule for load test 1 (%s) not found", loadTest1ID)
						}
						if !foundLoadTest2Schedule {
							return fmt.Errorf("schedule for load test 2 (%s) not found", loadTest2ID)
						}

						return nil
					},
				),
			},
		},
	})
}
