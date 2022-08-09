package grafana

import (
	"fmt"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAlertRule_basic(t *testing.T) {
	CheckOSSTestsEnabled(t)
	CheckOSSTestsSemver(t, ">=9.0.0")

	var group gapi.RuleGroup

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		// Implicitly tests deletion.
		CheckDestroy: testAlertRuleCheckDestroy(group),
		Steps: []resource.TestStep{
			// Test creation.
			{
				Config: testAccExample(t, "resources/grafana_alert_rule/resource.tf"),
				Check:  resource.ComposeTestCheckFunc(),
			},
		},
	})
}

func testAlertRuleCheckDestroy(group gapi.RuleGroup) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*client).gapi
		_, err := client.AlertRuleGroup(group.FolderUID, group.Title)
		if err == nil && strings.HasPrefix(err.Error(), "status: 404") {
			return fmt.Errorf("rule group still exists on the server")
		}
		return nil
	}
}
