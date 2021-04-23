package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceDashboard() *schema.Resource {
	return &schema.Resource{
		CreateContext: CreateDashboard,
		ReadContext:   ReadDashboard,
		UpdateContext: UpdateDashboard,
		DeleteContext: DeleteDashboard,
		Importer: &schema.ResourceImporter{
			State: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"slug": {
				Type:     schema.TypeString,
				Computed: true,
			},

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

func CreateDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}

	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string))

	dashboard.Folder = int64(d.Get("folder").(int))

	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resp.Slug)

	return ReadDashboard(ctx, d, meta)
}

func ReadDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	slug := d.Id()

	dashboard, err := client.Dashboard(slug)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing dashboard %s from state because it no longer exists in grafana", slug)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	configJSONBytes, err := json.Marshal(dashboard.Model)
	if err != nil {
		return diag.FromErr(err)
	}

	configJSON := NormalizeDashboardConfigJSON(string(configJSONBytes))

	d.SetId(dashboard.Meta.Slug)
	d.Set("slug", dashboard.Meta.Slug)
	d.Set("config_json", configJSON)
	d.Set("folder", dashboard.Folder)
	d.Set("dashboard_id", int64(dashboard.Model["id"].(float64)))

	return nil
}

func UpdateDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	dashboard := gapi.Dashboard{}

	dashboard.Model = prepareDashboardModel(d.Get("config_json").(string))

	dashboard.Folder = int64(d.Get("folder").(int))
	dashboard.Overwrite = true

	resp, err := client.NewDashboard(dashboard)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(resp.Slug)

	return ReadDashboard(ctx, d, meta)
}

func DeleteDashboard(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	slug := d.Id()
	if err := client.DeleteDashboard(slug); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func prepareDashboardModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic(fmt.Errorf("Invalid JSON got into prepare func"))
	}

	delete(configMap, "id")
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
	delete(configMap, "uid")

	ret, err := json.Marshal(configMap)
	if err != nil {
		// Should never happen.
		return configJSON
	}

	return string(ret)
}
