package grafana

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	gapi "github.com/grafana/grafana-api-golang-client"
)

func ResourceFolder() *schema.Resource {
	return &schema.Resource{

		Description: `
* [Official documentation](https://grafana.com/docs/grafana/latest/dashboards/dashboard_folders/)
* [HTTP API](https://grafana.com/docs/grafana/latest/http_api/folder/)
`,

		CreateContext: CreateFolder,
		DeleteContext: DeleteFolder,
		ReadContext:   ReadFolder,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			"uid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "Unique identifier.",
			},

			"title": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The title of the folder.",
			},
		},
	}
}

func CreateFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	model := d.Get("title").(string)

	resp, err := client.NewFolder(model)
	if err != nil {
		return diag.FromErr(err)
	}

	id := strconv.FormatInt(resp.ID, 10)
	d.SetId(id)
	d.Set("uid", resp.UID)
	d.Set("title", resp.Title)

	return ReadFolder(ctx, d, meta)
}

func ReadFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return diag.FromErr(err)
	}

	folder, err := client.Folder(id)
	if err != nil {
		if strings.HasPrefix(err.Error(), "status: 404") {
			log.Printf("[WARN] removing folder %d from state because it no longer exists in grafana", id)
			d.SetId("")
			return nil
		}

		return diag.FromErr(err)
	}

	d.SetId(strconv.FormatInt(folder.ID, 10))
	d.Set("title", folder.Title)
	d.Set("uid", folder.UID)

	return nil
}

func DeleteFolder(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*gapi.Client)

	if err := client.DeleteFolder(d.Get("uid").(string)); err != nil {
		return diag.FromErr(err)
	}

	return diag.Diagnostics{}
}

func prepareFolderModel(configJSON string) map[string]interface{} {
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		// The validate function should've taken care of this.
		panic(fmt.Errorf("Invalid JSON got into prepare func"))
	}

	return configMap
}

func ValidateFolderConfigJSON(configI interface{}, k string) ([]string, []error) {
	configJSON := configI.(string)
	configMap := map[string]interface{}{}
	err := json.Unmarshal([]byte(configJSON), &configMap)
	if err != nil {
		return nil, []error{err}
	}
	return nil, nil
}

func NormalizeFolderConfigJSON(configI interface{}) string {
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

	ret, err := json.Marshal(configMap)
	if err != nil {
		// Should never happen.
		return configJSON
	}

	return string(ret)
}
