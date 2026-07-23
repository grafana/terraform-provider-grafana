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
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	dbO11yConfigResourceType = "grafana_apps_productactivation_dbo11yconfig_v1alpha1"
	dbO11yConfigResourceName = dbO11yConfigResourceType + ".test"
	dbO11yConfigSingletonUID = "global"
)

func TestAccDBO11yConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		PreCheck:                 func() { testAccDeleteDBO11yConfigSingleton(t) },
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDBO11yConfigConfig(true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", dbO11yConfigSingletonUID),
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "spec.enabled", "true"),
					terraformresource.TestCheckResourceAttrSet(dbO11yConfigResourceName, "id"),
				),
			},
			{
				Config: testAccDBO11yConfigConfig(false),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", dbO11yConfigSingletonUID),
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "spec.enabled", "false"),
				),
			},
			{
				ResourceName:      dbO11yConfigResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(dbO11yConfigResourceName),
			},
		},
	})
}

func testAccDeleteDBO11yConfigSingleton(t *testing.T) {
	t.Helper()

	client := testutils.Provider.Meta().(*common.Client)

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.DBO11yConfigKind())
	if err != nil {
		t.Fatalf("failed to create app platform client: %v", err)
	}

	var ns string
	switch {
	case client.GrafanaStackID > 0:
		ns = claims.CloudNamespaceFormatter(client.GrafanaStackID)
	default:
		ns = claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	}

	namespacedClient := resource.NewNamespaced(
		resource.NewTypedClient[*appplatform.DBO11yConfig, *appplatform.DBO11yConfigList](rcli, appplatform.DBO11yConfigKind()),
		ns,
	)

	if err := namespacedClient.Delete(context.Background(), dbO11yConfigSingletonUID, resource.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("failed to delete stale dbO11yConfig %q: %v", dbO11yConfigSingletonUID, err)
	}
}

func testAccDBO11yConfigConfig(enabled bool) string {
	return fmt.Sprintf(`
resource "grafana_apps_productactivation_dbo11yconfig_v1alpha1" "test" {
  metadata {
    uid = "global"
  }

  spec {
    enabled = %v
  }
}
`, enabled)
}
