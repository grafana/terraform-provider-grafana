package grafana

import (
	"errors"
	"log"
	"net/http"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceOnCallEscalationChain() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_chains/)
`,
		Create: ResourceOnCallEscalationChainCreate,
		Read:   ResourceOnCallEscalationChainRead,
		Update: ResourceOnCallEscalationChainUpdate,
		Delete: ResourceOnCallEscalationChainDelete,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:        schema.TypeString,
				Required:    true,
				Description: "The name of the escalation chain.",
			},
			"team_id": &schema.Schema{
				Type:        schema.TypeString,
				Optional:    true,
				Description: "The ID of the team.",
			},
		},
	}
}

func ResourceOnCallEscalationChainCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		err := errors.New("Grafana OnCall api client is not configured")
		return err
	}

	nameData := d.Get("name").(string)
	teamIdData := d.Get("team_id").(string)

	createOptions := &onCallAPI.CreateEscalationChainOptions{
		Name:   nameData,
		TeamId: teamIdData,
	}

	escalationChain, _, err := client.EscalationChains.CreateEscalationChain(createOptions)
	if err != nil {
		return err
	}

	d.SetId(escalationChain.ID)

	return ResourceOnCallEscalationChainRead(d, m)
}

func ResourceOnCallEscalationChainRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		err := errors.New("Grafana OnCall api client is not configured")
		return err
	}

	escalationChain, r, err := client.EscalationChains.GetEscalationChain(d.Id(), &onCallAPI.GetEscalationChainOptions{})
	if err != nil {
		if r != nil && r.StatusCode == http.StatusNotFound {
			log.Printf("[WARN] removing escalation chain %s from state because it no longer exists", d.Get("name").(string))
			d.SetId("")
			return nil
		}
		return err
	}

	d.Set("name", escalationChain.Name)
	d.Set("team_id", escalationChain.TeamId)

	return nil
}

func ResourceOnCallEscalationChainUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		err := errors.New("Grafana OnCall api client is not configured")
		return err
	}

	nameData := d.Get("name").(string)

	updateOptions := &onCallAPI.UpdateEscalationChainOptions{
		Name: nameData,
	}

	escalationChain, _, err := client.EscalationChains.UpdateEscalationChain(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(escalationChain.ID)
	return ResourceOnCallEscalationChainRead(d, m)
}

func ResourceOnCallEscalationChainDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).onCallAPI
	if client == nil {
		err := errors.New("Grafana OnCall api client is not configured")
		return err
	}

	_, err := client.EscalationChains.DeleteEscalationChain(d.Id(), &onCallAPI.DeleteEscalationChainOptions{})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
