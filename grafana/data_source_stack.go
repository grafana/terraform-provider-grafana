package grafana

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceStack() *schema.Resource {
	return &schema.Resource{
		Description: "Data source for Grafana Stack",
		ReadContext: datasourceStackRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: `
Subdomain that the Grafana instance will be available at (i.e. setting slug to “<stack_slug>” will make the instance
available at “https://<stack_slug>.grafana.net".`,
			},
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. “<stack_slug>.grafana.net”).",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Description of stack.",
			},
			"region_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The region this stack is deployed to.",
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Custom URL for the Grafana instance. Must have a CNAME setup to point to `.grafana.net` before creating the stack",
			},
			"org_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the stack.",
			},
			"prometheus_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Promehteus user ID. Used for e.g. remote_write.",
			},
			"prometheus_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prometheus_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prometheus_remote_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana",
			},
			"prometheus_remote_write_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to send prometheus metrics to Grafana cloud",
			},
			"prometheus_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},
			"alertmanager_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "User ID of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base URL of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Alertmanager instance configured for this stack.",
			},
		},
	}
}

func datasourceStackRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	var diags diag.Diagnostics

	slug := d.Get("slug").(string)

	stack, err := client.StackBySlug(slug)
	if err != nil {
		return diag.FromErr(err)
	}

	FlattenStack(d, stack)

	return diags
}
