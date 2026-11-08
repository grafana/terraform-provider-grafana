package k6_test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccLoadTest_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
	)

	projectName := "Terraform Load Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project": projectName,
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.load_test_project", &project),
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test", &loadTest),
					resource.TestMatchResourceAttr("grafana_k6_load_test.test_load_test", "id", defaultIDRegexp),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "name", "Terraform Test Load Test"),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "script", "export default function() {\n  console.log('Hello from k6!');\n}\n"),
					testutils.CheckLister("grafana_k6_load_test.test_load_test"),
				),
			},
			{
				ResourceName:      "grafana_k6_load_test.test_load_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Delete the load test and check that TF sees a difference
			{
				PreConfig: func() {
					commonClient := testutils.Provider.Meta().(*common.Client)
					client := commonClient.K6APIClient
					config := commonClient.K6APIConfig

					ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
					deleteReq := client.LoadTestsAPI.LoadTestsDestroy(ctx, loadTest.Id).XStackId(config.StackID)

					_, err := deleteReq.Execute()
					if err != nil {
						t.Fatalf("error deleting load test: %s", err)
					}
				},
				RefreshState:       true,
				ExpectNonEmptyPlan: true,
			},
			// Recreate the test
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project": projectName,
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test", &loadTest),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "name", "Terraform Test Load Test"),
				),
			},
			// Change the name and script of a load test. This shouldn't recreate the load test.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project":   projectName,
					"Terraform Test Load Test":      "Terraform Test Load Test Updated",
					"console.log('Hello from k6!')": "console.log('Hello from updated k6!')",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccLoadTestWasntRecreated("grafana_k6_load_test.test_load_test", &loadTest),
					testAccLoadTestUnchangedAttr("grafana_k6_load_test.test_load_test", "id", func() string { return strconv.Itoa(int(loadTest.GetId())) }),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "name", "Terraform Test Load Test Updated"),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "script", "export default function() {\n  console.log('Hello from updated k6!');\n}\n"),
					testAccLoadTestUnchangedAttr("grafana_k6_load_test.test_load_test", "created", func() string { return loadTest.GetCreated().Truncate(time.Microsecond).Format(time.RFC3339Nano) }),
				),
			},
			// Change the project_id of a load test. This should recreate the load test.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project":                           projectName + " new",
					"resource \"grafana_k6_project\" \"load_test_project\"": "resource \"grafana_k6_project\" \"load_test_project_new\"",
					"grafana_k6_project.load_test_project.id":               "grafana_k6_project.load_test_project_new.id",
				}),
				Check: resource.ComposeTestCheckFunc(
					// The resource should be recreated with a new id
					resource.TestCheckResourceAttrWith("grafana_k6_load_test.test_load_test", "id", func(newVal string) error {
						if oldValue := strconv.Itoa(int(loadTest.GetId())); oldValue == newVal {
							return fmt.Errorf("id has not changed: %s", oldValue)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccLoadTest_k6Version(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	// The set of valid k6 version ids is environment-specific (the API
	// validates against the versions selectable in the target stack), so the
	// id to test with must be provided explicitly.
	versionID := os.Getenv("GRAFANA_K6_TEST_VERSION_ID")
	if versionID == "" {
		t.Skip("GRAFANA_K6_TEST_VERSION_ID must be set to a valid k6 version id to run this test")
	}

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
	)

	projectName := "Terraform Load Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			// Create with k6_version set.
			{
				Config: testAccLoadTestConfigK6Version(projectName, "Terraform Test Load Test", versionID),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.load_test_project", &project),
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test", &loadTest),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test", "k6_version", versionID),
				),
			},
			// Import and verify k6_version round-trips with no diff.
			{
				ResourceName:      "grafana_k6_load_test.test_load_test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Removing k6_version clears it without recreating the load test.
			{
				Config: testAccLoadTestConfigK6Version(projectName, "Terraform Test Load Test", ""),
				Check: resource.ComposeTestCheckFunc(
					testAccLoadTestWasntRecreated("grafana_k6_load_test.test_load_test", &loadTest),
					resource.TestCheckNoResourceAttr("grafana_k6_load_test.test_load_test", "k6_version"),
				),
			},
		},
	})
}

// testAccLoadTestConfigK6Version returns a load test config, including the
// k6_version attribute only when versionID is non-empty.
func testAccLoadTestConfigK6Version(projectName, testName, versionID string) string {
	k6VersionAttr := ""
	if versionID != "" {
		k6VersionAttr = fmt.Sprintf("\n  k6_version = %q", versionID)
	}
	return fmt.Sprintf(`
resource "grafana_k6_project" "load_test_project" {
  name = %q
}

resource "grafana_k6_load_test" "test_load_test" {
  project_id = grafana_k6_project.load_test_project.id
  name       = %q%s
  script     = <<-EOT
    export default function() {
      console.log('Hello from k6!');
    }
  EOT
}
`, projectName, testName, k6VersionAttr)
}

func TestAccLoadTest_StateUpgrade(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
	)

	projectName := "Terraform Test Project " + acctest.RandString(8)

	resource.ParallelTest(t, resource.TestCase{
		CheckDestroy: resource.ComposeTestCheckFunc(
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project": projectName,
				}),
				ExternalProviders: map[string]resource.ExternalProvider{
					"grafana": {
						Source:            "grafana/grafana",
						VersionConstraint: "<=3.25.2",
					},
				},
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.load_test_project", &project),
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test", &loadTest),
				),
			},
			// Test upgrading the provider version does not create a diff
			{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Load Test Project": projectName,
				}),
				ExpectNonEmptyPlan: false,
				PlanOnly:           true,
			},
		},
	})
}

func testAccLoadTestUnchangedAttr(resName, attrName string, oldValueGetter func() string) resource.TestCheckFunc {
	return resource.TestCheckResourceAttrWith(resName, attrName, func(newVal string) error {
		if oldValue := oldValueGetter(); oldValue != newVal {
			return fmt.Errorf("%s has changed: %s -> %s", attrName, oldValue, newVal)
		}
		return nil
	})
}

func testAccLoadTestWasntRecreated(rn string, oldLoadTest *k6.LoadTestApiModel) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		newLoadTestResource, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("load test not found: %s", rn)
		}
		if newLoadTestResource.Primary.ID == "" {
			return fmt.Errorf("load test id not set")
		}
		newLoadTestID, err := strconv.ParseInt(newLoadTestResource.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("could not convert load test id to integer: %s", err.Error())
		}

		client := testutils.Provider.Meta().(*common.Client).K6APIClient
		config := testutils.Provider.Meta().(*common.Client).K6APIConfig

		ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
		newLoadTest, _, err := client.LoadTestsAPI.LoadTestsRetrieve(ctx, newLoadTestID).
			XStackId(config.StackID).
			Execute()
		if err != nil {
			return fmt.Errorf("error getting load test: %s", err)
		}
		if newLoadTest.Created != oldLoadTest.Created {
			return fmt.Errorf("load test creation date has changed: %s -> %s", oldLoadTest.Created, newLoadTest.Created)
		}
		if !oldLoadTest.GetUpdated().Before(newLoadTest.GetUpdated()) {
			return fmt.Errorf("load test update date hasn't changed: %s -> %s", oldLoadTest.Updated, newLoadTest.Updated)
		}
		return nil
	}
}
