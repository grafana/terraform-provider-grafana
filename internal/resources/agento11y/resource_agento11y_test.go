package agento11y_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccAgento11yCollection_basic(t *testing.T) {
	testutils.CheckAgentObservabilityTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_agento11y_collection/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_agento11y_collection.test", "id"),
					resource.TestCheckResourceAttr("grafana_agento11y_collection.test", "name", "tf_acc_test_collection"),
					resource.TestCheckResourceAttr("grafana_agento11y_collection.test", "description", "Managed by the Terraform provider acceptance tests."),
					testutils.CheckLister("grafana_agento11y_collection.test"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_agento11y_collection/_acc_basic.tf", map[string]string{
					`"tf_acc_test_collection"`: `"tf_acc_test_collection_renamed"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_agento11y_collection.test", "name", "tf_acc_test_collection_renamed"),
				),
			},
			{
				// Removing description from configuration clears it server-side. The
				// name keeps its value from the previous step, so only the
				// description changes here.
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_agento11y_collection/_acc_basic.tf", map[string]string{
					`"tf_acc_test_collection"`: `"tf_acc_test_collection_renamed"`,
					"  description = \"Managed by the Terraform provider acceptance tests.\"\n": "",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_agento11y_collection.test", "name", "tf_acc_test_collection_renamed"),
					resource.TestCheckNoResourceAttr("grafana_agento11y_collection.test", "description"),
				),
			},
			{
				ResourceName:      "grafana_agento11y_collection.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAgento11yRuleAction_basic(t *testing.T) {
	testutils.CheckAgentObservabilityTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_agento11y_rule_action/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_agento11y_rule_action.test", "id"),
					resource.TestCheckResourceAttr("grafana_agento11y_rule_action.test", "condition", "all_evaluators_fail"),
					resource.TestCheckResourceAttr("grafana_agento11y_rule_action.test", "enabled", "true"),
					// The action points at the collection Terraform created, which is
					// the reference this resource exists to make possible.
					resource.TestCheckResourceAttrPair(
						"grafana_agento11y_rule_action.test", "collection_ids.0",
						"grafana_agento11y_collection.test", "id",
					),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_agento11y_rule_action/_acc_basic.tf", map[string]string{
					`"all_evaluators_fail"`: `"all_evaluators_pass"`,
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_agento11y_rule_action.test", "condition", "all_evaluators_pass"),
				),
			},
			{
				ResourceName:      "grafana_agento11y_rule_action.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources["grafana_agento11y_rule_action.test"]
					if !ok {
						return "", fmt.Errorf("resource grafana_agento11y_rule_action.test not found in state")
					}
					return rs.Primary.Attributes["rule_id"] + ":" + rs.Primary.Attributes["id"], nil
				},
			},
		},
	})
}

func TestAccAgento11yEvaluator_basic(t *testing.T) {
	testutils.CheckAgentObservabilityTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_agento11y_evaluator/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_agento11y_evaluator.test", "id"),
					resource.TestCheckResourceAttr("grafana_agento11y_evaluator.test", "evaluator_id", "tf_acc_test_evaluator"),
					resource.TestCheckResourceAttr("grafana_agento11y_evaluator.test", "kind", "regex"),
					testutils.CheckLister("grafana_agento11y_evaluator.test"),
				),
			},
			{
				ResourceName:      "grafana_agento11y_evaluator.test",
				ImportState:       true,
				ImportStateVerify: true,
				// config and output_keys are managed from configuration (the server
				// normalizes them), so they are not byte-for-byte comparable on import.
				ImportStateVerifyIgnore: []string{"config", "output_keys"},
			},
		},
	})
}

func TestAccAgento11yEvaluationRule_basic(t *testing.T) {
	testutils.CheckAgentObservabilityTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_agento11y_evaluation_rule/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_agento11y_evaluation_rule.test", "id"),
					resource.TestCheckResourceAttr("grafana_agento11y_evaluation_rule.test", "rule_id", "tf_acc_test_rule"),
					resource.TestCheckResourceAttr("grafana_agento11y_evaluation_rule.test", "selector", "user_visible_turn"),
					resource.TestCheckResourceAttr("grafana_agento11y_evaluation_rule.test", "sample_rate", "0.1"),
					testutils.CheckLister("grafana_agento11y_evaluation_rule.test"),
				),
			},
			{
				Config: testutils.TestAccExampleWithReplace(t, "resources/grafana_agento11y_evaluation_rule/_acc_basic.tf", map[string]string{
					"enabled       = true": "enabled       = false",
				}),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_agento11y_evaluation_rule.test", "enabled", "false"),
				),
			},
			{
				ResourceName:      "grafana_agento11y_evaluation_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccAgento11yHookRule_basic(t *testing.T) {
	testutils.CheckAgentObservabilityTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "resources/grafana_agento11y_hook_rule/_acc_basic.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttrSet("grafana_agento11y_hook_rule.test", "id"),
					resource.TestCheckResourceAttr("grafana_agento11y_hook_rule.test", "rule_id", "tf_acc_test_hook_rule"),
					resource.TestCheckResourceAttr("grafana_agento11y_hook_rule.test", "phase", "preflight"),
					resource.TestCheckResourceAttr("grafana_agento11y_hook_rule.test", "action_on_fail", "deny"),
					resource.TestCheckResourceAttr("grafana_agento11y_hook_rule.test", "blocked_tools.0", "delete_*"),
					testutils.CheckLister("grafana_agento11y_hook_rule.test"),
				),
			},
			{
				ResourceName:      "grafana_agento11y_hook_rule.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}
