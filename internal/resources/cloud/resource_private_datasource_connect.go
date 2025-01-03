package cloud

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

var (
	pdcNetworkID = common.NewResourceID(
		common.StringIDField("region"),
		common.StringIDField("policyId"),
	)
)

func resourcePDCNetwork() *common.Resource {
	schema := &schema.Resource{
		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/connect-externally-hosted/private-data-source-connect/)
* [API documentation](https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#create-an-access-policy)

Required access policy scopes:

* accesspolicies:read
* accesspolicies:write
* accesspolicies:delete
`,

		CreateContext: withClient[schema.CreateContextFunc](createPDCNetwork),
		UpdateContext: withClient[schema.UpdateContextFunc](updatePDCNetwork),
		DeleteContext: withClient[schema.DeleteContextFunc](deletePDCNetwork),
		ReadContext:   withClient[schema.ReadContextFunc](readPDCNetwork),

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"region": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				Description:  "Region where the API is deployed. Generally where the stack is deployed. Use the region list API to get the list of available regions: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-regions.",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the PDC network.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name of the PDC network. Defaults to the name.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" && old == d.Get("name").(string) {
						return true
					}
					return false
				},
			},
			"identifier": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the stack.",
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
		"grafana_cloud_private_datasource_connect",
		pdcNetworkID,
		schema,
	).
		WithLister(cloudListerFunction(listPDCNetworkIds)).
		WithResourceLister(cloudResourceListerFunction(listPDCNetworks)).
		WithPreferredResourceNameField("name")
}

func listPDCNetworkIds(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]string, error) {
	regionsReq := client.StackRegionsAPI.GetStackRegions(ctx)
	regionsResp, _, err := regionsReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	orgID, err := data.OrgID(ctx, client)
	if err != nil {
		return nil, err
	}

	var policies []string
	for _, region := range regionsResp.Items {
		regionSlug := region.FormattedApiStackRegionAnyOf.Slug
		req := client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(regionSlug).OrgId(orgID)
		resp, _, err := req.Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list access policies in region %s: %w", regionSlug, err)
		}

		for _, policy := range resp.Items {
			if slices.Contains(policy.Scopes, "pdc-signing:write") {
				policies = append(policies, resourceAccessPolicyID.Make(regionSlug, policy.Id))
			}
		}
	}

	return policies, nil
}

func listPDCNetworks(ctx context.Context, client *gcom.APIClient, data *ListerData) ([]any, error) {
	regionsReq := client.StackRegionsAPI.GetStackRegions(ctx)
	regionsResp, _, err := regionsReq.Execute()
	if err != nil {
		return nil, fmt.Errorf("failed to list regions: %w", err)
	}

	orgID, err := data.OrgID(ctx, client)
	if err != nil {
		return nil, err
	}

	var policies []any
	for _, region := range regionsResp.Items {
		regionSlug := region.FormattedApiStackRegionAnyOf.Slug
		req := client.AccesspoliciesAPI.GetAccessPolicies(ctx).Region(regionSlug).OrgId(orgID)
		resp, _, err := req.Execute()
		if err != nil {
			return nil, fmt.Errorf("failed to list access policies in region %s: %w", regionSlug, err)
		}

		for _, policy := range resp.Items {
			if slices.Contains(policy.Scopes, "pdc-signing:write") {
				policies = append(policies, policy)
			}
		}
	}

	return policies, nil
}

func createPDCNetwork(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	region := d.Get("region").(string)

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	req := client.AccesspoliciesAPI.PostAccessPolicies(ctx).Region(region).XRequestId(ClientRequestID()).
		PostAccessPoliciesRequest(gcom.PostAccessPoliciesRequest{
			Name:        d.Get("name").(string),
			DisplayName: &displayName,
			Scopes:      []string{"pdc-signing:write"},
			Realms:      []gcom.PostAccessPoliciesRequestRealmsInner{{Type: "stack", Identifier: d.Get("identifier").(string)}},
		})
	result, _, err := req.Execute()
	if err != nil {
		return apiError(err)
	}

	d.SetId(resourceAccessPolicyID.Make(region, result.Id))

	return readPDCNetwork(ctx, d, client)
}

func updatePDCNetwork(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
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
			Realms:      []gcom.PostAccessPoliciesRequestRealmsInner{{Type: "stack", Identifier: d.Get("identifier").(string)}},
		})
	if _, _, err = req.Execute(); err != nil {
		return apiError(err)
	}

	return readPDCNetwork(ctx, d, client)
}

func readPDCNetwork(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	result, _, err := client.AccesspoliciesAPI.GetAccessPolicy(ctx, id.(string)).Region(region.(string)).Execute()
	if err, shouldReturn := common.CheckReadError("access policy", d, err); shouldReturn {
		return err
	}

	d.Set("region", region)
	d.Set("policy_id", result.Id)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
	d.Set("scopes", result.Scopes)
	d.Set("realm", flattenCloudAccessPolicyRealm(result.Realms))
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	if updated := result.UpdatedAt; updated != nil {
		d.Set("updated_at", updated.Format(time.RFC3339))
	}
	d.SetId(resourceAccessPolicyID.Make(region, result.Id))

	return nil
}

func deletePDCNetwork(ctx context.Context, d *schema.ResourceData, client *gcom.APIClient) diag.Diagnostics {
	split, err := resourceAccessPolicyID.Split(d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	region, id := split[0], split[1]

	_, _, err = client.AccesspoliciesAPI.DeleteAccessPolicy(ctx, id.(string)).Region(region.(string)).XRequestId(ClientRequestID()).Execute()
	return apiError(err)
}
