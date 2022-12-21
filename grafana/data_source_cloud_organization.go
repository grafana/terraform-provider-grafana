package grafana

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func DatasourceCloudOrganization() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/administration/manage-organizations/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/org/)
`,
		ReadContext: datasourceCloudOrganizationRead,
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

func datasourceCloudOrganizationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	id := d.Get("id").(string)
	if id == "" {
		id = d.Get("slug").(string)
	}
	org, err := client.GetCloudOrg(id)
	if err != nil {
		return diag.FromErr(err)
	}

	id = strconv.FormatInt(org.ID, 10)
	d.SetId(id)
	d.Set("id", id)
	d.Set("name", org.Name)
	d.Set("slug", org.Slug)
	d.Set("url", org.URL)
	d.Set("created_at", org.CreatedAt)
	d.Set("updated_at", org.UpdatedAt)

	return nil
}
