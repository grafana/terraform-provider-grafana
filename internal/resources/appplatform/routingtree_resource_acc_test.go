package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	routingTreeResourceType = "grafana_apps_notifications_routingtree_v1beta1"
	routingTreeResourceName = routingTreeResourceType + ".test"
)

func TestAccRoutingTree(t *testing.T) {
	// RoutingTree (named/multiple routing trees) requires the `alertingMultiplePolicies`
	// feature toggle, available since Grafana 13.1. The toggle is enabled on the test
	// instance via GF_FEATURE_TOGGLES_ENABLE in docker-compose.yml.
	testutils.CheckOSSTestsEnabled(t, ">=13.1.0")

	t.Run("basic", func(t *testing.T) {
		uid := fmt.Sprintf("test-routing-tree-%s", acctest.RandString(6))

		// Routing trees are persisted in the global Alertmanager configuration, which
		// uses optimistic locking. Run sequentially (not ParallelTest) so concurrent
		// creates don't collide on the shared config.
		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckRoutingTreeDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccRoutingTreeConfigBasic(uid),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.defaults.receiver", "empty"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.defaults.group_by.#", "2"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.#", "1"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.continue", "false"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.matchers.0.label", "severity"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.disable_provenance", "false"),
						terraformresource.TestCheckResourceAttrSet(routingTreeResourceName, "id"),
					),
				},
				{
					ResourceName:      routingTreeResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(routingTreeResourceName),
				},
				{
					// Toggling disable_provenance updates the provenance annotation.
					Config: testAccRoutingTreeConfigProvenance(uid, true),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.disable_provenance", "true"),
					),
				},
			},
		})
	})

	t.Run("nested routes", func(t *testing.T) {
		uid := fmt.Sprintf("test-routing-tree-%s", acctest.RandString(6))

		// Routing trees are persisted in the global Alertmanager configuration, which
		// uses optimistic locking. Run sequentially (not ParallelTest) so concurrent
		// creates don't collide on the shared config.
		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckRoutingTreeDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccRoutingTreeConfigNested(uid),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.#", "1"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.continue", "true"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.routes.#", "1"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.routes.0.receiver", "empty"),
						terraformresource.TestCheckResourceAttr(routingTreeResourceName, "spec.routes.0.routes.0.matchers.0.type", "=~"),
					),
				},
			},
		})
	})
}

func testAccCheckRoutingTreeDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != routingTreeResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.RoutingTreeKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*v1beta1.RoutingTree, *v1beta1.RoutingTreeList](rcli, v1beta1.RoutingTreeKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("RoutingTree %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if RoutingTree %s exists: %w", uid, err)
		}
	}
	return nil
}

func testAccRoutingTreeConfigBasic(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_notifications_routingtree_v1beta1" "test" {
  metadata {
    uid = %q
  }

  spec {
    defaults {
      receiver        = "empty"
      group_by        = ["grafana_folder", "alertname"]
      group_wait      = "30s"
      group_interval  = "5m"
      repeat_interval = "4h"
    }

    routes {
      receiver = "empty"
      continue = false

      matchers = [
        {
          type  = "="
          label = "severity"
          value = "critical"
        }
      ]
    }
  }
}
`, uid)
}

func testAccRoutingTreeConfigProvenance(uid string, disable bool) string {
	return fmt.Sprintf(`
resource "grafana_apps_notifications_routingtree_v1beta1" "test" {
  metadata {
    uid = %q
  }

  spec {
    disable_provenance = %t

    defaults {
      receiver        = "empty"
      group_by        = ["grafana_folder", "alertname"]
      group_wait      = "30s"
      group_interval  = "5m"
      repeat_interval = "4h"
    }

    routes {
      receiver = "empty"
      continue = false

      matchers = [
        {
          type  = "="
          label = "severity"
          value = "critical"
        }
      ]
    }
  }
}
`, uid, disable)
}

func testAccRoutingTreeConfigNested(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_notifications_routingtree_v1beta1" "test" {
  metadata {
    uid = %q
  }

  spec {
    defaults {
      receiver = "empty"
      group_by = ["alertname"]
    }

    routes {
      receiver = "empty"
      continue = true

      matchers = [
        {
          type  = "="
          label = "team"
          value = "backend"
        }
      ]

      routes {
        receiver = "empty"
        continue = false

        matchers = [
          {
            type  = "=~"
            label = "service"
            value = "api|web"
          }
        ]
      }
    }
  }
}
`, uid)
}
