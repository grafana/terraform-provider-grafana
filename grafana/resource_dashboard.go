package grafana

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/nytm/go-grafana-api"
)

func ResourceDashboard() *schema.Resource {
	return &schema.Resource{
		Create: CreateDashboard,
		Read:   ReadDashboard,
		Update: UpdateDashboard,
		Delete: DeleteDashboard,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthrough,
		},
		Schema: map[string]*schema.Schema{
			"uid": {
				Type:     schema.TypeString,
				Computed: true,
				Optional: true,
				ForceNew: true,
			},

			"slug": {
				Type:       schema.TypeString,
				Computed:   true,
				Deprecated: "Slug is deprecated since Grafana v5. Use uid instead.",
			},

			"folder": {
				Type:     schema.TypeInt,
				Optional: true,
			},

			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: ValidateDashboardConfigJSON,
			},
		},
		SchemaVersion: 1,
		StateUpgraders: []schema.StateUpgrader{
			{
				Type:    resourceDashboardResourceV0().CoreConfigSchema().ImpliedType(),
				Upgrade: resourceDashboardStateUpgradeV0,
				Version: 0,
			},
		},
	}
}

func CreateDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}
	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string), d.Get("uid").(string))
	dashboard.Folder = int64(d.Get("folder").(int))

	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return fmt.Errorf("error creating dashboard: %s", err)
	}

	d.SetId(resp.Uid)
	return ReadDashboard(d, meta)
}

func ReadDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboard, err := client.DashboardByUID(d.Id())
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] Removing dashboard %s from state because it no longer exists in grafana", d.Id())
			d.SetId("")
			return nil
		}

		return fmt.Errorf("Failed to read dashboard '%s': %v", d.Id(), err)
	}

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return err
	}

	d.Set("uid", d.Id())
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("config_json", NormalizeDashboardConfigJSON(string(configJSONBytes)))
	d.Set("folder", dashboard.Folder)

	return nil
}

func UpdateDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}
	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string), d.Id())
	dashboard.Folder = int64(d.Get("folder").(int))
	dashboard.Overwrite = true

	_, err := client.NewDashboard(dashboard)
	if err != nil {
		return fmt.Errorf("error updating dashboard %q: %s", d.Id(), err)
	}

	return ReadDashboard(d, meta)
}

func DeleteDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	err := client.DeleteDashboardByUID(d.Id())
	if err != nil {
		return fmt.Errorf("error deleting dashboard %q: %s", d.Id(), err)
	}

	return nil
}

func prepareDashboardModel(configJSON string, uid string) map[string]interface{} {
	model := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &model)
	if err != nil {
		// The validate function should've taken care of this.
		panic("invalid JSON")
	}

	// Grafana's API is strange. We need to delete this field, otherwise Grafana wont create a new dashboard for us. Update also works without this field.
	delete(model, "id")

	// Let's use the uid from the terraform file. The one in config_file will be ignored.
	// Can be empty to create a dashboard.
	if len(uid) > 0 {
		model["uid"] = uid
	} else {
		delete(model, "uid")
	}

	delete(model, "version")

	return model
}

func ValidateDashboardConfigJSON(state interface{}, k string) ([]string, []error) {
	configJSON := state.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeDashboardConfigJSON(state interface{}) string {
	configJSON := state.(string)

	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic("invalid JSON")
	}

	// Some properties are managed by this provider and are thus not
	// significant when included in the JSON.
	delete(configMap, "id")
	delete(configMap, "version")
	delete(configMap, "uid")

	ret, err := json.Marshal(configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic("invalid JSON")
	}

	return string(ret)
}

func resourceDashboardResourceV0() *schema.Resource {
	return &schema.Resource{
		Schema: map[string]*schema.Schema{
			"slug": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"folder": {
				Type:     schema.TypeInt,
				Optional: true,
				ForceNew: true,
			},
			"config_json": {
				Type:         schema.TypeString,
				Required:     true,
				StateFunc:    NormalizeDashboardConfigJSON,
				ValidateFunc: ValidateDashboardConfigJSON,
			},
		},
	}
}

func resourceDashboardStateUpgradeV0(rawState map[string]interface{}, meta interface{}) (map[string]interface{}, error) {
	client := meta.(*gapi.Client)
	slug := rawState["slug"].(string)

	dashboard, err := client.Dashboard(slug)
	if err != nil {
		return nil, fmt.Errorf("error upgrading dashboard [slug=%q]: %s", slug, err)
	}

	uid, ok := dashboard.Model["uid"]
	if !ok {
		return nil, fmt.Errorf("error upgrading dashboard [slug=%q]: grafana_dashboard requires Grafana v5.0+. Please update Grafana or use a Terraform Grafana provider prior or equal to version v1.5.0", slug)
	}

	rawState["id"] = uid
	rawState["uid"] = uid

	return rawState, nil
}
