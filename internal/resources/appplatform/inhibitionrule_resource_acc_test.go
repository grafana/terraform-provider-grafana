package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	inhibitionRuleResourceType = "grafana_apps_notifications_inhibitionrule_v0alpha1"
	inhibitionRuleResourceName = inhibitionRuleResourceType + ".test"
)

func TestAccInhibitionRule(t *testing.T) {
	t.Skip("inhibition rules API requires Grafana >=13.0.0-22301942504; enable once a compatible instance is available in CI")
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0-22301942504")

	t.Run("basic", func(t *testing.T) {
		uid := fmt.Sprintf("test-inhibition-rule-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckInhibitionRuleDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccInhibitionRuleConfig(uid,
						[]inhibitionRuleMatcher{
							{matchType: "=", label: "alertname", value: "TargetDown"},
						},
						[]inhibitionRuleMatcher{
							{matchType: "=", label: "severity", value: "warning"},
						},
						[]string{"namespace", "pod"},
					),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.0.type", "="),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.0.label", "alertname"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.0.value", "TargetDown"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.0.type", "="),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.0.label", "severity"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.0.value", "warning"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.#", "2"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.0", "namespace"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.1", "pod"),
						terraformresource.TestCheckResourceAttrSet(inhibitionRuleResourceName, "id"),
					),
				},
				{
					ResourceName:      inhibitionRuleResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateIdFunc: importStateIDFunc(inhibitionRuleResourceName),
				},
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		uid := fmt.Sprintf("test-inhibition-rule-%s", acctest.RandString(6))

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckInhibitionRuleDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccInhibitionRuleConfig(uid,
						[]inhibitionRuleMatcher{
							{matchType: "=", label: "alertname", value: "TargetDown"},
						},
						[]inhibitionRuleMatcher{
							{matchType: "=", label: "severity", value: "warning"},
						},
						[]string{"namespace"},
					),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.#", "1"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.0", "namespace"),
					),
				},
				{
					Config: testAccInhibitionRuleConfig(uid,
						[]inhibitionRuleMatcher{
							{matchType: "=", label: "alertname", value: "TargetDown"},
							{matchType: "!=", label: "env", value: "test"},
						},
						[]inhibitionRuleMatcher{
							{matchType: "=~", label: "severity", value: "warning|critical"},
						},
						[]string{"namespace", "pod", "container"},
					),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.#", "2"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.1.type", "!="),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.1.label", "env"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.source_matchers.1.value", "test"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.0.type", "=~"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.target_matchers.0.value", "warning|critical"),
						terraformresource.TestCheckResourceAttr(inhibitionRuleResourceName, "spec.equal.#", "3"),
					),
				},
			},
		})
	})
}

func testAccCheckInhibitionRuleDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != inhibitionRuleResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v0alpha1.InhibitionRuleKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*v0alpha1.InhibitionRule, *v0alpha1.InhibitionRuleList](rcli, v0alpha1.InhibitionRuleKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("InhibitionRule %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if InhibitionRule %s exists: %w", uid, err)
		}
	}
	return nil
}

type inhibitionRuleMatcher struct {
	matchType string
	label     string
	value     string
}

func testAccInhibitionRuleConfig(uid string, sourceMatchers, targetMatchers []inhibitionRuleMatcher, equal []string) string {
	equalHCL := ""
	for i, e := range equal {
		if i > 0 {
			equalHCL += ", "
		}
		equalHCL += fmt.Sprintf("%q", e)
	}

	return fmt.Sprintf(`
resource "grafana_apps_notifications_inhibitionrule_v0alpha1" "test" {
  metadata {
    uid = %q
  }

  spec {
%s
%s
    equal = [%s]
  }
}
`, uid, buildInhibitionRuleMatchersHCL("source_matchers", sourceMatchers), buildInhibitionRuleMatchersHCL("target_matchers", targetMatchers), equalHCL)
}

func buildInhibitionRuleMatchersHCL(field string, matchers []inhibitionRuleMatcher) string {
	if len(matchers) == 0 {
		return ""
	}
	result := fmt.Sprintf("    %s = [\n", field)
	for _, m := range matchers {
		result += fmt.Sprintf("      { type = %q, label = %q, value = %q },\n", m.matchType, m.label, m.value)
	}
	result += "    ]"
	return result
}
