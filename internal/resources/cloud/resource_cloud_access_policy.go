package cloud

import (
	"context"
	"fmt"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAccessPolicy() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#create-an-access-policy)
`,

		CreateContext: CreateCloudAccessPolicy,
		UpdateContext: UpdateCloudAccessPolicy,
		DeleteContext: DeleteCloudAccessPolicy,
		ReadContext:   ReadCloudAccessPolicy,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region where the API is deployed. Generally where the stack is deployed. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/reference/cloud-api/#list-regions.",
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
				Description: "Scopes of the access policy. See https://grafana.com/docs/grafana-cloud/account-management/authentication-and-permissions/access-policies/#scopes for possible values.",
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
}

var cloudAccessPolicyRealmSchema = &schema.Resource{
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

func CreateCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	region := d.Get("region").(string)

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	result, err := client.CreateCloudAccessPolicy(region, gapi.CreateCloudAccessPolicyInput{
		Name:        d.Get("name").(string),
		DisplayName: displayName,
		Scopes:      common.ListToStringSlice(d.Get("scopes").(*schema.Set).List()),
		Realms:      expandCloudAccessPolicyRealm(d.Get("realm").(*schema.Set).List()),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", region, result.ID))

	return ReadCloudAccessPolicy(ctx, d, meta)
}

func UpdateCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	region, id, _ := strings.Cut(d.Id(), "/")

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	_, err := client.UpdateCloudAccessPolicy(region, id, gapi.UpdateCloudAccessPolicyInput{
		DisplayName: displayName,
		Scopes:      common.ListToStringSlice(d.Get("scopes").(*schema.Set).List()),
		Realms:      expandCloudAccessPolicyRealm(d.Get("realm").(*schema.Set).List()),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return ReadCloudAccessPolicy(ctx, d, meta)
}

func ReadCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI

	region, id, _ := strings.Cut(d.Id(), "/")
	result, err := client.CloudAccessPolicyByID(region, id)
	if err, shouldReturn := common.CheckReadError("access policy", d, err); shouldReturn {
		return err
	}

	d.Set("region", region)
	d.Set("policy_id", result.ID)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
	d.Set("scopes", result.Scopes)
	d.Set("realm", flattenCloudAccessPolicyRealm(result.Realms))
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	d.Set("updated_at", result.UpdatedAt.Format(time.RFC3339))

	return nil
}

func DeleteCloudAccessPolicy(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*common.Client).GrafanaCloudAPI
	region, id, _ := strings.Cut(d.Id(), "/")
	return diag.FromErr(client.DeleteCloudAccessPolicy(region, id))
}

func validateCloudAccessPolicyScope(v interface{}, path cty.Path) diag.Diagnostics {
	if strings.Count(v.(string), ":") != 1 {
		return diag.Errorf("invalid scope: %s. Should be in the `service:permission` format", v.(string))
	}

	return nil
}

func flattenCloudAccessPolicyRealm(realm []gapi.CloudAccessPolicyRealm) []interface{} {
	var result []interface{}

	for _, r := range realm {
		labelPolicy := []interface{}{}
		for _, lp := range r.LabelPolicies {
			labelPolicy = append(labelPolicy, map[string]interface{}{
				"selector": lp.Selector,
			})
		}

		result = append(result, map[string]interface{}{
			"type":         r.Type,
			"identifier":   r.Identifier,
			"label_policy": labelPolicy,
		})
	}
	return result
}

func expandCloudAccessPolicyRealm(realm []interface{}) []gapi.CloudAccessPolicyRealm {
	var result []gapi.CloudAccessPolicyRealm

	for _, r := range realm {
		r := r.(map[string]interface{})
		labelPolicy := []gapi.CloudAccessPolicyLabelPolicy{}
		for _, lp := range r["label_policy"].(*schema.Set).List() {
			lp := lp.(map[string]interface{})
			labelPolicy = append(labelPolicy, gapi.CloudAccessPolicyLabelPolicy{
				Selector: lp["selector"].(string),
			})
		}

		result = append(result, gapi.CloudAccessPolicyRealm{
			Type:          r["type"].(string),
			Identifier:    r["identifier"].(string),
			LabelPolicies: labelPolicy,
		})
	}
	return result
}
