package grafana

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceCloudAccessPolicyToken() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana-cloud/authentication-and-permissions/access-policies/)
* [API documentation](https://grafana.com/docs/grafana-cloud/reference/cloud-api/#create-a-token)
`,

		CreateContext: CreateCloudAccessPolicyToken,
		UpdateContext: UpdateCloudAccessPolicyToken,
		DeleteContext: DeleteCloudAccessPolicyToken,
		ReadContext:   ReadCloudAccessPolicyToken,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"access_policy_id": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "ID of the access policy for which to create a token.",
			},
			"region": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Region of the access policy. Should be set to the same region as the access policy.",
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "Name of the access policy token.",
			},
			"display_name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Display name of the access policy token. Defaults to the name.",
				DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
					if new == "" && old == d.Get("name").(string) {
						return true
					}
					return false
				},
			},
			"expires_at": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				Description:  "Expiration date of the access policy token. Does not expire by default.",
				ValidateFunc: validation.IsRFC3339Time,
			},

			// Computed
			"token": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"created_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Creation date of the access policy token.",
			},
			"updated_at": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Last update date of the access policy token.",
			},
		},
	}
}

func CreateCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi
	region := d.Get("region").(string)

	tokenInput := gapi.CreateCloudAccessPolicyTokenInput{
		AccessPolicyID: d.Get("access_policy_id").(string),
		Name:           d.Get("name").(string),
		DisplayName:    d.Get("display_name").(string),
	}

	if v, ok := d.GetOk("expires_at"); ok {
		expiresAt, err := time.Parse(time.RFC3339, v.(string))
		if err != nil {
			return diag.FromErr(err)
		}
		tokenInput.ExpiresAt = &expiresAt
	}

	result, err := client.CreateCloudAccessPolicyToken(region, tokenInput)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(fmt.Sprintf("%s/%s", region, result.ID))
	d.Set("token", result.Token)

	return ReadCloudAccessPolicyToken(ctx, d, meta)
}

func UpdateCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi
	region, id, _ := strings.Cut(d.Id(), "/")

	displayName := d.Get("display_name").(string)
	if displayName == "" {
		displayName = d.Get("name").(string)
	}

	_, err := client.UpdateCloudAccessPolicyToken(region, id, gapi.UpdateCloudAccessPolicyTokenInput{
		DisplayName: displayName,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	return ReadCloudAccessPolicyToken(ctx, d, meta)
}

func ReadCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi

	region, id, _ := strings.Cut(d.Id(), "/")

	result, err := client.CloudAccessPolicyTokenByID(region, id)

	if result.ID == "" || (err != nil && strings.HasPrefix(err.Error(), "status: 404")) {
		log.Printf("[WARN] removing cloud access policy token %s from state because it no longer exists", d.Get("name").(string))
		d.SetId("")
		return nil
	}

	if err != nil {
		return diag.FromErr(err)
	}

	d.Set("access_policy_id", result.AccessPolicyID)
	d.Set("region", region)
	d.Set("name", result.Name)
	d.Set("display_name", result.DisplayName)
	d.Set("created_at", result.CreatedAt.Format(time.RFC3339))
	if result.ExpiresAt != nil {
		d.Set("expires_at", result.ExpiresAt.Format(time.RFC3339))
	}
	if result.UpdatedAt != nil {
		d.Set("updated_at", result.UpdatedAt.Format(time.RFC3339))
	}

	return nil
}

func DeleteCloudAccessPolicyToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*client).gcloudapi
	region, id, _ := strings.Cut(d.Id(), "/")

	return diag.FromErr(client.DeleteCloudAccessPolicyToken(region, id))
}
