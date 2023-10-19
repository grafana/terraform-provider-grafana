package grafana

import (
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/client"

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

// MakeOrgResourceID creates a resource ID for an org-scoped resource
func MakeOrgResourceID(orgID int64, resourceID interface{}) string {
	return fmt.Sprintf("%d:%s", orgID, fmt.Sprint(resourceID))
}

// SplitOrgResourceID splits into two parts (org ID and resource ID) the ID of an org-scoped resource
func SplitOrgResourceID(id string) (int64, string) {
	if strings.ContainsRune(id, ':') {
		parts := strings.SplitN(id, ":", 2)
		orgID, _ := strconv.ParseInt(parts[0], 10, 64)
		return orgID, parts[1]
	}

	return 0, id
}

// ClientFromExistingOrgResource creates a client from the ID of an org-scoped resource
// Those IDs are in the <orgID>:<resourceID> format
func ClientFromExistingOrgResource(meta interface{}, id string) (*gapi.Client, int64, string) {
	orgID, restOfID := SplitOrgResourceID(id)
	client := meta.(*common.Client).GrafanaAPI
	if orgID == 0 {
		orgID = meta.(*common.Client).GrafanaAPIConfig.OrgID // It's configured globally. TODO: Remove this once we drop support for the global org_id
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

// ClientFromNewOrgResource creates a client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func ClientFromNewOrgResource(meta interface{}, d *schema.ResourceData) (*gapi.Client, int64) {
	orgID := parseOrgID(d)
	client := meta.(*common.Client).GrafanaAPI
	if orgID == 0 {
		orgID = meta.(*common.Client).GrafanaAPIConfig.OrgID // It's configured globally. TODO: Remove this once we drop support for the global org_id
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}

// OAPIClientFromExistingOrgResource creates a client from the ID of an org-scoped resource
// Those IDs are in the <orgID>:<resourceID> format
func OAPIClientFromExistingOrgResource(meta interface{}, id string) (*goapi.GrafanaHTTPAPI, int64, string) {
	orgID, restOfID := SplitOrgResourceID(id)
	client := meta.(*common.Client).GrafanaOAPI
	if orgID == 0 {
		orgID = meta.(*common.Client).GrafanaOAPI.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

// OAPIClientFromNewOrgResource creates an OpenAPI client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func OAPIClientFromNewOrgResource(meta interface{}, d *schema.ResourceData) (*goapi.GrafanaHTTPAPI, int64) {
	orgID := parseOrgID(d)
	client := meta.(*common.Client).GrafanaOAPI
	if orgID == 0 {
		orgID = meta.(*common.Client).GrafanaOAPI.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}

func parseOrgID(d *schema.ResourceData) int64 {
	orgID, _ := strconv.ParseInt(d.Get("org_id").(string), 10, 64)
	return orgID
}
