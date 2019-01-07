package grafana

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/hashicorp/terraform/helper/schema"

	gapi "github.com/nytm/go-grafana-api"
)

func ResourceFolder() *schema.Resource {
	return &schema.Resource{
		Create: CreateFolder,
		Delete: DeleteFolder,
		Read:   ReadFolder,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeInt,
				Computed: true,
			},

			"uid": {
				Type:     schema.TypeString,
				Computed: true,
			},

			"title": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},
		},
	}
}

func CreateFolder(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	model := d.Get("title").(string)

	resp, err := client.NewFolder(model)
	if err != nil {
		return err
	}

	id := strconv.FormatInt(resp.Id, 10)
	d.SetId(id)
	d.Set("uid", resp.Uid)
	d.Set("title", resp.Title)

	return ReadFolder(d, meta)
}

func ReadFolder(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	id, err := strconv.ParseInt(d.Id(), 10, 64)
	if err != nil {
		return err
	}

	folder, err := client.Folder(id)
	if err != nil {
		if err.Error() == "404 Not Found" {
			log.Printf("[WARN] removing folder %d from state because it no longer exists in grafana", id)
			d.SetId("")
			return nil
		}

		return err
	}

	d.SetId(strconv.FormatInt(folder.Id, 10))
	d.Set("title", folder.Title)

	return nil
}

func DeleteFolder(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*gapi.Client)

	return client.DeleteFolder(d.Get("uid").(string))
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
