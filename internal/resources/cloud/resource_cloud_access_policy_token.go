package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
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

		Schema: map[string]*schema.Schema{
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
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the access policy token.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// Do not suppress as this is the first time the name is set.
					if _, ok := d.GetOk("computed_name"); !ok {
						return false
					}

					// If name is being reverted back to its original state and computed_name has been set,
					// we'll want to suppress the diff to avoid forcing a new token to be created.
					return new == d.Get("name").(string) && old == d.Get("computed_name").(string)
				},
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
			"expires_at": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "Expiration date of the access policy token. Does not expire by default. Computed automatically when using rotate_after and post_rotation_lifetime.",
				ValidateFunc: validation.IsRFC3339Time,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// If expires_at has not been set but computed_expires_at has been, suppress the diff.
					return new == "" && old == d.Get("computed_expires_at").(string)
				},
			},

			"rotate_after": {
				Type:          schema.TypeString,
				Optional:      true,
				ForceNew:      true,
				Description:   "The time after which the token will be rotated. If defined, `name` will be suffixed with the timestamp of the rotation.",
				ConflictsWith: []string{"expires_at"},
				ValidateFunc:  validation.IsRFC3339Time,
			},

			"post_rotation_lifetime": {
				Type:         schema.TypeString,
				Optional:     true,
				Description:  "Duration that the token should live after rotation (e.g. '24h', '30m', '1h30m'). If defined, `expires_at` will be set to the time of the rotation plus this duration. Must be used together with `rotate_after`.",
				RequiredWith: []string{"rotate_after"},
				ValidateFunc: func(v interface{}, k string) (warnings []string, errors []error) {
					value := v.(string)
					if value == "" {
						return
					}
					if _, err := time.ParseDuration(value); err != nil {
						errors = append(errors, fmt.Errorf("%s must be a valid duration string (e.g. '24h', '30m', '1h30m'): %v", k, err))
					}
					return
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
			"computed_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Computed name of the access policy token. Only set when `rotate_after` is defined.",
			},
			"computed_expires_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Computed expiration date of the access policy token. Only set when `rotate_after` and `post_rotation_lifetime` are defined.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_access_policy_token",
		resourceAccessPolicyTokenID,
		schema,
	)
}

func createCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	region := d.Get("region").(string)

	tokenInput := gcom.PostTokensRequest{
		AccessPolicyId: d.Get("access_policy_id").(string),
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

	if v, ok := d.GetOk("rotate_after"); ok {
		rotateAfter, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		tokenInput.Name = fmt.Sprintf("%s-%d", d.Get("name").(string), rotateAfter.Unix())
		d.Set("computed_name", tokenInput.Name)

		if postRotationLifetime, ok := d.GetOk("post_rotation_lifetime"); ok {
			duration, err := time.ParseDuration(postRotationLifetime.(string))
			if err != nil {
				return diag.FromErr(err)
			}
			expiresAt := rotateAfter.Add(duration)
			tokenInput.ExpiresAt = &expiresAt
			d.Set("computed_expires_at", tokenInput.ExpiresAt.Format(time.RFC3339))
		}
	}

	req := client.TokensAPI.PostTokens(ctx).Region(region).XRequestId(ClientRequestID()).PostTokensRequest(tokenInput)
	result, _, err := req.Execute()
	if err != nil {
		return apiError(err)
	}

	d.SetId(resourceAccessPolicyTokenID.Make(region, result.Id))
	d.Set("token", result.Token)

	return readCloudAccessPolicyToken(ctx, d, client)
}

func updateCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyTokenID.Split(d.Id())
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
	split, err := resourceAccessPolicyTokenID.Split(d.Id())
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

	tokenID := resourceAccessPolicyTokenID.Make(region, result.Id)

	d.SetId(tokenID)
	return nil
}

func deleteCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	if d.Get("rotate_after").(string) != "" {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Token rotation is enabled",
				Detail:   "The token will not be deleted and will expire automatically if it has an expiration set. If it does not have an expiration, it will need to be deleted manually.",
			},
		}
	}

	split, err := resourceAccessPolicyTokenID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	_, _, err = client.TokensAPI.DeleteToken(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}
