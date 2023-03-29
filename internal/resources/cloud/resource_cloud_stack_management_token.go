package cloud

import (
	"context"
	"log"
	"strconv"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceStackManagementToken() *schema.Resource {
	return &schema.Resource{

		Description: `
**Note:** This resource is available only with Grafana 9.1+.

Manages service accounts and the associated service account tokens of a Grafana Cloud stack using the Cloud API
This can be used to bootstrap a management service account token for a new stack

* [Official documentation](https://grafana.com/docs/grafana/latest/administration/service-accounts/)
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/serviceaccount/#service-account-api)`,

		CreateContext: createStackManagementToken,
		ReadContext:   readStackManagementToken,
		UpdateContext: updateStackManagementToken,
		DeleteContext: deleteStackManagementToken,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},
		Schema: map[string]*schema.Schema{
			"stack_slug": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the associated service account and service account token.",
			},
			"role": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
				Description:  "The basic role of the service account in the organization.",
			},
			"is_disabled": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "The disabled status for the service account.",
			},
			"seconds_to_live": {
				Type:        schema.TypeInt,
				Optional:    true,
				ForceNew:    true,
				Description: "Duration for which the service account token will be valid.",
			},
			"token": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "Service account token value.",
			},
			"expiration": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Time when service account token expires.",
			},
			"has_expired": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "True if service account token has expired.",
			},
		},
	}
}

func createStackManagementToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, cleanup, err := getClientForTokenManagement(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	isDisabled := d.Get("is_disabled").(bool)
	req := gapi.CreateServiceAccountRequest{
		Name:       d.Get("name").(string),
		Role:       d.Get("role").(string),
		IsDisabled: &isDisabled,
	}
	sa, err := client.CreateServiceAccount(req)
	if err != nil {
		return diag.FromErr(err)
	}

	tokenReq := gapi.CreateServiceAccountTokenRequest{
		Name:             d.Get("name").(string),
		ServiceAccountID: sa.ID,
		SecondsToLive:    int64(d.Get("seconds_to_live").(int)),
	}
	response, err := client.CreateServiceAccountToken(tokenReq)
	if err != nil {
		return diag.FromErr(err)
	}
	err = d.Set("token", response.Key)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(sa.ID, 10))
	return readStackManagementToken(ctx, d, meta)
}

func readStackManagementToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, cleanup, err := getClientForTokenManagement(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	sas, err := client.GetServiceAccounts()
	if err != nil {
		return diag.FromErr(err)
	}

	for _, sa := range sas {
		if sa.ID == id {
			err = d.Set("name", sa.Name)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("role", sa.Role)
			if err != nil {
				return diag.FromErr(err)
			}
			err = d.Set("is_disabled", sa.IsDisabled)
			if err != nil {
				return diag.FromErr(err)
			}

			tokens, err := client.GetServiceAccountTokens(sa.ID)
			if err != nil {
				return diag.FromErr(err)
			}
			foundToken := false
			for _, t := range tokens {
				if t.Name == d.Get("name") {
					if t.Expiration != nil && !t.Expiration.IsZero() {
						err = d.Set("expiration", t.Expiration.String())
						if err != nil {
							return diag.FromErr(err)
						}
					}
					err = d.Set("has_expired", t.HasExpired)
					if err != nil {
						return diag.FromErr(err)
					}
					foundToken = true
					break
				}
			}
			if !foundToken {
				log.Printf("[WARN] removing service account %d from state because no tokens associated with it exist in grafana", id)
				d.SetId("")
			}
			return nil
		}
	}
	log.Printf("[WARN] removing service account %d from state because it no longer exists in grafana", id)
	d.SetId("")

	return nil
}

func updateStackManagementToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, cleanup, err := getClientForTokenManagement(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	updateRequest := gapi.UpdateServiceAccountRequest{}
	if d.HasChange("role") {
		updateRequest.Role = d.Get("role").(string)
	}
	if d.HasChange("is_disabled") {
		isDisabled := d.Get("is_disabled").(bool)
		updateRequest.IsDisabled = &isDisabled
	}

	if _, err := client.UpdateServiceAccount(id, updateRequest); err != nil {
		return diag.FromErr(err)
	}

	return readStackManagementToken(ctx, d, meta)
}

func deleteStackManagementToken(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client, cleanup, err := getClientForTokenManagement(d, meta)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	_, err = client.DeleteServiceAccount(id)
	return diag.FromErr(err)
}

func getClientForTokenManagement(d *schema.ResourceData, m interface{}) (c *gapi.Client, cleanup func() error, err error) {
	cloudClient := m.(*common.Client).GrafanaCloudAPI
	return cloudClient.CreateTemporaryStackGrafanaClient(d.Get("stack_slug").(string), "terraform-temp-sa-", 60*time.Second)
}
