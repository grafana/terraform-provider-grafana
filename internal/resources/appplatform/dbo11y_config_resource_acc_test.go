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
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
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
		CheckDestroy:             testAccCheckDBO11yConfigDestroy,
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

func dbO11yConfigNamespacedClient() (*resource.NamespacedClient[*appplatform.DBO11yConfig, *appplatform.DBO11yConfigList], error) {
	client := testutils.Provider.Meta().(*common.Client)

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.DBO11yConfigKind())
	if err != nil {
		return nil, fmt.Errorf("failed to create app platform client: %w", err)
	}

	var ns string
	switch {
	case client.GrafanaStackID > 0:
		ns = claims.CloudNamespaceFormatter(client.GrafanaStackID)
	default:
		ns = claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	}

	return resource.NewNamespaced(
		resource.NewTypedClient[*appplatform.DBO11yConfig, *appplatform.DBO11yConfigList](rcli, appplatform.DBO11yConfigKind()),
		ns,
	), nil
}

func testAccDeleteDBO11yConfigSingleton(t *testing.T) {
	t.Helper()

	namespacedClient, err := dbO11yConfigNamespacedClient()
	if err != nil {
		t.Fatal(err)
	}

	if err := namespacedClient.Delete(context.Background(), dbO11yConfigSingletonUID, resource.DeleteOptions{}); err != nil && !apierrors.IsNotFound(err) {
		t.Fatalf("failed to delete stale dbO11yConfig %q: %v", dbO11yConfigSingletonUID, err)
	}
}

func testAccCheckDBO11yConfigDestroy(s *terraform.State) error {
	for _, r := range s.RootModule().Resources {
		if r.Type != dbO11yConfigResourceType {
			continue
		}

		namespacedClient, err := dbO11yConfigNamespacedClient()
		if err != nil {
			return err
		}

		uid := r.Primary.Attributes["metadata.uid"]
		if _, err := namespacedClient.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("DBO11yConfig %s still exists", uid)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if DBO11yConfig %s exists: %w", uid, err)
		}
	}
	return nil
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
