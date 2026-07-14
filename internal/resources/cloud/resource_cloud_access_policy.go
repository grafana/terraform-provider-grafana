package cloud

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	resourceAccessPolicyID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("policyId"),
	)
)

func resourceAccessPolicy() *common.Resource {
	cloudAccessPolicyConditionSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"allowed_subnets": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Conditions that apply to the access policy,such as IP Allow lists.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateCloudAccessPolicyAllowedSubnets,
				},
			},
		},
	}
	cloudAccessPolicyRealmSchema := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "Whether a policy applies to a Cloud org or a specific stack. Should be one of `org` or `stack`.",
				ValidateFunc: validation.StringInSlice([]string{"org", "stack"}, false),
			},
			"identifier": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the org or stack. For orgs, this is the slug, for stacks, this is the stack ID.",
			},
			"label_policy": {
				Type:     schema.TypeSet,
				Optional: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"selector": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label selector to match in metrics or logs query. Should be in PromQL or LogQL format.",
						},
					},
				},
			},
		},
	}

	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-an-access-policy)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete
`,

		CreateContext: withClient[schema.CreateContextFunc](createCloudAccessPolicy),
		UpdateContext: withClient[schema.UpdateContextFunc](updateCloudAccessPolicy),
		DeleteContext: withClient[schema.DeleteContextFunc](deleteCloudAccessPolicy),
		ReadContext:   withClient[schema.ReadContextFunc](readCloudAccessPolicy),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region where the API is deployed. Generally where the stack is deployed. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the access policy.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name of the access policy. Defaults to the name.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" && old == d.Get("name").(string) {
						return true
					}
					return false
				},
			},
			"scopes": {
				Type:        schema.TypeSet,
				Required:    true,
				Description: "Scopes of the access policy. See https://grafana.com/docs/grafana-cloud/security-and-account-management/authentication-and-permissions/access-policies/#scopes for possible values.",
				Elem: &schema.Schema{
					Type:             schema.TypeString,
					ValidateDiagFunc: validateCloudAccessPolicyScope,
				},
			},
			"realm": {
				Type:     schema.TypeSet,
				Required: true,
				Elem:     cloudAccessPolicyRealmSchema,
			},
			"conditions": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Conditions for the access policy.",
				Elem:        cloudAccessPolicyConditionSchema,
			},

			// Computed
			"policy_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the access policy.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation date of the access policy.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update date of the access policy.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_access_policy",
		resourceAccessPolicyID,
		schema,
	).
		WithLister(cloudListerFunction(listAccessPolicies)).
		WithPreferredResourceNameField("name")
}

func listAccessPolicies(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	var regionsResp *gcom.GetStackRegions200Response
	if err := common.RetryRequest(ctx, "list stack regions", func() (*http.Response, error) {
		r, httpResp, err := client.StackRegionsAPI.GetStackRegions(ctx).Execute()
		regionsResp = r
		return httpResp, err
	}); err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	orgID, err := data.OrgID(ctx, client)
	if err != nil {
		return nil, err
	}

	var policies []string
	for _, region := range regionsResp.Items {
		regionSlug := region.FormattedApiStackRegionAnyOf.Slug
		var resp *gcom.GetAccessPolicies200Response
		if err := common.RetryRequest(ctx, "list access policies", func() (*http.Response, error) {
			r, httpResp, err := client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(regionSlug).OrgId(orgID).Execute()
			resp = r
			return httpResp, err
		}); err != nil {
			return nil, fmt.Errorf("failed to list access policies in region %s: %w", regionSlug, err)
		}

		for _, policy := range resp.Items {
			policies = append(policies, resourceAccessPolicyID.Make(regionSlug, policy.Id))
		}
	}

	return policies, nil
}

func createCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	region := d.Get("region").(string)
	name := d.Get("name").(string)

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = name
	}

	req := client.AccesspoliciesAPI.PostAccessPolicies(ctx).Region(region).XRequestId(ClientRequestID()).
		PostAccessPoliciesRequest(gcom.PostAccessPoliciesRequest{
			Name:        name,
			DisplayName: &displayName,
			Scopes:      common.ListToStringSlice(d.Get("scopes").(*schema.Set).List()),
			Realms:      expandCloudAccessPolicyRealm(d.Get("realm").(*schema.Set).List()),
			Conditions:  expandCloudAccessPolicyConditions(d.Get("conditions").(*schema.Set).List()),
		})

	// Retry transient server and rate-limit failures; 400/409 responses are client/conflict errors and return immediately.
	var result *gcom.AuthAccessPolicy
	attempt := 0
	cfg := common.DefaultHTTPRequestRetryConfig()
	cfg.Operation = "create cloud access policy"
	// Make create retries idempotent: a previous attempt may have created the policy
	// server-side even though we never saw the response (a transient 5xx or a dropped
	// connection), so blindly re-issuing the POST could create a duplicate or fail with a
	// conflict. From the second attempt onward, look the policy up by name+region and, if it
	// already exists, adopt it instead of creating again. The first attempt is left
	// untouched so a genuine pre-existing name conflict still surfaces to the user.
	cfg.ErrorAnalyzer = func(resp *http.Response, err error) error {
		if err == nil || attempt <= 1 {
			return err
		}
		existing, lookupErr := findAccessPolicyByName(ctx, client, region, name)
		if lookupErr != nil {
			log.Printf("[WARN] Grafana Cloud API: could not check whether access policy %q already exists before retrying create in region %s: %v", name, region, lookupErr)
			return err
		}
		if existing != nil {
			log.Printf("[INFO] Grafana Cloud API: access policy %q already exists in region %s after a failed create attempt; adopting it instead of retrying", name, region)
			result = existing
			return nil
		}
		return err
	}
	if err := common.RetryHTTPRequest(ctx, cfg, func() (*http.Response, error) {
		attempt++
		r, httpResp, err := req.Execute()
		if r != nil {
			result = r
		}
		return httpResp, err
	}); err != nil {
		return apiError(err)
	}

	d.SetId(resourceAccessPolicyID.Make(region, result.Id))

	return readCloudAccessPolicy(ctx, d, client)
}

// findAccessPolicyByName returns the access policy with the given name in the region, or nil if none
// exists. It makes create retries idempotent: a prior attempt may have created the policy even when its
// HTTP response was lost. The name+region pair uniquely identifies a policy within the authenticated org.
func findAccessPolicyByName(ctx context.Context, client *gcom.APIClient, region, name string) (*gcom.AuthAccessPolicy, error) {
	var resp *gcom.GetAccessPolicies200Response
	if err := common.RetryRequest(ctx, "find access policy by name", func() (*http.Response, error) {
		r, httpResp, err := client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(region).Name(name).Execute()
		resp = r
		return httpResp, err
	}); err != nil {
		return nil, err
	}
	for i := range resp.Items {
		if resp.Items[i].Name == name {
			return &resp.Items[i], nil
		}
	}
	return nil, nil
}

func updateCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	req := client.AccesspoliciesAPI.PostAccessPolicy(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).
		PostAccessPolicyRequest(gcom.PostAccessPolicyRequest{
			DisplayName: &displayName,
			Scopes:      common.ListToStringSlice(d.Get("scopes").(*schema.Set).List()),
			Realms:      expandCloudAccessPolicyRealm(d.Get("realm").(*schema.Set).List()),
			Conditions:  expandCloudAccessPolicyConditions(d.Get("conditions").(*schema.Set).List()),
		})
	if err := common.RetryRequest(ctx, "update cloud access policy", func() (*http.Response, error) {
		_, httpResp, err := req.Execute()
		return httpResp, err
	}); err != nil {
		return apiError(err)
	}

	return readCloudAccessPolicy(ctx, d, client)
}

func readCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	var result *gcom.AuthAccessPolicy
	getErr := common.RetryRequest(ctx, "read cloud access policy", func() (*http.Response, error) {
		r, httpResp, err := client.AccesspoliciesAPI.GetAccessPolicy(ctx, id.(string)).Region(region.(string)).Execute()
		result = r
		return httpResp, err
	})
	if err, shouldReturn := common.CheckReadError("access policy", d, getErr); shouldReturn {
		return err
	}

	d.Set("region", region)
	d.Set("policy_id", result.Id)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
	d.Set("scopes", result.Scopes)
	d.Set("realm", flattenCloudAccessPolicyRealm(result.Realms))
	d.Set("conditions", flattenCloudAccessPolicyConditions(result.Conditions))
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	if updated := result.UpdatedAt; updated != nil {
		d.Set("updated_at", updated.Format(time.RFC3339))
	}
	d.SetId(resourceAccessPolicyID.Make(region, result.Id))

	return nil
}

func deleteCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	cfg := common.DefaultHTTPRequestRetryConfig()
	cfg.Operation = "delete cloud access policy"
	cfg.ErrorAnalyzer = common.AcceptNotFound
	if err := common.RetryHTTPRequest(ctx, cfg, func() (*http.Response, error) {
		return client.AccesspoliciesAPI.DeleteAccessPolicy(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	}); err != nil {
		return apiError(err)
	}
	return nil
}

func validateCloudAccessPolicyScope(v any, path cty.Path) diag.Diagnostics {
	if strings.Count(v.(string), ":") != 1 {
		return diag.Errorf("invalid scope: %s. Should be in the `service:permission` format", v.(string))
	}

	return nil
}

func validateCloudAccessPolicyAllowedSubnets(v any, path cty.Path) diag.Diagnostics {
	_, _, err := net.ParseCIDR(v.(string))
	if err == nil {
		return nil
	}
	return diag.Errorf("Invalid IP CIDR : %s.", v.(string))
}

func flattenCloudAccessPolicyRealm(realm []gcom.AuthAccessPolicyRealmsInner) []any {
	var result []any

	for _, r := range realm {
		labelPolicy := []any{}
		for _, lp := range r.LabelPolicies {
			labelPolicy = append(labelPolicy, map[string]any{
				"selector": lp.Selector,
			})
		}

		result = append(result, map[string]any{
			"type":         r.Type,
			"identifier":   r.Identifier,
			"label_policy": labelPolicy,
		})
	}

	return result
}

func flattenCloudAccessPolicyConditions(condition *gcom.AuthAccessPolicyConditions) []any {
	if condition == nil || len(condition.GetAllowedSubnets()) == 0 {
		return nil
	}
	var result []any
	var allowedSubnets []string

	for _, sn := range condition.GetAllowedSubnets() {
		allowedSubnets = append(allowedSubnets, *sn.String)
	}

	result = append(result, map[string]any{
		"allowed_subnets": allowedSubnets,
	})

	return result
}

func expandCloudAccessPolicyConditions(condition []any) gcom.NullablePostAccessPoliciesRequestConditions {
	var result gcom.PostAccessPoliciesRequestConditions

	for _, c := range condition {
		c := c.(map[string]any)
		for _, as := range c["allowed_subnets"].(*schema.Set).List() {
			result.AllowedSubnets = append(result.AllowedSubnets, as.(string))
		}
	}

	return *gcom.NewNullablePostAccessPoliciesRequestConditions(&result)
}

func expandCloudAccessPolicyRealm(realm []any) []gcom.PostAccessPoliciesRequestRealmsInner {
	var result []gcom.PostAccessPoliciesRequestRealmsInner

	for _, r := range realm {
		r := r.(map[string]any)
		labelPolicy := []gcom.PostAccessPoliciesRequestRealmsInnerLabelPoliciesInner{}
		for _, lp := range r["label_policy"].(*schema.Set).List() {
			lp := lp.(map[string]any)
			labelPolicy = append(labelPolicy, gcom.PostAccessPoliciesRequestRealmsInnerLabelPoliciesInner{
				Selector: lp["selector"].(string),
			})
		}

		result = append(result, gcom.PostAccessPoliciesRequestRealmsInner{
			Type:          r["type"].(string),
			Identifier:    r["identifier"].(string),
			LabelPolicies: labelPolicy,
		})
	}
	return result
}
