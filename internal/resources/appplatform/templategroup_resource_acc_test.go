package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	templateGroupResourceType = "grafana_apps_notifications_templategroup_v1beta1"
	templateGroupResourceName = templateGroupResourceType + ".test"
)

func TestAccTemplateGroup(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.4.0")

	// The subtests deliberately use terraformresource.Test instead of ParallelTest and don't call
	// t.Parallel(): template groups live in the org's Alertmanager config, which uses optimistic
	// locking, so parallel writes get a 409.

	t.Run("basic", func(t *testing.T) {
		uid := fmt.Sprintf("test-template-group-%s", acctest.RandString(6))
		title := "Title " + uid

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckTemplateGroupDestroy,
			Steps: []terraformresource.TestStep{
				{
					// disable_provenance is omitted to cover its documented default (false).
					Config: testAccTemplateGroupConfig(uid, title, "hello", nil),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.title", title),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.content", testAccTemplateGroupContent(uid, "hello")),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.disable_provenance", "false"),
						terraformresource.TestCheckResourceAttrSet(templateGroupResourceName, "id"),
					),
				},
				testAccTemplateGroupImportStep(),
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		uid := fmt.Sprintf("test-template-group-%s", acctest.RandString(6))
		title := "Title " + uid

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckTemplateGroupDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccTemplateGroupConfig(uid, title, "before", boolPtr(false)),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.content", testAccTemplateGroupContent(uid, "before")),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.disable_provenance", "false"),
					),
				},
				{
					// false -> true: allow editing from other sources.
					Config: testAccTemplateGroupConfig(uid, title, "after", boolPtr(true)),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.content", testAccTemplateGroupContent(uid, "after")),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.disable_provenance", "true"),
					),
				},
				testAccTemplateGroupImportStep(),
				{
					// true -> false: restore Terraform-only management.
					Config: testAccTemplateGroupConfig(uid, title, "after", boolPtr(false)),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.content", testAccTemplateGroupContent(uid, "after")),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.disable_provenance", "false"),
					),
				},
				testAccTemplateGroupImportStep(),
			},
		})
	})

	t.Run("rename keeps uid", func(t *testing.T) {
		uid := fmt.Sprintf("test-template-group-%s", acctest.RandString(6))
		var uuid string

		terraformresource.Test(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccCheckTemplateGroupDestroy,
			Steps: []terraformresource.TestStep{
				{
					Config: testAccTemplateGroupConfig(uid, "Before "+uid, "body", nil),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.title", "Before "+uid),
						captureAttr(templateGroupResourceName, "metadata.uuid", &uuid),
					),
				},
				{
					Config: testAccTemplateGroupConfig(uid, "After "+uid, "body", nil),
					// metadata.uuid is derived from the resource identity, so an unchanged value doesn't prove
					// the rename was an in-place update; title has no RequiresReplace, so it is by schema.
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "metadata.uid", uid),
						terraformresource.TestCheckResourceAttr(templateGroupResourceName, "spec.title", "After "+uid),
						func(s *terraform.State) error {
							got := s.RootModule().Resources[templateGroupResourceName].Primary.Attributes["metadata.uuid"]
							if got != uuid {
								return fmt.Errorf("renaming changed the resource identity: metadata.uuid changed from %q to %q", uuid, got)
							}
							return nil
						},
					),
				},
				// Importing reads title, content and provenance back from Grafana, so this verifies the
				// rename was persisted and not only planned.
				testAccTemplateGroupImportStep(),
			},
		})
	})
}

// testAccTemplateGroupImportStep verifies persisted values through import because
// the shared engine doesn't refresh spec during Read.
func testAccTemplateGroupImportStep() terraformresource.TestStep {
	return terraformresource.TestStep{
		ResourceName:      templateGroupResourceName,
		ImportState:       true,
		ImportStateVerify: true,
		ImportStateVerifyIgnore: []string{
			"options.%",
			"options.overwrite",
		},
		ImportStateIdFunc: importStateIDFunc(templateGroupResourceName),
	}
}

func testAccCheckTemplateGroupDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != templateGroupResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.TemplateGroupKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := sdkresource.NewNamespaced(
			sdkresource.NewTypedClient[*v1beta1.TemplateGroup, *v1beta1.TemplateGroupList](rcli, v1beta1.TemplateGroupKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("TemplateGroup %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if TemplateGroup %s exists: %w", uid, err)
		}
	}
	return nil
}

func boolPtr(b bool) *bool { return &b }

// testAccTemplateGroupContent returns trimmed content with a define block: the server trims
// content and wraps it in a define block when missing, which would not round-trip through import.
func testAccTemplateGroupContent(uid, body string) string {
	return fmt.Sprintf(`{{ define "tmpl_%s" }}%s{{ end }}`, uid, body)
}

// testAccTemplateGroupConfig renders the resource. A nil disableProvenance omits the attribute.
func testAccTemplateGroupConfig(uid, title, body string, disableProvenance *bool) string {
	provenance := ""
	if disableProvenance != nil {
		provenance = fmt.Sprintf("    disable_provenance = %t\n", *disableProvenance)
	}
	return fmt.Sprintf(`
resource "grafana_apps_notifications_templategroup_v1beta1" "test" {
  metadata {
    uid = %[1]q
  }

  spec {
    title   = %[2]q
%[3]s    content = %[4]q
  }
}
`, uid, title, provenance, testAccTemplateGroupContent(uid, body))
}
