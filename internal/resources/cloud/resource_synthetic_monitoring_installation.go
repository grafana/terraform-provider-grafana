package cloud

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	SMAPI "github.com/grafana/synthetic-monitoring-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO: Automate finding the correct API URL based on the stack region
var smAPIURLs = map[string]string{
	"us":                  "https://synthetic-monitoring-api.grafana.net",
	"us-azure":            "https://synthetic-monitoring-api-us-central2.grafana.net",
	"eu":                  "https://synthetic-monitoring-api-eu-west.grafana.net",
	"au":                  "https://synthetic-monitoring-api-au-southeast.grafana.net",
	"prod-ap-southeast-0": "https://synthetic-monitoring-api-ap-southeast-0.grafana.net",
	"prod-gb-south-0":     "https://synthetic-monitoring-api-gb-south.grafana.net",
	"prod-eu-west-2":      "https://synthetic-monitoring-api-eu-west-2.grafana.net",
	"prod-eu-west-3":      "https://synthetic-monitoring-api-eu-west-3.grafana.net",
	"prod-ap-south-0":     "https://synthetic-monitoring-api-ap-south-0.grafana.net",
	"prod-sa-east-0":      "https://synthetic-monitoring-api-sa-east-0.grafana.net",
	"prod-us-east-0":      "https://synthetic-monitoring-api-us-east-0.grafana.net",
}

func ResourceInstallation() *schema.Resource {
	return &schema.Resource{

		Description: `
Sets up Synthetic Monitoring on a Grafana cloud stack and generates a token. 
Once a Grafana Cloud stack is created, a user can either use this resource or go into the UI to install synthetic monitoring.
This resource cannot be imported but it can be used on an existing Synthetic Monitoring installation without issues.

**Note that this resource must be used on a provider configured with Grafana Cloud credentials.**

* [Official documentation](https://grafana.com/docs/grafana-cloud/synthetic-monitoring/installation/)
* [API documentation](https://github.com/grafana/synthetic-monitoring-api-go-client/blob/main/docs/API.md#apiv1registerinstall)
`,
		CreateContext: ResourceInstallationCreate,
		ReadContext:   ResourceInstallationRead,
		DeleteContext: ResourceInstallationDelete,

		Schema: map[string]*schema.Schema{
			"metrics_publisher_key": {
				Type:        schema.TypeString,
				Sensitive:   true,
				Required:    true,
				ForceNew:    true,
				Description: "The Cloud API Key with the `MetricsPublisher` role used to publish metrics to the SM API",
			},
			"stack_sm_api_url": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
				ForceNew:    true,
				Description: "The URL of the SM API to install SM on. This depends on the stack region, find the list of API URLs here: https://grafana.com/docs/grafana-cloud/synthetic-monitoring/private-probes/#probe-api-server-url. A static mapping exists in the provider but it may not contain all the regions. If it does contain the stack's region, this field is computed automatically and readable.",
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
}

func ResourceInstallationCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	cloudClient := meta.(*common.Client).GrafanaCloudAPI
	var stack gapi.Stack

	stackIDInt, err := strconv.ParseInt(d.Get("stack_id").(string), 10, 64)
	if err == nil {
		stack, err = cloudClient.StackByID(stackIDInt)
	} else {
		stack, err = cloudClient.StackBySlug(d.Get("stack_id").(string))
	}
	if err != nil {
		return diag.FromErr(err)
	}

	apiURL := d.Get("stack_sm_api_url").(string)
	if apiURL == "" {
		apiURL = smAPIURLs[stack.RegionSlug]
	}
	if apiURL == "" {
		return diag.Errorf("could not find a valid SM API URL for stack region %s", stack.RegionSlug)
	}

	smClient := SMAPI.NewClient(apiURL, "", nil)
	stackID, metricsID, logsID := stack.ID, int64(stack.HmInstancePromID), int64(stack.HlInstanceID)
	resp, err := smClient.Install(ctx, stackID, metricsID, logsID, d.Get("metrics_publisher_key").(string))
	if err != nil {
		return diag.FromErr(err)
	}
	d.SetId(fmt.Sprintf("%s;%d", apiURL, stackID))
	d.Set("sm_access_token", resp.AccessToken)
	d.Set("stack_sm_api_url", apiURL)
	return ResourceInstallationRead(ctx, d, meta)
}

// Management of the installation is a one-off operation. The state cannot be updated through a read operation.
// This read function will only invalidate the state (forcing recreation) if the installation has been deleted.
func ResourceInstallationRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiURL := strings.Split(d.Id(), ";")[0]
	tempClient := SMAPI.NewClient(apiURL, d.Get("sm_access_token").(string), nil)
	if err := tempClient.ValidateToken(ctx); err != nil {
		log.Printf("[WARN] removing SM installation from state because it is no longer valid")
		d.SetId("")
	}

	return nil
}

func ResourceInstallationDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	apiURL := strings.Split(d.Id(), ";")[0]
	tempClient := SMAPI.NewClient(apiURL, d.Get("sm_access_token").(string), nil)
	if err := tempClient.DeleteToken(ctx); err != nil {
		return diag.FromErr(err)
	}
	d.SetId("")
	return nil
}
