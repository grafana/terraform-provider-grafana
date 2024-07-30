package cloud_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/grafana/terraform-provider-grafana/v2/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v2/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccCloudApiKey_Basic(t *testing.T) {
	testutils.CheckCloudAPITestsEnabled(t)

	prefix := "testcloudkey-"
	testAccDeleteExistingCloudAPIKeys(t, prefix)

	var tests = []struct {
		role string
	}{
		{"Viewer"},
		{"Editor"},
		{"Admin"},
		{"MetricsPublisher"},
		{"PluginPublisher"},
	}

	for _, tt := range tests {
		t.Run(tt.role, func(t *testing.T) {
			resourceName := prefix + acctest.RandStringFromCharSet(5, acctest.CharSetAlphaNum)

			resource.ParallelTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             testAccCheckCloudAPIKeyDestroy(resourceName),
				Steps: []resource.TestStep{
					{
						Config: testAccCloudAPIKeyConfig(resourceName, tt.role),
						Check: resource.ComposeTestCheckFunc(
							testAccCheckCloudAPIKeyExists(resourceName),
							resource.TestCheckResourceAttrSet("grafana_cloud_api_key.test", "id"),
							resource.TestCheckResourceAttrSet("grafana_cloud_api_key.test", "key"),
							resource.TestCheckResourceAttr("grafana_cloud_api_key.test", "name", resourceName),
							resource.TestCheckResourceAttr("grafana_cloud_api_key.test", "role", tt.role),
						),
					},
					{
						ResourceName:            "grafana_cloud_api_key.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"key"},
					},
					// Test import/read with the ID format from version 2.12.2 and earlier
					{
						ResourceName:            "grafana_cloud_api_key.test",
						ImportState:             true,
						ImportStateVerify:       true,
						ImportStateVerifyIgnore: []string{"key"},
						ImportStateId:           fmt.Sprintf("%s-%s", os.Getenv("GRAFANA_CLOUD_ORG"), resourceName),
					},
				},
			})
		})
	}
}

func testAccCheckCloudAPIKeyExists(apiKeyName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
		res, _, err := client.OrgsAPI.GetApiKeys(context.Background(), os.Getenv("GRAFANA_CLOUD_ORG")).Execute()
		if err != nil {
			return err
		}

		for _, apiKey := range res.Items {
			if apiKey.Name == apiKeyName {
				return nil
			}
		}

		return fmt.Errorf("API Key `%s` not found via API", apiKeyName)
	}
}

func testAccCheckCloudAPIKeyDestroy(apiKeyName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		getErr := testAccCheckCloudAPIKeyExists(apiKeyName)(s)
		if getErr == nil {
			return fmt.Errorf("API Key `%s` still exists via API", apiKeyName)
		}
		return nil
	}
}

func testAccDeleteExistingCloudAPIKeys(t *testing.T, prefix string) {
	org := os.Getenv("GRAFANA_CLOUD_ORG")
	client := testutils.Provider.Meta().(*common.Client).GrafanaCloudAPI
	resp, _, err := client.OrgsAPI.GetApiKeys(context.Background(), org).Execute()
	if err != nil {
		t.Error(err)
	}

	for _, key := range resp.Items {
		if strings.HasPrefix(key.Name, prefix) {
			_, err := client.OrgsAPI.DelApiKey(context.Background(), key.Name, org).XRequestId(cloud.ClientRequestID()).Execute()
			if err != nil {
				t.Error(err)
			}
		}
	}
}

func testAccCloudAPIKeyConfig(resourceName, role string) string {
	// GRAFANA_CLOUD_ORG is required from the `testutils.CheckCloudAPITestsEnabled` function
	return fmt.Sprintf(`
resource "grafana_cloud_api_key" "test" {
  cloud_org_slug = "%s"
  name           = "%s"
  role           = "%s"
}
`, os.Getenv("GRAFANA_CLOUD_ORG"), resourceName, role)
}
