package cloud_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGrafanaServiceAccountFromCloud(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	prefix := "tfsatest"
	slug := GetRandomStackName(prefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, true, "Admin"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthCheckServiceAccounts(&stack, []string{"management-sa"}),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "name", "management-sa"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "role", "Admin"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "is_disabled", "true"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_token.management_token", "name", "management-sa-token"),
					resource.TestCheckNoResourceAttr("grafana_cloud_stack_service_account_token.management_token", "expiration"),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_service_account_token.management_token", "key"),
				),
			},
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, false, "Editor"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "is_disabled", "false"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "role", "Editor"),
				),
			},
			{
				ImportState:       true,
				ResourceName:      "grafana_cloud_stack_service_account.management",
				ImportStateVerify: true,
			},
			{
				Config: testAccStackConfigBasic(slug, slug, "description"),
				Check:  testAccGrafanaAuthCheckServiceAccounts(&stack, []string{}),
			},
		},
	})
}

func TestAccGrafanaServiceAccountFromCloud_AssignRoleOrPermissions(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	prefix := "tfsatest"
	slug := GetRandomStackName(prefix)

	resource.Test(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			// SA permission item
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, false, "Admin") + `
				provider "grafana" {
					alias = "stack"
					auth = grafana_cloud_stack_service_account_token.management_token.key
					url  = grafana_cloud_stack.test.url
				}

				resource "grafana_team" "test" {
					provider = grafana.stack
					name = "test"
				}
			
				resource "grafana_service_account_permission_item" "test" {
					provider = grafana.stack
					service_account_id = grafana_cloud_stack_service_account.management.id
					permission = "Admin"
					team = grafana_team.test.id
				}
				`,
			},
			// Role assignment
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, false, "Admin") + `
				provider "grafana" {
					alias = "stack"
					auth = grafana_cloud_stack_service_account_token.management_token.key
					url  = grafana_cloud_stack.test.url
				}
			
				resource "grafana_role" "test" {
					provider = grafana.stack
					name  = "test"
					description = "test desc"
					version = 1
					uid = "test"
					group = "testgroup"
					display_name = "testdisplay"
				}
			
				resource "grafana_role_assignment" "test" {
					provider = grafana.stack
					role_uid = grafana_role.test.uid
					service_accounts = [grafana_cloud_stack_service_account.management.id]
				}
				`,
			},
			// SA permission
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, false, "Admin") + `
				provider "grafana" {
					alias = "stack"
					auth = grafana_cloud_stack_service_account_token.management_token.key
					url  = grafana_cloud_stack.test.url
				}

				resource "grafana_team" "test" {
					provider = grafana.stack
					name = "test"
				}
			
				resource "grafana_service_account_permission" "test" {
					provider = grafana.stack
					service_account_id = grafana_cloud_stack_service_account.management.id
			
					permissions {
						team_id = grafana_team.test.id
						permission = "Admin"
					}
				}
				`,
			},
		},
	})
}

func TestAccGrafanaServiceAccountFromCloudNoneRole(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	var stack gcom.FormattedApiInstance
	prefix := "tfsanone"
	slug := GetRandomStackName(prefix)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccDeleteExistingStacks(t, prefix)
		},
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccStackCheckDestroy(&stack),
		Steps: []resource.TestStep{
			{
				Config: testAccGrafanaServiceAccountFromCloud(slug, slug, true, "None"),
				Check: resource.ComposeTestCheckFunc(
					testAccStackCheckExists("grafana_cloud_stack.test", &stack),
					testAccGrafanaAuthCheckServiceAccounts(&stack, []string{"management-sa"}),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "name", "management-sa"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "role", "None"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account.management", "is_disabled", "true"),
					resource.TestCheckResourceAttr("grafana_cloud_stack_service_account_token.management_token", "name", "management-sa-token"),
					resource.TestCheckNoResourceAttr("grafana_cloud_stack_service_account_token.management_token", "expiration"),
					resource.TestCheckResourceAttrSet("grafana_cloud_stack_service_account_token.management_token", "key"),
				),
			},
		},
	})
}

func testAccGrafanaServiceAccountFromCloud(name, slug string, disabled bool, role string) string {
	return testAccStackConfigBasic(name, slug, "description") + fmt.Sprintf(`
	resource "grafana_cloud_stack_service_account" "management" {
		stack_slug = grafana_cloud_stack.test.slug
		name        = "management-sa"
		role        = "%s"
		is_disabled = %t
	}

	resource "grafana_cloud_stack_service_account_token" "management_token" {
		stack_slug = grafana_cloud_stack.test.slug
		service_account_id = grafana_cloud_stack_service_account.management.id
		name       = "management-sa-token"
	}
	`, role, disabled)
}

func testAccGrafanaAuthCheckServiceAccounts(stack *gcom.FormattedApiInstance, expectedSAs []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		cloudClient := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		c, cleanup, err := cloud.CreateTemporaryStackGrafanaClient(context.Background(), cloudClient, stack.Slug, "test-api-key-")
		if err != nil {
			return err
		}
		defer cleanup()

		response, err := c.ServiceAccounts.SearchOrgServiceAccountsWithPaging(service_accounts.NewSearchOrgServiceAccountsWithPagingParams())
		if err != nil {
			return fmt.Errorf("failed to get service accounts: %w", err)
		}

		for _, expectedSA := range expectedSAs {
			found := false
			for _, sa := range response.Payload.ServiceAccounts {
				if sa.Name == expectedSA {
					found = true
					if sa.Tokens == 0 {
						return fmt.Errorf("expected to find at least one token for service account %s", sa.Name)
					}
					break
				}
			}
			if !found {
				return fmt.Errorf("expected to find key %s, but it was not found", expectedSA)
			}
		}

		return nil
	}
}
