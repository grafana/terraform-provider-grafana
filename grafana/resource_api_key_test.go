package grafana

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaAuthKey(t *testing.T) {
	CheckOSSTestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccGrafanaAuthKeyCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyBasicConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.foo", "foo-name", "Admin", false),
				),
			},
			{
				Config: testAccGrafanaAuthKeyExpandedConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.bar", "bar-name", "Viewer", true),
				),
			},
		},
	})
}

func TestAccGrafanaAuthKeyFromCloud(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	var stack gapi.Stack
	prefix := "tfapikeytest"
	slug := GetRandomStackName(prefix)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testAccProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaAuthKeyFromCloud(slug, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthKeyCheckFields("grafana_api_key.management", "management-key", "Admin", false),

					// TODO: Check how we can remove this sleep
					// Sometimes the stack is not ready to be deleted at the end of the test
					func(s *terraform.State) error {
						time.Sleep(time.Second * 15)
						return nil
					},
				),
			},
			{
				Config: testAccStackConfigBasic(slug, slug),
				Check:  testAccGrafanaAuthKeyCheckDestroyCloud,
			},
		},
	})
}

func testAccGrafanaAuthKeyCheckDestroy(s *terraform.State) error {
	c := testAccProvider.Meta().(*client).gapi

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_api_key" {
			continue
		}

		idStr := rs.Primary.ID
		id, err := strconv.ParseInt(idStr, 10, 32)
		if err != nil {
			return err
		}

		keys, err := c.GetAPIKeys(false)
		if err != nil {
			return err
		}

		for _, key := range keys {
			if key.ID == id {
				return fmt.Errorf("API key still exists")
			}
		}
	}

	return nil
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

		cloudClient := testAccProvider.Meta().(*client).gcloudapi
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

const testAccGrafanaAuthKeyBasicConfig = `
resource "grafana_api_key" "foo" {
	name = "foo-name"
	role = "Admin"
}
`

const testAccGrafanaAuthKeyExpandedConfig = `
resource "grafana_api_key" "bar" {
	name 			= "bar-name"
	role 			= "Viewer"
	seconds_to_live = 300
}
`

func testAccGrafanaAuthKeyFromCloud(name, slug string) string {
	return testAccStackConfigBasic(name, slug) + `
	resource "grafana_api_key" "management" {
		cloud_stack_slug = grafana_cloud_stack.test.slug
		name             = "management-key"
		role             = "Admin"
	}
	`
}
