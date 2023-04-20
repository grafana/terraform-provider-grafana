package oncall

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceOutgoingWebhook() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/oncall/latest/oncall-api-reference/outgoing_webhooks/)
`,
		CreateContext: ResourceOutgoingWebhookCreate,
		ReadContext:   ResourceOutgoingWebhookRead,
		UpdateContext: ResourceOutgoingWebhookUpdate,
		DeleteContext: ResourceOutgoingWebhookDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the outgoing webhook.",
			},
			"team_id": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the OnCall team. To get one, create a team in Grafana, and navigate to the OnCall plugin (to sync the team with OnCall). You can then get the ID using the `grafana_oncall_team` datasource.",
			},
			"url": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The webhook URL.",
			},
			"data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The data of the webhook.",
			},
			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth data of the webhook. Used for Basic authentication.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth data of the webhook. Used for Basic authentication",
			},
			"authorization_header": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth data of the webhook. Used in Authorization header instead of user/password auth.",
			},
			"forward_whole_payload": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Forwards whole payload of the alert to the webhook's url as POST data.",
			},
		},
	}
}

func ResourceOutgoingWebhookCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	name := d.Get("name").(string)
	teamID := d.Get("team_id").(string)
	url := d.Get("url").(string)
	forwardWholePayload := d.Get("forward_whole_payload").(bool)

	createOptions := &onCallAPI.CreateCustomActionOptions{
		Name:                name,
		TeamId:              teamID,
		Url:                 url,
		ForwardWholePayload: forwardWholePayload,
	}

	data, dataOk := d.GetOk("data")
	if dataOk {
		dd := data.(string)
		createOptions.Data = &dd
	}
	user, userOk := d.GetOk("user")
	if userOk {
		u := user.(string)
		createOptions.User = &u
	}

	password, passwordOk := d.GetOk("password")
	if passwordOk {
		p := password.(string)
		createOptions.Password = &p
	}
	authHeader, authHeaderOk := d.GetOk("authorization_header")
	if authHeaderOk {
		a := authHeader.(string)
		createOptions.AuthorizationHeader = &a
	}

	outgoingWebhook, _, err := client.CustomActions.CreateCustomAction(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)

	return ResourceOutgoingWebhookRead(ctx, d, m)
}

func ResourceOutgoingWebhookRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	outgoingWebhook, r, err := client.CustomActions.GetCustomAction(d.Id(), &onCallAPI.GetCustomActionOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing outgoingWebhook %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return diag.FromErr(err)
	}

	d.Set("name", outgoingWebhook.Name)
	d.Set("team_id", outgoingWebhook.TeamId)
	d.Set("url", outgoingWebhook.Url)
	d.Set("data", outgoingWebhook.Data)
	d.Set("user", outgoingWebhook.User)
	d.Set("password", outgoingWebhook.Password)
	d.Set("authorization_header", outgoingWebhook.AuthorizationHeader)
	d.Set("forward_whole_payload", outgoingWebhook.ForwardWholePayload)

	return nil
}

func ResourceOutgoingWebhookUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	name := d.Get("name").(string)
	url := d.Get("url").(string)
	forwardWholePayload := d.Get("forward_whole_payload").(bool)

	updateOptions := &onCallAPI.UpdateCustomActionOptions{
		Name:                name,
		Url:                 url,
		ForwardWholePayload: forwardWholePayload,
	}

	data, dataOk := d.GetOk("data")
	if dataOk {
		dd := data.(string)
		updateOptions.Data = &dd
	}
	user, userOk := d.GetOk("user")
	if userOk {
		u := user.(string)
		updateOptions.User = &u
	}

	password, passwordOk := d.GetOk("password")
	if passwordOk {
		p := password.(string)
		updateOptions.Password = &p
	}
	authHeader, authHeaderOk := d.GetOk("authorization_header")
	if authHeaderOk {
		a := authHeader.(string)
		updateOptions.AuthorizationHeader = &a
	}

	outgoingWebhook, _, err := client.CustomActions.UpdateCustomAction(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)
	return ResourceOutgoingWebhookRead(ctx, d, m)
}

func ResourceOutgoingWebhookDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*common.Client).OnCallClient

	_, err := client.CustomActions.DeleteCustomAction(d.Id(), &onCallAPI.DeleteCustomActionOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
