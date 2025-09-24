package cloud

import (
	"context"
	"fmt"
	"strings"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var smAPIURLsExceptions = map[string]string{
	"au":              "https://synthetic-monitoring-api-au-southeast.grafana.net",
	"eu":              "https://synthetic-monitoring-api-eu-west.grafana.net",
	"prod-gb-south-0": "https://synthetic-monitoring-api-gb-south.grafana.net",
	"us":              "https://synthetic-monitoring-api.grafana.net",
	"us-azure":        "https://synthetic-monitoring-api-us-central2.grafana.net",
}

// createSMClient creates a new SMAPI client with proper client-id and client-version settings
func createSMClient(apiURL, accessToken string) *SMAPI.Client {
	client := SMAPI.NewClient(apiURL, accessToken, nil)
	client.SetCustomClientID("terraform")
	client.SetCustomClientVersion("unknown") // TODO: see if we can get the provider version here
	return client
}

func resourceSyntheticMonitoringInstallation() *common.Resource {
	schema := &schema.Resource{

		Description: `
Sets up Synthetic Monitoring on a Grafana cloud stack and generates a token. 
Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install synthetic monitoring.
This resource cannot be imported but it can be used on an existing Synthetic Monitoring installation without issues.

**Note that this resource must be used on a provider configured with Grafana Cloud credentials.**

* [Official documentation](https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/)
* [API documentation](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#apiv1registerinstall)

Required access policy scopes:

* stacks:read
`,
		CreateContext: withClient[schema.CreateContextFunc](resourceInstallationCreate),
		ReadContext:   resourceInstallationRead,
		DeleteContext: resourceInstallationDelete,

		Schema: map[string]*schema.Schema{
			"metrics_publisher_key": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				ForceNew:    true,
				Description: "The [Grafana Cloud access policy](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/) with the following scopes: `stacks:read`, `metrics:write`, `logs:write`, `traces:write`. This is used to publish metrics and logs to Grafana Cloud stack.",
			},
			"stack_sm_api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The URL of the SM API to install SM on. This depends on the stack region, find the list of API URLs here: https://grafana.com/docs/grafana-cloud/testing/synthetic-monitoring/set-up/set-up-private-probes/#probe-api-server-url. A static mapping exists in the provider but it may not contain all the regions. If it does contain the stack's region, this field is computed automatically and readable.",
			},
			"stack_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The ID or slug of the stack to install SM on.",
			},
			"sm_access_token": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Generated token to access the SM API.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategorySyntheticMonitoring,
		"grafana_synthetic_monitoring_installation",
		nil,
		schema,
	)
}

func resourceInstallationCreate(ctx context.Context, d *schema.ResourceData, cloudClient *gcom.APIClient) diag.Diagnostics {
	req := cloudClient.InstancesAPI.GetInstance(ctx, d.Get("stack_id").(string))
	stack, _, err := req.Execute()
	if err != nil {
		return apiError(err)
	}

	// TODO: Get this URL programatically
	// Perhaps it should be exposed to users through /api/stack-regions?
	apiURL := d.Get("stack_sm_api_url").(string)
	if apiURL == "" {
		apiURL = smAPIURLsExceptions[stack.RegionSlug]
	}
	if apiURL == "" {
		apiURL = fmt.Sprintf("https://synthetic-monitoring-api-%s.grafana.net", strings.TrimPrefix(stack.RegionSlug, "prod-"))
	}

	smClient := createSMClient(apiURL, "")
	stackID, metricsID, logsID := int64(stack.Id), int64(stack.HmInstancePromId), int64(stack.HlInstanceId)
	resp, err := smClient.Install(ctx, stackID, metricsID, logsID, d.Get("metrics_publisher_key").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s;%d", apiURL, stackID))
	d.Set("sm_access_token", resp.AccessToken)
	d.Set("stack_sm_api_url", apiURL)
	return resourceInstallationRead(ctx, d, nil)
}

// Management of the installation is a one-off operation. The state cannot be updated through a read operation.
// This read function will only invalidate the state (forcing recreation) if the installation has been deleted.
func resourceInstallationRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	apiURL := strings.Split(d.Id(), ";")[0]
	tempClient := createSMClient(apiURL, d.Get("sm_access_token").(string))
	if err := tempClient.ValidateToken(ctx); err != nil {
		return common.WarnMissing("synthetic monitoring installation", d)
	}

	return nil
}

func resourceInstallationDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	apiURL := strings.Split(d.Id(), ";")[0]
	tempClient := createSMClient(apiURL, d.Get("sm_access_token").(string))
	err := tempClient.DeleteToken(ctx)
	return diag.FromErr(err)
}
