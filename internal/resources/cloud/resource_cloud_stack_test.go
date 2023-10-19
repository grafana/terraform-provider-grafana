package cloud_test

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceStack_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	prefix := "tfresourcetest"

	var stack gapi.Stack
	resourceName := GetRandomStackName(prefix)
	stackDescription := "This is a test stack"

	firstStepChecks := resource.ComposeTestCheckFunc(
		testAccStackCheckExists("grafana_cloud_stack.test", &stack),
		resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", common.IDRegexp),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "status", "active"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "prometheus_remote_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "prometheus_remote_write_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom/push"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "prometheus_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "alertmanager_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "logs_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "traces_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "graphite_user_id"),
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			// Create a basic stack
			{
				Config: testAccStackConfigBasic(resourceName, resourceName),
				Check:  firstStepChecks,
			},
			// Check that we can't takeover a stack without importing it
			// The retrying logic for creation is very permissive,
			// but it shouldn't allow to apply an already existing stack on a new resource
			{
				Config: testAccStackConfigBasic(resourceName, resourceName) +
					testAccStackConfigBasicWithCustomResourceName(resourceName, resourceName, "test2"), // new stack with same name/slug
				ExpectError: regexp.MustCompile(fmt.Sprintf(".*a stack with the name '%s' already exists.*", resourceName)),
			},
			// Test that the stack is correctly recreated if it's tainted and reapplied
			// This is a special case because stack deletion is asynchronous
			{
				Config: testAccStackConfigBasic(resourceName, resourceName),
				Check:  firstStepChecks,
				Taint:  []string{"grafana_cloud_stack.test"},
			},
			{
				// Delete the stack outside of the test and make sure it is recreated
				// Terraform should detect that it's gone and recreate it (status should be active at all times)
				PreConfig: func() {
					testAccDeleteExistingStacks(t, prefix)
					time.Sleep(10 * time.Second)
				},
				Config: testAccStackConfigBasic(resourceName, resourceName),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "status", "active"),
				),
			},
			// Update the stack
			{
				Config: testAccStackConfigUpdate(resourceName+"new", resourceName, stackDescription),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", common.IDRegexp),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName+"new"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "description", stackDescription),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "status", "active"),
				),
			},
			// Test import from ID
			{
				ResourceName:      "grafana_cloud_stack.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Test import from slug
			{
				ResourceName:      "grafana_cloud_stack.test",
				ImportStateId:     resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccDeleteExistingStacks(t *testing.T, prefix string) {
	client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
	resp, err := client.Stacks()
	if err != nil {
		t.Error(err)
	}

	for _, stack := range resp.Items {
		if strings.HasPrefix(stack.Name, prefix) {
			err := client.DeleteStack(stack.Slug)
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func testAccStackCheckExists(rn string, a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		id, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		stack, err := client.StackByID(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = stack

		return nil
	}
}

func testAccStackCheckDestroy(a *gapi.Stack) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		stack, err := client.StackBySlug(a.Slug)
		if err == nil && stack.Name != "" {
			return fmt.Errorf("stack `%s` with ID `%d` still exists after destroy", stack.Name, stack.ID)
		}

		return nil
	}
}

func testAccStackConfigBasic(name string, slug string) string {
	return testAccStackConfigBasicWithCustomResourceName(name, slug, "test")
}

func testAccStackConfigBasicWithCustomResourceName(name string, slug string, resourceName string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "%s" {
		name  = "%s"
		slug  = "%s"
		region_slug = "eu"
	  }
	`, resourceName, name, slug)
}

func testAccStackConfigUpdate(name string, slug string, description string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "test" {
		name  = "%s"
		slug  = "%s"
		region_slug = "eu"
		description = "%s"
	  }
	`, name, slug, description)
}

// Prefix a character as stack name can't start with a number
func GetRandomStackName(prefix string) string {
	return prefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}
