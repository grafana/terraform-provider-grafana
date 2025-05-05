package k6_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

func TestAccLoadTest_basic(t *testing.T) {
	//testutils.CheckCloudInstanceTestsEnabled(t)

	var (
		project  k6.ProjectApiModel
		loadTest k6.LoadTestApiModel
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			loadTestCheckExists.destroyed(&loadTest),
			projectCheckExists.destroyed(&project),
		),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"${path.module}/archive.tar": "../../../examples/resources/grafana_k6_load_test/archive.tar",
				}),
				Check: resource.ComposeTestCheckFunc(
					projectCheckExists.exists("grafana_k6_project.load_test_project", &project),
					// inline
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test_inline", &loadTest),
					resource.TestMatchResourceAttr("grafana_k6_load_test.test_load_test_inline", "id", defaultIDRegexp),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test_inline", "name", "Terraform Test Load Test Inline"),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test_inline", "script", "export default function() {\n  console.log('Hello from k6!');\n}\n"),
					// archive
					loadTestCheckExists.exists("grafana_k6_load_test.test_load_test_archive", &loadTest),
					resource.TestMatchResourceAttr("grafana_k6_load_test.test_load_test_archive", "id", defaultIDRegexp),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test_archive", "name", "Terraform Test Load Test Archive"),
					resource.TestCheckResourceAttrSet("grafana_k6_load_test.test_load_test_archive", "script_file"),
					// lister
					testutils.CheckLister("grafana_k6_load_test.test_load_test_inline"),
					testutils.CheckLister("grafana_k6_load_test.test_load_test_archive"),
				),
			},
			{
				ResourceName:      "grafana_k6_load_test.test_load_test_inline",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Change the name and script of a load test. This shouldn't recreate the load test.
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_k6_load_test/resource.tf", map[string]string{
					"Terraform Test Load Test Inline": "Terraform Test Load Test Updated",
					"console.log('Hello from k6!')":   "console.log('Hello from updated k6!')",
					"${path.module}/archive.tar":      "../../../examples/resources/grafana_k6_load_test/archive.tar",
				}),
				Check: resource.ComposeTestCheckFunc(
					testAccLoadTestWasntRecreated("grafana_k6_load_test.test_load_test_inline", &loadTest),
					testAccLoadTestUnchangedAttr("grafana_k6_load_test.test_load_test_inline", "id", func() string { return strconv.Itoa(int(loadTest.GetId())) }),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test_inline", "name", "Terraform Test Load Test Updated"),
					resource.TestCheckResourceAttr("grafana_k6_load_test.test_load_test_inline", "script", "export default function() {\n  console.log('Hello from updated k6!');\n}\n"),
					testAccLoadTestUnchangedAttr("grafana_k6_load_test.test_load_test_inline", "created", func() string { return loadTest.GetCreated().Truncate(time.Microsecond).Format(time.RFC3339Nano) }),
				),
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
		var newLoadTestID int32
		if loadTestID, err := strconv.Atoi(newLoadTestResource.Primary.ID); err != nil {
			return fmt.Errorf("could not convert load test id to integer: %s", err.Error())
		} else if newLoadTestID, err = common.ToInt32(loadTestID); err != nil {
			return fmt.Errorf("could not convert load test id to int32: %s", err.Error())
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
