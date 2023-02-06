package cloud_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaAuthKeyFromCloud(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gapi.Stack
	prefix := "tfapikeytest"
	slug := GetRandomStackName(prefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyFromCloud(slug, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.management", "management-key", "Admin", false),
				),
			},
			{
				Config: testAccStackConfigBasic(slug, slug),
				Check:  testAccGrafanaAuthKeyCheckDestroyCloud,
			},
		},
	})
}

func testAccGrafanaAuthKeyFromCloud(name, slug string) string {
	return testAccStackConfigBasic(name, slug) + `
	resource "grafana_api_key" "management" {
		cloud_stack_slug = grafana_cloud_stack.test.slug
		name             = "management-key"
		role             = "Admin"
	}
	`
}

func testAccGrafanaAuthKeyCheckFields(n string, name string, role string, expires bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("no ID is set")
		}

		if rs.Primary.Attributes["key"] == "" {
			return fmt.Errorf("no API key is set")
		}

		if rs.Primary.Attributes["name"] != name {
			return fmt.Errorf("incorrect name field found: %s", rs.Primary.Attributes["name"])
		}

		if rs.Primary.Attributes["role"] != role {
			return fmt.Errorf("incorrect role field found: %s", rs.Primary.Attributes["role"])
		}

		expiration := rs.Primary.Attributes["expiration"]
		if expires && expiration == "" {
			return fmt.Errorf("no expiration date set")
		}

		if !expires && expiration != "" {
			return fmt.Errorf("expiration date set")
		}

		return nil
	}
}

// Checks that all API keys are deleted, to be called before the stack is completely destroyed
func testAccGrafanaAuthKeyCheckDestroyCloud(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_cloud_stack" {
			continue
		}

		cloudClient := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		c, cleanup, err := cloudClient.CreateTemporaryStackGrafanaClient(rs.Primary.Attributes["slug"], "test-api-key-", 60*time.Second)
		if err != nil {
			return err
		}
		defer cleanup()

		response, err := c.GetAPIKeys(true)
		if err != nil {
			return err
		}

		for _, key := range response {
			if !strings.HasPrefix(key.Name, "test-api-key-") {
				return fmt.Errorf("Found unexpected API key: %s", key.Name)
			}
		}
		return nil
	}

	return errors.New("no cloud stack created")
}
