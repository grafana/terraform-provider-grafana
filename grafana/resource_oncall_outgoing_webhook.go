package grafana

import (
	"context"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceOnCallOutgoingWebhook() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/outgoing_webhooks/)
`,
		CreateContext: ResourceOnCallOutgoingWebhookCreate,
		ReadContext:   ResourceOnCallOutgoingWebhookRead,
		UpdateContext: ResourceOnCallOutgoingWebhookUpdate,
		DeleteContext: ResourceOnCallOutgoingWebhookDelete,
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
				Description: "The ID of the team.",
			},
			"webhook": {
				Type:        schema.TypeString,
				Required:    true,
				Description: "The url of the webhook.",
			},
			"data": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The data of the webhook.",
			},
			"user": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth data of the webhook.",
			},
			"password": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The auth data of the webhook.",
			},
			"authorization_header": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The authorization header.",
			},
			"forward_whole_payload": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Forwards whole payload of the alert to the webhook's url as POST data.",
			},
		},
	}
}

func ResourceOnCallOutgoingWebhookCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	nameData := d.Get("name").(string)
	teamIdData := d.Get("team_id").(string)
	webhookData := d.Get("webhook").(string)
	forwardWholePayloadData := d.Get("forward_whole_payload").(bool)

	createOptions := &onCallAPI.CreateCustomActionOptions{
		Name:                nameData,
		TeamId:              teamIdData,
		Webhook:             webhookData,
		ForwardWholePayload: forwardWholePayloadData,
	}

	dataData, dataDataOk := d.GetOk("data")
    if dataDataOk {
		dd := dataData.(string)
		createOptions.Data = &dd
	}
	userData, userDataOk := d.GetOk("user")
    if userDataOk {
		u := userData.(string)
		createOptions.User = &u
	}

	passwordData, passwordDataOk := d.GetOk("password")
    if passwordDataOk {
		p := passwordData.(string)
		createOptions.Password = &p
	}
	authHeaderData, authHeaderDataOk := d.GetOk("authorization_header")
    if authHeaderDataOk {
		a := authHeaderData.(string)
		createOptions.AuthorizationHeader = &a
	}

	outgoingWebhook, _, err := client.CustomActions.CreateCustomAction(createOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)

	return ResourceOnCallOutgoingWebhookRead(ctx, d, m)
}

func ResourceOnCallOutgoingWebhookRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

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
	d.Set("webhook", outgoingWebhook.Webhook)
	d.Set("data", outgoingWebhook.Data)
	d.Set("user", outgoingWebhook.User)
	d.Set("password", outgoingWebhook.Password)
	d.Set("authorization_header", outgoingWebhook.AuthorizationHeader)
	d.Set("forward_whole_payload", outgoingWebhook.ForwardWholePayload)

	return nil
}

func ResourceOnCallOutgoingWebhookUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	nameData := d.Get("name").(string)
	webhookData := d.Get("webhook").(string)
	forwardWholePayloadData := d.Get("forward_whole_payload").(bool)

	updateOptions := &onCallAPI.UpdateCustomActionOptions{
		Name:                nameData,
        Webhook:             webhookData,
		ForwardWholePayload: forwardWholePayloadData,
	}

		dataData, dataDataOk := d.GetOk("data")
    if dataDataOk {
		dd := dataData.(string)
		updateOptions.Data = &dd
	}
	userData, userDataOk := d.GetOk("user")
    if userDataOk {
		u := userData.(string)
		updateOptions.User = &u
	}

	passwordData, passwordDataOk := d.GetOk("password")
    if passwordDataOk {
		p := passwordData.(string)
		updateOptions.Password = &p
	}
	authHeaderData, authHeaderDataOk := d.GetOk("authorization_header")
    if authHeaderDataOk {
		a := authHeaderData.(string)
		updateOptions.AuthorizationHeader = &a
	}

	outgoingWebhook, _, err := client.CustomActions.UpdateCustomAction(d.Id(), updateOptions)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(outgoingWebhook.ID)
	return ResourceOnCallOutgoingWebhookRead(ctx, d, m)
}

func ResourceOnCallOutgoingWebhookDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client := m.(*client).onCallAPI
	if client == nil {
		return diag.Errorf("grafana OnCall api client is not configured")
	}

	_, err := client.CustomActions.DeleteCustomAction(d.Id(), &onCallAPI.DeleteCustomActionOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return nil
}
