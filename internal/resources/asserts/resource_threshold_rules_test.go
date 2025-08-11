package asserts_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAssertsThresholdRules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-acc-tr-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsThresholdRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsThresholdRulesConfig(stackID, rName, "resource"),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdRulesCheckExists("grafana_asserts_threshold_rules.test", stackID, "resource"),
					resource.TestCheckResourceAttr("grafana_asserts_threshold_rules.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_threshold_rules.test", "scope", "resource"),
				),
			},
			{
				// Test import
				ResourceName:      "grafana_asserts_threshold_rules.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Test update
				Config: testAccAssertsThresholdRulesConfigUpdated(stackID, rName, "resource"),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsThresholdRulesCheckExists("grafana_asserts_threshold_rules.test", stackID, "resource"),
					resource.TestCheckResourceAttr("grafana_asserts_threshold_rules.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_threshold_rules.test", "scope", "resource"),
				),
			},
		},
	})
}

func testAccAssertsThresholdRulesCheckExists(rn string, stackID int64, scope string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
		ctx := context.Background()

		if scope == "resource" {
			_, _, err := client.ThresholdRulesConfigControllerAPI.GetResourceThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
			if err != nil {
				return fmt.Errorf("error getting resource threshold rules: %s", err)
			}
		} else {
			_, _, err := client.ThresholdRulesConfigControllerAPI.GetRequestThresholdRules(ctx).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
			if err != nil {
				return fmt.Errorf("error getting request threshold rules: %s", err)
			}
		}

		return nil
	}
}

func testAccAssertsThresholdRulesCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_threshold_rules" {
			continue
		}

		scope, _, err := parseThresholdRuleID(rs.Primary.ID)
		if err != nil {
			return err
		}

		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)
		if scope == "resource" {
			_, _, err := client.ThresholdRulesConfigControllerAPI.GetResourceThresholdRules(ctx).XScopeOrgID(stackID).Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking resource threshold rules destruction: %s", err)
			}
		} else {
			_, _, err := client.ThresholdRulesConfigControllerAPI.GetRequestThresholdRules(ctx).XScopeOrgID(stackID).Execute()
			if err != nil {
				if common.IsNotFoundError(err) {
					continue
				}
				return fmt.Errorf("error checking request threshold rules destruction: %s", err)
			}
		}

		return fmt.Errorf("threshold rules for scope %s still exists", scope)
	}

	return nil
}

func parseThresholdRuleID(id string) (string, string, error) {
	parts := strings.SplitN(id, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("unexpected ID format (%s), expected scope/name", id)
	}
	return parts[0], parts[1], nil
}

func testAccAssertsThresholdRulesConfig(stackID int64, name, scope string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_threshold_rules" "test" {
  name  = "%s"
  scope = "%s"
  rules = <<-EOT
    groups:
    - name: "custom-thresholds"
      rules:
      - alert: "custom-latency"
        expr: "sum(rate(http_requests_total{job='api-server'}[5m])) > 100"
        for: "1m"
        labels:
          severity: "page"
  EOT
}
`, name, scope)
}

func testAccAssertsThresholdRulesConfigUpdated(stackID int64, name, scope string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_threshold_rules" "test" {
  name  = "%s"
  scope = "%s"
  rules = <<-EOT
    groups:
    - name: "custom-thresholds"
      rules:
      - alert: "custom-latency"
        expr: "sum(rate(http_requests_total{job='api-server'}[5m])) > 200"
        for: "5m"
        labels:
          severity: "critical"
  EOT
}
`, name, scope)
}
