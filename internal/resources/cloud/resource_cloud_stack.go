package cloud

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/customdiff"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const defaultReadinessTimeout = time.Minute * 5

var (
	stackLabelRegex = regexp.MustCompile(`^[a-zA-Z0-9/\-.]+$`)
	stackSlugRegex  = regexp.MustCompile(`^[a-z][a-z0-9]+$`)
	resourceStackID = common.NewResourceID(common.StringIDField("stackSlugOrID"))
)

func resourceStack() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#stacks/)

Required access policy scopes:

* stacks:read
* stacks:write
* stacks:delete
`,

		CreateContext: withClient[schema.CreateContextFunc](createStack),
		UpdateContext: withClient[schema.UpdateContextFunc](updateStack),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteStack),
		ReadContext:   withClient[schema.ReadContextFunc](readStack),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The stack id assigned to this stack by Grafana.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. `<stack_slug>.grafana.net`).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of stack.",
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: "Subdomain that the Grafana instance will be available at. " +
					"Setting slug to `<stack_slug>` will make the instance available at `https://<stack_slug>.grafana.net`.",
				ValidateFunc: validation.All(
					validation.StringMatch(stackSlugRegex, "must be a lowercase alphanumeric string and must start with a letter."),
					validation.StringLenBetween(1, 29),
				),
			},
			"region_slug": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: `Region slug to assign to this stack. Changing region will destroy the existing stack and create a new one in the desired region. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.`,
				DiffSuppressFunc: func(_, oldValue, newValue string, _ *schema.ResourceData) bool {
					return oldValue == newValue || newValue == "" // Ignore default region
				},
			},
			"url": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Custom URL for the Grafana instance. Must have a CNAME setup to point to `.grafana.net` before creating the stack",
				DiffSuppressFunc: func(k, oldValue, newValue string, d *schema.ResourceData) bool {
					return oldValue == newValue ||
						// No diff if we're using the default URL
						(oldValue == defaultStackURL(d.Get("slug").(string)) && newValue == "")
				},
			},
			"wait_for_readiness": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to wait for readiness of the stack after creating it. The check is a HEAD request to the stack URL (Grafana instance).",
				// Suppress the diff if the stack is already created
				DiffSuppressFunc: func(_, _, _ string, d *schema.ResourceData) bool { return !d.IsNewResource() },
			},
			"wait_for_readiness_timeout": {
				Type:             schema.TypeString,
				Optional:         true,
				Default:          defaultReadinessTimeout.String(),
				ValidateDiagFunc: common.ValidateDuration,
				// Only used when wait_for_readiness is true
				DiffSuppressFunc: func(_, _, newValue string, d *schema.ResourceData) bool {
					return newValue == defaultReadinessTimeout.String()
				},
				Description: "How long to wait for readiness (if enabled).",
			},
			"org_id":   common.ComputedIntWithDescription("Organization id to assign to this stack."),
			"org_slug": common.ComputedStringWithDescription("Organization slug to assign to this stack."),
			"org_name": common.ComputedStringWithDescription("Organization name to assign to this stack."),
			"status":   common.ComputedStringWithDescription("Status of the stack."),
			"labels": {
				Type:        schema.TypeMap,
				Optional:    true,
				Description: fmt.Sprintf("A map of labels to assign to the stack. Label keys and values must match the following regexp: %q and stacks cannot have more than 10 labels.", stackLabelRegex.String()),
				Elem:        &schema.Schema{Type: schema.TypeString},
				ValidateFunc: func(i interface{}, s string) ([]string, []error) {
					labels := i.(map[string]interface{})
					if len(labels) > 10 {
						return nil, []error{fmt.Errorf("stacks cannot have more than 10 labels")}
					}
					for k, v := range labels {
						if !stackLabelRegex.MatchString(k) {
							return nil, []error{fmt.Errorf("label key %q does not match %q", k, stackLabelRegex.String())}
						}
						if !stackLabelRegex.MatchString(v.(string)) {
							return nil, []error{fmt.Errorf("label value %q does not match %q", v, stackLabelRegex.String())}
						}
					}
					return nil, nil
				},
			},

			// Metrics (Mimir/Prometheus)
			"prometheus_user_id":               common.ComputedIntWithDescription("Prometheus user ID. Used for e.g. remote_write."),
			"prometheus_url":                   common.ComputedStringWithDescription("Prometheus url for this instance."),
			"prometheus_name":                  common.ComputedStringWithDescription("Prometheus name for this instance."),
			"prometheus_remote_endpoint":       common.ComputedStringWithDescription("Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana"),
			"prometheus_remote_write_endpoint": common.ComputedStringWithDescription("Use this URL to send prometheus metrics to Grafana cloud"),
			"prometheus_status":                common.ComputedStringWithDescription("Prometheus status for this instance."),

			// Alertmanager
			"alertmanager_user_id": common.ComputedIntWithDescription("User ID of the Alertmanager instance configured for this stack."),
			"alertmanager_name":    common.ComputedStringWithDescription("Name of the Alertmanager instance configured for this stack."),
			"alertmanager_url":     common.ComputedStringWithDescription("Base URL of the Alertmanager instance configured for this stack."),
			"alertmanager_status":  common.ComputedStringWithDescription("Status of the Alertmanager instance configured for this stack."),

			// Logs (Loki)
			"logs_user_id": common.ComputedInt(),
			"logs_name":    common.ComputedString(),
			"logs_url":     common.ComputedString(),
			"logs_status":  common.ComputedString(),

			// Traces (Tempo)
			"traces_user_id": common.ComputedInt(),
			"traces_name":    common.ComputedString(),
			"traces_url":     common.ComputedStringWithDescription("Base URL of the Traces instance configured for this stack. To use this in the Tempo data source in Grafana, append `/tempo` to the URL."),
			"traces_status":  common.ComputedString(),

			// Profiles (Pyroscope)
			"profiles_user_id": common.ComputedInt(),
			"profiles_name":    common.ComputedString(),
			"profiles_url":     common.ComputedString(),
			"profiles_status":  common.ComputedString(),

			// Graphite
			"graphite_user_id": common.ComputedInt(),
			"graphite_name":    common.ComputedString(),
			"graphite_url":     common.ComputedString(),
			"graphite_status":  common.ComputedString(),

			// Fleet Management
			"fleet_management_user_id": common.ComputedIntWithDescription("User ID of the Fleet Management instance configured for this stack."),
			"fleet_management_name":    common.ComputedStringWithDescription("Name of the Fleet Management instance configured for this stack."),
			"fleet_management_url":     common.ComputedStringWithDescription("Base URL of the Fleet Management instance configured for this stack."),
			"fleet_management_status":  common.ComputedStringWithDescription("Status of the Fleet Management instance configured for this stack."),

			// Connections
			"influx_url": common.ComputedStringWithDescription("Base URL of the InfluxDB instance configured for this stack. The username is the same as the metrics' (`prometheus_user_id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/metrics/metrics-influxdb/push-from-telegraf/ for docs on how to use this."),
			"otlp_url":   common.ComputedStringWithDescription("Base URL of the OTLP instance configured for this stack. The username is the stack's ID (`id` attribute of this resource). See https://grafana.com/docs/grafana-cloud/send-data/otlp/send-data-otlp/ for docs on how to use this."),
		},
		CustomizeDiff: customdiff.All(
			customdiff.ComputedIf("url", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("slug")
			}),
			customdiff.ComputedIf("alertmanager_name", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("slug")
			}),
			customdiff.ComputedIf("logs_name", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("slug")
			}),
			customdiff.ComputedIf("traces_name", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("slug")
			}),
			customdiff.ComputedIf("prometheus_name", func(_ context.Context, diff *schema.ResourceDiff, meta interface{}) bool {
				return diff.HasChange("slug")
			}),
		),
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_stack",
		resourceStackID,
		schema,
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

func createStack(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	stack := gcom.PostInstancesRequest{
		Name:        d.Get("name").(string),
		Slug:        common.Ref(d.Get("slug").(string)),
		Url:         common.Ref(d.Get("url").(string)),
		Region:      common.Ref(d.Get("region_slug").(string)),
		Description: common.Ref(d.Get("description").(string)),
		Labels:      common.Ref(common.UnpackMap[string](d.Get("labels"))),
	}

	err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		req := client.InstancesAPI.PostInstances(ctx).PostInstancesRequest(stack).XRequestId(ClientRequestID())
		createdStack, _, err := req.Execute()
		switch {
		case err != nil && strings.Contains(strings.ToLower(err.Error()), "conflict"):
			// If the API returns a conflict error, it means that the stack already exists
			// It may also mean that the stack was recently deleted and is still in the process of being deleted
			// In that case, we want to retry
			time.Sleep(10 * time.Second) // Do not retry too fast, default is 500ms
			return retry.RetryableError(err)
		case err != nil:
			// If we had an error that isn't a a conflict error (already exists), try to read the stack
			// Sometimes, the stack is created but the API returns an error (e.g. 504)
			readReq := client.InstancesAPI.GetInstance(ctx, *stack.Slug)
			readStack, _, readErr := readReq.Execute()
			if readErr == nil {
				d.SetId(strconv.FormatInt(int64(readStack.Id), 10))
				return nil
			}
			time.Sleep(10 * time.Second) // Do not retry too fast, default is 500ms
			return retry.RetryableError(fmt.Errorf("failed to create stack: %w", err))
		default:
			d.SetId(strconv.FormatInt(int64(createdStack.Id), 10))
		}
		return nil
	})
	if err != nil {
		return apiError(err)
	}

	if diag := readStack(ctx, d, client); diag != nil {
		return diag
	}

	if d.Get("wait_for_readiness").(bool) {
		timeout := defaultReadinessTimeout
		if timeoutVal := d.Get("wait_for_readiness_timeout").(string); timeoutVal != "" {
			timeout, _ = time.ParseDuration(timeoutVal)
		}
		return waitForStackReadiness(ctx, timeout, d.Get("url").(string))
	}
	return nil
}

func updateStack(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	id, err := resourceStackID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	// Default to the slug if the URL is not set
	url := d.Get("url").(string)
	if url == "" {
		url = defaultStackURL(d.Get("slug").(string))
	}

	stack := gcom.PostInstanceRequest{
		Name:        common.Ref(d.Get("name").(string)),
		Slug:        common.Ref(d.Get("slug").(string)),
		Description: common.Ref(d.Get("description").(string)),
		Url:         &url,
		Labels:      common.Ref(common.UnpackMap[string](d.Get("labels"))),
	}
	req := client.InstancesAPI.PostInstance(ctx, id.(string)).PostInstanceRequest(stack).XRequestId(ClientRequestID())
	_, _, err = req.Execute()
	if err != nil {
		return apiError(err)
	}

	if diag := readStack(ctx, d, client); diag != nil {
		return diag
	}

	if d.Get("wait_for_readiness").(bool) {
		timeout := defaultReadinessTimeout
		if timeoutVal := d.Get("wait_for_readiness_timeout").(string); timeoutVal != "" {
			timeout, _ = time.ParseDuration(timeoutVal)
		}
		return waitForStackReadiness(ctx, timeout, d.Get("url").(string))
	}
	return nil
}

func deleteStack(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	id, err := resourceStackID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := client.InstancesAPI.DeleteInstance(ctx, id.(string)).XRequestId(ClientRequestID())
	_, _, err = req.Execute()
	return apiError(err)
}

func readStack(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	id, err := resourceStackID.Single(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}

	req := client.InstancesAPI.GetInstance(ctx, id.(string))
	stack, _, err := req.Execute()
	if err, shouldReturn := common.CheckReadError("stack", d, err); shouldReturn {
		return err
	}

	if stack.Status == "deleted" {
		return common.WarnMissing("stack", d)
	}

	connectionsReq := client.InstancesAPI.GetConnections(ctx, id.(string))
	connections, _, err := connectionsReq.Execute()
	if err != nil {
		return apiError(err)
	}

	if err := flattenStack(d, stack, connections); err != nil {
		return diag.FromErr(err)
	}
	// Always set the wait attribute to true after creation
	// It no longer matters and this will prevent drift if the stack was imported
	// The "if" condition is here to allow using the same Read function for the data source
	if v, ok := d.GetOk("wait_for_readiness"); ok && !v.(bool) {
		d.Set("wait_for_readiness", true)
	}

	return nil
}

func flattenStack(d *schema.ResourceData, stack *gcom.FormattedApiInstance, connections *gcom.FormattedApiInstanceConnections) error {
	id := strconv.FormatInt(int64(stack.Id), 10)
	d.SetId(id)
	d.Set("name", stack.Name)
	d.Set("slug", stack.Slug)
	d.Set("url", stack.Url)
	d.Set("status", stack.Status)
	d.Set("region_slug", stack.RegionSlug)
	d.Set("description", stack.Description)
	d.Set("labels", stack.Labels)

	d.Set("org_id", stack.OrgId)
	d.Set("org_slug", stack.OrgSlug)
	d.Set("org_name", stack.OrgName)

	d.Set("prometheus_user_id", stack.HmInstancePromId)
	d.Set("prometheus_url", stack.HmInstancePromUrl)
	d.Set("prometheus_name", stack.HmInstancePromName)
	reURL, err := appendPath(stack.HmInstancePromUrl, "/api/prom")
	if err != nil {
		return err
	}
	d.Set("prometheus_remote_endpoint", reURL)
	rweURL, err := appendPath(stack.HmInstancePromUrl, "/api/prom/push")
	if err != nil {
		return err
	}
	d.Set("prometheus_remote_write_endpoint", rweURL)
	d.Set("prometheus_status", stack.HmInstancePromStatus)

	d.Set("logs_user_id", stack.HlInstanceId)
	d.Set("logs_url", stack.HlInstanceUrl)
	d.Set("logs_name", stack.HlInstanceName)
	d.Set("logs_status", stack.HlInstanceStatus)

	d.Set("alertmanager_user_id", stack.AmInstanceId)
	d.Set("alertmanager_name", stack.AmInstanceName)
	d.Set("alertmanager_url", stack.AmInstanceUrl)
	d.Set("alertmanager_status", stack.AmInstanceStatus)

	d.Set("traces_user_id", stack.HtInstanceId)
	d.Set("traces_name", stack.HtInstanceName)
	d.Set("traces_url", stack.HtInstanceUrl)
	d.Set("traces_status", stack.HtInstanceStatus)

	d.Set("profiles_user_id", stack.HpInstanceId)
	d.Set("profiles_name", stack.HpInstanceName)
	d.Set("profiles_url", stack.HpInstanceUrl)
	d.Set("profiles_status", stack.HpInstanceStatus)

	d.Set("graphite_user_id", stack.HmInstanceGraphiteId)
	d.Set("graphite_name", stack.HmInstanceGraphiteName)
	d.Set("graphite_url", stack.HmInstanceGraphiteUrl)
	d.Set("graphite_status", stack.HmInstanceGraphiteStatus)

	d.Set("fleet_management_user_id", stack.AgentManagementInstanceId)
	d.Set("fleet_management_name", stack.AgentManagementInstanceName)
	d.Set("fleet_management_url", stack.AgentManagementInstanceUrl)
	d.Set("fleet_management_status", stack.AgentManagementInstanceStatus)

	if otlpURL := connections.OtlpHttpUrl; otlpURL.IsSet() {
		d.Set("otlp_url", otlpURL.Get())
	}

	if influxURL := connections.InfluxUrl; influxURL.IsSet() {
		d.Set("influx_url", influxURL.Get())
	}

	return nil
}

// Append path to baseurl
func appendPath(baseURL, path string) (string, error) {
	bu, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	u, err := bu.Parse(path)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// waitForStackReadiness retries until the stack is ready, verified by querying the Grafana URL
func waitForStackReadiness(ctx context.Context, timeout time.Duration, stackURL string) diag.Diagnostics {
	healthURL, joinErr := url.JoinPath(stackURL, "api", "health")
	if joinErr != nil {
		return diag.FromErr(joinErr)
	}
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		// Query the instance URL directly. This makes the stack wake-up if it has been paused.
		// The health endpoint is helpful to check that the stack is ready, but it doesn't wake up the stack.
		stackReq, err := http.NewRequestWithContext(ctx, http.MethodGet, stackURL, nil)
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
		return diag.Errorf("error waiting for stack (URL: %s) to be ready: %v", healthURL, err)
	}

	return nil
}

func waitForStackReadinessFromSlug(ctx context.Context, timeout time.Duration, slug string, client *gcom.APIClient) diag.Diagnostics {
	stack, _, err := client.InstancesAPI.GetInstance(ctx, slug).Execute()
	if err != nil {
		return apiError(err)
	}

	return waitForStackReadiness(ctx, timeout, stack.Url)
}

func defaultStackURL(slug string) string {
	return fmt.Sprintf("https://%s.grafana.net", slug)
}
