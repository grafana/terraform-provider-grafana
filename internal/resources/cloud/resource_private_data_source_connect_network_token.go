package cloud

import (
	"context"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	pdcNetworkTokenID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("tokenId"),
	)
)

func resourcePDCNetworkToken() *common.Resource {
	schema := &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-a-token)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete
`,

		CreateContext: withClient[schema.CreateContextFunc](createPDCNetworkToken),
		UpdateContext: withClient[schema.UpdateContextFunc](updatePDCNetworkToken),
		DeleteContext: withClient[schema.DeleteContextFunc](deletePDCNetworkToken),
		ReadContext:   withClient[schema.ReadContextFunc](readPDCNetworkToken),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"pdc_network_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the private data source network for which to create a token.",
			},
			"region": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Region of the private data source network. Should be set to the same region as the private data source network. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the private data source network token.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name of the private data source network token. Defaults to the name.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" && old == d.Get("name").(string) {
						return true
					}
					return false
				},
			},
			"expires_at": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "Expiration date of the private data source network token. Does not expire by default.",
				ValidateFunc: validation.IsRFC3339Time,
			},

			// Computed
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation date of the private data source network token.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update date of the private data source network token.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_private_data_source_connect_network_token",
		pdcNetworkTokenID,
		schema,
	)
}

func createPDCNetworkToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	region := d.Get("region").(string)

	tokenInput := gcom.PostTokensRequest{
		AccessPolicyId: d.Get("pdc_network_id").(string),
		Name:           d.Get("name").(string),
		DisplayName:    common.Ref(d.Get("display_name").(string)),
	}

	if v, ok := d.GetOk("expires_at"); ok {
		expiresAt, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		tokenInput.ExpiresAt = &expiresAt
	}

	req := client.TokensAPI.PostTokens(ctx).Region(region).XRequestId(ClientRequestID()).PostTokensRequest(tokenInput)
	result, _, err := req.Execute()
	if err != nil {
		return apiError(err)
	}

	d.SetId(pdcNetworkTokenID.Make(region, result.Id))
	d.Set("token", result.Token)

	return readPDCNetworkToken(ctx, d, client)
}

func updatePDCNetworkToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := pdcNetworkTokenID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	req := client.TokensAPI.PostToken(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).PostTokenRequest(gcom.PostTokenRequest{
		DisplayName: &displayName,
	})
	if _, _, err := req.Execute(); err != nil {
		return apiError(err)
	}

	return readPDCNetworkToken(ctx, d, client)
}

func readPDCNetworkToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := pdcNetworkTokenID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	result, _, err := client.TokensAPI.GetToken(ctx, id.(string)).Region(region.(string)).Execute()
	if err, shouldReturn := common.CheckReadError("policy token", d, err); shouldReturn {
		return err
	}

	d.Set("pdc_network_id", result.AccessPolicyId)
	d.Set("region", region)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	if result.ExpiresAt != nil {
		d.Set("expires_at", result.ExpiresAt.Format(time.RFC3339))
	}
	if result.UpdatedAt != nil {
		d.Set("updated_at", result.UpdatedAt.Format(time.RFC3339))
	}
	d.SetId(pdcNetworkTokenID.Make(region, result.Id))

	return nil
}

func deletePDCNetworkToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := pdcNetworkTokenID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	_, _, err = client.TokensAPI.DeleteToken(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}
