package grafana

import (
	"context"
	"fmt"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceNotificationPolicy() *schema.Resource {
	return &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/notifications/)
* [HTTP API](https://grafana.com/docs/grafana/next/developers/http_api/alerting_provisioning/#notification-policies)
`,

		CreateContext: createNotificationPolicy,
		ReadContext:   readNotificationPolicy,
		UpdateContext: updateNotificationPolicy,
		DeleteContext: deleteNotificationPolicy,
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
					Type: schema.TypeString,
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
const supportedPolicyTreeDepth = 2

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
				Required:    true,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group alerts by all labels, effectively disabling grouping.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"matcher": {
				Type:        schema.TypeList,
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
							Type:        schema.TypeString,
							Required:    true,
							Description: "The operator to apply when matching values of the given label. Allowed operators are `=` for equality, `!=` for negated equality, `=~` for regex equality, and `!~` for negated regex equality.",
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
			Elem:        policySchema(supportedPolicyTreeDepth - 1),
		}
	}

	return resource
}

func readNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt, err := client.NotificationPolicyTree()
	if err != nil {
		return diag.FromErr(err)
	}

	packNotifPolicy(npt, data)
	data.SetId(PolicySingletonID)
	return nil
}

func createNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt, err := unpackNotifPolicy(data)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.SetNotificationPolicyTree(&npt); err != nil {
		return diag.FromErr(err)
	}

	data.SetId(PolicySingletonID)
	return readNotificationPolicy(ctx, data, meta)
}

func updateNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	npt, err := unpackNotifPolicy(data)
	if err != nil {
		return diag.FromErr(err)
	}

	if err := client.SetNotificationPolicyTree(&npt); err != nil {
		return diag.FromErr(err)
	}

	return readNotificationPolicy(ctx, data, meta)
}

func deleteNotificationPolicy(ctx context.Context, data *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gapi

	if err := client.ResetNotificationPolicyTree(); err != nil {
		return diag.FromErr(err)
	}
	return diag.Diagnostics{}
}

func packNotifPolicy(npt gapi.NotificationPolicyTree, data *schema.ResourceData) {
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

func packSpecificPolicy(p gapi.SpecificPolicy, depth uint) interface{} {
	result := map[string]interface{}{
		"contact_point": p.Receiver,
		"group_by":      p.GroupBy,
		"continue":      p.Continue,
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

func packPolicyMatcher(m gapi.Matcher) interface{} {
	return map[string]interface{}{
		"label": m.Name,
		"match": m.Type.String(),
		"value": m.Value,
	}
}

func unpackNotifPolicy(data *schema.ResourceData) (gapi.NotificationPolicyTree, error) {
	groupBy := data.Get("group_by").([]interface{})
	groups := make([]string, 0, len(groupBy))
	for _, g := range groupBy {
		groups = append(groups, g.(string))
	}

	var children []gapi.SpecificPolicy
	nested, ok := data.GetOk("policy")
	if ok {
		routes := nested.([]interface{})
		for _, r := range routes {
			unpacked, err := unpackSpecificPolicy(r)
			if err != nil {
				return gapi.NotificationPolicyTree{}, err
			}
			children = append(children, unpacked)
		}
	}

	return gapi.NotificationPolicyTree{
		Receiver:       data.Get("contact_point").(string),
		GroupBy:        groups,
		GroupWait:      data.Get("group_wait").(string),
		GroupInterval:  data.Get("group_interval").(string),
		RepeatInterval: data.Get("repeat_interval").(string),
		Routes:         children,
	}, nil
}

func unpackSpecificPolicy(p interface{}) (gapi.SpecificPolicy, error) {
	json := p.(map[string]interface{})
	policy := gapi.SpecificPolicy{
		Receiver: json["contact_point"].(string),
		GroupBy:  listToStringSlice(json["group_by"].([]interface{})),
		Continue: json["continue"].(bool),
	}

	if v, ok := json["matcher"]; ok && v != nil {
		ms := v.([]interface{})
		matchers := make([]gapi.Matcher, 0, len(ms))
		for _, m := range ms {
			matcher, err := unpackPolicyMatcher(m)
			if err != nil {
				return gapi.SpecificPolicy{}, err
			}
			matchers = append(matchers, matcher)
		}
		policy.ObjectMatchers = matchers
	}
	if v, ok := json["mute_timings"]; ok && v != nil {
		policy.MuteTimeIntervals = listToStringSlice(v.([]interface{}))
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
		policies := make([]gapi.SpecificPolicy, 0, len(ps))
		for _, p := range ps {
			unpacked, err := unpackSpecificPolicy(p)
			if err != nil {
				return gapi.SpecificPolicy{}, err
			}
			policies = append(policies, unpacked)
		}
		policy.Routes = policies
	}

	return policy, nil
}

func unpackPolicyMatcher(m interface{}) (gapi.Matcher, error) {
	json := m.(map[string]interface{})

	var matchType gapi.MatchType
	switch json["match"].(string) {
	case "=":
		matchType = gapi.MatchEqual
	case "!=":
		matchType = gapi.MatchNotEqual
	case "=~":
		matchType = gapi.MatchRegexp
	case "!~":
		matchType = gapi.MatchNotRegexp
	default:
		return gapi.Matcher{}, fmt.Errorf("unknown match operator: %s", json["match"].(string))
	}
	return gapi.Matcher{
		Name:  json["label"].(string),
		Type:  matchType,
		Value: json["value"].(string),
	}, nil
}
