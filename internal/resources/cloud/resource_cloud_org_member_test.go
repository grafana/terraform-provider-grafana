package cloud_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// We need an actual user to test the org member resource
// This is a user created from my personal email, but it can be replaced by any existing user
const testOrgMemberUser = "julienduchesne1"

func TestAccResourceOrgMember(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	org := os.Getenv("GRAFANA_CLOUD_ORG")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:                 func() { testAccDeleteExistingOrgMember(t, org, testOrgMemberUser) },
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,

		Steps: []resource.TestStep{
			{
				Config: testAccCloudOrgMember(org, testOrgMemberUser, "Admin", true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgMember(org, testOrgMemberUser, true),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "org", org),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "user", testOrgMemberUser),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "role", "Admin"),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "receive_billing_emails", "true"),
				),
			},
			{
				Config: testAccCloudOrgMember(org, testOrgMemberUser, "Editor", false),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckOrgMember(org, testOrgMemberUser, true),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "org", org),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "user", testOrgMemberUser),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_cloud_org_member.test", "receive_billing_emails", "false"),
				),
			},
			{
				ResourceName:      "grafana_cloud_org_member.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
		CheckDestroy: testAccCheckOrgMember(org, testOrgMemberUser, false),
	})
}

func testAccCheckOrgMember(org, user string, shouldExist bool) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		commonClient, ok := testutils.Provider.Meta().(*client.Client)
		if !ok {
			return fmt.Errorf("failed to get common client")
		}
		if commonClient.GrafanaCloudAPI == nil {
			return fmt.Errorf("GrafanaCloudAPI is nil")
		}
		client := commonClient.GrafanaCloudAPI
		resp, _, err := client.OrgsAPI.GetOrgMembers(context.Background(), org).Execute()
		if err != nil {
			return err
		}

		for _, member := range resp.Items {
			if member.UserName == user {
				if !shouldExist {
					return fmt.Errorf("org member %s still exists", user)
				}
				return nil
			}
		}

		if shouldExist {
			return fmt.Errorf("org member %s does not exist", user)
		}
		return nil
	}
}

func testAccDeleteExistingOrgMember(t *testing.T, org, name string) {
	t.Helper()

	client := testutils.Provider.Meta().(*client.Client).GrafanaCloudAPI
	resp, _, err := client.OrgsAPI.GetOrgMembers(context.Background(), org).Execute()
	if err != nil {
		t.Error(err)
	}

	for _, member := range resp.Items {
		if member.UserName == name {
			_, err := client.OrgsAPI.DeleteOrgMember(context.Background(), org, member.UserName).Execute()
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func testAccCloudOrgMember(org string, user string, role string, receiveBillingEmails bool) string {
	return fmt.Sprintf(`
resource "grafana_cloud_org_member" "test" {
	  org = "%s"
	  user = "%s"
	  role = "%s"
	  receive_billing_emails = %t
}
`, org, user, role, receiveBillingEmails)
}
