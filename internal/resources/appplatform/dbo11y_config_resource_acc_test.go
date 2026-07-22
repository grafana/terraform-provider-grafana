package appplatform_test

import (
	"context"
	"fmt"
	"testing"

	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	dbO11yConfigResourceType = "grafana_apps_productactivation_dbo11yconfig_v1alpha1"
	dbO11yConfigResourceName = dbO11yConfigResourceType + ".test"
	dbO11yConfigUID          = "global"
)

func TestAccDBO11yConfig_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	terraformresource.Test(t, terraformresource.TestCase{
		PreCheck:                 func() { testAccDeleteExistingDBO11yConfig(t) },
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckDBO11yConfigDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccDBO11yConfigConfig(true),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", dbO11yConfigUID),
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "spec.enabled", "true"),
					terraformresource.TestCheckResourceAttrSet(dbO11yConfigResourceName, "id"),
				),
			},
			{
				Config: testAccDBO11yConfigConfig(false),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(dbO11yConfigResourceName, "metadata.uid", dbO11yConfigUID),
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

// testAccDeleteExistingDBO11yConfig removes a leftover singleton from a prior run.
// DbO11yConfig is namespaced-unique with uid "global"; create fails with HTTP 409
// if a previous cloudinstance job left it behind.
func testAccDeleteExistingDBO11yConfig(t *testing.T) {
	t.Helper()

	client, err := testAccDBO11yConfigClient(testutils.Provider.Meta().(*common.Client))
	if err != nil {
		t.Fatalf("failed to create dbo11yconfig client: %v", err)
	}

	err = client.Delete(context.Background(), dbO11yConfigUID, sdkresource.DeleteOptions{})
	if err := testAccIgnoreNotFound(err); err != nil {
		t.Fatalf("failed to delete existing dbo11yconfig %q: %v", dbO11yConfigUID, err)
	}
}

func testAccCheckDBO11yConfigDestroy(s *terraform.State) error {
	client, err := testAccDBO11yConfigClient(testutils.Provider.Meta().(*common.Client))
	if err != nil {
		return fmt.Errorf("failed to create dbo11yconfig client: %w", err)
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != dbO11yConfigResourceType {
			continue
		}

		uid := rs.Primary.Attributes["metadata.uid"]
		if uid == "" {
			uid = dbO11yConfigUID
		}

		if _, err := client.Get(context.Background(), uid); err == nil {
			return fmt.Errorf("dbo11yconfig %q still exists", uid)
		} else if !testAccIsNotFound(err) {
			return fmt.Errorf("error checking if dbo11yconfig %q exists: %w", uid, err)
		}
	}
	return nil
}

func testAccDBO11yConfigClient(commonClient *common.Client) (*sdkresource.NamespacedClient[*appplatform.DBO11yConfig, *appplatform.DBO11yConfigList], error) {
	namespace, err := testAccNamespace(commonClient)
	if err != nil {
		return nil, err
	}

	kind := appplatform.DBO11yConfigKind()
	rcli, err := commonClient.GrafanaAppPlatformAPI.ClientFor(kind)
	if err != nil {
		return nil, err
	}

	return sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*appplatform.DBO11yConfig, *appplatform.DBO11yConfigList](rcli, kind),
		namespace,
	), nil
}

func testAccDBO11yConfigConfig(enabled bool) string {
	return fmt.Sprintf(`
resource "grafana_apps_productactivation_dbo11yconfig_v1alpha1" "test" {
  metadata {
    uid = %q
  }

  spec {
    enabled = %v
  }
}
`, dbO11yConfigUID, enabled)
}
