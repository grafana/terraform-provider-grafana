package cloud

import (
	"context"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DataSourceOrganization() *schema.Resource {
	return &schema.Resource{
		ReadContext: DataSourceOrganizationRead,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"slug": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"created_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"updated_at": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func DataSourceOrganizationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	id := d.Get("id").(string)
	if id == "" {
		id = d.Get("slug").(string)
	}
	org, _, err := client.OrgsAPI.GetOrg(ctx, id).Execute()
	if err != nil {
		return apiError(err)
	}

	id = strconv.FormatInt(int64(*org.Id), 10)
	d.SetId(id)
	d.Set("id", id)
	d.Set("name", org.Name)
	d.Set("slug", org.Slug)
	d.Set("url", org.Url)
	d.Set("created_at", org.CreatedAt)
	d.Set("updated_at", org.UpdatedAt.Get())

	return nil
}
