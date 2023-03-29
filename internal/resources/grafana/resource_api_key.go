package grafana

import (
	"context"
	"strconv"
	"time"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func ResourceAPIKey() *schema.Resource {
	return &schema.Resource{
		Description: `
Manages Grafana API Keys.

* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/auth/)
`,

		CreateContext: resourceAPIKeyCreate,
		ReadContext:   resourceAPIKeyRead,
		DeleteContext: resourceAPIKeyDelete,

		Schema: map[string]*schema.Schema{
			"org_id": orgIDAttribute(),
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
			"role": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{"Viewer", "Editor", "Admin"}, false),
			},
			"seconds_to_live": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"cloud_stack_slug": {
				Type:        schema.TypeString,
				Optional:    true,
				ForceNew:    true,
				Description: "Deprecated: Use `grafana_cloud_stack_management_token` resource instead",
				Deprecated:  "Use `grafana_cloud_stack_management_token` resource instead",
			},

			"id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"key": {
				Type:      schema.TypeString,
				Computed:  true,
				Sensitive: true,
			},
			"expiration": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func resourceAPIKeyCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	name := d.Get("name").(string)
	role := d.Get("role").(string)
	ttl := d.Get("seconds_to_live").(int)

	c, orgID, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	request := gapi.CreateAPIKeyRequest{Name: name, Role: role, SecondsToLive: int64(ttl)}
	response, err := c.CreateAPIKey(request)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(MakeOrgResourceID(orgID, response.ID))
	d.Set("key", response.Key)

	// Fill the true resource's state after a create by performing a read
	return resourceAPIKeyRead(ctx, d, m)
}

func resourceAPIKeyRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	c, _, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	response, err := c.GetAPIKeys(true)
	if err != nil {
		return diag.FromErr(err)
	}

	_, idStr := SplitOrgResourceID(d.Id())
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}
	for _, key := range response {
		if id == key.ID {
			d.Set("name", key.Name)
			d.Set("role", key.Role)

			if !key.Expiration.IsZero() {
				d.Set("expiration", key.Expiration.String())
			}

			return nil
		}
	}

	// Resource was not found via the client. Have Terraform destroy it.
	d.SetId("")

	return nil
}

func resourceAPIKeyDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	_, idStr := SplitOrgResourceID(d.Id())
	id, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		return diag.FromErr(err)
	}

	c, _, cleanup, err := getClientForAPIKeyManagement(d, m)
	if err != nil {
		return diag.FromErr(err)
	}
	defer cleanup()

	_, err = c.DeleteAPIKey(id)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func getClientForAPIKeyManagement(d *schema.ResourceData, m interface{}) (c *gapi.Client, orgID int64, cleanup func() error, err error) {
	// TODO: Remove this client management once `cloud_stack_slug` is removed
	if cloudStackSlug, ok := d.GetOk("cloud_stack_slug"); ok && cloudStackSlug.(string) != "" {
		cloudClient := m.(*common.Client).GrafanaCloudAPI
		c, cleanup, err = cloudClient.CreateTemporaryStackGrafanaClient(cloudStackSlug.(string), "terraform-temp-", 60*time.Second)
		return
	}

	cleanup = func() error { return nil }
	if d.Id() != "" {
		c, orgID, _ = ClientFromExistingOrgResource(m, d.Id())
	} else {
		c, orgID = ClientFromNewOrgResource(m, d)
	}

	return
}
