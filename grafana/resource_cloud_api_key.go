package grafana

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var cloudApiKeyRoles = []string{"Viewer", "Editor", "Admin", "MetricsPublisher", "PluginPublisher"}

func ResourceCloudApiKey() *schema.Resource {
	return &schema.Resource{
		Description: `Manages a single API key on the Grafana Cloud portal (on the organization level)
* [API documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#api-keys)
`,
		CreateContext: resourceCloudApiKeyCreate,
		ReadContext:   resourceCloudApiKeyRead,
		DeleteContext: resourceCloudApiKeyDelete,
		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the API key.",
			},
			"cloud_org_slug": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The slug of the organization to create the API key in. This is the same slug as the organization name in the URL.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the API key.",
			},
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  fmt.Sprintf("Role of the API key. Might be one of %s. See https://grafana.com/docs/grafana-cloud/api/#create-api-key for details.", cloudApiKeyRoles),
				ValidateFunc: validation.StringInSlice(cloudApiKeyRoles, false),
			},
			"key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The generated API key.",
			},
		},
	}
}

func resourceCloudApiKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).gcloudapi

	req := &gapi.CreateCloudAPIKeyInput{
		Name: d.Get("name").(string),
		Role: d.Get("role").(string),
	}

	resp, err := c.CreateCloudAPIKey(d.Get("cloud_org_slug").(string), req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("key", resp.Token)
	d.SetId(resp.Name)
	d.Set("id", resp.Name)

	return resourceCloudApiKeyRead(ctx, d, meta)
}

func resourceCloudApiKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).gcloudapi

	resp, err := c.ListCloudAPIKeys(d.Get("cloud_org_slug").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	for _, apiKey := range resp.Items {
		if apiKey.Name == d.Id() {
			d.Set("name", apiKey.Name)
			d.Set("role", apiKey.Role)
			break
		}
	}

	return nil
}

func resourceCloudApiKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*client).gcloudapi

	if err := c.DeleteCloudAPIKey(d.Get("cloud_org_slug").(string), d.Id()); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
