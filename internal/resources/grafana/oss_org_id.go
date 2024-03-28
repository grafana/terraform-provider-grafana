package grafana

import (
	"fmt"
	"strconv"
	"strings"

	goapi "github.com/grafana/grafana-openapi-client-go/client"

	"github.com/grafana/terraform-provider-grafana/v2/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Helpers for org-scoped resource IDs
func orgResourceIDString(fieldName string) *common.ResourceID {
	return common.NewResourceID(common.OptionalIntIDField("orgID"), common.StringIDField(fieldName))
}

func orgResourceIDInt(fieldName string) *common.ResourceID {
	return common.NewResourceID(common.OptionalIntIDField("orgID"), common.IntIDField(fieldName))
}

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

// OAPIClientFromExistingOrgResource creates a client from the ID of an org-scoped resource
// Those IDs are in the <orgID>:<resourceID> format
func OAPIClientFromExistingOrgResource(meta interface{}, id string) (*goapi.GrafanaHTTPAPI, int64, string) {
	orgID, restOfID := SplitOrgResourceID(id)
	client := meta.(*common.Client).GrafanaOAPI.Clone()
	if orgID == 0 {
		orgID = client.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

// OAPIClientFromNewOrgResource creates an OpenAPI client from the `org_id` attribute of a resource
// This client is meant to be used in `Create` functions when the ID hasn't already been baked into the resource ID
func OAPIClientFromNewOrgResource(meta interface{}, d *schema.ResourceData) (*goapi.GrafanaHTTPAPI, int64) {
	orgID := parseOrgID(d)
	client := meta.(*common.Client).GrafanaOAPI.Clone()
	if orgID == 0 {
		orgID = client.OrgID()
	} else if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}

func OAPIGlobalClient(meta interface{}) *goapi.GrafanaHTTPAPI {
	return meta.(*common.Client).GrafanaOAPI.Clone().WithOrgID(0)
}

func parseOrgID(d *schema.ResourceData) int64 {
	orgID, _ := strconv.ParseInt(d.Get("org_id").(string), 10, 64)
	return orgID
}
