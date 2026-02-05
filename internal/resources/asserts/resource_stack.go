package asserts

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	assertsapi "github.com/grafana/grafana-asserts-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func makeResourceStack() *common.Resource {
	resourceSchema := &schema.Resource{
		Description: `Manages the Asserts Stack configuration.

This resource configures the Asserts stack with the required API tokens for integration
with Grafana Cloud services.

The ` + "`cloud_access_policy_token`" + ` is used internally for GCom API access, Mimir metrics 
authentication, and assertion detector webhook authentication. Create a Cloud Access Policy 
with the following scopes: ` + "`stacks:read`" + `, ` + "`metrics:read`" + `, ` + "`metrics:write`" + `.

The ` + "`grafana_token`" + ` is a Grafana Service Account token used for installing dashboards
and Grafana Managed Alerts.`,

		CreateContext: resourceStackUpsert,
		ReadContext:   resourceStackRead,
		UpdateContext: resourceStackUpsert,
		DeleteContext: resourceStackDelete,

		Timeouts: &schema.ResourceTimeout{
			Create: schema.DefaultTimeout(5 * time.Minute),
			Read:   schema.DefaultTimeout(2 * time.Minute),
			Update: schema.DefaultTimeout(5 * time.Minute),
			Delete: schema.DefaultTimeout(5 * time.Minute),
		},

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// Primary token - used for gcom, mimir, and assertion detector
			"cloud_access_policy_token": {
				Type:        schema.TypeString,
				Required:    true,
				Sensitive:   true,
				Description: "A Grafana Cloud Access Policy token with the following scopes: `stacks:read`, `metrics:read`, `metrics:write`. This token is used for GCom API access, Mimir authentication, and assertion detector webhook authentication.",
			},
			// Grafana Service Account token for dashboards and Grafana Managed Alerts
			"grafana_token": {
				Type:        schema.TypeString,
				Optional:    true,
				Sensitive:   true,
				Description: "A Grafana Service Account token for installing dashboards and Grafana Managed Alerts. Required permissions: `dashboards:create`, `dashboards:write`, `dashboards:read`, `folders:create`, `folders:write`, `folders:read`, `folders:delete`, `datasources:read`, `datasources:query`, `alert.provisioning:write`, `alert.notifications.provisioning:write`, `alert.notifications:write`, `alert.rules:read`, `alert.rules:create`, `alert.rules:delete`. Create using `grafana_cloud_stack_service_account_token` resource.",
			},
			// Computed fields
			"enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the stack is currently enabled.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Current onboarding status of the stack.",
			},
			"version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Configuration version number.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAsserts,
		"grafana_asserts_stack",
		common.NewResourceID(common.StringIDField("id")),
		resourceSchema,
	).WithLister(assertsListerFunction(listStack))
}

// resourceStackUpsert creates or updates the stack using the V2 API endpoints.
// The full onboarding flow is:
// 1. PUT /v2/stack - provision tokens
// 2. POST /v2/stack/datasets/auto-setup - auto-configure datasets
// 3. POST /v2/stack/enable - enable the stack with configured datasets
func resourceStackUpsert(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	stackIDStr := fmt.Sprintf("%d", stackID)

	// Step 1: Provision tokens via PUT /v2/stack
	stackDto := buildStackDto(d)
	putRequest := client.StackControllerAPI.PutV2Stack(ctx).
		StackDto(*stackDto).
		XScopeOrgID(stackIDStr)

	_, err := putRequest.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to provision stack tokens", err))
	}

	// Step 2: Auto-configure datasets via POST /v2/stack/datasets/auto-setup
	autoConfigRequest := client.StackControllerAPI.DetectAndAutoConfigureDatasets(ctx).
		XScopeOrgID(stackIDStr)

	_, _, err = autoConfigRequest.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to auto-configure datasets", err))
	}

	// Step 3: Enable the stack via POST /v2/stack/enable
	// Add Content-Type header since the endpoint requires it even without a body
	cfg := client.GetConfig()
	cfg.AddDefaultHeader("Content-Type", "application/json")

	enableRequest := client.StackControllerAPI.EnableV2Stack(ctx).
		XScopeOrgID(stackIDStr)

	// The enable endpoint returns HTTP 409 Conflict if there are blockers in the sanity checks
	_, _, err = enableRequest.Execute()
	if err != nil {
		return diag.FromErr(formatAPIError("failed to enable stack", err))
	}

	d.SetId(stackIDStr)
	return resourceStackRead(ctx, d, meta)
}

// resourceStackRead reads the stack configuration using the GetStatus API endpoint.
// Note: We must use the V1 endpoint (/v1/stack/status) because there is no V2 equivalent
// for reading stack status. The V2 API only provides create/update/enable/disable operations.
func resourceStackRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Retry logic for read operation to handle eventual consistency
	var stackStatus *assertsapi.StackStatusDto
	err := withRetryRead(ctx, func(retryCount, maxRetries int) *retry.RetryError {
		request := client.StackControllerAPI.GetStatus(ctx).
			XScopeOrgID(fmt.Sprintf("%d", stackID))

		status, _, err := request.Execute()
		if err != nil {
			return createAPIError("get stack status", retryCount, maxRetries, err)
		}

		stackStatus = status
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	if stackStatus == nil {
		d.SetId("")
		return nil
	}

	// Set computed fields (tokens are write-only, so we don't read them back)
	if stackStatus.HasEnabled() {
		if err := d.Set("enabled", stackStatus.GetEnabled()); err != nil {
			return diag.FromErr(err)
		}
	}

	if stackStatus.HasStatus() {
		if err := d.Set("status", stackStatus.GetStatus()); err != nil {
			return diag.FromErr(err)
		}
	}

	if stackStatus.HasVersion() {
		if err := d.Set("version", int(stackStatus.GetVersion())); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}

// resourceStackDelete disables the stack using the V2 API endpoint.
// Note: The V2 disable endpoint requires Content-Type: application/json header even though
// it doesn't have a request body. We add it to DefaultHeader before calling the API.
func resourceStackDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, stackID, diags := validateAssertsClient(meta)
	if diags.HasError() {
		return diags
	}

	// Add Content-Type header to satisfy the backend requirement
	// The V2 disable endpoint requires this header even without a body
	cfg := client.GetConfig()
	cfg.AddDefaultHeader("Content-Type", "application/json")

	request := client.StackControllerAPI.DisableV2Stack(ctx).
		XScopeOrgID(fmt.Sprintf("%d", stackID))

	_, err := request.Execute()
	if err != nil {
		return diag.FromErr(fmt.Errorf("failed to disable stack: %w", err))
	}

	d.SetId("")
	return nil
}

// buildStackDto constructs a StackDto from the Terraform schema.
// The cloud_access_policy_token is used for all three backend token fields:
// gcomToken, mimirToken, and assertionDetectorToken.
func buildStackDto(d *schema.ResourceData) *assertsapi.StackDto {
	dto := assertsapi.NewStackDto()

	// Use the single cloud_access_policy_token for all three backend tokens
	if v, ok := d.GetOk("cloud_access_policy_token"); ok {
		token := v.(string)
		dto.SetGcomToken(token)
		dto.SetMimirToken(token)
		dto.SetAssertionDetectorToken(token)
	}

	// Grafana token is separate (it's a Service Account token, not a CAP token)
	if v, ok := d.GetOk("grafana_token"); ok {
		dto.SetGrafanaToken(v.(string))
	}

	return dto
}

// listStack retrieves the stack ID for listing.
func listStack(ctx context.Context, client *assertsapi.APIClient, stackID string) ([]string, error) {
	request := client.StackControllerAPI.GetStatus(ctx).
		XScopeOrgID(stackID)

	_, _, err := request.Execute()
	if err != nil {
		// If stack doesn't exist, return empty list
		return []string{}, nil
	}

	// Return the stack ID as the single resource ID
	return []string{stackID}, nil
}
