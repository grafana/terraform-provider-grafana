package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	serviceModelComponentResourceType = "grafana_apps_servicemodel_component_v1alpha1"
	serviceModelComponentResourceName = serviceModelComponentResourceType + ".test"
)

func TestAccServiceModelComponent_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	uid := fmt.Sprintf("tf-test-component-%s", acctest.RandString(6))

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckServiceModelComponentDestroy,
		Steps: []terraformresource.TestStep{
			{
				// Minimal config: uid and title only; everything else null or defaulted.
				Config: testAccServiceModelComponentMinimalConfig(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttrSet(serviceModelComponentResourceName, "id"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.title", "Minimal"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.type", "service"),
				),
			},
			{
				Config: testAccServiceModelComponentConfig(uid, "Checkout Service", "Handles checkout.", false),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttrSet(serviceModelComponentResourceName, "id"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.title", "Checkout Service"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.description", "Handles checkout."),
					// Provider-side default.
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.type", "service"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.identifiers.#", "1"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.identifiers.0.key", "service_name"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.identifiers.0.value", "checkout"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.depends_on_refs.#", "1"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.links.#", "1"),
					// Ref defaults materialized without being configured.
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.owner_ref.name", "team-checkout"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.owner_ref.api_version", "iam.grafana.app/v0alpha1"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.owner_ref.kind", "Team"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.depends_on_refs.0.name", "payments-api"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.depends_on_refs.0.api_version", "servicemodel.ext.grafana.com/v1alpha1"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.depends_on_refs.0.kind", "Component"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.links.0.url", "https://example.com/checkout"),
				),
			},
			{
				Config: testAccServiceModelComponentConfig(uid, "Checkout", "Handles checkout and payments.", true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.title", "Checkout"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.description", "Handles checkout and payments."),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.identifiers.#", "2"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.identifiers.1.key", "namespace"),
					// Computed default must survive updates.
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.type", "service"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.links.#", "1"),
					terraformresource.TestCheckResourceAttr(serviceModelComponentResourceName, "spec.links.0.title", "Repository"),
				),
			},
			{
				ResourceName:      serviceModelComponentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(serviceModelComponentResourceName),
			},
		},
	})
}

func testAccCheckServiceModelComponentDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != serviceModelComponentResourceType {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.ServiceModelComponentKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		var ns string
		switch {
		case client.GrafanaStackID > 0:
			ns = claims.CloudNamespaceFormatter(client.GrafanaStackID)
		default:
			ns = claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		}

		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*appplatform.ServiceModelComponent, *appplatform.ServiceModelComponentList](rcli, appplatform.ServiceModelComponentKind()),
			ns,
		)

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("service model component %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if service model component %s exists: %w", uid, err)
		}
	}
	return nil
}

func testAccServiceModelComponentMinimalConfig(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_servicemodel_component_v1alpha1" "test" {
  metadata {
    uid = %[1]q
  }

  spec {
    title = "Minimal"
  }
}
`, uid)
}

func testAccServiceModelComponentConfig(uid, title, description string, extraIdentifier bool) string {
	extra := ""
	linkTitle := "Source code"
	if extraIdentifier {
		extra = `
    identifiers {
      key   = "namespace"
      value = "checkout-prod"
    }
`
		linkTitle = "Repository"
	}

	return fmt.Sprintf(`
resource "grafana_apps_servicemodel_component_v1alpha1" "test" {
  metadata {
    uid = %[1]q
  }

  spec {
    title       = %[2]q
    description = %[3]q

    identifiers {
      key   = "service_name"
      value = "checkout"
    }
%[4]s
    owner_ref {
      name = "team-checkout"
    }

    depends_on_refs {
      name = "payments-api"
    }

    links {
      url   = "https://example.com/checkout"
      title = %[5]q
      type  = "repository"
    }
  }
}
`, uid, title, description, extra, linkTitle)
}
