package cloud

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceStackServiceAccountToken() *common.Resource {
	schema := &schema.Resource{
		Description: `
Manages service account tokens of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account token for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)

Required access policy scopes:

* stack-service-accounts:write
`,

		CreateContext: withClient[schema.CreateContextFunc](stackServiceAccountTokenCreate),
		ReadContext:   withClient[schema.ReadContextFunc](stackServiceAccountTokenRead),
		DeleteContext: withClient[schema.DeleteContextFunc](stackServiceAccountTokenDelete),

		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"service_account_id": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					// The service account ID is now possibly a composite ID that includes the stack slug
					oldID, _ := getStackServiceAccountID(old)
					newID, _ := getStackServiceAccountID(new)
					return oldID == newID && oldID != 0 && newID != 0
				},
			},
			"seconds_to_live": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"expiration": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"has_expired": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_stack_service_account_token",
		nil,
		schema,
	)
}

func stackServiceAccountTokenCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if err := waitForStackReadinessFromSlug(ctx, 5*time.Minute, d.Get("stack_slug").(string), cloudClient); err != nil {
		return err
	}

	stackSlug := d.Get("stack_slug").(string)
	serviceAccountID, err := getStackServiceAccountID(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	req := gcom.PostInstanceServiceAccountTokensRequest{
		Name:          d.Get("name").(string),
		SecondsToLive: common.Ref(int32(d.Get("seconds_to_live").(int))), //nolint:gosec
	}

	resp, _, err := cloudClient.InstancesAPI.PostInstanceServiceAccountTokens(ctx, stackSlug, strconv.FormatInt(serviceAccountID, 10)).
		PostInstanceServiceAccountTokensRequest(req).
		XRequestId(ClientRequestID()).
		Execute()
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(*resp.Id, 10))
	err = d.Set("key", resp.Key)
	if err != nil {
		return diag.FromErr(err)
	}

	// Fill the true resource's state by performing a read
	return stackServiceAccountTokenRead(ctx, d, cloudClient)
}

func stackServiceAccountTokenRead(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	stackSlug := d.Get("stack_slug").(string)
	serviceAccountID, err := getStackServiceAccountID(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	if err := waitForStackReadinessFromSlug(ctx, 5*time.Minute, stackSlug, cloudClient); err != nil {
		return err
	}

	response, _, err := cloudClient.InstancesAPI.GetInstanceServiceAccountTokens(ctx, stackSlug, strconv.FormatInt(serviceAccountID, 10)).Execute()
	if err != nil {
		return diag.FromErr(err)
	}

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response {
		if id == *key.Id {
			d.SetId(strconv.FormatInt(*key.Id, 10))
			err = d.Set("name", key.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			if key.Expiration != nil && !key.Expiration.IsZero() {
				err = d.Set("expiration", key.Expiration.String())
				if err != nil {
					return diag.FromErr(err)
				}
			}
			err = d.Set("has_expired", key.HasExpired)

			return diag.FromErr(err)
		}
	}

	return common.WarnMissing("stack service account token", d)
}

func stackServiceAccountTokenDelete(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if err := waitForStackReadinessFromSlug(ctx, 5*time.Minute, d.Get("stack_slug").(string), cloudClient); err != nil {
		return err
	}

	stackSlug := d.Get("stack_slug").(string)
	serviceAccountID, err := getStackServiceAccountID(d.Get("service_account_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = cloudClient.InstancesAPI.DeleteInstanceServiceAccountToken(ctx, stackSlug, strconv.FormatInt(serviceAccountID, 10), d.Id()).
		XRequestId(ClientRequestID()).
		Execute()
	return diag.FromErr(err)
}

func getStackServiceAccountID(id string) (int64, error) {
	split, splitErr := resourceStackServiceAccountID.Split(id)
	if splitErr != nil {
		serviceAccountID, parseErr := strconv.ParseInt(id, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("failed to parse ID (%s) as stackSlug:serviceAccountID: %v and failed to parse as serviceAccountID: %v", id, splitErr, parseErr)
		}
		return serviceAccountID, nil
	}
	return split[1].(int64), nil
}
