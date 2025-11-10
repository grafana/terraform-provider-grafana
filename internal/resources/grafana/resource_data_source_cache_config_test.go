package grafana_test

import (
	"fmt"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	// Helper to check existence of cache config via Enterprise API
	cacheConfigCheckExists = newCheckExistsHelper(
		func(c *models.CacheConfigResponse) string { return c.DataSourceUID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.CacheConfigResponse, error) {
			resp, err := client.Enterprise.GetDataSourceCacheConfig(id)
			return payloadOrError(resp, err)
		},
	)
)

func cacheConfigDestroyed(v *models.CacheConfigResponse, org *models.OrgDetailsDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var orgID int64 = 1
		if org != nil {
			orgID = org.ID
		}
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		resp, err := client.Enterprise.GetDataSourceCacheConfig(v.DataSourceUID)
		if err != nil {
			// If API says not found, consider destroyed
			if common.IsNotFoundError(err) {
				return nil
			}
			return fmt.Errorf("error checking cache config for datasource %s in org %d: %s", v.DataSourceUID, orgID, err)
		}
		// When resource is deleted, we disable cache, so treat enabled=false as destroyed
		if resp.Payload.Enabled {
			return fmt.Errorf("cache still enabled for datasource %s in org %d", v.DataSourceUID, orgID)
		}
		return nil
	}
}

func testAccDataSourceCacheConfigWithTTLs(dsName string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "prom" {
  type = "prometheus"
  name = "%[1]s"
  url  = "http://acc-test.invalid/"
}

resource "grafana_data_source_cache_config" "test" {
  datasource_uid   = grafana_data_source.prom.uid
  enabled          = true
  use_default_ttl  = false
  ttl_queries_ms   = 60000
  ttl_resources_ms = 300000
}
`, dsName)
}

func testAccDataSourceCacheConfigWithDefaultTTL(dsName string) string {
	return fmt.Sprintf(`
resource "grafana_data_source" "prom" {
  type = "prometheus"
  name = "%[1]s"
  url  = "http://acc-test.invalid/"
}

resource "grafana_data_source_cache_config" "test" {
  datasource_uid   = grafana_data_source.prom.uid
  enabled          = true
  use_default_ttl  = true
}
`, dsName)
}

func TestAccDataSourceCacheConfig_CreateWithCustomTTLs(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	var cfg models.CacheConfigResponse
	dsName := acctest.RandString(10)
	config := testAccDataSourceCacheConfigWithTTLs(dsName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             cacheConfigDestroyed(&cfg, nil),
		Steps: []resource.TestStep{
			{
				// Create with explicit TTLs
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					cacheConfigCheckExists.exists("grafana_data_source_cache_config.test", &cfg),
					resource.TestMatchResourceAttr("grafana_data_source_cache_config.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "use_default_ttl", "false"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "ttl_queries_ms", "60000"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "ttl_resources_ms", "300000"),
				),
			},
		},
	})
}

func TestAccDataSourceCacheConfig_UpdateToUseDefaultTTL(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	var cfg models.CacheConfigResponse
	dsName := acctest.RandString(10)
	configInitial := testAccDataSourceCacheConfigWithTTLs(dsName)
	configUpdate := testAccDataSourceCacheConfigWithDefaultTTL(dsName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             cacheConfigDestroyed(&cfg, nil),
		Steps: []resource.TestStep{
			{
				// Create with explicit TTLs
				Config: configInitial,
				Check: resource.ComposeTestCheckFunc(
					cacheConfigCheckExists.exists("grafana_data_source_cache_config.test", &cfg),
					resource.TestMatchResourceAttr("grafana_data_source_cache_config.test", "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "use_default_ttl", "false"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "ttl_queries_ms", "60000"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "ttl_resources_ms", "300000"),
				),
			},
			{
				// Update to use default TTLs
				Config: configUpdate,
				Check: resource.ComposeTestCheckFunc(
					cacheConfigCheckExists.exists("grafana_data_source_cache_config.test", &cfg),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "enabled", "true"),
					resource.TestCheckResourceAttr("grafana_data_source_cache_config.test", "use_default_ttl", "true"),
				),
			},
		},
	})
}

func TestAccDataSourceCacheConfig_Import(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=10.0.0")

	var cfg models.CacheConfigResponse
	dsName := acctest.RandString(10)
	config := testAccDataSourceCacheConfigWithTTLs(dsName)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             cacheConfigDestroyed(&cfg, nil),
		Steps: []resource.TestStep{
			{
				// Create for import
				Config: config,
				Check:  cacheConfigCheckExists.exists("grafana_data_source_cache_config.test", &cfg),
			},
			{
				ResourceName:            "grafana_data_source_cache_config.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{}, // Nothing sensitive here
			},
		},
	})
}
