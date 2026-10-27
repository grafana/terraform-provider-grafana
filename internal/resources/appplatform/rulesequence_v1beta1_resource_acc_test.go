package appplatform_test

import (
	"context"
	"fmt"
	"strings"
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

const ruleSequenceResourceTypePrefix = "grafana_apps_rules_rulesequence_"

func TestAccRuleSequence(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.1.0")

	for _, tc := range []struct {
		version         string
		grafanaVersions string
		alertRulesAttr  string
	}{
		{version: "v0alpha1", grafanaVersions: ">=13.1.0", alertRulesAttr: "alerting_rules"},
		{version: "v1beta1", grafanaVersions: ">=13.3.0", alertRulesAttr: "alert_rules"},
	} {
		t.Run(tc.version, func(t *testing.T) {
			testutils.CheckOSSTestsEnabled(t, tc.grafanaVersions)
			testAccRuleSequenceBasic(t, tc.version, tc.alertRulesAttr)
		})
	}
}

func testAccRuleSequenceBasic(t *testing.T, version, alertRulesAttr string) {
	ruleSequenceResourceName := ruleSequenceResourceTypePrefix + version + ".test"

	t.Run("basic", func(t *testing.T) {
		suffix := acctest.RandString(6)
		uid := fmt.Sprintf("test-rule-sequence-%s", suffix)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckRuleSequenceDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccRuleSequenceConfig(version, alertRulesAttr, suffix, "1m"),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.trigger.interval", "1m"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.recording_rules.#", "1"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec.recording_rules.0.name", fmt.Sprintf("test-seq-recording-rule-%s", suffix)),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec."+alertRulesAttr+".#", "1"),
						terraformresource.TestCheckResourceAttr(ruleSequenceResourceName, "spec."+alertRulesAttr+".0.name", fmt.Sprintf("test-seq-alert-rule-%s", suffix)),
						terraformresource.TestCheckResourceAttrSet(ruleSequenceResourceName, "id"),
					),
				},
				{
					Config: testAccRuleSequenceConfig(version, alertRulesAttr, suffix, "5m"),
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
					Config: testAccRuleSequenceRulesConfig(version, suffix),
					Check:  testAccCheckRuleSequenceDeleted(uid),
				},
			},
		})
	})
}

func testAccCheckRuleSequenceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		// v0alpha1 stays served alongside v1beta1, so its client checks sequences created through either version.
		if !strings.HasPrefix(r.Type, ruleSequenceResourceTypePrefix) {
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

// testAccRuleSequenceRulesConfig renders the rules a sequence references for the given API version.
// v1beta1 types the expressions instead of encoding each one as JSON, so each version has its own config.
func testAccRuleSequenceRulesConfig(version, suffix string) string {
	if version == "v0alpha1" {
		return testAccRuleSequenceDependenciesConfig(suffix) + testAccRuleSequenceRulesV0Alpha1Config(suffix)
	}
	return testAccRuleSequenceDependenciesConfig(suffix) + testAccRuleSequenceRulesV1Beta1Config(suffix)
}

func testAccRuleSequenceDependenciesConfig(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_folder" "test" {
  title = "test-rule-sequence-folder-%[1]s"
}

resource "grafana_data_source" "test" {
  type = "prometheus"
  name = "test-rule-sequence-ds-%[1]s"
  url  = "http://localhost:9090"
}

resource "grafana_contact_point" "test" {
  name = "test-rule-sequence-cp-%[1]s"
  email {
    addresses = ["test@example.com"]
  }
}
`, suffix)
}

func testAccRuleSequenceRulesV1Beta1Config(suffix string) string {
	return fmt.Sprintf(`
resource "grafana_apps_rules_recordingrule_v1beta1" "test" {
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
    }
  }
}

resource "grafana_apps_rules_alertrule_v1beta1" "test" {
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
    notification_settings {
      simplified_routing {
        contact_point = grafana_contact_point.test.name
      }
    }
    expressions = {
      "A" = {
        model = jsonencode({
          datasource = {
            type = "prometheus"
            uid  = grafana_data_source.test.uid
          }
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
    }
  }
}
`, suffix)
}

func testAccRuleSequenceRulesV0Alpha1Config(suffix string) string {
	return fmt.Sprintf(`
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
    notification_settings {
      simplified_routing {
        contact_point = grafana_contact_point.test.name
      }
    }
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

// testAccRuleSequenceConfig renders the sequence config for the given API version. v1beta1 renamed
// alerting_rules to alert_rules, so alertRulesAttr names that attribute.
func testAccRuleSequenceConfig(version, alertRulesAttr, suffix, interval string) string {
	return testAccRuleSequenceRulesConfig(version, suffix) + strings.Replace(withRulesVersion(fmt.Sprintf(`
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
`, suffix, interval), version), "alerting_rules =", alertRulesAttr+" =", 1)
}

// withRulesVersion rewrites a config written against the v0alpha1 rules resources to use the given API version.
// The configs stay literal v0alpha1 HCL so that tools parsing them (such as genreferences) see real resource types.
func withRulesVersion(config, version string) string {
	return strings.ReplaceAll(config, "_v0alpha1", "_"+version)
}
