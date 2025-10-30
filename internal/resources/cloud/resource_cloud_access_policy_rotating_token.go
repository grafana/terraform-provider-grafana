package cloud

import (
	"context"
	"fmt"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

var (
	resourceAccessPolicyRotatingTokenID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("tokenId"),
	)
)

func resourceAccessPolicyRotatingToken() *common.Resource {
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

		CreateContext: withClient[schema.CreateContextFunc](createCloudAccessPolicyRotatingToken),
		UpdateContext: withClient[schema.UpdateContextFunc](updateCloudAccessPolicyRotatingToken),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteCloudAccessPolicyRotatingToken),
		ReadContext:   withClient[schema.ReadContextFunc](readCloudAccessPolicyRotatingToken),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			expireAfter, err := getDurationFromKey(d, "expire_after")
			if err != nil {
				return fmt.Errorf("error while calculating custom diff: %w", err)
			}

			earlyRotationWindow, err := getDurationFromKey(d, "early_rotation_window")
			if err != nil {
				return fmt.Errorf("error while calculating custom diff: %w", err)
			}

			if earlyRotationWindow > expireAfter {
				return fmt.Errorf("`early_rotation_window` cannot be bigger than `expire_after`")
			}

			// We need to use GetChange() to get `expires_at` from the state because Get() omits computed values
			// that are not being changed.
			expiresAtState, _ := d.GetChange("expires_at")
			if expiresAtState != nil && expiresAtState.(string) != "" {
				expiresAt, err := time.Parse(time.RFC3339, expiresAtState.(string))
				if err != nil {
					return fmt.Errorf("could not parse 'expires_at' while calculating custom diff: %w", err)
				}
				if Now().After(expiresAt.Add(-1 * earlyRotationWindow)) {
					// Token can be rotated. We rotate it by modifying `ready_for_rotation` instead of
					// calling d.ForceNew(field) because the latter only works on fields that are being
					// updated and we do not have any in this case.
					d.SetNew("ready_for_rotation", true)
				}
			}

			return nil
		},

		Schema: tokenResourceWithCustomSchema(map[string]*schema.Schema{
			"name_prefix": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Prefix for the name of the access policy token. The actual name will be stored in the computed field `name`, which will be in the format '<name_prefix>-<expiration_timestamp>'",
				ValidateFunc: validation.StringLenBetween(1, 200),
			},
			"expire_after": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Duration after which the token will expire (e.g. '24h', '30m', '1h30m').",
				ValidateFunc: validatePositiveDuration,
			},
			"early_rotation_window": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Duration of the window before expiring where the token can be rotated (e.g. '24h', '30m', '1h30m').",
				ValidateFunc: validatePositiveDuration,
			},
			"delete_on_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				Description: "Deletes the token in Grafana Cloud when the resource is destroyed in Terraform, " +
					"instead of leaving it to expire at its `expires_at` time. Use it with " +
					"`lifecycle { create_before_destroy = true }` to make sure that the new token is created before " +
					"the old one is deleted.",
			},

			// Computed
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
			"ready_for_rotation": {
				Type:     schema.TypeBool,
				Computed: true,
				ForceNew: true,
				Description: "Signals that the token is either expired " +
					"or within the period to be early rotated.",
			},
		}),
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_access_policy_rotating_token",
		resourceAccessPolicyRotatingTokenID,
		schema,
	)
}

func createCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	expireAfter, err := getDurationFromKey(d, "expire_after")
	if err != nil {
		return diag.FromErr(err)
	}

	expiresAt := Now().Add(expireAfter)
	name := fmt.Sprintf("%s-%d", d.Get("name_prefix").(string), expiresAt.Unix())

	return createTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID, name, &expiresAt)
}

func updateCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return updateTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID)
}

func readCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return readTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID)
}

func deleteCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	if !d.Get("delete_on_destroy").(bool) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Rotating tokens do not get deleted by default.",
				Detail: "The token will not be deleted and will expire automatically at its expiration time. " +
					"If it does not have an expiration, it will need to be deleted manually. To change this behaviour " +
					"enable `delete_on_destroy`.",
			},
		}
	}
	return deleteTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID)
}

func validatePositiveDuration(v interface{}, k string) (warnings []string, errors []error) {
	value := v.(string)
	if value == "" {
		return
	}
	dur, err := time.ParseDuration(value)
	if err != nil {
		errors = append(errors, fmt.Errorf("`%s` must be a valid duration string (e.g. '24h', '30m', '1h30m'): %v", k, err))
	}
	if dur < 0 {
		errors = append(errors, fmt.Errorf("`%s` must be 0 or a positive duration string", k))
	}
	return
}

func getDurationFromKey(d getter, key string) (time.Duration, error) {
	durationStr, ok := d.GetOk(key)
	if !ok {
		return 0, fmt.Errorf("%s is not set", key)
	}
	duration, err := time.ParseDuration(durationStr.(string))
	if err != nil {
		return 0, fmt.Errorf("could not parse duration from '%s': %w", key, err)
	}
	return duration, nil
}
