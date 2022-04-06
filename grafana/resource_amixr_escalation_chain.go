package grafana

import (
	amixrAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func ResourceAmixrEscalationChain() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana-cloud/oncall/oncall-api-reference/escalation_chains/)
`,
		Create: resourceAmixrEscalationChainCreate,
		Read:   resourceAmixrEscalationChainRead,
		Update: resourceAmixrEscalationChainUpdate,
		Delete: resourceAmixrEscalationChainDelete,
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

func resourceAmixrEscalationChainCreate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	nameData := d.Get("name").(string)
	teamIdData := d.Get("team_id").(string)

	createOptions := &amixrAPI.CreateEscalationChainOptions{
		Name:   nameData,
		TeamId: teamIdData,
	}

	escalationChain, _, err := client.EscalationChains.CreateEscalationChain(createOptions)
	if err != nil {
		return err
	}

	d.SetId(escalationChain.ID)

	return resourceAmixrEscalationChainRead(d, m)
}

func resourceAmixrEscalationChainRead(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	escalationChain, _, err := client.EscalationChains.GetEscalationChain(d.Id(), &amixrAPI.GetEscalationChainOptions{})
	if err != nil {
		return err
	}

	d.Set("name", escalationChain.Name)
	d.Set("team_id", escalationChain.TeamId)

	return nil
}

func resourceAmixrEscalationChainUpdate(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	nameData := d.Get("name").(string)

	updateOptions := &amixrAPI.UpdateEscalationChainOptions{
		Name: nameData,
	}

	escalationChain, _, err := client.EscalationChains.UpdateEscalationChain(d.Id(), updateOptions)
	if err != nil {
		return err
	}

	d.SetId(escalationChain.ID)
	return resourceAmixrEscalationChainRead(d, m)
}

func resourceAmixrEscalationChainDelete(d *schema.ResourceData, m interface{}) error {
	client := m.(*client).amixrAPI

	_, err := client.EscalationChains.DeleteEscalationChain(d.Id(), &amixrAPI.DeleteEscalationChainOptions{})
	if err != nil {
		return err
	}

	d.SetId("")

	return nil
}
