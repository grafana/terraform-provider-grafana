package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceCloudRegion_Basic(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_cloud_region/resource.tf"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.grafana_cloud_region.us", "id", "1"),
					resource.TestCheckResourceAttr("data.grafana_cloud_region.us", "slug", "us"),
					resource.TestCheckResourceAttr("data.grafana_cloud_region.us", "name", "United States"),
					resource.TestCheckResourceAttr("data.grafana_cloud_region.us", "description", "United States"),
					resource.TestCheckResourceAttr("data.grafana_cloud_region.us", "visibility", "public"),

					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "stack_state_service_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "synthetic_monitoring_api_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "integrations_api_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hosted_exporters_api_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "machine_learning_api_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "incidents_api_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hg_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hg_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hg_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hg_cluster_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_prometheus_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_prometheus_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_prometheus_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_prometheus_cluster_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_graphite_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_graphite_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_graphite_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hm_graphite_cluster_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hl_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hl_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hl_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "hl_cluster_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "am_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "am_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "am_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "am_cluster_url"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "ht_cluster_id"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "ht_cluster_slug"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "ht_cluster_name"),
					resource.TestCheckResourceAttrSet("data.grafana_cloud_region.us", "ht_cluster_url"),
				),
			},
		},
	})
}
