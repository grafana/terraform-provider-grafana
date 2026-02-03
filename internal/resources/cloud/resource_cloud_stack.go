package cloud

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework-validators/mapvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	sdkdiag "github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
)

const defaultReadinessTimeout = time.Minute * 5

var (
	stackLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9/\-._]+$`)
	stackSlugRegex  = regexp.MustCompile(`^[a-z][a-z0-9]+$`)
)

var _ resource.Resource = &CloudStackResource{}
var _ resource.ResourceWithConfigure = &CloudStackResource{}
var _ resource.ResourceWithImportState = &CloudStackResource{}

var resourceCloudStackName = "grafana_cloud_stack"

type CloudStackResource struct {
	basePluginFrameworkResource
}

func resourceStack() *common.Resource {
	return common.NewResource(
		common.CategoryCloud,
		resourceCloudStackName,
		common.NewResourceID(common.StringIDField("stackSlugOrID")),
		&CloudStackResource{},
	).
		WithLister(cloudListerFunction(listStacks)).
		WithPreferredResourceNameField("name")
}

func listStacks(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	stacks, err := data.Stacks(ctx, client)
	if err != nil {
		return nil, err
	}

	var stackSlugs []string
	for _, stack := range stacks {
		stackSlugs = append(stackSlugs, stack.Slug)
	}
	return stackSlugs, nil
}

func (r *CloudStackResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = resourceCloudStackName
}

func (r *CloudStackResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#stacks/)

Required access policy scopes:

* stacks:read
* stacks:write
* stacks:delete
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. `<stack_slug>.grafana.net`).",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Description of stack.",
			},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "Subdomain that the Grafana instance will be available at. Setting slug to `<stack_slug>` will make the instance available at `https://<stack_slug>.grafana.net`.",
				Validators: []validator.String{
					stringvalidator.RegexMatches(stackSlugRegex, "must be a lowercase alphanumeric string and must start with a letter."),
					stringvalidator.LengthBetween(1, 29),
				},
			},
			"region_slug": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Region slug to assign to this stack. Changing region will destroy the existing stack and create a new one in the desired region. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Slug of the cluster where this stack resides.",
			},
			"cluster_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the cluster where this stack resides.",
			},
			"url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Custom URL for the Grafana instance. Must have a CNAME setup to point to `.grafana.net` before creating the stack",
			},
			"wait_for_readiness": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to wait for readiness of the stack after creating it. The check is a HEAD request to the stack URL (Grafana instance).",
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"wait_for_readiness_timeout": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(defaultReadinessTimeout.String()),
				Description: "How long to wait for readiness (if enabled).",
			},
			"org_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": schema.StringAttribute{
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": schema.StringAttribute{
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the stack.",
			},
			"labels": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: fmt.Sprintf("A map of labels to assign to the stack. Label keys and values must match the following regexp: %q and stacks cannot have more than 10 labels.", stackLabelRegex.String()),
				Validators: []validator.Map{
					mapvalidator.SizeAtMost(10),
					mapvalidator.KeysAre(stringvalidator.RegexMatches(stackLabelRegex, "label key must match "+stackLabelRegex.String())),
					mapvalidator.ValueStringsAre(stringvalidator.RegexMatches(stackLabelRegex, "label value must match "+stackLabelRegex.String())),
				},
			},
			"delete_protection": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether to enable delete protection for the stack, preventing accidental deletion.",
			},

			// IP Allow List CNAMEs
			"grafanas_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the grafana instance (Optional)",
			},

			// Prometheus (Metrics/Mimir) - all computed
			"prometheus_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "Prometheus user ID. Used for e.g. remote_write.",
			},
			"prometheus_url": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prometheus_name": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prometheus_remote_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana",
			},
			"prometheus_remote_write_endpoint": schema.StringAttribute{
				Computed:    true,
				Description: "Use this URL to send prometheus metrics to Grafana cloud",
			},
			"prometheus_status": schema.StringAttribute{
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},
			"prometheus_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Prometheus when using AWS PrivateLink (only for AWS stacks)",
			},
			"prometheus_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Prometheus when using AWS PrivateLink (only for AWS stacks)",
			},
			"prometheus_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Prometheus instance (Optional)",
			},

			// Alertmanager - all computed
			"alertmanager_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "User ID of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Alertmanager instances (Optional)",
			},

			// OnCall
			"oncall_api_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the OnCall API instance configured for this stack.",
			},

			// Logs (Loki) - all computed
			"logs_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"logs_name": schema.StringAttribute{
				Computed: true,
			},
			"logs_url": schema.StringAttribute{
				Computed: true,
			},
			"logs_status": schema.StringAttribute{
				Computed: true,
			},
			"logs_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Logs when using AWS PrivateLink (only for AWS stacks)",
			},
			"logs_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Logs when using AWS PrivateLink (only for AWS stacks)",
			},
			"logs_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Logs instance (Optional)",
			},

			// Traces (Tempo) - all computed
			"traces_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"traces_name": schema.StringAttribute{
				Computed: true,
			},
			"traces_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Traces instance configured for this stack. To use this in the Tempo data source in Grafana, append `/tempo` to the URL.",
			},
			"traces_status": schema.StringAttribute{
				Computed: true,
			},
			"traces_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Traces when using AWS PrivateLink (only for AWS stacks)",
			},
			"traces_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Traces when using AWS PrivateLink (only for AWS stacks)",
			},
			"traces_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Traces instance (Optional)",
			},

			// Profiles (Pyroscope) - all computed
			"profiles_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"profiles_name": schema.StringAttribute{
				Computed: true,
			},
			"profiles_url": schema.StringAttribute{
				Computed: true,
			},
			"profiles_status": schema.StringAttribute{
				Computed: true,
			},
			"profiles_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Profiles when using AWS PrivateLink (only for AWS stacks)",
			},
			"profiles_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Profiles when using AWS PrivateLink (only for AWS stacks)",
			},
			"profiles_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Profiles instance (Optional)",
			},

			// Graphite - all computed
			"graphite_user_id": schema.Int64Attribute{
				Computed: true,
			},
			"graphite_name": schema.StringAttribute{
				Computed: true,
			},
			"graphite_url": schema.StringAttribute{
				Computed: true,
			},
			"graphite_status": schema.StringAttribute{
				Computed: true,
			},
			"graphite_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Graphite when using AWS PrivateLink (only for AWS stacks)",
			},
			"graphite_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Graphite when using AWS PrivateLink (only for AWS stacks)",
			},
			"graphite_ip_allow_list_cname": schema.StringAttribute{
				Computed:    true,
				Description: "Comma-separated list of CNAMEs that can be whitelisted to access the Graphite instance (Optional)",
			},

			// Fleet Management - all computed
			"fleet_management_user_id": schema.Int64Attribute{
				Computed:    true,
				Description: "User ID of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_name": schema.StringAttribute{
				Computed:    true,
				Description: "Name of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_status": schema.StringAttribute{
				Computed:    true,
				Description: "Status of the Fleet Management instance configured for this stack.",
			},
			"fleet_management_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for Fleet Management when using AWS PrivateLink (only for AWS stacks)",
			},
			"fleet_management_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for Fleet Management when using AWS PrivateLink (only for AWS stacks)",
			},

			// Connections
			"influx_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the InfluxDB instance configured for this stack. The username is the same as the metrics' (`prometheus_user_id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-influxdb/push-from-telegraf/ for docs on how to use this.",
			},
			"otlp_url": schema.StringAttribute{
				Computed:    true,
				Description: "Base URL of the OTLP instance configured for this stack. The username is the stack's ID (`id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/ for docs on how to use this.",
			},
			"otlp_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for OTLP when using AWS PrivateLink (only for AWS stacks)",
			},
			"otlp_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for OTLP when using AWS PrivateLink (only for AWS stacks)",
			},

			// PDC
			"pdc_api_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for PDC's API when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_api_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for PDC's API when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_gateway_private_connectivity_info_private_dns": schema.StringAttribute{
				Computed:    true,
				Description: "Private DNS for PDC's Gateway when using AWS PrivateLink (only for AWS stacks)",
			},
			"pdc_gateway_private_connectivity_info_service_name": schema.StringAttribute{
				Computed:    true,
				Description: "Service Name for PDC's Gateway when using AWS PrivateLink (only for AWS stacks)",
			},
		},
	}
}

func (r *CloudStackResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data StackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert labels if present
	var labels map[string]string
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	stack := gcom.PostInstancesRequest{
		Name:             data.Name.ValueString(),
		Slug:             common.Ref(data.Slug.ValueString()),
		Url:              common.Ref(data.URL.ValueString()),
		Region:           common.Ref(data.RegionSlug.ValueString()),
		Description:      common.Ref(data.Description.ValueString()),
		Labels:           common.Ref(labels),
		DeleteProtection: common.Ref(data.DeleteProtection.ValueBool()),
	}

	fmt.Printf("Starting to create: %+v\n", stack)

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		createReq := r.client.InstancesAPI.PostInstances(ctx).PostInstancesRequest(stack).XRequestId(ClientRequestID())
		createdStack, httpResp, err := createReq.Execute()

		// Helper function to get response body
		getResponseBody := func(resp *http.Response) string {
			if resp == nil || resp.Body == nil {
				return ""
			}
			body, readErr := io.ReadAll(resp.Body)
			resp.Body.Close()
			if readErr != nil {
				return fmt.Sprintf("(error reading body: %v)", readErr)
			}
			return string(body)
		}

		if httpResp != nil {
			fmt.Printf("Response Status: %+v\n", httpResp)
		}

		switch {
		case err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict"):
			time.Sleep(10 * time.Second)
			errMsg := fmt.Sprintf("conflict error: %v", err)
			if httpResp != nil {
				bodyMsg := getResponseBody(httpResp)
				errMsg = fmt.Sprintf("conflict error (status %d): %v - response: %s", httpResp.StatusCode, err, bodyMsg)
			}
			return retry.RetryableError(fmt.Errorf("%s", errMsg))
		case err != nil:
			// If we had an error that isn't a conflict, try to read the stack
			readReq := r.client.InstancesAPI.GetInstance(ctx, *stack.Slug)
			readStack, _, readErr := readReq.Execute()
			if readErr == nil {
				data.ID = types.StringValue(strconv.FormatInt(int64(readStack.Id), 10))
				return nil
			}
			time.Sleep(10 * time.Second)
			errMsg := fmt.Sprintf("failed to create stack: %v", err)
			if httpResp != nil {
				bodyMsg := getResponseBody(httpResp)
				errMsg = fmt.Sprintf("failed to create stack (status %d): %v - response: %s", httpResp.StatusCode, err, bodyMsg)
			}
			return retry.RetryableError(fmt.Errorf("%s", errMsg))
		default:
			data.ID = types.StringValue(strconv.FormatInt(int64(createdStack.Id), 10))
		}
		return nil
	})
	if err != nil {
		resp.Diagnostics.AddError("Failed to create stack", err.Error())
		return
	}

	// Read back the full stack data
	model, diags := ReadStackData(ctx, r.client, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve the wait_for_readiness settings from the plan
	model.WaitForReadiness = data.WaitForReadiness
	model.WaitForReadinessTimeout = data.WaitForReadinessTimeout

	// Wait for readiness if requested
	if model.WaitForReadiness.ValueBool() {
		timeout := defaultReadinessTimeout
		if !model.WaitForReadinessTimeout.IsNull() && model.WaitForReadinessTimeout.ValueString() != "" {
			timeout, _ = time.ParseDuration(model.WaitForReadinessTimeout.ValueString())
		}
		diags = waitForStackReadinessFramework(ctx, timeout, model.URL.ValueString())
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *CloudStackResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data StackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Use the shared read function
	model, diags := ReadStackData(ctx, r.client, data.ID.ValueString())
	if diags.HasError() {
		// Check if it's a not found error
		for _, d := range diags.Errors() {
			if strings.Contains(d.Summary(), "not found") || strings.Contains(d.Summary(), "deleted") {
				resp.State.RemoveResource(ctx)
				return
			}
		}
		resp.Diagnostics.Append(diags...)
		return
	}

	// Preserve wait_for_readiness settings from state
	model.WaitForReadiness = data.WaitForReadiness
	model.WaitForReadinessTimeout = data.WaitForReadinessTimeout

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *CloudStackResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data StackModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Convert labels if present
	var labels map[string]string
	if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
		resp.Diagnostics.Append(data.Labels.ElementsAs(ctx, &labels, false)...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	// Default to the slug if the URL is not set
	url := data.URL.ValueString()
	if url == "" {
		url = defaultStackURL(data.Slug.ValueString())
	}

	stack := gcom.PostInstanceRequest{
		Name:             common.Ref(data.Name.ValueString()),
		Slug:             common.Ref(data.Slug.ValueString()),
		Description:      common.Ref(data.Description.ValueString()),
		Url:              &url,
		Labels:           common.Ref(labels),
		DeleteProtection: common.Ref(data.DeleteProtection.ValueBool()),
	}

	updateReq := r.client.InstancesAPI.PostInstance(ctx, data.ID.ValueString()).PostInstanceRequest(stack).XRequestId(ClientRequestID())
	_, httpResp, err := updateReq.Execute()
	if err != nil {
		errMsg := fmt.Sprintf("error: %v", err)
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			errMsg = fmt.Sprintf("error (status %d): %v - response: %s", httpResp.StatusCode, err, string(body))
		} else if httpResp != nil {
			errMsg = fmt.Sprintf("error (status %d): %v", httpResp.StatusCode, err)
		}
		resp.Diagnostics.AddError("Failed to update stack", errMsg)
		return
	}

	// Read back the full stack data
	model, diags := ReadStackData(ctx, r.client, data.ID.ValueString())
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Preserve wait_for_readiness settings from plan
	model.WaitForReadiness = data.WaitForReadiness
	model.WaitForReadinessTimeout = data.WaitForReadinessTimeout

	// Wait for readiness if requested
	if model.WaitForReadiness.ValueBool() {
		timeout := defaultReadinessTimeout
		if !model.WaitForReadinessTimeout.IsNull() && model.WaitForReadinessTimeout.ValueString() != "" {
			timeout, _ = time.ParseDuration(model.WaitForReadinessTimeout.ValueString())
		}
		diags = waitForStackReadinessFramework(ctx, timeout, model.URL.ValueString())
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, model)...)
}

func (r *CloudStackResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data StackModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteReq := r.client.InstancesAPI.DeleteInstance(ctx, data.ID.ValueString()).XRequestId(ClientRequestID())
	_, httpResp, err := deleteReq.Execute()
	if err != nil {
		errMsg := fmt.Sprintf("error: %v", err)
		if httpResp != nil && httpResp.Body != nil {
			body, _ := io.ReadAll(httpResp.Body)
			httpResp.Body.Close()
			errMsg = fmt.Sprintf("error (status %d): %v - response: %s", httpResp.StatusCode, err, string(body))
		} else if httpResp != nil {
			errMsg = fmt.Sprintf("error (status %d): %v", httpResp.StatusCode, err)
		}
		resp.Diagnostics.AddError("Failed to delete stack", errMsg)
		return
	}
}

func (r *CloudStackResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Set the ID from the import identifier
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("id"), req.ID)...)
}

// waitForStackReadinessFramework is the Framework SDK version of waitForStackReadiness
func waitForStackReadinessFramework(ctx context.Context, timeout time.Duration, stackURL string) diag.Diagnostics {
	var diags diag.Diagnostics

	healthURL, joinErr := url.JoinPath(stackURL, "api", "health")
	if joinErr != nil {
		diags.AddError("Failed to construct health URL", joinErr.Error())
		return diags
	}
	wakePath, joinErr := url.JoinPath(stackURL, "login")
	if joinErr != nil {
		diags.AddError("Failed to construct wake URL", joinErr.Error())
		return diags
	}
	wakeURL := wakePath + "?disableAutoLogin=true"

	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		// Query the instance login page directly. This makes the stack wake-up if it has been paused.
		stackReq, err := http.NewRequestWithContext(ctx, http.MethodGet, wakeURL, nil)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		stackResp, err := http.DefaultClient.Do(stackReq)
		if err != nil {
			return retry.RetryableError(err)
		}
		defer stackResp.Body.Close()

		healthReq, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		healthResp, err := http.DefaultClient.Do(healthReq)
		if err != nil {
			return retry.RetryableError(err)
		}
		defer healthResp.Body.Close()
		if healthResp.StatusCode != 200 {
			buf := new(bytes.Buffer)
			body := ""
			_, err = buf.ReadFrom(healthResp.Body)
			if err != nil {
				body = "unable to read response body, error: " + err.Error()
			} else {
				body = buf.String()
			}
			return retry.RetryableError(fmt.Errorf("stack was not ready in %s. Status code: %d, Body: %s", timeout, healthResp.StatusCode, body))
		}

		return nil
	})
	if err != nil {
		diags.AddError("Stack readiness timeout", fmt.Sprintf("Error waiting for stack (URL: %s) to be ready: %v", healthURL, err))
	}

	return diags
}

func defaultStackURL(slug string) string {
	return fmt.Sprintf("https://%s.grafana.net", slug)
}

// waitForStackReadinessFromSlug is a helper for SDK v2 code that needs stack readiness checking
// This returns SDK v2 diagnostics for compatibility with existing SDK v2 resources
func waitForStackReadinessFromSlug(ctx context.Context, timeout time.Duration, slug string, client *gcom.APIClient) sdkdiag.Diagnostics {
	stack, _, err := client.InstancesAPI.GetInstance(ctx, slug).Execute()
	if err != nil {
		return sdkdiag.FromErr(err)
	}

	// Call the legacy waitForStackReadiness function
	return waitForStackReadiness(ctx, timeout, stack.Url)
}

// waitForStackReadiness returns SDK v2 diagnostics for compatibility with SDK v2 resources
func waitForStackReadiness(ctx context.Context, timeout time.Duration, stackURL string) sdkdiag.Diagnostics {
	healthURL, joinErr := url.JoinPath(stackURL, "api", "health")
	if joinErr != nil {
		return sdkdiag.FromErr(joinErr)
	}
	wakePath, joinErr := url.JoinPath(stackURL, "login")
	if joinErr != nil {
		return sdkdiag.FromErr(joinErr)
	}
	wakeURL := wakePath + "?disableAutoLogin=true"

	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		// Query the instance login page directly. This makes the stack wake-up if it has been paused.
		// The health endpoint is helpful to check that the stack is ready, but it doesn't wake up the stack.
		stackReq, err := http.NewRequestWithContext(ctx, http.MethodGet, wakeURL, nil)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		stackResp, err := http.DefaultClient.Do(stackReq)
		if err != nil {
			return retry.RetryableError(err)
		}
		defer stackResp.Body.Close()

		healthReq, err := http.NewRequestWithContext(ctx, http.MethodGet, healthURL, nil)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		healthResp, err := http.DefaultClient.Do(healthReq)
		if err != nil {
			return retry.RetryableError(err)
		}
		defer healthResp.Body.Close()
		if healthResp.StatusCode != 200 {
			buf := new(bytes.Buffer)
			body := ""
			_, err = buf.ReadFrom(healthResp.Body)
			if err != nil {
				body = "unable to read response body, error: " + err.Error()
			} else {
				body = buf.String()
			}
			return retry.RetryableError(fmt.Errorf("stack was not ready in %s. Status code: %d, Body: %s", timeout, healthResp.StatusCode, body))
		}

		return nil
	})
	if err != nil {
		return sdkdiag.Errorf("error waiting for stack (URL: %s) to be ready: %v", healthURL, err)
	}

	return nil
}
