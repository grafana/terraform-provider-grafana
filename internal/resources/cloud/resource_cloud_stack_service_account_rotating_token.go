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

func resourceStackServiceAccountRotatingToken() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages and rotates service account tokens of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account token for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)

Required access policy scopes:

* stack-service-accounts:write
`,

		// We use the same function for Read and Update because fields are only updated in the Terraform state,
		// not in Grafana, for this resource.
		CreateContext: withClient[schema.CreateContextFunc](stackServiceAccountRotatingTokenCreate),
		ReadContext:   withClient[schema.ReadContextFunc](stackServiceAccountRotatingTokenRead),
		UpdateContext: withClient[schema.UpdateContextFunc](stackServiceAccountRotatingTokenRead),
		DeleteContext: withClient[schema.DeleteContextFunc](stackServiceAccountRotatingTokenDelete),

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			secondsToLive := d.Get("seconds_to_live").(int)
			earlyRotationWindowSec := d.Get("early_rotation_window_seconds").(int)

			if earlyRotationWindowSec > secondsToLive {
				return fmt.Errorf("`early_rotation_window_seconds` cannot be greater than `seconds_to_live`")
			}

			// We need to use GetChange() to get the value from the state because Get() omits computed values that are
			// not being changed.
			hasExpired, _ := d.GetChange("has_expired")
			if hasExpired != nil && hasExpired.(bool) {
				return d.SetNew("ready_for_rotation", true)
			}

			expirationState, _ := d.GetChange("expiration")
			if expirationState != nil && expirationState.(string) != "" {
				// We save the expiration time in Golang's native layout when we read the resource, so we
				// need to use that same layout here. We should ideally switch to a standard format like
				// RFC3339.
				expiration, err := time.Parse("2006-01-02 15:04:05 -0700 MST", expirationState.(string))
				if err != nil {
					return fmt.Errorf("could not parse 'expiration' while calculating custom diff: %w", err)
				}
				if Now().After(expiration.Add(-1 * time.Duration(earlyRotationWindowSec) * time.Second)) {
					return d.SetNew("ready_for_rotation", true)
				}
			}

			return nil
		},

		Schema: stackServiceAccountTokenResourceWithCustomSchema(map[string]*schema.Schema{
			"name_prefix": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				Description: "Prefix for the name of the service account tokens created by this resource. " +
					"The actual name will be stored in the computed field `name`, which will be in the format " +
					"`<name_prefix>-<additional_characters>`.",
			},
			"seconds_to_live": {
				Type:         schema.TypeInt,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "The token expiration in seconds.",
			},
			"early_rotation_window_seconds": {
				Type:         schema.TypeInt,
				Required:     true,
				ValidateFunc: validation.IntAtLeast(0),
				Description:  "Duration of the time window before expiring where the token can be rotated, in seconds.",
			},
			"delete_on_destroy": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
				Description: "Deletes the service account token in Grafana when the resource " +
					"is destroyed in Terraform, instead of leaving it to expire at its `expiration` " +
					"time. Use it with `lifecycle { create_before_destroy = true }` to make sure " +
					"that the new token is created before the old one is deleted.",
			},
			// Computed
			"name": {
				Type:     schema.TypeString,
				Computed: true,
				Description: "The name of the service account token. It will start with `<name_prefix>-` and will have " +
					"characters appended to it to make the name unique.",
			},
			"ready_for_rotation": {
				Type:     schema.TypeBool,
				Computed: true,
				ForceNew: true,
				Description: "Signals that the service account token is expired or " +
					"within the period to be early rotated.",
			},
		}),
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_stack_service_account_rotating_token",
		nil,
		schema,
	)
}

func stackServiceAccountRotatingTokenCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	namePrefix := d.Get("name_prefix").(string)
	ttl := d.Get("seconds_to_live").(int)

	expiration := Now().Add(time.Duration(ttl) * time.Second)
	name := fmt.Sprintf("%s-%d", namePrefix, expiration.Unix())

	errDiag := stackServiceAccountTokenCreateHelper(ctx, d, cloudClient, name)
	if errDiag.HasError() {
		return errDiag
	}

	// Fill the true resource's state by performing a read
	return stackServiceAccountRotatingTokenRead(ctx, d, cloudClient)
}

func stackServiceAccountRotatingTokenRead(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	return stackServiceAccountTokenRead(ctx, d, cloudClient)
}

func stackServiceAccountRotatingTokenDelete(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if !d.Get("delete_on_destroy").(bool) {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  "Rotating tokens do not get deleted by default.",
				Detail: "The Service Account token will not be deleted and will expire automatically at its expiration time. " +
					"If it does not have an expiration, it will need to be deleted manually. To change this behaviour " +
					"enable `delete_on_destroy`.",
			},
		}
	}
	return stackServiceAccountTokenDelete(ctx, d, cloudClient)
}
