package appplatform_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/rules/pkg/apis/alerting/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	ruleSequenceResourceType = "grafana_apps_rules_rulesequence_v0alpha1"
	ruleSequenceResourceName = ruleSequenceResourceType + ".test"
)

func TestAccRuleSequence(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.1.0")

	t.Run("basic", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := fmt.Sprintf("test-rule-sequence-%s", suffix)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckRuleSequenceDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccRuleSequenceConfig(suffix, "1m"),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.trigger.interval", "1m"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.recording_rules.#", "1"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.recording_rules.0.name", fmt.Sprintf("test-seq-recording-rule-%s", suffix)),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.alerting_rules.#", "1"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.alerting_rules.0.name", fmt.Sprintf("test-seq-alert-rule-%s", suffix)),
						terraformresource.TestCheckResourceAttrSet(ruleSequenceResourceName, "id"),
					),
				},
				{
					Config: testAccRuleSequenceConfig(suffix, "5m"),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.trigger.interval", "5m"),
					),
				},
				{
					ResourceName:      ruleSequenceResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(ruleSequenceResourceName),
				},
				// This destroys the sequence since we currently can't delete the rules while they are associated with a sequence
				// TODO: Remove this when we update the rule sequence API to handle this gracefully
				{
					Config: testAccRuleSequenceRulesConfig(suffix),
					Check:  testAccCheckRuleSequenceDeleted(uid),
				},
			},
		})
	})
}

func testAccCheckRuleSequenceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != ruleSequenceResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v0alpha1.RuleSequenceKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*v0alpha1.RuleSequence, *v0alpha1.RuleSequenceList](rcli, v0alpha1.RuleSequenceKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err != nil {
			if apierrors.IsNotFound(err) {
				continue
			}
			return fmt.Errorf("error checking if RuleSequence %s exists: %w", uid, err)
		}

		return fmt.Errorf("RuleSequence %s still exists", uid)
	}
	return nil
}

func testAccCheckRuleSequenceDeleted(uid string) terraformresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v0alpha1.RuleSequenceKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*v0alpha1.RuleSequence, *v0alpha1.RuleSequenceList](rcli, v0alpha1.RuleSequenceKind()),
			ns,
		)

		// Wait up to 30 seconds until the sequence is fully deleted
		for range 30 {
			_, err := namespacedClient.Get(context.Background(), uid)
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return fmt.Errorf("error polling RuleSequence %s: %w", uid, err)
			}
			time.Sleep(time.Second)
		}

		return fmt.Errorf("RuleSequence %s was not deleted within the timeout", uid)
	}
}

func testAccRuleSequenceRulesConfig(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
  title = "test-rule-sequence-folder-%[1]s"
}

resource "grafana_data_source" "test" {
  type = "prometheus"
  name = "test-rule-sequence-ds-%[1]s"
  url  = "http://localhost:9090"
}

resource "grafana_apps_rules_recordingrule_v0alpha1" "test" {
  metadata {
    uid        = "test-seq-recording-rule-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    title = "Test Sequence Recording Rule %[1]s"
    trigger {
      interval = "1m"
    }
    target_datasource_uid = grafana_data_source.test.uid
    metric                = "tf_seq_metric_%[1]s"
    expressions = {
      "A" = jsonencode({
        model = {
          editorMode = "code"
          expr       = "count(up{})"
          instant    = true
          intervalMs = 1000
          refId      = "A"
        }
        datasource_uid = grafana_data_source.test.uid
        relative_time_range = {
          from = "600s"
          to   = "0s"
        }
        query_type = ""
        source     = true
      })
    }
  }
}

resource "grafana_apps_rules_alertrule_v0alpha1" "test" {
  metadata {
    uid        = "test-seq-alert-rule-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    title = "Test Sequence Alert Rule %[1]s"
    trigger {
      interval = "1m"
    }
    no_data_state  = "KeepLast"
    exec_err_state = "KeepLast"
    expressions = {
      "A" = jsonencode({
        model = {
          datasource = {
            type = "prometheus"
            uid  = grafana_data_source.test.uid
          }
          editorMode = "code"
          expr       = "count(up{})"
          instant    = true
          intervalMs = 1000
          refId      = "A"
        }
        datasource_uid = grafana_data_source.test.uid
        relative_time_range = {
          from = "600s"
          to   = "0s"
        }
        query_type = ""
        source     = true
      })
    }
  }
}
`, suffix)
}

func testAccRuleSequenceConfig(suffix, interval string) string {
	return testAccRuleSequenceRulesConfig(suffix) + fmt.Sprintf(`
resource "grafana_apps_rules_rulesequence_v0alpha1" "test" {
  metadata {
    uid        = "test-rule-sequence-%[1]s"
    folder_uid = grafana_folder.test.uid
  }
  spec {
    trigger {
      interval = %[2]q
    }
    recording_rules = [
      { name = grafana_apps_rules_recordingrule_v0alpha1.test.metadata.uid },
    ]
    alerting_rules = [
      { name = grafana_apps_rules_alertrule_v0alpha1.test.metadata.uid },
    ]
  }
}
`, suffix, interval)
}
