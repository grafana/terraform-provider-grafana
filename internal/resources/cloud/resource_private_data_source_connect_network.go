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
				Description:  "The region where your stack is deployed. Use the instances list API to get the region for your instance - use the regionSlug property: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#list-stacks",
				ValidateFunc: validation.StringIsNotEmpty,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the PDC network." + "**Note:** The name must be lowercase and can contain hyphens or underscores. See full requirements here: https://grafana.com/docs/grafana-cloud/developer-resources/api-reference/cloud-api/#request-body",
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
			"stack_identifier": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The identifier of the stack.",
			},

			// Computed
			"pdc_network_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "ID of the private data source connect network.",
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation date of the private data source connect network.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update date of the private data source connect network.",
			},
		},
	}

	return common.NewLegacySDKResource(
		common.CategoryCloud,
		"grafana_cloud_private_data_source_connect_network",
		pdcNetworkID,
		schema,
	).
		WithLister(cloudListerFunction(listPDCNetworkIds)).
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
			Realms:      []gcom.PostAccessPoliciesRequestRealmsInner{{Type: "stack", Identifier: d.Get("stack_identifier").(string)}},
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
			Realms:      []gcom.PostAccessPoliciesRequestRealmsInner{{Type: "stack", Identifier: d.Get("stack_identifier").(string)}},
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
	d.Set("pdc_network_id", result.Id)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
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
