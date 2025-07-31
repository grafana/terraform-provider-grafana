package cloud

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func resourceK6Installation() *common.Resource {
	schema := &schema.Resource{

		Description: `
Sets up the k6 App on a Grafana Cloud instance and generates a token. 
Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install k6.
This resource cannot be imported but it can be used on an existing k6 App installation without issues.

**Note that this resource must be used on a provider configured with Grafana Cloud credentials.**

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/k6/)

Required access policy scopes:

* stacks:read
* stacks:write
* subscriptions:read
* orgs:read
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceK6InstallationCreate),
		ReadContext:   resourceK6InstallationRead,
		DeleteContext: resourceK6InstallationDelete,

		Schema: map[string]*schema.Schema{
			"cloud_access_policy_token": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				ForceNew:    true,
				Description: "The [Grafana Cloud access policy](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/).",
			},
			"stack_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The identifier of the stack to install k6 on.",
			},
			"grafana_sa_token": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				ForceNew:    true,
				Description: "The [service account](https://grafana.com/docs/grafana/latest/administration/service-accounts/) token.",
			},
			"grafana_user": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The user to use for the installation.",
			},
			"k6_access_token": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Computed:    true,
				Description: "Generated token to access the k6 API.",
			},
			"k6_organization": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The identifier of the k6 organization.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryK6,
		"grafana_k6_installation",
		nil,
		schema,
	)
}

func resourceK6InstallationCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	const url = "https://api.k6.io/v3/account/grafana-app/start"
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	stackID, ok := d.Get("stack_id").(string)
	if !ok || len(stackID) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid stack_id")
	}

	cloudAccessPolicyToken, ok := d.Get("cloud_access_policy_token").(string)
	if !ok || len(cloudAccessPolicyToken) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid cloud_access_policy_token")
	}

	grafanaServiceAccountToken, ok := d.Get("grafana_sa_token").(string)
	if !ok || len(grafanaServiceAccountToken) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid grafana_sa_token")
	}

	grafanaUser, ok := d.Get("grafana_user").(string)
	if !ok || len(grafanaUser) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid grafana_user")
	}

	req.Header.Set("X-Stack-Id", stackID)
	req.Header.Set("X-Grafana-Key", cloudAccessPolicyToken)
	req.Header.Set("X-Grafana-Service-Token", grafanaServiceAccountToken)
	req.Header.Set("X-Grafana-User", grafanaUser)

	resp, err := cloudClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	var installationRes struct {
		V3GrafanaToken string `json:"v3_grafana_token"`
		OrganizationID string `json:"organization_id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&installationRes)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(installationRes.OrganizationID)

	if err := d.Set("k6_access_token", installationRes.V3GrafanaToken); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("k6_organization", installationRes.OrganizationID); err != nil {
		return diag.FromErr(err)
	}

	return resourceK6InstallationRead(ctx, d, nil)
}

// Management of the installation is a one-off operation. The state cannot be updated through a read operation.
// This read function will only invalidate the state (forcing recreation) if the installation has been deleted.
func resourceK6InstallationRead(ctx context.Context, d *schema.ResourceData, _ interface{}) diag.Diagnostics {
	var stackID int32
	if intStackID, err := strconv.Atoi(d.Get("stack_id").(string)); err != nil {
		return diag.Errorf("could not convert stack_id to integer: %s", err.Error())
	} else if stackID, err = common.ToInt32(intStackID); err != nil {
		return diag.Errorf("could not convert stack_id to int32: %s", err.Error())
	}

	tempClient := k6.NewAPIClient(k6.NewConfiguration())
	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.Get("k6_access_token").(string))
	if _, _, err := tempClient.ProjectsAPI.ProjectsList(ctx).XStackId(stackID).Execute(); err != nil {
		return common.WarnMissing("k6 installation", d)
	}

	return nil
}

func resourceK6InstallationDelete(_ context.Context, _ *schema.ResourceData, _ interface{}) diag.Diagnostics {
	// To be implemented, not supported yet
	return nil
}
