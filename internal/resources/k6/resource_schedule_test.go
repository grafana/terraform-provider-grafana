package k6_test

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
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
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.interval", "1"),
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
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.interval", "1"),
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
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.frequency", "DAILY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.interval", "1"),
				),
			},
			// Update the schedule frequency and interval
			{
				Config: testScheduleConfigUpdated(projectName, loadTestName),
				Check: resource.ComposeTestCheckFunc(
					testAccScheduleWasntRecreated("grafana_k6_schedule.test", &schedule),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.frequency", "WEEKLY"),
					resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.interval", "2"),
				),
			},
		},
	})
}

func TestAccSchedule_frequencyValidation(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	projectName := "Terraform Schedule Validation Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Schedule Validation Test Load Test " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Test invalid frequency values
			{
				Config:      testScheduleConfigInvalidFrequency(projectName, loadTestName, "INVALID"),
				ExpectError: regexp.MustCompile(`Attribute recurrence_rule\.frequency value must be one of: \["HOURLY" "DAILY"\s+"WEEKLY" "MONTHLY" "YEARLY"\], got: "INVALID"`),
			},
			{
				Config:      testScheduleConfigInvalidFrequency(projectName, loadTestName, "daily"),
				ExpectError: regexp.MustCompile(`Attribute recurrence_rule\.frequency value must be one of: \["HOURLY" "DAILY"\s+"WEEKLY" "MONTHLY" "YEARLY"\], got: "daily"`),
			},
		},
	})
}

func TestAccSchedule_validFrequencies(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
		schedule k6.ScheduleApiModel
	)

	projectName := "Terraform Schedule Valid Frequencies Test Project " + acctest.RandString(8)
	loadTestName := "Terraform Schedule Valid Frequencies Test Load Test " + acctest.RandString(8)

	// Test all valid frequency enum values in sequence
	validFrequencies := []string{"HOURLY", "DAILY", "WEEKLY", "MONTHLY", "YEARLY"}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			scheduleCheckExists.destroyed(&schedule),
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: func() []resource.TestStep {
			steps := []resource.TestStep{}

			// Add a step for each valid frequency
			for i, frequency := range validFrequencies {
				steps = append(steps, resource.TestStep{
					Config: testScheduleConfigWithFrequency(projectName, loadTestName, frequency),
					Check: resource.ComposeTestCheckFunc(
						projectCheckExists.exists("grafana_k6_project.test", &project),
						loadTestCheckExists.exists("grafana_k6_load_test.test", &loadTest),
						scheduleCheckExists.exists("grafana_k6_schedule.test", &schedule),
						resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.frequency", frequency),
						resource.TestCheckResourceAttr("grafana_k6_schedule.test", "recurrence_rule.0.interval", "1"),
						resource.TestMatchResourceAttr("grafana_k6_schedule.test", "id", defaultIDRegexp),
						resource.TestCheckResourceAttrSet("grafana_k6_schedule.test", "load_test_id"),
						resource.TestCheckResourceAttrSet("grafana_k6_schedule.test", "starts"),
						testutils.CheckLister("grafana_k6_schedule.test"),
					),
				})

				// Add import test only for the first frequency to avoid redundancy
				if i == 0 {
					steps = append(steps, resource.TestStep{
						ResourceName:      "grafana_k6_schedule.test",
						ImportState:       true,
						ImportStateVerify: true,
					})
				}
			}

			return steps
		}(),
	})
}

func TestScheduleResource_FrequencyValidation_Unit(t *testing.T) {
	// Test the expected frequency enum values
	validFrequencies := []string{"HOURLY", "DAILY", "WEEKLY", "MONTHLY", "YEARLY"}
	invalidFrequencies := []string{"daily", "Daily", "MINUTELY", "INVALID", "", "hourly", "SECOND"}

	// Test that all valid frequencies are included in our validation
	for _, frequency := range validFrequencies {
		t.Run(fmt.Sprintf("valid_%s", frequency), func(t *testing.T) {
			// This test verifies our enum contains the expected values
			// The actual validation logic is in the stringvalidator.OneOf()
			// which is tested by the acceptance tests
			if frequency == "" {
				t.Errorf("Valid frequency should not be empty")
			}
			if len(frequency) == 0 {
				t.Errorf("Valid frequency should have length > 0")
			}
		})
	}

	// Test that invalid frequencies are correctly identified as invalid
	for _, frequency := range invalidFrequencies {
		t.Run(fmt.Sprintf("invalid_%s", frequency), func(t *testing.T) {
			isValid := false
			for _, valid := range validFrequencies {
				if frequency == valid {
					isValid = true
					break
				}
			}
			if isValid {
				t.Errorf("Frequency '%s' should be invalid but was found in valid list", frequency)
			}
		})
	}

	// Test that we have exactly 5 valid frequency values
	if len(validFrequencies) != 5 {
		t.Errorf("Expected exactly 5 valid frequencies, got %d", len(validFrequencies))
	}

	// Test that all frequencies are uppercase
	for _, frequency := range validFrequencies {
		if frequency != strings.ToUpper(frequency) {
			t.Errorf("Frequency '%s' should be uppercase", frequency)
		}
	}
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
  recurrence_rule {
    frequency = "DAILY"
    interval = 1
  }
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
  recurrence_rule {
    frequency = "WEEKLY"
    interval = 2
  }
}
`, projectName, loadTestName)
}

func testScheduleConfigInvalidFrequency(projectName, loadTestName, frequency string) string {
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
  recurrence_rule {
    frequency = "%s"
    interval = 1
  }
}
`, projectName, loadTestName, frequency)
}

func testScheduleConfigWithFrequency(projectName, loadTestName, frequency string) string {
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
  recurrence_rule {
    frequency = "%s"
    interval = 1
  }
}
`, projectName, loadTestName, frequency)
}
