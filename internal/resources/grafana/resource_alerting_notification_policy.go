package grafana

import (
	"context"

	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/grafana/terraform-provider-grafana/internal/common"
)

func ResourceNotificationPolicy() *schema.Resource {
	return &schema.Resource{
		Description: `
Sets the global notification policy for Grafana.

!> This resource manages the entire notification policy tree, and will overwrite any existing policies.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/manage-notifications/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/alerting_provisioning/)

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
				Required:    true,
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
				Description: "A list of mute timing names to apply to alerts that match this policy.",
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

func readNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta) // TODO: Support org-scoped policies

	resp, err := client.Provisioning.GetPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}

	packNotifPolicy(resp.Payload, data)
	data.SetId(PolicySingletonID)
	return nil
}

func putNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta) // TODO: Support org-scoped policies

	npt, err := unpackNotifPolicy(data)
	if err != nil {
		return diag.FromErr(err)
	}

	params := provisioning.NewPutPolicyTreeParams().WithBody(npt)
	if _, err := client.Provisioning.PutPolicyTree(params); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(PolicySingletonID)
	return readNotificationPolicy(ctx, data, meta)
}

func deleteNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := OAPIGlobalClient(meta) // TODO: Support org-scoped policies

	if _, err := client.Provisioning.ResetPolicyTree(); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func packNotifPolicy(npt *models.Route, data *schema.ResourceData) {
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

	if p.ObjectMatchers != nil && len(p.ObjectMatchers) > 0 {
		matchers := make([]interface{}, 0, len(p.ObjectMatchers))
		for _, m := range p.ObjectMatchers {
			matchers = append(matchers, packPolicyMatcher(m))
		}
		result["matcher"] = matchers
	}
	if p.MuteTimeIntervals != nil && len(p.MuteTimeIntervals) > 0 {
		result["mute_timings"] = p.MuteTimeIntervals
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
