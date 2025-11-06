package cloud

import (
	"context"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var (
	resourceAccessPolicyTokenID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("tokenId"),
	)
)

func resourceAccessPolicyToken() *common.Resource {
	schema := &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-a-token)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete
`,

		CreateContext: withClient[schema.CreateContextFunc](createCloudAccessPolicyToken),
		UpdateContext: withClient[schema.UpdateContextFunc](updateCloudAccessPolicyToken),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteCloudAccessPolicyToken),
		ReadContext:   withClient[schema.ReadContextFunc](readCloudAccessPolicyToken),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: tokenResourceWithCustomSchema(map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the access policy token.",
			},
			"expires_at": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "Expiration date of the access policy token. Does not expire by default.",
				ValidateFunc: validation.IsRFC3339Time,
			},
		}),
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_access_policy_token",
		resourceAccessPolicyTokenID,
		schema,
	)
}

func createCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	var expiresAt *time.Time
	if v, ok := d.GetOk("expires_at"); ok {
		t, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		expiresAt = &t
	}

	return createTokenHelper(ctx, d, client, resourceAccessPolicyTokenID, d.Get("name").(string), expiresAt)
}

func createTokenHelper(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient, tokenResourceID *common.ResourceID, name string, expiresAt *time.Time) diag.Diagnostics {
	region := d.Get("region").(string)

	tokenInput := gcom.PostTokensRequest{
		AccessPolicyId: d.Get("access_policy_id").(string),
		Name:           name,
		DisplayName:    common.Ref(d.Get("display_name").(string)),
		ExpiresAt:      expiresAt,
	}

	req := client.TokensAPI.PostTokens(ctx).Region(region).XRequestId(ClientRequestID()).PostTokensRequest(tokenInput)
	result, _, err := req.Execute()
	if err != nil {
		return apiError(err)
	}

	d.SetId(tokenResourceID.Make(region, result.Id))
	d.Set("token", result.Token)

	return readCloudAccessPolicyToken(ctx, d, client)
}

func updateCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return updateTokenHelper(ctx, d, client, resourceAccessPolicyTokenID)
}

// updateTokenHelper is a helper function meant to update token-related resources, like tokens or token rotations
func updateTokenHelper(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient, tokenResourceID *common.ResourceID) diag.Diagnostics {
	// Avoid sending update requests to the API if the fields that are being updated do not need to be updated in
	// Grafana Cloud, only in the Terraform state.
	if !d.HasChanges("display_name") {
		return nil
	}

	split, err := tokenResourceID.Split(d.Id())
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

	return readCloudAccessPolicyToken(ctx, d, client)
}

func readCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return readTokenHelper(ctx, d, client, resourceAccessPolicyTokenID)
}

// readTokenHelper is a helper function meant to read token-related resources, like tokens or token rotations
func readTokenHelper(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient, tokenResourceID *common.ResourceID) diag.Diagnostics {
	split, err := tokenResourceID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	result, _, err := client.TokensAPI.GetToken(ctx, id.(string)).Region(region.(string)).Execute()
	if err, shouldReturn := common.CheckReadError("policy token", d, err); shouldReturn {
		return err
	}

	d.Set("access_policy_id", result.AccessPolicyId)
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
	d.SetId(tokenResourceID.Make(region, result.Id))

	return nil
}

func deleteCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return deleteTokenHelper(ctx, d, client, resourceAccessPolicyTokenID)
}

// deleteTokenHelper is a helper function meant to delete token-related resources, like tokens or token rotations
func deleteTokenHelper(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient, tokenResourceID *common.ResourceID) diag.Diagnostics {
	split, err := tokenResourceID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	_, _, err = client.TokensAPI.DeleteToken(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}

// tokenResourceWithCustomSchema returns a map that has the fields common to all token-related resources, like tokens
// and token rotations, plus the specified custom fields.
func tokenResourceWithCustomSchema(customFields map[string]*schema.Schema) map[string]*schema.Schema {
	// preset shared common fields
	fields := map[string]*schema.Schema{
		"access_policy_id": {
			Type:        schema.TypeString,
			Required:    true,
			ForceNew:    true,
			Description: "ID of the access policy for which to create a token.",
		},
		"region": {
			Type:         schema.TypeString,
			Required:     true,
			ForceNew:     true,
			Description:  "Region of the access policy. Should be set to the same region as the access policy. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.",
			ValidateFunc: validation.StringIsNotEmpty,
		},
		"display_name": {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Display name of the access policy token. Defaults to the name.",
			DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
				if new == "" && old == d.Get("name").(string) {
					return true
				}
				return false
			},
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
			Description: "Creation date of the access policy token.",
		},
		"updated_at": {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "Last update date of the access policy token.",
		},
	}
	for k, v := range customFields {
		fields[k] = v
	}
	return fields
}
