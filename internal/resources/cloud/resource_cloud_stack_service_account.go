package cloud

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	resourceStackServiceAccountID = common.NewResourceID(
		common.StringIDField("stackSlug"),
		common.IntIDField("serviceAccountID"),
	)
)

func resourceStackServiceAccount() *common.Resource {
	schema := &schema.Resource{

		Description: `
Manages service accounts of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developer-resources/api-reference/http-api/api-legacy/serviceaccount/#service-account-api)

Required access policy scopes:

* stacks:read
* stack-service-accounts:write
`,

		CreateContext: withClient[schema.CreateContextFunc](createStackServiceAccount),
		ReadContext:   withClient[schema.ReadContextFunc](readStackServiceAccount),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteStackServiceAccount),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the service account.",
				ForceNew:    true,
			},
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin", "None"}, false),
				Description:  "The basic role of the service account in the organization.",
				ForceNew:     true, // The grafana API does not support updating the service account
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The disabled status for the service account.",
				ForceNew:    true, // The grafana API does not support updating the service account
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_stack_service_account",
		resourceStackServiceAccountID,
		schema,
	)
}

func createStackServiceAccount(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if err := waitForStackReadinessFromSlug(ctx, 10*time.Minute, d.Get("stack_slug").(string), cloudClient); err != nil {
		return err
	}

	stackSlug := d.Get("stack_slug").(string)
	name := d.Get("name").(string)
	if existing, err := findStackServiceAccountByExactName(ctx, cloudClient, stackSlug, name); err != nil {
		return diag.FromErr(err)
	} else if existing != nil {
		return diag.Errorf("a stack service account named %q already exists in stack %q (id %d)", name, stackSlug, existing.ID)
	}

	req := gcom.PostInstanceServiceAccountsRequest{
		Name:       name,
		Role:       d.Get("role").(string),
		IsDisabled: common.Ref(d.Get("is_disabled").(bool)),
	}

	var resp *gcom.GrafanaServiceAccountDTO
	var sawPrior5xxOrNetwork bool
	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		var httpResp *http.Response
		var callErr error
		resp, httpResp, callErr = cloudClient.InstancesAPI.PostInstanceServiceAccounts(ctx, stackSlug).
			PostInstanceServiceAccountsRequest(req).
			XRequestId(ClientRequestID()).
			Execute()
		if callErr == nil {
			return nil
		}
		if !shouldRetryServiceAccountOperation(httpResp, callErr) {
			return retry.NonRetryableError(callErr)
		}

		if shouldAdoptResource(sawPrior5xxOrNetwork, httpResp) {
			if adopted, adoptErr := findStackServiceAccountByExactName(ctx, cloudClient, stackSlug, name); adoptErr != nil {
				return retry.NonRetryableError(adoptErr)
			} else if adopted != nil {
				var getErr error
				resp, _, getErr = cloudClient.InstancesAPI.GetInstanceServiceAccount(ctx, stackSlug, strconv.FormatInt(adopted.ID, 10)).Execute()
				if getErr != nil {
					code := 0
					if httpResp != nil {
						code = httpResp.StatusCode
					}
					log.Printf("[WARN] PostInstanceServiceAccounts failed for stack %q (HTTP %d); adopt-after-400: read existing service account %q (id %d) failed: %v",
						stackSlug, code, name, adopted.ID, getErr)
					return retry.RetryableError(getErr)
				}
				return nil
			}
		}

		if is5xxOrNetworkError(httpResp, callErr) {
			sawPrior5xxOrNetwork = true
		}

		code := 0
		if httpResp != nil {
			code = httpResp.StatusCode
		}
		log.Printf("[WARN] PostInstanceServiceAccounts failed for stack %q (HTTP %d), retrying: %v", stackSlug, code, callErr)
		time.Sleep(5 * time.Second)
		return retry.RetryableError(callErr)
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resourceStackServiceAccountID.Make(stackSlug, resp.Id))
	return readStackServiceAccount(ctx, d, cloudClient)
}

func readStackServiceAccount(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	var stackSlug string
	var serviceAccountID int64
	split, splitErr := resourceStackServiceAccountID.Split(d.Id())
	if splitErr != nil {
		stackSlug = d.Get("stack_slug").(string)
		var parseErr error
		if serviceAccountID, parseErr = strconv.ParseInt(d.Id(), 10, 64); parseErr != nil {
			return diag.Errorf("failed to parse ID (%s) as stackSlug:serviceAccountID: %v and failed to parse as serviceAccountID: %v", d.Id(), splitErr, parseErr)
		}
	} else {
		stackSlug, serviceAccountID = split[0].(string), split[1].(int64)
	}

	if err := waitForStackReadinessFromSlug(ctx, 10*time.Minute, stackSlug, cloudClient); err != nil {
		return err
	}

	resp, httpResp, err := cloudClient.InstancesAPI.GetInstanceServiceAccount(ctx, stackSlug, strconv.FormatInt(serviceAccountID, 10)).Execute()
	if httpResp != nil && httpResp.StatusCode == 404 {
		return common.WarnMissing("stack service account", d)
	}
	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("stack_slug", stackSlug)
	d.Set("name", resp.Name)
	d.Set("role", resp.Role)
	d.Set("is_disabled", resp.IsDisabled)
	d.SetId(resourceStackServiceAccountID.Make(stackSlug, serviceAccountID))

	return nil
}

func deleteStackServiceAccount(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if err := waitForStackReadinessFromSlug(ctx, 10*time.Minute, d.Get("stack_slug").(string), cloudClient); err != nil {
		return err
	}

	split, err := resourceStackServiceAccountID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	stackSlug, serviceAccountID := split[0].(string), split[1].(int64)

	httpResp, err := cloudClient.InstancesAPI.DeleteInstanceServiceAccount(ctx, stackSlug, strconv.FormatInt(serviceAccountID, 10)).
		XRequestId(ClientRequestID()).
		Execute()
	if httpResp != nil && httpResp.StatusCode == 404 {
		return nil
	}
	return diag.FromErr(err)
}

func CreateTemporaryStackGrafanaClient(ctx context.Context, cloudClient *gcom.APIClient, stackSlug, tempSaPrefix string) (*goapi.GrafanaHTTPAPI, func() error, error) {
	stack, _, err := cloudClient.InstancesAPI.GetInstance(ctx, stackSlug).Execute()
	if err != nil {
		return nil, nil, err
	}

	name := fmt.Sprintf("%s%d", tempSaPrefix, time.Now().UnixNano())

	req := gcom.PostInstanceServiceAccountsRequest{
		Name: name,
		Role: "Admin",
	}

	sa, _, err := cloudClient.InstancesAPI.PostInstanceServiceAccounts(ctx, stackSlug).
		PostInstanceServiceAccountsRequest(req).
		XRequestId(ClientRequestID()).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	tokenRequest := gcom.PostInstanceServiceAccountTokensRequest{
		Name:          name,
		SecondsToLive: common.Ref(int32(60)),
	}
	token, _, err := cloudClient.InstancesAPI.PostInstanceServiceAccountTokens(ctx, stackSlug, fmt.Sprintf("%d", int(*sa.Id))).
		PostInstanceServiceAccountTokensRequest(tokenRequest).
		XRequestId(ClientRequestID()).
		Execute()
	if err != nil {
		return nil, nil, err
	}

	stackURLParsed, err := url.Parse(stack.Url)
	if err != nil {
		return nil, nil, err
	}

	client := goapi.NewHTTPClientWithConfig(nil, &goapi.TransportConfig{
		Host:         stackURLParsed.Host,
		Schemes:      []string{stackURLParsed.Scheme},
		BasePath:     "api",
		APIKey:       *token.Key,
		NumRetries:   5,
		RetryTimeout: 10 * time.Second,
	})

	cleanup := func() error {
		_, err = client.ServiceAccounts.DeleteServiceAccount(*sa.Id)
		return err
	}

	return client, cleanup, nil
}
