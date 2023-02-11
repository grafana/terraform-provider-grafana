package grafana

import (
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func orgIDAttribute() *schema.Schema {
	return &schema.Schema{
		Type:        schema.TypeString,
		Optional:    true,
		Description: "The Organization ID. If not set, the Org ID defined in the provider block will be used.",
		ForceNew:    true,
		DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool {
			return new == "" // Ignore the case where we have a global org_id set
		},
	}
}

// makeOrgResourceID creates a resource ID for an org-scoped resource
func makeOrgResourceID(orgID int64, resourceID interface{}) string {
	return fmt.Sprintf("%d:%s", orgID, fmt.Sprint(resourceID))
}

// splitOrgResourceID splits into two parts (org ID and resource ID) the ID of an org-scoped resource
func splitOrgResourceID(id string) (int64, string) {
	if strings.ContainsRune(id, ':') {
		parts := strings.SplitN(id, ":", 2)
		orgID, _ := strconv.ParseInt(parts[0], 10, 64)
		return orgID, parts[1]
	}

	return 0, id
}

// clientFromExistingOrgResource creates a client from the ID of an org-scoped resource
// Those IDs are in the <orgID>:<resourceID> format
func clientFromExistingOrgResource(meta interface{}, id string) (*gapi.Client, int64, string) {
	orgID, restOfID := splitOrgResourceID(id)
	client := meta.(*common.Client).GrafanaAPI
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

// clientFromNewOrgResource creates a client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func clientFromNewOrgResource(meta interface{}, d *schema.ResourceData) (*gapi.Client, int64) {
	orgID, _ := strconv.ParseInt(d.Get("org_id").(string), 10, 64)
	client := meta.(*common.Client).GrafanaAPI
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}
