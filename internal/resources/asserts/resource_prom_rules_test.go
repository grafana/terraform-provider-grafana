package asserts_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAssertsPromRules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-acc-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.#", "1"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.name", "test_rules"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.#", "1"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.record", "custom:test:metric"),
					testutils.CheckLister("grafana_asserts_prom_rule_file.test"),
				),
			},
			{
				// Test import
				ResourceName:      "grafana_asserts_prom_rule_file.test",
				ImportState:       true,
				ImportStateVerify: true,
				// Ignore active field - API may not return it if it's the default (true)
				ImportStateVerifyIgnore: []string{"active"},
			},
			{
				// Test update
				Config: testAccAssertsPromRulesConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.#", "2"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.name", "test_rules"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.1.name", "additional_rules"),
				),
			},
		},
	})
}

func TestAccAssertsPromRules_recordingRule(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-recording-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesRecordingConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.record", "custom:requests:rate"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.labels.source", "custom"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.labels.severity", "info"),
				),
			},
		},
	})
}

func TestAccAssertsPromRules_alertingRule(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-alerting-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesAlertingConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "name", rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.alert", "TestAlert"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.rule.0.expr", "up == 0"),
				),
			},
		},
	})
}

func TestAccAssertsPromRules_multipleGroups(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-multi-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesMultiGroupConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.#", "3"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.0.name", "latency_rules"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.1.name", "error_rules"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "group.2.name", "throughput_rules"),
				),
			},
		},
	})
}

func TestAccAssertsPromRules_inactive(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := getTestStackID(t)
	rName := fmt.Sprintf("test-inactive-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesInactiveConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test", "active", "false"),
				),
			},
		},
	})
}

func TestAccAssertsPromRules_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	stackID := getTestStackID(t)
	baseName := fmt.Sprintf("stress-test-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsPromRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsPromRulesStressConfig(stackID, baseName),
				Check: resource.ComposeTestCheckFunc(
					// Check that all resources were created successfully
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test1", stackID, baseName+"-1"),
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test2", stackID, baseName+"-2"),
					testAccAssertsPromRulesCheckExists("grafana_asserts_prom_rule_file.test3", stackID, baseName+"-3"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test1", "name", baseName+"-1"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test2", "name", baseName+"-2"),
					resource.TestCheckResourceAttr("grafana_asserts_prom_rule_file.test3", "name", baseName+"-3"),
				),
			},
		},
	})
}

func testAccAssertsPromRulesCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
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

		// Get specific rules file
		request := client.PromRulesConfigControllerAPI.GetPromRules(ctx, name).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		_, resp, err := request.Execute()
		if err != nil {
			if resp != nil && resp.StatusCode == 404 {
				return fmt.Errorf("Prometheus rules file %s not found", name)
			}
			return fmt.Errorf("error getting Prometheus rules file: %s", err)
		}

		return nil
	}
}

func testAccAssertsPromRulesCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	deadline := time.Now().Add(60 * time.Second)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_prom_rule_file" {
			continue
		}

		// Resource ID is just the name now
		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		for {
			// Try to get the rules file
			request := client.PromRulesConfigControllerAPI.GetPromRules(ctx, name).
				XScopeOrgID(stackID)

			_, resp, err := request.Execute()
			if err != nil {
				// If 404, resource is deleted - that's what we want
				if resp != nil && resp.StatusCode == 404 {
					break
				}
				// If we can't get it for other reasons, assume it's deleted
				if common.IsNotFoundError(err) {
					break
				}
				return fmt.Errorf("error checking Prometheus rules file destruction: %s", err)
			}

			// Resource still exists
			if time.Now().After(deadline) {
				return fmt.Errorf("Prometheus rules file %s still exists", name)
			}
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

func testAccAssertsPromRulesConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name = "%s"

  group {
    name = "test_rules"

    rule {
      record = "custom:test:metric"
      expr   = "up"
    }
  }
}
`, name)
}

func testAccAssertsPromRulesConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name = "%s"

  group {
    name = "test_rules"

    rule {
      record = "custom:test:metric:v2"
      expr   = "up"
    }

    rule {
      record = "custom:new:metric"
      expr   = "up"
    }
  }

  group {
    name = "additional_rules"

    rule {
      record = "custom:another:metric"
      expr   = "up"
    }
  }
}
`, name)
}

func testAccAssertsPromRulesRecordingConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name = "%s"

  group {
    name = "recording_rules"

    rule {
      record = "custom:requests:rate"
      expr   = "up"
      labels = {
        source   = "custom"
        severity = "info"
      }
    }
  }
}
`, name)
}

func testAccAssertsPromRulesAlertingConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name = "%s"

  group {
    name = "alerting_rules"

    rule {
      alert = "TestAlert"
      expr  = "up == 0"
    }
  }
}
`, name)
}

func testAccAssertsPromRulesMultiGroupConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name = "%s"

  group {
    name = "latency_rules"

    rule {
      record = "custom:latency:p95"
      expr   = "up"
    }
  }

  group {
    name = "error_rules"

    rule {
      record = "custom:error:rate"
      expr   = "up"
    }
  }

  group {
    name = "throughput_rules"

    rule {
      record = "custom:throughput:total"
      expr   = "up"
    }
  }
}
`, name)
}

func testAccAssertsPromRulesInactiveConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test" {
  name   = "%s"
  active = false

  group {
    name = "inactive_rules"

    rule {
      record = "custom:test:metric"
      expr   = "up"
    }
  }
}
`, name)
}

func testAccAssertsPromRulesStressConfig(stackID int64, baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_prom_rule_file" "test1" {
  name   = "%s-1"
  active = true

  group {
    name = "stress_test_group_1"

    rule {
      record = "stress:test:metric1"
      expr   = "up"
    }
  }
}

resource "grafana_asserts_prom_rule_file" "test2" {
  name   = "%s-2"
  active = true

  group {
    name = "stress_test_group_2"

    rule {
      record = "stress:test:metric2"
      expr   = "up"
    }
  }
}

resource "grafana_asserts_prom_rule_file" "test3" {
  name   = "%s-3"
  active = true

  group {
    name = "stress_test_group_3"

    rule {
      record = "stress:test:metric3"
      expr   = "up"
    }
  }
}
`, baseName, baseName, baseName)
}
