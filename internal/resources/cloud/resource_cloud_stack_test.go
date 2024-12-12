package cloud_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestResourceStack_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	prefix := "tfresourcetest"

	var stack gcom.FormattedApiInstance
	resourceName := GetRandomStackName(prefix)
	stackDescription := "This is a test stack"

	firstStepChecks := resource.ComposeTestCheckFunc(
		testAccStackCheckExists("grafana_cloud_stack.test", &stack),
		resource.TestMatchResourceAttr("grafana_cloud_stack.test", "id", common.IDRegexp),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "name", resourceName),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "slug", resourceName),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "url", "https://"+resourceName+".grafana.net"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "description", stackDescription),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "status", "active"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.tf", "true"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.source", "terraform"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.todelete", "true"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.%", "3"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "prometheus_remote_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom"),
		resource.TestCheckResourceAttr("grafana_cloud_stack.test", "prometheus_remote_write_endpoint", "https://prometheus-prod-01-eu-west-0.grafana.net/api/prom/push"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "prometheus_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "alertmanager_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "logs_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "traces_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "graphite_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "profiles_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "profiles_name"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "profiles_url"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "profiles_status"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "fleet_management_user_id"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "fleet_management_name"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "fleet_management_url"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "fleet_management_status"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "otlp_url"),
		resource.TestCheckResourceAttrSet("grafana_cloud_stack.test", "influx_url"),
	)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			// Create a basic stack
			{
				Config: testAccStackConfigBasic(resourceName, resourceName, stackDescription),
				Check:  firstStepChecks,
			},
			// Check that we can't takeover a stack without importing it
			// The retrying logic for creation is very permissive,
			// but it shouldn't allow to apply an already existing stack on a new resource
			{
				Config: testAccStackConfigBasic(resourceName, resourceName, stackDescription) +
					testAccStackConfigBasicWithCustomResourceName(resourceName, resourceName, "eu", "test2", stackDescription), // new stack with same name/slug
				ExpectError: regexp.MustCompile(".*That URL has already been taken.*"),
			},
			// Test that the stack is correctly recreated if it's tainted and reapplied
			// This is a special case because stack deletion is asynchronous
			{
				Config: testAccStackConfigBasic(resourceName, resourceName, stackDescription),
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
				Config: testAccStackConfigBasic(resourceName, resourceName, stackDescription),
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
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.tf", "true"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.source", "terraform-updated"),
					resource.TestCheckResourceAttr("grafana_cloud_stack.test", "labels.%", "2"),
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

func TestResourceStack_Invalid(t *testing.T) {
	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: `resource "grafana_cloud_stack" "test" { 
					name = "test" 
					slug = "ABC" // Can't start with an uppercase letter
				}`,
				ExpectError: regexp.MustCompile(`.*invalid value for slug \(must be a lowercase alphanumeric string and must start with a letter.*`),
			},
			{
				Config: `resource "grafana_cloud_stack" "test" {
					name = "test"
					slug = "test"
					labels = {
						invalid_key = "true" // Can't have an underscore
					}
				}`,
				ExpectError: regexp.MustCompile(`Error: label key "invalid_key" does not match .+"`),
			},
			{
				Config: `resource "grafana_cloud_stack" "test" {
					name = "test"
					slug = "test"
					labels = {
						"key" = "invalid$"
					}
				}`,
				ExpectError: regexp.MustCompile(`Error: label value "invalid\$" does not match .+"`),
			},
			{
				Config: `resource "grafana_cloud_stack" "test" {
					name = "test"
					slug = "test"
					labels = {
						"1" = "1"
						"2" = "2"
						"3" = "3"
						"4" = "4"
						"5" = "5"
						"6" = "6"
						"7" = "7"
						"8" = "8"
						"9" = "9"
						"10" = "10"
						"11" = "11"
					}
				}`,
				ExpectError: regexp.MustCompile("Error: stacks cannot have more than 10 labels"),
			},
		},
	})
}

func testAccDeleteExistingStacks(t *testing.T, prefix string) {
	client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
	resp, _, err := client.InstancesAPI.GetInstances(context.Background()).Execute()
	if err != nil {
		t.Error(err)
	}

	for _, stack := range resp.Items {
		if strings.HasPrefix(stack.Name, prefix) {
			_, _, err := client.InstancesAPI.DeleteInstance(context.Background(), stack.Slug).XRequestId(cloud.ClientRequestID()).Execute()
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func testAccStackCheckExists(rn string, a *gcom.FormattedApiInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		stack, _, err := client.InstancesAPI.GetInstance(context.Background(), rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = *stack

		if destroyErr := testAccStackCheckDestroy(a)(s); destroyErr == nil {
			return fmt.Errorf("expected the stack's destroy check to fail, but it didn't")
		}

		return nil
	}
}

func testAccStackCheckDestroy(a *gcom.FormattedApiInstance) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		stack, _, err := client.InstancesAPI.GetInstance(context.Background(), a.Slug).Execute()
		if err == nil && stack.Name != "" && stack.Status != "deleting" {
			return fmt.Errorf("stack `%s` with ID `%d` still exists after destroy. Status: %s", stack.Name, int(stack.Id), stack.Status)
		}

		return nil
	}
}

func testAccStackConfigBasic(name string, slug string, description string) string {
	return testAccStackConfigBasicWithCustomResourceName(name, slug, "eu", "test", description)
}

func testAccStackConfigBasicWithCustomResourceName(name, slug, region, resourceName, description string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "%s" {
		name  = "%s"
		slug  = "%s"
		region_slug = "%s"
		description = "%s"
		labels = {
			tf        = "true"
			source    = "terraform"
			todelete = "true"
		}
	  }
	`, resourceName, name, slug, region, description)
}

func testAccStackConfigUpdate(name string, slug string, description string) string {
	return fmt.Sprintf(`
	resource "grafana_cloud_stack" "test" {
		name  = "%s"
		slug  = "%s"
		region_slug = "eu"
		description = "%s"
		labels = {
			tf     = "true"
			source = "terraform-updated"
		}
	  }
	`, name, slug, description)
}

// Prefix a character as stack name can't start with a number
func GetRandomStackName(prefix string) string {
	return prefix + acctest.RandStringFromCharSet(10, acctest.CharSetAlphaNum)
}
