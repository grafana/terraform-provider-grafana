package appplatform_test

import (
	"fmt"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// TestAccAlertmanagerResources_concurrentCreate applies many Alertmanager-backed resources in a
// single apply. They have no dependencies on each other, so Terraform creates them concurrently,
// and they all read-modify-write the org's shared Alertmanager config under optimistic locking.
// Without RetryCreateOnConflict, this reliably hits 409 Conflict on create.
//
// The destroy at the end deletes them concurrently as well, so a failure there means deletes
// need the same conflict handling.
func TestAccAlertmanagerResources_concurrentCreate(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.4.0")

	const count = 6
	prefix := fmt.Sprintf("test-concurrent-%s", acctest.RandString(6))

	checks := []terraformresource.TestCheckFunc{}
	for i := 0; i < count; i++ {
		checks = append(checks,
			terraformresource.TestCheckResourceAttr(fmt.Sprintf("%s.test.%d", templateGroupResourceType, i), "metadata.uid", fmt.Sprintf("%s-tg-%d", prefix, i)),
			terraformresource.TestCheckResourceAttr(fmt.Sprintf("%s.test.%d", timeIntervalResourceType, i), "metadata.uid", fmt.Sprintf("%s-ti-%d", prefix, i)),
			terraformresource.TestCheckResourceAttr(fmt.Sprintf("%s.test.%d", inhibitionRuleResourceType, i), "metadata.uid", fmt.Sprintf("%s-ir-%d", prefix, i)),
		)
	}

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: func(s *terraform.State) error {
			for _, check := range []terraformresource.TestCheckFunc{
				testAccCheckTemplateGroupDestroy,
				testAccCheckTimeIntervalDestroy,
				testAccCheckInhibitionRuleDestroy,
			} {
				if err := check(s); err != nil {
					return err
				}
			}
			return nil
		},
		Steps: []terraformresource.TestStep{
			{
				Config: testAccAlertmanagerConcurrentConfig(prefix, count),
				Check:  terraformresource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

func testAccAlertmanagerConcurrentConfig(prefix string, count int) string {
	return fmt.Sprintf(`
locals {
  prefix = %[1]q
  count  = %[2]d
}

resource "grafana_apps_notifications_templategroup_v1beta1" "test" {
  count = local.count

  metadata {
    uid = "${local.prefix}-tg-${count.index}"
  }

  spec {
    title   = "Concurrent ${local.prefix} ${count.index}"
    content = "{{ define \"${local.prefix}_tg_${count.index}\" }}hello{{ end }}"
  }
}

resource "grafana_apps_notifications_timeinterval_v1beta1" "test" {
  count = local.count

  metadata {
    uid = "${local.prefix}-ti-${count.index}"
  }

  spec {
    name = "Concurrent ${local.prefix} ${count.index}"

    time_intervals {
      weekdays = ["monday:friday"]
    }
  }
}

resource "grafana_apps_notifications_inhibitionrule_v1beta1" "test" {
  count = local.count

  metadata {
    uid = "${local.prefix}-ir-${count.index}"
  }

  spec {
    source_matchers = [{ type = "=", label = "alertname", value = "Source${count.index}" }]
    target_matchers = [{ type = "=", label = "alertname", value = "Target${count.index}" }]
    equal           = ["namespace"]
  }
}
`, prefix, count)
}
