package cloud

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceStackServiceAccount() *schema.Resource {
	return &schema.Resource{

		Description: `
**Note:** This resource is available only with Grafana 9.1+.

Manages service accounts of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)`,

		CreateContext: createStackServiceAccount,
		ReadContext:   readStackServiceAccount,
		UpdateContext: updateStackServiceAccount,
		DeleteContext: deleteStackServiceAccount,
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
				ForceNew:    true,
				Description: "The name of the service account.",
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
				Description:  "The basic role of the service account in the organization.",
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The disabled status for the service account.",
			},
		},
	}
}

func createStackServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloudClient := meta.(*common.Client).GrafanaCloudAPI
	client, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	req := service_accounts.NewCreateServiceAccountParams().WithBody(&models.CreateServiceAccountForm{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: d.Get("is_disabled").(bool),
	})
	resp, err := client.ServiceAccounts.CreateServiceAccount(req)
	if err != nil {
		return diag.FromErr(err)
	}
	sa := resp.Payload

	d.SetId(strconv.FormatInt(sa.ID, 10))
	return readStackServiceAccountWithClient(client, d)
}

func readStackServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloudClient := meta.(*common.Client).GrafanaCloudAPI
	client, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	return readStackServiceAccountWithClient(client, d)
}

func readStackServiceAccountWithClient(client *goapi.GrafanaHTTPAPI, d *schema.ResourceData) diag.Diagnostics {
	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	resp, err := client.ServiceAccounts.RetrieveServiceAccount(id)
	if err != nil {
		return diag.FromErr(err)
	}
	sa := resp.Payload

	err = d.Set("name", sa.Name)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("role", sa.Role)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("is_disabled", sa.IsDisabled)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func updateStackServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloudClient := meta.(*common.Client).GrafanaCloudAPI
	client, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := service_accounts.NewUpdateServiceAccountParams().
		WithBody(&models.UpdateServiceAccountForm{}).
		WithServiceAccountID(id)
	if d.HasChange("name") {
		updateRequest.Body.Name = d.Get("name").(string)
	}
	if d.HasChange("role") {
		updateRequest.Body.Role = d.Get("role").(string)
	}
	if d.HasChange("is_disabled") {
		updateRequest.Body.IsDisabled = d.Get("is_disabled").(bool)
	}

	if _, err := client.ServiceAccounts.UpdateServiceAccount(updateRequest); err != nil {
		return diag.FromErr(err)
	}

	return readStackServiceAccountWithClient(client, d)
}

func deleteStackServiceAccount(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloudClient := meta.(*common.Client).GrafanaCloudAPI
	client, cleanup, err := CreateTemporaryStackGrafanaClient(ctx, cloudClient, d.Get("stack_slug").(string), "terraform-temp-")
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.ServiceAccounts.DeleteServiceAccount(id)
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
