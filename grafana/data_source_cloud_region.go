package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceCloudRegion() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for Grafana Cloud Region",
		ReadContext: datasourceCloudRegionRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"description": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"visibility": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Service URLs
			"stack_state_service_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"synthetic_monitoring_api_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"integrations_api_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosted_exporters_api_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_learning_api_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"incidents_api_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Hosted Grafana
			"hg_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hg_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hg_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hg_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Hosted Metrics: Prometheus
			"hm_prometheus_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hm_prometheus_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hm_prometheus_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hm_prometheus_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Hosted Metrics: Graphite
			"hm_graphite_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hm_graphite_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hm_graphite_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hm_graphite_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Hosted Logs
			"hl_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hl_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hl_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hl_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Alertmanager
			"am_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"am_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"am_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"am_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Hosted Traces
			"ht_cluster_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"ht_cluster_slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ht_cluster_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"ht_cluster_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func datasourceCloudRegionRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	slug := d.Get("slug").(string)
	region, err := client.GetCloudRegionBySlug(slug)
	if err != nil {
		return diag.FromErr(err)
	}
	d.Set("id", region.ID)
	d.Set("slug", region.Slug)
	d.Set("name", region.Name)
	d.Set("description", region.Description)
	d.Set("visibility", region.Visibility)

	d.Set("stack_state_service_url", region.StackStateServiceURL)
	d.Set("synthetic_monitoring_api_url", region.SyntheticMonitoringAPIURL)
	d.Set("integrations_api_url", region.IntegrationsAPIURL)
	d.Set("hosted_exporters_api_url", region.HostedExportersAPIURL)
	d.Set("machine_learning_api_url", region.MachineLearningAPIURL)
	d.Set("incidents_api_url", region.IncidentsAPIURL)

	// Hosted Grafana
	d.Set("hg_cluster_id", region.HGClusterID)
	d.Set("hg_cluster_slug", region.HGClusterSlug)
	d.Set("hg_cluster_name", region.HGClusterName)
	d.Set("hg_cluster_url", region.HGClusterURL)

	// Hosted Metrics: Prometheus
	d.Set("hm_prometheus_cluster_id", region.HMPromClusterID)
	d.Set("hm_prometheus_cluster_slug", region.HMPromClusterSlug)
	d.Set("hm_prometheus_cluster_name", region.HMPromClusterName)
	d.Set("hm_prometheus_cluster_url", region.HMPromClusterURL)

	// Hosted Metrics: Graphite
	d.Set("hm_graphite_cluster_id", region.HMGraphiteClusterID)
	d.Set("hm_graphite_cluster_slug", region.HMGraphiteClusterSlug)
	d.Set("hm_graphite_cluster_name", region.HMGraphiteClusterName)
	d.Set("hm_graphite_cluster_url", region.HMGraphiteClusterURL)

	// Hosted Logs
	d.Set("hl_cluster_id", region.HLClusterID)
	d.Set("hl_cluster_slug", region.HLClusterSlug)
	d.Set("hl_cluster_name", region.HLClusterName)
	d.Set("hl_cluster_url", region.HLClusterURL)

	// Alertmanager
	d.Set("am_cluster_id", region.AMClusterID)
	d.Set("am_cluster_slug", region.AMClusterSlug)
	d.Set("am_cluster_name", region.AMClusterName)
	d.Set("am_cluster_url", region.AMClusterURL)

	// Hosted Traces
	d.Set("ht_cluster_id", region.HTClusterID)
	d.Set("ht_cluster_slug", region.HTClusterSlug)
	d.Set("ht_cluster_name", region.HTClusterName)
	d.Set("ht_cluster_url", region.HTClusterURL)

	return nil
}
