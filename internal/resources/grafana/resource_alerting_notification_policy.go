package grafana

import (
	"context"
	"strconv"
	"time"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/retry"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
)

func resourceNotificationPolicy() *common.Resource {
	schema := &schema.Resource{
		Description: `
Sets the global notification policy for Grafana.

!> This resource manages the entire notification policy tree and overwrites its policies. However, it does not overwrite internal policies created when alert rules directly set a contact point for notifications.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/set-up/provision-alerting-resources/terraform-provisioning/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/#notification-policies)

This resource requires Grafana 9.1.0 or later.
`,

		CreateContext: common.WithAlertingMutex[schema.CreateContextFunc](putNotificationPolicy),
		ReadContext:   readNotificationPolicy,
		UpdateContext: common.WithAlertingMutex[schema.UpdateContextFunc](putNotificationPolicy),
		DeleteContext: common.WithAlertingMutex[schema.DeleteContextFunc](deleteNotificationPolicy),
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		SchemaVersion: 0,
		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"disable_provenance": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Allow modifying the notification policy from other sources than Terraform or the Grafana API.",
			},
			"contact_point": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The default contact point to route all unmatched notifications to.",
			},
			"group_by": {
				Type:        schema.TypeList,
				Required:    true,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping.",
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.StringIsNotEmpty,
				},
			},
			"group_wait": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Time to wait to buffer alerts of the same group before sending a notification. Default is 30 seconds.",
			},
			"group_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval between two notifications for the same group. Default is 5 minutes.",
			},
			"repeat_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval for re-sending a notification if an alert is still firing. Default is 4 hours.",
			},

			"policy": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Routing rules for specific label sets.",
				Elem:        policySchema(supportedPolicyTreeDepth),
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryAlerting,
		"grafana_notification_policy",
		orgResourceIDString("anyString"),
		schema,
	).WithLister(listerFunctionOrgResource(listNotificationPolicies))
}

// The maximum depth of policy tree that the provider supports, as Terraform does not allow for infinitely recursive schemas.
// This can be increased without breaking backwards compatibility.
const supportedPolicyTreeDepth = 4

const PolicySingletonID = "policy"

// policySchema recursively builds a resource schema for the policy resource. Each policy contains a list of policies.
// Since Terraform does not support infinitely recursive schemas, we instead define the resource to a finite depth.
func policySchema(depth uint) *schema.Resource {
	if depth == 0 {
		panic("there is no valid Terraform schema for a policy tree with depth 0")
	}

	resource := &schema.Resource{
		Schema: map[string]*schema.Schema{
			"contact_point": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The contact point to route notifications that match this rule to.",
			},
			"group_by": {
				Type:        schema.TypeList,
				Required:    depth == 1,
				Optional:    depth > 1,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping. Required for root policy only. If empty, the parent grouping is used.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"matcher": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "Describes which labels this rule should match. When multiple matchers are supplied, an alert must match ALL matchers to be accepted by this policy. When no matchers are supplied, the rule will match all alert instances.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"label": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the label to match against.",
						},
						"match": {
							Type:         schema.TypeString,
							Required:     true,
							Description:  "The operator to apply when matching values of the given label. Allowed operators are `=` for equality, `!=` for negated equality, `=~` for regex equality, and `!~` for negated regex equality.",
							ValidateFunc: validation.StringInSlice([]string{"=", "!=", "=~", "!~"}, false),
						},
						"value": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The label value to match against.",
						},
					},
				},
			},
			"mute_timings": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of time intervals to apply to alerts that match this policy to mute them for the specified time.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"active_timings": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "A list of time interval names to apply to alerts that match this policy to suppress them unless they are sent at the specified time. Supported in Grafana 12.1.0 and later",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"continue": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Whether to continue matching subsequent rules if an alert matches the current rule. Otherwise, the rule will be 'consumed' by the first policy to match it.",
			},
			"group_wait": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Time to wait to buffer alerts of the same group before sending a notification. Default is 30 seconds.",
			},
			"group_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval between two notifications for the same group. Default is 5 minutes.",
			},
			"repeat_interval": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Minimum time interval for re-sending a notification if an alert is still firing. Default is 4 hours.",
			},
		},
	}

	if depth > 1 {
		resource.Schema["policy"] = &schema.Schema{
			Type:        schema.TypeList,
			Optional:    true,
			Description: "Routing rules for specific label sets.",
			Elem:        policySchema(depth - 1),
		}
	}

	return resource
}

func listNotificationPolicies(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
	var ids []string
	// Retry if the API returns 500 because it may be that the alertmanager is not ready in the org yet.
	// The alertmanager is provisioned asynchronously when the org is created.
	if err := retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, err := client.Provisioning.GetPolicyTree()
		if err != nil {
			if orgID > 1 && (err.(*runtime.APIError).IsCode(500) || err.(*runtime.APIError).IsCode(403)) {
				return retry.RetryableError(err)
			}
			return retry.NonRetryableError(err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	ids = append(ids, MakeOrgResourceID(orgID, PolicySingletonID))

	return ids, nil
}

func readNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID, _ := OAPIClientFromExistingOrgResource(meta, data.Id())

	resp, err := client.Provisioning.GetPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}

	packNotifPolicy(resp.Payload, data)
	data.SetId(MakeOrgResourceID(orgID, PolicySingletonID))
	data.Set("org_id", strconv.FormatInt(orgID, 10))
	return nil
}

func putNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, orgID := OAPIClientFromNewOrgResource(meta, data)

	npt, err := unpackNotifPolicy(data)
	if err != nil {
		return diag.FromErr(err)
	}

	putParams := provisioning.NewPutPolicyTreeParams().WithBody(npt)
	if data.Get("disable_provenance").(bool) {
		putParams.SetXDisableProvenance(&provenanceDisabled)
	}

	err = retry.RetryContext(ctx, 2*time.Minute, func() *retry.RetryError {
		_, err := client.Provisioning.PutPolicyTree(putParams)
		if orgID > 1 && err != nil {
			if apiError, ok := err.(*runtime.APIError); ok && (apiError.IsCode(500) || apiError.IsCode(404)) {
				return retry.RetryableError(err)
			}
		}
		if err != nil {
			return retry.NonRetryableError(err)
		}
		return nil
	})

	if err != nil {
		return diag.FromErr(err)
	}

	data.SetId(MakeOrgResourceID(orgID, PolicySingletonID))
	return readNotificationPolicy(ctx, data, meta)
}

func deleteNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, _, _ := OAPIClientFromExistingOrgResource(meta, data.Id())

	if _, err := client.Provisioning.ResetPolicyTree(); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func packNotifPolicy(npt *models.Route, data *schema.ResourceData) {
	data.Set("disable_provenance", npt.Provenance == "")
	data.Set("contact_point", npt.Receiver)
	data.Set("group_by", npt.GroupBy)
	data.Set("group_wait", npt.GroupWait)
	data.Set("group_interval", npt.GroupInterval)
	data.Set("repeat_interval", npt.RepeatInterval)

	if len(npt.Routes) > 0 {
		policies := make([]interface{}, 0, len(npt.Routes))
		for _, r := range npt.Routes {
			policies = append(policies, packSpecificPolicy(r, supportedPolicyTreeDepth))
		}
		data.Set("policy", policies)
	}
}

func packSpecificPolicy(p *models.Route, depth uint) interface{} {
	result := map[string]interface{}{
		"contact_point": p.Receiver,
		"continue":      p.Continue,
	}
	if len(p.GroupBy) > 0 {
		result["group_by"] = p.GroupBy
	}

	if len(p.ObjectMatchers) > 0 {
		matchers := make([]interface{}, 0, len(p.ObjectMatchers))
		for _, m := range p.ObjectMatchers {
			matchers = append(matchers, packPolicyMatcher(m))
		}
		result["matcher"] = matchers
	}
	if len(p.MuteTimeIntervals) > 0 {
		result["mute_timings"] = p.MuteTimeIntervals
	}
	if len(p.ActiveTimeIntervals) > 0 {
		result["active_timings"] = p.ActiveTimeIntervals
	}
	if p.GroupWait != "" {
		result["group_wait"] = p.GroupWait
	}
	if p.GroupInterval != "" {
		result["group_interval"] = p.GroupInterval
	}
	if p.RepeatInterval != "" {
		result["repeat_interval"] = p.RepeatInterval
	}
	if depth > 1 && p.Routes != nil && len(p.Routes) > 0 {
		policies := make([]interface{}, 0, len(p.Routes))
		for _, r := range p.Routes {
			policies = append(policies, packSpecificPolicy(r, depth-1))
		}
		result["policy"] = policies
	}
	return result
}

func packPolicyMatcher(m models.ObjectMatcher) interface{} {
	return map[string]interface{}{
		"label": m[0],
		"match": m[1],
		"value": m[2],
	}
}

func unpackNotifPolicy(data *schema.ResourceData) (*models.Route, error) {
	groupBy := data.Get("group_by").([]interface{})
	groups := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groups = append(groups, g.(string))
	}

	var children []*models.Route
	nested, ok := data.GetOk("policy")
	if ok {
		routes := nested.([]interface{})
		for _, r := range routes {
			unpacked, err := unpackSpecificPolicy(r)
			if err != nil {
				return nil, err
			}
			children = append(children, unpacked)
		}
	}

	return &models.Route{
		Receiver:       data.Get("contact_point").(string),
		GroupBy:        groups,
		GroupWait:      data.Get("group_wait").(string),
		GroupInterval:  data.Get("group_interval").(string),
		RepeatInterval: data.Get("repeat_interval").(string),
		Routes:         children,
	}, nil
}

func unpackSpecificPolicy(p interface{}) (*models.Route, error) {
	json := p.(map[string]interface{})

	var groupBy []string
	if g, ok := json["group_by"]; ok {
		groupBy = common.ListToStringSlice(g.([]interface{}))
	}

	policy := models.Route{
		Receiver: json["contact_point"].(string),
		GroupBy:  groupBy,
		Continue: json["continue"].(bool),
	}

	if v, ok := json["matcher"]; ok && v != nil {
		ms := v.(*schema.Set).List()
		matchers := make(models.ObjectMatchers, 0, len(ms))
		for _, m := range ms {
			matchers = append(matchers, unpackPolicyMatcher(m))
		}
		policy.ObjectMatchers = matchers
	}
	if v, ok := json["mute_timings"]; ok && v != nil {
		policy.MuteTimeIntervals = common.ListToStringSlice(v.([]interface{}))
	}
	if v, ok := json["active_timings"]; ok && v != nil {
		policy.ActiveTimeIntervals = common.ListToStringSlice(v.([]interface{}))
	}
	if v, ok := json["continue"]; ok && v != nil {
		policy.Continue = v.(bool)
	}
	if v, ok := json["group_wait"]; ok && v != nil {
		policy.GroupWait = v.(string)
	}
	if v, ok := json["group_interval"]; ok && v != nil {
		policy.GroupInterval = v.(string)
	}
	if v, ok := json["repeat_interval"]; ok && v != nil {
		policy.RepeatInterval = v.(string)
	}
	if v, ok := json["policy"]; ok && v != nil {
		ps := v.([]interface{})
		policies := make([]*models.Route, 0, len(ps))
		for _, p := range ps {
			unpacked, err := unpackSpecificPolicy(p)
			if err != nil {
				return nil, err
			}
			policies = append(policies, unpacked)
		}
		policy.Routes = policies
	}

	return &policy, nil
}

func unpackPolicyMatcher(m interface{}) models.ObjectMatcher {
	json := m.(map[string]interface{})
	return models.ObjectMatcher{json["label"].(string), json["match"].(string), json["value"].(string)}
}
