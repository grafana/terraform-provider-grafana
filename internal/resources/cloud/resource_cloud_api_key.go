package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	cloudAPIKeyRoles = []string{"Viewer", "Editor", "Admin", "MetricsPublisher", "PluginPublisher"}
	//nolint:staticcheck
	resourceAPIKeyID = common.NewResourceIDWithLegacySeparator("-",
		common.StringIDField("orgSlug"),
		common.StringIDField("apiKeyName"),
	)
)

func resourceAPIKey() *common.Resource {
	schema := &schema.Resource{
		Description: `This resource is deprecated and will be removed in a future release. Please use grafana_cloud_access_policy instead.

Manages a single API key on the Grafana Cloud portal (on the organization level)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#api-keys)

Required access policy scopes:

* api-keys:read
* api-keys:write
* api-keys:delete
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceAPIKeyCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceAPIKeyRead),
		DeleteContext: withClient[schema.DeleteContextFunc](resourceAPIKeyDelete),
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

	return common.NewLegacySDKResource(
		"grafana_cloud_api_key",
		resourceAPIKeyID,
		schema,
	).WithLister(cloudListerFunction(listAPIKeys))
}

func listAPIKeys(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	org := data.OrgSlug()
	req := client.OrgsAPI.GetApiKeys(ctx, org)
	resp, _, err := req.Execute()
	if err != nil {
		return nil, err
	}

	var keys []string
	for _, key := range resp.Items {
		keys = append(keys, resourceAPIKeyID.Make(key.OrgSlug, key.Name))
	}

	return keys, nil
}

func resourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, c *gcom.APIClient) diag.Diagnostics {
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
	d.SetId(resourceAPIKeyID.Make(org, resp.Name))

	return resourceAPIKeyRead(ctx, d, c)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, c *gcom.APIClient) diag.Diagnostics {
	org, name, err := resourceAPIKeySplitID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	resp, _, err := c.OrgsAPI.GetApiKey(ctx, name, org).Execute()
	if err != nil {
		return apiError(err)
	}

	d.Set("name", resp.Name)
	d.Set("role", resp.Role)
	d.Set("cloud_org_slug", org)
	d.SetId(resourceAPIKeyID.Make(org, resp.Name))

	return nil
}

func resourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, c *gcom.APIClient) diag.Diagnostics {
	org, name, err := resourceAPIKeySplitID(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = c.OrgsAPI.DelApiKey(ctx, name, org).XRequestId(ClientRequestID()).Execute()
	d.SetId("")
	return apiError(err)
}

func resourceAPIKeySplitID(id string) (string, string, error) {
	var org, name string
	if strings.Contains(id, common.ResourceIDSeparator) {
		split, err := resourceAPIKeyID.Split(id)
		if err != nil {
			return "", "", err
		}
		org, name = split[0].(string), split[1].(string)
	} else {
		splitID := strings.SplitN(id, "-", 2)
		org, name = splitID[0], splitID[1]
	}
	return org, name, nil
}
