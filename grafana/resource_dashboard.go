package grafana

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
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
			"dashboard_id": {
				Type:     schema.TypeInt,
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

func CreateDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}

	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string))

	dashboard.Folder = int64(d.Get("folder").(int))

	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return err
	}

	d.SetId(resp.UID)

	return ReadDashboard(d, meta)
}

func ReadDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	uid := d.Id()

	dashboard, err := client.DashboardByUID(uid)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing dashboard %s from state because it no longer exists in grafana", uid)
			d.SetId("")
			return nil
		}

		return err
	}

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return err
	}

	configJSON := NormalizeDashboardConfigJSON(string(configJSONBytes))

	d.SetId(uid)
	d.Set("config_json", configJSON)
	d.Set("folder", dashboard.Folder)
	d.Set("dashboard_id", int64(dashboard.Model["id"].(float64)))

	return nil
}

func UpdateDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}

	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string))

	dashboard.Folder = int64(d.Get("folder").(int))
	dashboard.Overwrite = true

	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return err
	}

	d.SetId(resp.UID)

	return ReadDashboard(d, meta)
}

func DeleteDashboard(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	uid := d.Id()
	return client.DeleteDashboardByUID(uid)
}

func prepareDashboardModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic(fmt.Errorf("Invalid JSON got into prepare func"))
	}

	delete(configMap, "id")
	// Only exists in 5.0+
	delete(configMap, "uid")
	configMap["version"] = 0

	return configMap
}

func ValidateDashboardConfigJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeDashboardConfigJSON(configI interface{}) string {
	configJSON := configI.(string)

	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		return ""
	}

	// Some properties are managed by this provider and are thus not
	// significant when included in the JSON.
	delete(configMap, "id")
	delete(configMap, "version")
	// Only exists in 5.0+
	delete(configMap, "uid")

	ret, err := json.Marshal(configMap)
	if err != nil {
		// Should never happen.
		return configJSON
	}

	return string(ret)
}
