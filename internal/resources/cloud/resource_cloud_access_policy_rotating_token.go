package cloud

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
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
	emptyTokenRotationValueError = errors.New("empty value for required field")
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
			customDiffErr := errors.New("error while generating the customized diff")

			name, err := computeRotatingTokenName(d)
			if err != nil && !errors.Is(err, emptyTokenRotationValueError) {
				return fmt.Errorf("%w: %w", customDiffErr, err)
			}
			if name != "" {
				if err = d.SetNew("name", name); err != nil {
					return fmt.Errorf("%w: %w", customDiffErr, err)
				}
			}

			expiresAt, err := computeRotatingTokenExpiresAt(d)
			if err != nil && !errors.Is(err, emptyTokenRotationValueError) {
				return fmt.Errorf("%w: %w", customDiffErr, err)
			}
			if expiresAt != nil {
				if err = d.SetNew("expires_at", expiresAt.Format(time.RFC3339)); err != nil {
					return fmt.Errorf("%w: error setting 'expires_at': %w", customDiffErr, err)
				}
			}
			return nil
		},

		Schema: tokenResourceWithCustomSchema(map[string]*schema.Schema{
			"name_prefix": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Prefix for the name of the access policy token. The actual name will be stored in the computed field `name`, which will be in the format '<name_prefix>-<rotate_after>-<post_rotation_lifetime>'",
				ValidateFunc: validation.StringLenBetween(1, 200),
			},
			"rotate_after": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				Description:  "The time after which the token will be rotated, as a unix timestamp (number of seconds elapsed since epoch time - January 1, 1970 UTC).",
				ValidateFunc: validation.IntAtLeast(0),
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
					postRotationLifetime, err := time.ParseDuration(value)
					if err != nil {
						errors = append(errors, fmt.Errorf("%s must be a valid duration string (e.g. '24h', '30m', '1h30m'): %v", k, err))
					}
					if postRotationLifetime < 0 {
						errors = append(errors, fmt.Errorf("%s must be 0 or a positive duration string", k))
					}
					return
				},
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
				Description: "Expiration date of the access policy token. This is the result of adding `rotate_after` and `post_rotation_lifetime`",
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
	expiresAt, err := computeRotatingTokenExpiresAt(d)
	if err != nil {
		return diag.FromErr(err)
	}

	name, err := computeRotatingTokenName(d)
	if err != nil {
		return diag.FromErr(err)
	}

	return createTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID, name, expiresAt)
}

func updateCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return updateTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID)
}

func readCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	if di := readTokenHelper(ctx, d, client, resourceAccessPolicyRotatingTokenID); di.HasError() {
		return di
	}

	name := d.Get("name").(string)
	attrs, err := rotatingTokenAttributesFromName(name)
	if err != nil {
		return diag.Errorf("error while parsing attributes from rotating token name '%s': %s", name, err)
	}

	err = errors.Join(
		d.Set("post_rotation_lifetime", attrs.postRotationLifetime),
		d.Set("rotate_after", attrs.rotateAfter),
		d.Set("name_prefix", attrs.namePrefix),
	)
	return diag.FromErr(err)
}

func deleteCloudAccessPolicyRotatingToken(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "Rotating tokens do not get deleted.",
			Detail:   "The token will not be deleted and will expire automatically at its expiration time. If it does not have an expiration, it will need to be deleted manually.",
		},
	}
}

func getRotatingTokenPostRotationLifetime(d getter) (*time.Duration, error) {
	durationString := d.Get("post_rotation_lifetime").(string)
	if durationString == "" {
		return nil, emptyTokenRotationValueError
	}
	duration, err := time.ParseDuration(durationString)
	if err != nil {
		return nil, err
	}
	rounded := duration.Round(time.Second)
	return &rounded, nil
}

func getRotatingTokenRotateAfter(d getter) (*time.Time, error) {
	rotateAfterInt, ok := d.Get("rotate_after").(int)
	if !ok || rotateAfterInt == 0 {
		return nil, emptyTokenRotationValueError
	}
	rotateAfter := time.Unix(int64(rotateAfterInt), 0).UTC()
	return &rotateAfter, nil
}

func computeRotatingTokenName(d getter) (string, error) {
	namePrefix := d.Get("name_prefix").(string)
	if namePrefix == "" {
		return "", emptyTokenRotationValueError
	}

	postRotationLifetime := d.Get("post_rotation_lifetime").(string)
	if postRotationLifetime == "" {
		return "", emptyTokenRotationValueError
	}

	rotateAfterInt, ok := d.Get("rotate_after").(int)
	if !ok || rotateAfterInt == 0 {
		return "", emptyTokenRotationValueError
	}

	attrs := rotatingTokenNameAttributes{
		namePrefix:           namePrefix,
		postRotationLifetime: postRotationLifetime,
		rotateAfter:          rotateAfterInt,
	}

	return attrs.computedName(), nil
}

func computeRotatingTokenExpiresAt(d getter) (*time.Time, error) {
	rotateAfter, err := getRotatingTokenRotateAfter(d)
	if err != nil {
		return nil, fmt.Errorf("error parsing 'rotate_after' while computing 'expires_at': %w", err)
	}

	postRotationLifetime, err := getRotatingTokenPostRotationLifetime(d)
	if err != nil {
		return nil, fmt.Errorf("error parsing 'post_rotation_lifetime' while computing 'expires_at': %w", err)
	}

	expiresAt := rotateAfter.Add(*postRotationLifetime)
	return &expiresAt, nil
}

type rotatingTokenNameAttributes struct {
	namePrefix           string
	postRotationLifetime string
	rotateAfter          int
}

func (r rotatingTokenNameAttributes) computedName() string {
	return fmt.Sprintf("%s-%d-%s", r.namePrefix, r.rotateAfter, r.postRotationLifetime)
}

func rotatingTokenAttributesFromName(name string) (*rotatingTokenNameAttributes, error) {
	parts := strings.Split(name, "-")
	if len(parts) < 3 {
		return nil, fmt.Errorf("rotating token name does not follow the expected pattern '<name_prefix>-<rotate_after>-<post_rotation_lifetime>': %s", name)
	}

	// rotateAfter
	rotateAfterStr := parts[len(parts)-2]
	rotateAfter, err := strconv.ParseInt(rotateAfterStr, 10, 64)
	if err != nil || rotateAfter <= 0 {
		return nil, fmt.Errorf("could not infer 'rotate_after' from rotating token name: %w", err)
	}

	// postRotationLifetime
	postRotationLifetime := parts[len(parts)-1]
	if _, err = time.ParseDuration(postRotationLifetime); err != nil {
		return nil, fmt.Errorf("could not infer 'post_rotation_lifetime' from rotating token name: %w", err)
	}

	// namePrefix
	namePrefix := strings.Join(parts[:len(parts)-2], "-")
	if namePrefix == "" {
		return nil, errors.New("could not infer 'name_prefix' from rotating token name")
	}

	return &rotatingTokenNameAttributes{
		namePrefix:           namePrefix,
		postRotationLifetime: postRotationLifetime,
		rotateAfter:          int(rotateAfter),
	}, nil
}
