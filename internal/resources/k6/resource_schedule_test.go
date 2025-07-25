package k6_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccSchedule_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
		schedule k6.ScheduleApiModel
	)

	projectName := "Terraform Schedule Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Schedule Test Load Test " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			scheduleCheckExists.destroyed(&schedule),
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testScheduleConfigBasic(projectName, loadTestName),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test", &project),
					loadTestCheckExists.exists("grafana_k6_load_test.test", &loadTest),
					scheduleCheckExists.exists("grafana_k6_schedule.test", &schedule),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "interval", "1"),
					resource.TestMatchResourceAttr("grafana_k6_schedule.test", "id", defaultIDRegexp),
					resource.TestCheckResourceAttrSet("grafana_k6_schedule.test", "load_test_id"),
					resource.TestCheckResourceAttrSet("grafana_k6_schedule.test", "starts"),
					testutils.CheckLister("grafana_k6_schedule.test"),
				),
			},
			{
				ResourceName:      "grafana_k6_schedule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete the schedule and check that TF sees a difference
			{
				PreConfig: func() {
					commonClient := testutils.Provider.Meta().(*common.Client)
					client := commonClient.K6APIClient
					config := commonClient.K6APIConfig

					ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
					deleteReq := client.SchedulesAPI.SchedulesDestroy(ctx, schedule.Id).XStackId(config.StackID)

					_, err := deleteReq.Execute()
					if err != nil {
						t.Fatalf("error deleting schedule: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Recreate the schedule
			{
				Config: testScheduleConfigBasic(projectName, loadTestName),
				Check: resource.ComposeAggregateTestCheckFunc(
					scheduleCheckExists.exists("grafana_k6_schedule.test", &schedule),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "interval", "1"),
				),
			},
		},
	})
}

func TestAccSchedule_update(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
		schedule k6.ScheduleApiModel
	)

	projectName := "Terraform Schedule Update Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Schedule Update Test Load Test " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			scheduleCheckExists.destroyed(&schedule),
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testScheduleConfigBasic(projectName, loadTestName),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.test", &project),
					loadTestCheckExists.exists("grafana_k6_load_test.test", &loadTest),
					scheduleCheckExists.exists("grafana_k6_schedule.test", &schedule),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "interval", "1"),
				),
			},
			// Update the schedule frequency and interval
			{
				Config: testScheduleConfigUpdated(projectName, loadTestName),
				Check: resource.ComposeTestCheckFunc(
					testAccScheduleWasntRecreated("grafana_k6_schedule.test", &schedule),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "frequency", "WEEKLY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "interval", "2"),
				),
			},
		},
	})
}

func testAccScheduleWasntRecreated(rn string, oldSchedule *k6.ScheduleApiModel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		newScheduleResource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("schedule not found: %s", rn)
		}
		if newScheduleResource.Primary.ID == "" {
			return fmt.Errorf("schedule id not set")
		}
		var newScheduleID int32
		if scheduleID, err := strconv.Atoi(newScheduleResource.Primary.ID); err != nil {
			return fmt.Errorf("could not convert schedule id to integer: %s", err.Error())
		} else if newScheduleID, err = common.ToInt32(scheduleID); err != nil {
			return fmt.Errorf("could not convert schedule id to int32: %s", err.Error())
		}

		if oldSchedule.GetId() != newScheduleID {
			return fmt.Errorf("schedule was recreated: old id %d, new id %d", oldSchedule.GetId(), newScheduleID)
		}

		return nil
	}
}

func testScheduleConfigBasic(projectName, loadTestName string) string {
	return fmt.Sprintf(`
resource "grafana_k6_project" "test" {
  name = "%s"
}

resource "grafana_k6_load_test" "test" {
  name = "%s"
  project_id = grafana_k6_project.test.id
  script = "export default function() { console.log('Hello, k6!'); }"
}

resource "grafana_k6_schedule" "test" {
  load_test_id = grafana_k6_load_test.test.id
  starts = "2024-12-25T10:00:00Z"
  frequency = "DAILY"
  interval = 1
}
`, projectName, loadTestName)
}

func testScheduleConfigUpdated(projectName, loadTestName string) string {
	return fmt.Sprintf(`
resource "grafana_k6_project" "test" {
  name = "%s"
}

resource "grafana_k6_load_test" "test" {
  name = "%s"
  project_id = grafana_k6_project.test.id
  script = "export default function() { console.log('Hello, k6!'); }"
}

resource "grafana_k6_schedule" "test" {
  load_test_id = grafana_k6_load_test.test.id
  starts = "2024-12-25T10:00:00Z"
  frequency = "WEEKLY"
  interval = 2
}
`, projectName, loadTestName)
}
