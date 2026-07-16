package cloud

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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

The publisher token (` + "`publisher_token`" + `) is a stack-scoped access policy token with the following scopes, used by Grafana Cloud k6 to publish test metrics to the stack and process thresholds:

* metrics:read
* metrics:write
* rules:read
* rules:write

It is required when creating new installations and can be updated in place afterwards.
Grafana Cloud also manages and rotates this token internally, so the value tracked in
Terraform may be superseded outside of Terraform over time.
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceK6InstallationCreate),
		ReadContext:   withClient[schema.ReadContextFunc](resourceK6InstallationRead),
		UpdateContext: withClient[schema.UpdateContextFunc](resourceK6InstallationUpdate),
		DeleteContext: resourceK6InstallationDelete,

		CustomizeDiff: func(ctx context.Context, d *schema.ResourceDiff, meta any) error {
			// Skip destroy plans: a resource present in config but absent from
			// state would otherwise fail validation and block terraform destroy.
			if d.GetRawPlan().IsNull() {
				return nil
			}
			// The publisher token is required for new installations only: existing
			// installations (d.Id() != "") predate the attribute and keep working,
			// but without it new stacks are provisioned unable to publish test metrics.
			if d.Id() != "" {
				return nil
			}
			// Values unknown at plan time (e.g. a token created in the same apply)
			// are validated at apply time by resourceK6InstallationCreate instead.
			if !d.NewValueKnown("publisher_token") {
				return nil
			}
			if v, ok := d.Get("publisher_token").(string); !ok || v == "" {
				return fmt.Errorf("publisher_token is required when creating a new k6 installation: create a stack-scoped access policy token with metrics:read, metrics:write, rules:read and rules:write scopes")
			}
			return nil
		},

		Schema: map[string]*schema.Schema{
			"cloud_access_policy_token": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Optional:    true,
				Deprecated:  "This attribute is no longer used by the k6 Cloud API and will be removed in the next major release. It can be safely removed from your configuration.",
				Description: "Deprecated: The [Grafana Cloud access policy](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/) token. It is no longer used to install the k6 App and can be safely removed.",
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
				Description: "The [service account](https://grafana.com/docs/grafana/latest/administration/service-accounts/) token. Updates are propagated to the existing installation.",
			},
			"grafana_user": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The user to use for the installation.",
			},
			"publisher_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A [Grafana Cloud access policy](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/) token with `metrics:read`, `metrics:write`, `rules:read` and `rules:write` scopes on the stack, used by Grafana Cloud k6 to publish test metrics to the stack and process thresholds. Required when creating new installations; updates are propagated to the existing installation. Grafana Cloud also manages this token internally, so it may be rotated outside of Terraform, and removing this attribute does not clear it from the installation.",
			},
			"k6_api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The Grafana Cloud k6 API url.",
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
	k6ApiURL := getk6ApiURL(d)

	url := fmt.Sprintf("%s/v3/account/grafana-app/start", k6ApiURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	publisherToken, ok := d.Get("publisher_token").(string)
	if !ok || len(publisherToken) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid publisher_token: create a stack-scoped access policy token with metrics:read, metrics:write, rules:read and rules:write scopes")
	}

	if diags := setK6InstallationHeaders(d, cloudClient, req); diags.HasError() {
		return diags
	}

	resp, err := cloudClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return diag.Errorf("failed to install the k6 App: %s returned %s: %s", url, resp.Status, string(body))
	}

	var installationRes struct {
		V3GrafanaToken string `json:"v3_grafana_token"`
		OrganizationID string `json:"organization_id"`
	}

	err = json.NewDecoder(resp.Body).Decode(&installationRes)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(installationRes.OrganizationID)

	if err := d.Set("k6_api_url", k6ApiURL); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("k6_access_token", installationRes.V3GrafanaToken); err != nil {
		return diag.FromErr(err)
	}

	if err := d.Set("k6_organization", installationRes.OrganizationID); err != nil {
		return diag.FromErr(err)
	}

	// The /start endpoint is a no-op for stacks that already have the k6 App
	// installed and ignores the tokens sent to it. /initialized stores them in
	// that case, mirroring the k6 App plugin's own bootstrap sequence.
	if diags := k6InstallationSyncTokens(ctx, d, cloudClient); diags.HasError() {
		return diags
	}

	return resourceK6InstallationRead(ctx, d, cloudClient)
}

func resourceK6InstallationUpdate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	if d.HasChanges("publisher_token", "grafana_sa_token") {
		if diags := k6InstallationSyncTokens(ctx, d, cloudClient); diags.HasError() {
			return diags
		}
	}
	return resourceK6InstallationRead(ctx, d, cloudClient)
}

// k6InstallationSyncTokens calls the /initialized endpoint, which updates the
// service account and publisher tokens stored by the k6 API when they differ
// from the ones sent. This is the same call the k6 App plugin makes on load.
func k6InstallationSyncTokens(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	url := fmt.Sprintf("%s/v3/account/grafana-app/initialized", getk6ApiURL(d))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return diag.FromErr(err)
	}

	if diags := setK6InstallationHeaders(d, cloudClient, req); diags.HasError() {
		return diags
	}

	resp, err := cloudClient.GetConfig().HTTPClient.Do(req)
	if err != nil {
		return diag.FromErr(err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return diag.Errorf("failed to sync the k6 installation tokens: %s returned %s: %s", url, resp.Status, string(body))
	}

	return nil
}

// Management of the installation is a one-off operation. The state cannot be updated through a read operation.
// This read function will only invalidate the state (forcing recreation) if the installation has been deleted.
func resourceK6InstallationRead(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	var stackID int32
	if intStackID, err := strconv.Atoi(d.Get("stack_id").(string)); err != nil {
		return diag.Errorf("could not convert stack_id to integer: %s", err.Error())
	} else if stackID, err = common.ToInt32(intStackID); err != nil {
		return diag.Errorf("could not convert stack_id to int32: %s", err.Error())
	}

	k6ApiURL := getk6ApiURL(d)

	k6Cfg := k6.NewConfiguration()
	k6Cfg.Servers = []k6.ServerConfiguration{
		{URL: k6ApiURL},
	}
	k6Cfg.UserAgent = cloudClient.GetConfig().UserAgent
	k6Cfg.HTTPClient = cloudClient.GetConfig().HTTPClient

	tempClient := k6.NewAPIClient(k6Cfg)

	ctx = context.WithValue(ctx, k6.ContextAccessToken, d.Get("k6_access_token").(string))
	if _, _, err := tempClient.ProjectsAPI.ProjectsList(ctx).XStackId(stackID).Execute(); err != nil {
		return common.WarnMissing("k6 installation", d)
	}

	return nil
}

func resourceK6InstallationDelete(_ context.Context, _ *schema.ResourceData, _ any) diag.Diagnostics {
	// To be implemented, not supported yet
	return nil
}

// setK6InstallationHeaders validates the installation attributes and sets the headers
// for the /start k6 API call.
func setK6InstallationHeaders(d *schema.ResourceData, cloudClient *gcom.APIClient, req *http.Request) diag.Diagnostics {
	stackID, ok := d.Get("stack_id").(string)
	if !ok || len(stackID) == 0 {
		return diag.Errorf("the grafana_k6_installation must have a valid stack_id")
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
	// Deprecated: the k6 Cloud API no longer uses this header. Keep sending it
	// when set until the attribute is removed in the next major release.
	if cloudAccessPolicyToken, ok := d.Get("cloud_access_policy_token").(string); ok && len(cloudAccessPolicyToken) > 0 {
		req.Header.Set("X-Grafana-Key", cloudAccessPolicyToken)
	}
	req.Header.Set("X-Grafana-Service-Token", grafanaServiceAccountToken)
	req.Header.Set("X-Grafana-User", grafanaUser)
	req.Header.Set("User-Agent", cloudClient.GetConfig().UserAgent)

	if publisherToken, ok := d.Get("publisher_token").(string); ok && len(publisherToken) > 0 {
		req.Header.Set("X-Publisher-Token", publisherToken)
	}

	return nil
}

func getk6ApiURL(d *schema.ResourceData) string {
	k6APIURL, ok := d.Get("k6_api_url").(string)
	if !ok || len(k6APIURL) == 0 {
		k6APIURL = "https://api.k6.io"
	}
	return k6APIURL
}
