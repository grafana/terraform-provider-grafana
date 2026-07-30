package agento11y_test

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

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
