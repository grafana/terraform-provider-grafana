package cloud

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const defaultReadinessTimeout = time.Minute * 5

var stackSlugRegex = regexp.MustCompile("^[a-z][a-z0-9]+$")

func ResourceStack() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#stacks/)
`,

		CreateContext: CreateStack,
		UpdateContext: UpdateStack,
		DeleteContext: DeleteStack,
		ReadContext:   ReadStack,
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
				Description: "Name of stack. Conventionally matches the url of the instance (e.g. “<stack_slug>.grafana.net”).",
			},
			"description": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Description of stack.",
			},
			"slug": {
				Type:     schema.TypeString,
				Required: true,
				Description: `
Subdomain that the Grafana instance will be available at (i.e. setting slug to “<stack_slug>” will make the instance
available at “https://<stack_slug>.grafana.net".`,
				ValidateFunc: validation.StringMatch(stackSlugRegex, "must be a lowercase alphanumeric string and must start with a letter."),
			},
			"region_slug": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: `Region slug to assign to this stack. Changing region will destroy the existing stack and create a new one in the desired region. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/reference/cloud-api/#list-regions.`,
			},
			"url": {
				Type:        schema.TypeString,
				Computed:    true,
				Optional:    true,
				Description: "Custom URL for the Grafana instance. Must have a CNAME setup to point to `.grafana.net` before creating the stack",
			},
			"wait_for_readiness": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     true,
				Description: "Whether to wait for readiness of the stack after creating it. The check is a HEAD request to the stack URL (Grafana instance).",
				// Suppress the diff if the new value is "false" because this attribute is only used at creation-time
				// If the diff is suppress for a "true" value, the attribute cannot be read at all
				DiffSuppressFunc: func(_, _, newValue string, _ *schema.ResourceData) bool { return newValue == "false" },
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
			"org_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Organization id to assign to this stack.",
			},
			"org_slug": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization slug to assign to this stack.",
			},
			"org_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Organization name to assign to this stack.",
			},
			"status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the stack.",
			},

			// Hosted Metrics
			"prometheus_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Prometheus user ID. Used for e.g. remote_write.",
			},
			"prometheus_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus url for this instance.",
			},
			"prometheus_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus name for this instance.",
			},
			"prometheus_remote_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to query hosted metrics data e.g. Prometheus data source in Grafana",
			},
			"prometheus_remote_write_endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Use this URL to send prometheus metrics to Grafana cloud",
			},
			"prometheus_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Prometheus status for this instance.",
			},

			// Alerting
			"alertmanager_user_id": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "User ID of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Name of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base URL of the Alertmanager instance configured for this stack.",
			},
			"alertmanager_status": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Status of the Alertmanager instance configured for this stack.",
			},

			// Hosted Logs
			"logs_user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"logs_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"logs_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"logs_status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Traces
			"traces_user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"traces_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"traces_url": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Base URL of the Traces instance configured for this stack. To use this in the Tempo data source in Grafana, append `/tempo` to the URL.",
			},
			"traces_status": {
				Type:     schema.TypeString,
				Computed: true,
			},

			// Graphite
			"graphite_user_id": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"graphite_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"graphite_url": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"graphite_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func CreateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	stack := &gapi.CreateStackInput{
		Name:   d.Get("name").(string),
		Slug:   d.Get("slug").(string),
		URL:    d.Get("url").(string),
		Region: d.Get("region_slug").(string),
	}

	stackID, err := client.NewStack(stack)
	if err != nil && err.Error() == "409 Conflict" {
		return diag.Errorf("Error: A Grafana stack with the name '%s' already exists.", stack.Name)
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(stackID, 10))

	if diag := ReadStack(ctx, d, meta); diag != nil {
		return diag
	}

	return waitForStackReadiness(ctx, d)
}

func UpdateStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	stackID, _ := strconv.ParseInt(d.Id(), 10, 64)

	// The underlying API olnly allows to update the name and description.
	allowedChanges := []string{"name", "description", "slug"}
	if d.HasChangesExcept(allowedChanges...) {
		return diag.Errorf("Error: Only name, slug and description can be updated.")
	}

	if d.HasChange("name") || d.HasChange("description") || d.HasChanges("slug") {
		stack := &gapi.UpdateStackInput{
			Name:        d.Get("name").(string),
			Slug:        d.Get("slug").(string),
			Description: d.Get("description").(string),
		}
		err := client.UpdateStack(stackID, stack)
		if err != nil {
			return diag.FromErr(err)
		}
	}

	if diag := ReadStack(ctx, d, meta); diag != nil {
		return diag
	}

	return waitForStackReadiness(ctx, d)
}

func DeleteStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	if err := client.DeleteStack(d.Id()); err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func ReadStack(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	stack, err := getStackFromIDOrSlug(client, d.Id())

	if err != nil {
		if strings.Contains(err.Error(), "404") {
			log.Printf("[WARN] removing stack %s from state because it no longer exists in grafana", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	if stack.Status == "deleted" {
		log.Printf("[WARN] removing stack %s from state because it was deleted outside of Terraform", stack.Name)
		d.SetId("")
		return nil
	}

	if err := FlattenStack(d, *stack); err != nil {
		return diag.FromErr(err)
	}
	// Always set the wait attribute to true after creation
	// It no longer matters and this will prevent drift if the stack was imported
	d.Set("wait_for_readiness", true)

	return nil
}

func FlattenStack(d *schema.ResourceData, stack gapi.Stack) error {
	id := strconv.FormatInt(stack.ID, 10)
	d.SetId(id)
	d.Set("name", stack.Name)
	d.Set("slug", stack.Slug)
	d.Set("url", stack.URL)
	d.Set("status", stack.Status)
	d.Set("region_slug", stack.RegionSlug)
	d.Set("description", stack.Description)

	d.Set("org_id", stack.OrgID)
	d.Set("org_slug", stack.OrgSlug)
	d.Set("org_name", stack.OrgName)

	d.Set("prometheus_user_id", stack.HmInstancePromID)
	d.Set("prometheus_url", stack.HmInstancePromURL)
	d.Set("prometheus_name", stack.HmInstancePromName)
	reURL, err := appendPath(stack.HmInstancePromURL, "/api/prom")
	if err != nil {
		return err
	}
	d.Set("prometheus_remote_endpoint", reURL)
	rweURL, err := appendPath(stack.HmInstancePromURL, "/api/prom/push")
	if err != nil {
		return err
	}
	d.Set("prometheus_remote_write_endpoint", rweURL)
	d.Set("prometheus_status", stack.HmInstancePromStatus)

	d.Set("logs_user_id", stack.HlInstanceID)
	d.Set("logs_url", stack.HlInstanceURL)
	d.Set("logs_name", stack.HlInstanceName)
	d.Set("logs_status", stack.HlInstanceStatus)

	d.Set("alertmanager_user_id", stack.AmInstanceID)
	d.Set("alertmanager_name", stack.AmInstanceName)
	d.Set("alertmanager_url", stack.AmInstanceURL)
	d.Set("alertmanager_status", stack.AmInstanceStatus)

	d.Set("traces_user_id", stack.HtInstanceID)
	d.Set("traces_name", stack.HtInstanceName)
	d.Set("traces_url", stack.HtInstanceURL)
	d.Set("traces_status", stack.HtInstanceStatus)

	d.Set("graphite_user_id", stack.HmInstanceGraphiteID)
	d.Set("graphite_name", stack.HmInstanceGraphiteName)
	d.Set("graphite_url", stack.HmInstanceGraphiteURL)
	d.Set("graphite_status", stack.HmInstanceGraphiteStatus)

	return nil
}

func getStackFromIDOrSlug(client *gapi.Client, id string) (*gapi.Stack, error) {
	numericalID, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		// If the ID is not a number, then it may be a slug
		stack, err := client.StackBySlug(id)
		if err != nil {
			return nil, fmt.Errorf("failed to find stack by ID or slug '%s': %w", id, err)
		}
		return &stack, nil
	}

	stack, err := client.StackByID(numericalID)
	if err != nil {
		return nil, err
	}

	return &stack, nil
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
func waitForStackReadiness(ctx context.Context, d *schema.ResourceData) diag.Diagnostics {
	if wait := d.Get("wait_for_readiness").(bool); !wait {
		return nil
	}

	timeout := defaultReadinessTimeout
	if timeoutVal := d.Get("wait_for_readiness_timeout").(string); timeoutVal != "" {
		timeout, _ = time.ParseDuration(timeoutVal)
	}
	err := retry.RetryContext(ctx, timeout, func() *retry.RetryError {
		req, err := http.NewRequestWithContext(ctx, http.MethodHead, d.Get("url").(string), nil)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return retry.NonRetryableError(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			buf := new(bytes.Buffer)
			body := ""
			_, err = buf.ReadFrom(resp.Body)
			if err != nil {
				body = "unable to read response body, error: " + err.Error()
			} else {
				body = buf.String()
			}
			return retry.RetryableError(fmt.Errorf("stack was not ready in %s. Status code: %d, Body: %s", timeout, resp.StatusCode, body))
		}

		return nil
	})
	if err != nil {
		return diag.Errorf("error waiting for stack to be ready: %v", err)
	}

	return nil
}
