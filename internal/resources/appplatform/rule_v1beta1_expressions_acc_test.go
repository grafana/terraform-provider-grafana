package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// TestAccRuleV1Beta1Expressions_optionalFields checks that expressions written the way the first
// schema required (every field set, null when unused) and with the unused fields left out are the
// same config, so moving expressions to a nested attribute doesn't change existing configs or state.
func TestAccRuleV1Beta1Expressions_optionalFields(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.3.0")

	suffix := acctest.RandString(6)
	alertRule := "grafana_apps_rules_alertrule_v1beta1.test"
	recordingRule := "grafana_apps_rules_recordingrule_v1beta1.test"

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV6ProviderFactories: testutils.ProtoV6ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccRuleV1Beta1ExpressionsConfig(suffix, true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertRule, "spec.expressions.B.datasource_uid", "__expr__"),
					terraformresource.TestCheckNoResourceAttr(alertRule, "spec.expressions.B.query_type"),
					terraformresource.TestCheckNoResourceAttr(alertRule, "spec.expressions.B.relative_time_range.from"),
					terraformresource.TestCheckNoResourceAttr(alertRule, "spec.expressions.B.source"),
					terraformresource.TestCheckResourceAttr(alertRule, "spec.panel_ref.panel_id", "5"),
					terraformresource.TestCheckNoResourceAttr(recordingRule, "spec.expressions.B.query_type"),
				),
			},
			{
				Config:   testAccRuleV1Beta1ExpressionsConfig(suffix, false),
				PlanOnly: true,
			},
			{
				Config: testAccRuleV1Beta1ExpressionsConfig(suffix, false),
			},
			{
				Config:   testAccRuleV1Beta1ExpressionsConfig(suffix, true),
				PlanOnly: true,
			},
		},
	})
}

// testAccRuleV1Beta1ExpressionsConfig renders an alert rule and a recording rule whose B
// expressions use only some fields. With explicitNulls, the unused fields are set to null.
func testAccRuleV1Beta1ExpressionsConfig(suffix string, explicitNulls bool) string {
	unused := ""
	if explicitNulls {
		unused = `
        query_type          = null
        relative_time_range = null
        source              = null`
	}

	return fmt.Sprintf(`
resource "grafana_folder" "test" {
  title = "test-rule-expressions-folder-%[1]s"
}

resource "grafana_data_source" "test" {
  type = "prometheus"
  name = "test-rule-expressions-ds-%[1]s"
  url  = "http://localhost:9090"
}

resource "grafana_contact_point" "test" {
  name = "test-rule-expressions-cp-%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}

resource "grafana_apps_rules_alertrule_v1beta1" "test" {
  metadata {
    uid        = "test-expr-alert-rule-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    title = "Test Expressions Alert Rule %[1]s"
    trigger {
      interval = "1m"
    }
    paused         = true
    no_data_state  = "KeepLast"
    exec_err_state = "KeepLast"
    notification_settings {
      simplified_routing {
        contact_point = grafana_contact_point.test.name
      }
    }
    panel_ref = {
      dashboard_uid = "dashboard123"
      panel_id      = 5
    }
    expressions = {
      "A" = {
        model = jsonencode({
          editorMode = "code"
          expr       = "count(up{})"
          instant    = true
          intervalMs = 1000
          refId      = "A"
        })
        datasource_uid = grafana_data_source.test.uid
        relative_time_range = {
          from = "600s"
          to   = "0s"
        }
        query_type = ""
        source     = true
      }
      "B" = {
        model = jsonencode({
          expression = "A"
          refId      = "B"
          type       = "threshold"
          conditions = [{ evaluator = { params = [1], type = "gt" } }]
        })
        datasource_uid = "__expr__"%[2]s
      }
    }
  }
}

resource "grafana_apps_rules_recordingrule_v1beta1" "test" {
  metadata {
    uid        = "test-expr-recording-rule-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    title = "Test Expressions Recording Rule %[1]s"
    trigger {
      interval = "1m"
    }
    paused                = true
    target_datasource_uid = grafana_data_source.test.uid
    metric                = "tf_expr_metric_%[1]s"
    expressions = {
      "A" = {
        model = jsonencode({
          editorMode = "code"
          expr       = "count(up{})"
          instant    = true
          intervalMs = 1000
          refId      = "A"
        })
        datasource_uid = grafana_data_source.test.uid
        relative_time_range = {
          from = "600s"
          to   = "0s"
        }
        source = true
      }
      "B" = {
        model = jsonencode({
          expression = "$A * 2"
          refId      = "B"
          type       = "math"
        })
        datasource_uid = "__expr__"%[2]s
      }
    }
  }
}
`, suffix, unused)
}
