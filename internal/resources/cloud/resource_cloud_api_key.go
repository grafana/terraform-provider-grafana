package cloud

import (
	"context"
	"fmt"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var cloudAPIKeyRoles = []string{"Viewer", "Editor", "Admin", "MetricsPublisher", "PluginPublisher"}

func ResourceAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `Manages a single API key on the Grafana Cloud portal (on the organization level)
* [API documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#api-keys)
`,
		CreateContext: ResourceAPIKeyCreate,
		ReadContext:   ResourceAPIKeyRead,
		DeleteContext: ResourceAPIKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		DeprecationMessage: "Use `grafana_cloud_stack_service_account` together with `grafana_cloud_stack_service_account_token` resources instead see https://grafana.com/docs/grafana/next/administration/api-keys/#migrate-api-keys-to-grafana-service-accounts-using-terraform",

		Schema: map[string]*schema.Schema{
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
				Description:  fmt.Sprintf("Role of the API key. Should be one of %s. See https://grafana.com/docs/grafana-cloud/api/#create-api-key for details.", cloudAPIKeyRoles),
				ValidateFunc: validation.StringInSlice(cloudAPIKeyRoles, false),
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

func ResourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).GrafanaCloudAPI

	req := &gapi.CreateCloudAPIKeyInput{
		Name: d.Get("name").(string),
		Role: d.Get("role").(string),
	}
	org := d.Get("cloud_org_slug").(string)

	resp, err := c.CreateCloudAPIKey(org, req)
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("key", resp.Token)
	d.SetId(org + "-" + resp.Name)

	return ResourceAPIKeyRead(ctx, d, meta)
}

func ResourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).GrafanaCloudAPI

	splitID := strings.SplitN(d.Id(), "-", 2)
	org, name := splitID[0], splitID[1]

	resp, err := c.ListCloudAPIKeys(org)
	if err != nil {
		return diag.FromErr(err)
	}

	for _, apiKey := range resp.Items {
		if apiKey.Name == name {
			d.Set("name", apiKey.Name)
			d.Set("role", apiKey.Role)
			break
		}
	}
	d.Set("cloud_org_slug", org)

	return nil
}

func ResourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).GrafanaCloudAPI

	if err := c.DeleteCloudAPIKey(d.Get("cloud_org_slug").(string), d.Get("name").(string)); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}
