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

func TestAccGrafanaServiceAccountFromCloud(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gapi.Stack
	prefix := "tfsatest"
	slug := GetRandomStackName(prefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "name", "management-sa"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "role", "Admin"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "is_disabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_token.management_token", "name", "management-sa-token"),
					resource.TestCheckNoResourceAttr("grafana_cloud_stack_service_account_token.management_token", "expiration"),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_service_account_token.management_token", "key"),
				),
			},
			{
				Config: testAccStackConfigBasic(slug, slug, "description"),
				Check:  testAccGrafanaServiceAccountCheckDestroyCloud,
			},
		},
	})
}

func testAccGrafanaServiceAccountFromCloud(name, slug string) string {
	return testAccStackConfigBasic(name, slug, "description") + `
	resource "grafana_cloud_stack_service_account" "management" {
		stack_slug = grafana_cloud_stack.test.slug
		name       = "management-sa"
		role       = "Admin"
	}

	resource "grafana_cloud_stack_service_account_token" "management_token" {
		stack_slug = grafana_cloud_stack.test.slug
		service_account_id = grafana_cloud_stack_service_account.management.id
		name       = "management-sa-token"
	}
	`
}

// Checks that all service accounts and service account tokens are deleted, to be called before the stack is completely destroyed
func testAccGrafanaServiceAccountCheckDestroyCloud(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_cloud_stack" {
			continue
		}

		cloudClient := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		c, cleanup, err := cloudClient.CreateTemporaryStackGrafanaClient(rs.Primary.Attributes["slug"], "test-service-account-", 60*time.Second)
		if err != nil {
			return err
		}
		defer cleanup()

		response, err := c.GetServiceAccounts()
		if err != nil {
			return err
		}

		for _, sa := range response {
			if strings.HasPrefix(sa.Name, "test-service-account-") {
				continue // this is a service account created by this test
			}

			tokens, err := c.GetServiceAccountTokens(sa.ID)
			if err != nil {
				return err
			}
			if len(tokens) > 0 {
				return fmt.Errorf("found unexpected service account tokens for service account %s: %v", sa.Name, tokens)
			}

			return fmt.Errorf("found unexpected service account: %v", sa)
		}

		return nil
	}

	return errors.New("no cloud stack created")
}
