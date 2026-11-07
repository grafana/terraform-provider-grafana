package slo_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/slo-openapi-client/go/slo"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// notificationSettingsConfig is an SLO whose fast and slow burn rates carry the given
// notification_settings bodies (an empty string omits the block). The contact point is
// created in the same config so the SLO can reference it.
func notificationSettingsConfig(name, contactPoint, fastBody, slowBody string) string {
	block := func(body string) string {
		if body == "" {
			return ""
		}
		return fmt.Sprintf("notification_settings {\n%s\n}", body)
	}
	return fmt.Sprintf(`
resource "grafana_contact_point" "cp" {
  name = "%[2]s"
  email {
    addresses = ["slo-notification-settings@example.com"]
  }
}

resource "grafana_slo" "test" {
  name        = "%[1]s"
  description = "%[1]s"
  query {
    type = "freeform"
    freeform {
      query = "sum(rate(apiserver_request_total{code!=\"500\"}[$__rate_interval])) / sum(rate(apiserver_request_total[$__rate_interval]))"
    }
  }
  objectives {
    value  = 0.995
    window = "28d"
  }
  destination_datasource {
    uid = "grafanacloud-prom"
  }
  alerting {
    fastburn {
      annotation {
        key   = "name"
        value = "fast"
      }
      %[3]s
    }
    slowburn {
      annotation {
        key   = "name"
        value = "slow"
      }
      %[4]s
    }
  }
}
`, name, contactPoint, block(fastBody), block(slowBody))
}

// testNotificationSettingsOnAPI checks what the SLO API actually stores for a burn rate,
// independent of Terraform state. wantContactPoint "" means no notification settings.
func testNotificationSettingsOnAPI(rn, burn, wantContactPoint string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs := s.RootModule().Resources[rn]
		client := testutils.Provider.Meta().(*common.Client).SLOClient
		got, _, err := client.DefaultAPI.V1SloIdGet(context.Background(), rs.Primary.ID).Execute()
		if err != nil {
			return fmt.Errorf("error getting SLO: %w", err)
		}
		var meta *slo.SloV00AlertingMetadata
		if got.Alerting != nil {
			meta = got.Alerting.FastBurn
			if burn == "slowburn" {
				meta = got.Alerting.SlowBurn
			}
		}
		var ns *slo.SloV00NotificationSettings
		if meta != nil {
			ns = meta.NotificationSettings
		}
		if wantContactPoint == "" {
			if ns != nil {
				return fmt.Errorf("%s: expected no notification settings, got %+v", burn, ns)
			}
			return nil
		}
		if ns == nil || ns.ContactPoint == nil || *ns.ContactPoint != wantContactPoint {
			return fmt.Errorf("%s: expected contact point %q, got %+v", burn, wantContactPoint, ns)
		}
		if ns.Policy != nil {
			return fmt.Errorf("%s: expected no policy, got %q", burn, *ns.Policy)
		}
		return nil
	}
}

// TestAccResourceSloNotificationSettings covers the whole cycle for a contact point:
// create, an empty plan afterwards (no drift), import, and removing a block. The policy
// shape has no Terraform resource to create a named policy tree, so it is covered by the
// unit tests and the validation test below.
func TestAccResourceSloNotificationSettings(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	name := acctest.RandomWithPrefix("SLO Terraform Testing")
	cp := acctest.RandomWithPrefix("slo-notification-settings")
	const (
		fast = `contact_point = grafana_contact_point.cp.name`
		slow = `contact_point = grafana_contact_point.cp.name`
	)

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: notificationSettingsConfig(name, cp, fast, slow),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_slo.test", "alerting.0.fastburn.0.notification_settings.0.contact_point", cp),
					resource.TestCheckNoResourceAttr("grafana_slo.test", "alerting.0.fastburn.0.notification_settings.0.policy"),
					resource.TestCheckResourceAttr("grafana_slo.test", "alerting.0.slowburn.0.notification_settings.0.contact_point", cp),
					testNotificationSettingsOnAPI("grafana_slo.test", "fastburn", cp),
					testNotificationSettingsOnAPI("grafana_slo.test", "slowburn", cp),
				),
			},
			{
				// Re-planning the same config must produce no diff.
				Config:   notificationSettingsConfig(name, cp, fast, slow),
				PlanOnly: true,
			},
			{
				ResourceName:      "grafana_slo.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			{
				// Removing the slow burn block clears it on the API; fast burn is untouched.
				Config: notificationSettingsConfig(name, cp, fast, ""),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_slo.test", "alerting.0.slowburn.0.notification_settings.#", "0"),
					testNotificationSettingsOnAPI("grafana_slo.test", "fastburn", cp),
					testNotificationSettingsOnAPI("grafana_slo.test", "slowburn", ""),
				),
			},
			{
				Config:   notificationSettingsConfig(name, cp, fast, ""),
				PlanOnly: true,
			},
		},
	})
}

// TestAccResourceSloNotificationSettingsInvalid checks that bad input is rejected at
// plan time, before any API call: the API would reject all of it, and the empty string
// would otherwise round-trip as unset and fail after apply.
func TestAccResourceSloNotificationSettingsInvalid(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	name := acctest.RandomWithPrefix("SLO Terraform Testing")
	cases := map[string]struct {
		body      string
		wantError string
	}{
		"both contact_point and policy":    {`contact_point = "a"` + "\n" + `policy = "b"`, "and only one"},
		"neither contact_point nor policy": {`# empty block`, "and only one"},
		"empty contact_point":              {`contact_point = ""`, "contact_point must be non-empty if set"},
		"empty policy":                     {`policy = ""`, "policy must be non-empty if set"},
	}

	steps := []resource.TestStep{}
	for _, tc := range cases {
		steps = append(steps, resource.TestStep{
			Config:      notificationSettingsConfig(name, "unused", tc.body, ""),
			PlanOnly:    true,
			ExpectError: wrappedMessage(tc.wantError),
		})
	}

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps:                    steps,
	})
}

// wrappedMessage matches msg whatever way Terraform wraps it across lines.
func wrappedMessage(msg string) *regexp.Regexp {
	words := strings.Fields(msg)
	for i, w := range words {
		words[i] = regexp.QuoteMeta(w)
	}
	return regexp.MustCompile(strings.Join(words, `\s+`))
}
