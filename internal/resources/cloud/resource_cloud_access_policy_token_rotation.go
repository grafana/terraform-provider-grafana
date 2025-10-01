package cloud

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var (
	resourceAccessPolicyTokenRotationID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("tokenId"),
	)
	emptyValueError = errors.New("empty value for required field")
)

func resourceAccessPolicyTokenRotation() *common.Resource {
	schema := &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-a-token)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete

This is similar to the grafana_cloud_access_policy_token resource, but it represents a token that will be rotated automatically over time.
`,

		CreateContext: withClient[schema.CreateContextFunc](createCloudAccessPolicyTokenRotation),
		UpdateContext: withClient[schema.UpdateContextFunc](updateCloudAccessPolicyTokenRotation),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteCloudAccessPolicyTokenRotation),
		ReadContext:   withClient[schema.ReadContextFunc](readCloudAccessPolicyTokenRotation),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			name, err := computeName(d)
			if err != nil && !errors.Is(err, emptyValueError) {
				return fmt.Errorf("error while calculating the customized diff: %w", err)
			}
			if name != "" {
				if err = d.SetNew("name", name); err != nil {
					return fmt.Errorf("error while calculating the customized diff: %w", err)
				}
			}

			expiresAt, err := computeExpiresAt(d)
			if err != nil && !errors.Is(err, emptyValueError) {
				return fmt.Errorf("error while calculating the customized diff: error computing expires_at: %w", err)
			}
			if expiresAt != nil {
				if err = d.SetNew("expires_at", expiresAt.Format(time.RFC3339)); err != nil {
					return fmt.Errorf("error while calculating the customized diff: setting expires_at: %w", err)
				}
			}
			return nil
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
			"name_prefix": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Prefix for the name of the access policy token. The actual name will be stored in the computed field `name`, which will be in the format '<name_prefix>-<rotation_timestamp>'",
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
			"rotate_after": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "The time after which the token will be rotated.",
				ValidateFunc: validation.IsRFC3339Time,
			},
			"post_rotation_lifetime": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Duration that the token should live after rotation (e.g. '24h', '30m', '1h30m'). `expires_at` will be set to the time of the rotation plus this duration.",
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
			"delete_after_creation": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Whether to delete the old token in Grafana Cloud after being rotated or to leave it to expire at its `expires_at` time.",
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
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the access policy token.",
			},
			"expires_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Expiration date of the access policy token.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_access_policy_token_rotation",
		resourceAccessPolicyTokenRotationID,
		schema,
	)
}

func createCloudAccessPolicyTokenRotation(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	region := d.Get("region").(string)

	expiresAt, err := computeExpiresAt(d)
	if err != nil {
		return diag.FromErr(err)
	}

	name, err := computeName(d)
	if err != nil {
		return diag.FromErr(err)
	}

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

	d.SetId(resourceAccessPolicyTokenRotationID.Make(region, result.Id))
	d.Set("token", result.Token)
	d.Set("name", tokenInput.Name)
	d.Set("expires_at", tokenInput.ExpiresAt.Format(time.RFC3339))

	return readCloudAccessPolicyTokenRotation(ctx, d, client)
}

func updateCloudAccessPolicyTokenRotation(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyTokenRotationID.Split(d.Id())
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

	return readCloudAccessPolicyTokenRotation(ctx, d, client)
}

func readCloudAccessPolicyTokenRotation(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyTokenRotationID.Split(d.Id())
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
	d.Set("display_name", result.DisplayName)
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	if result.ExpiresAt != nil {
		d.Set("expires_at", result.ExpiresAt.Format(time.RFC3339))
	}
	if result.UpdatedAt != nil {
		d.Set("updated_at", result.UpdatedAt.Format(time.RFC3339))
	}
	d.Set("name", result.Name)

	tokenID := resourceAccessPolicyTokenRotationID.Make(region, result.Id)

	d.SetId(tokenID)
	return nil
}

func deleteCloudAccessPolicyTokenRotation(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	// TODO: Possibly combine it with: lifecycle { create_before_destroy = true }
	if !d.Get("delete_after_creation").(bool) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "delete_after_creation is disabled",
				Detail:   "The token will not be deleted and will expire automatically if it has an expiration set. If it does not have an expiration, it will need to be deleted manually.",
			},
		}
	}

	split, err := resourceAccessPolicyTokenRotationID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	_, _, err = client.TokensAPI.DeleteToken(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}

func computeName(d getter) (string, error) {
	rotateAfterString := d.Get("rotate_after").(string)
	if rotateAfterString == "" {
		return "", fmt.Errorf("error parsing 'rotate_after': %w", emptyValueError)
	}
	rotateAfter, err := time.Parse(time.RFC3339, rotateAfterString)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%d", d.Get("name_prefix").(string), rotateAfter.Unix()), nil
}

func computeExpiresAt(d getter) (*time.Time, error) {
	rotateAfterString := d.Get("rotate_after").(string)
	if rotateAfterString == "" {
		return nil, fmt.Errorf("error parsing 'rotate_after': %w", emptyValueError)
	}
	rotateAfter, err := time.Parse(time.RFC3339, rotateAfterString)
	if err != nil {
		return nil, fmt.Errorf("error parsing 'rotate_after' to compute 'expires_at': %w", err)
	}

	durationString := d.Get("post_rotation_lifetime").(string)
	if durationString == "" {
		return nil, fmt.Errorf("error parsing 'post_rotation_lifetime': %w", emptyValueError)
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return nil, fmt.Errorf("error parsing 'post_rotation_lifetime' to compute 'expires_at': %w", err)
	}
	expiresAt := rotateAfter.Add(duration)
	return &expiresAt, nil
}
