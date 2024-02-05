package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var cloudAPIKeyRoles = []string{"Viewer", "Editor", "Admin", "MetricsPublisher", "PluginPublisher"}

func ResourceAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `This resource is deprecated and will be removed in a future release. Please use grafana_cloud_access_policy instead.

Manages a single API key on the Grafana Cloud portal (on the organization level)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#api-keys)
`,
		CreateContext: ResourceAPIKeyCreate,
		ReadContext:   ResourceAPIKeyRead,
		DeleteContext: ResourceAPIKeyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		DeprecationMessage: "This resource is deprecated and will be removed in a future release. Please use `grafana_cloud_access_policy` instead.",

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

	req := gcom.PostApiKeysRequest{
		Name: d.Get("name").(string),
		Role: d.Get("role").(string),
	}
	org := d.Get("cloud_org_slug").(string)

	resp, _, err := c.OrgsAPI.PostApiKeys(ctx, org).
		PostApiKeysRequest(req).
		XRequestId(ClientRequestID()).
		Execute()
	if err != nil {
		return apiError(err)
	}

	d.Set("key", *resp.Token)
	d.SetId(org + "-" + resp.Name)

	return ResourceAPIKeyRead(ctx, d, meta)
}

func ResourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).GrafanaCloudAPI

	splitID := strings.SplitN(d.Id(), "-", 2)
	org, name := splitID[0], splitID[1]

	resp, _, err := c.OrgsAPI.GetApiKey(ctx, name, org).Execute()
	if err != nil {
		return apiError(err)
	}

	d.Set("name", resp.Name)
	d.Set("role", resp.Role)
	d.Set("cloud_org_slug", org)

	return nil
}

func ResourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := meta.(*common.Client).GrafanaCloudAPI

	_, err := c.OrgsAPI.DelApiKey(ctx, d.Get("name").(string), d.Get("cloud_org_slug").(string)).XRequestId(ClientRequestID()).Execute()
	d.SetId("")
	return apiError(err)
}
