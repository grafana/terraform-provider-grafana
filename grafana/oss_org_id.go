package grafana

import (
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func makeOSSOrgID(orgID int64, resourceID interface{}) string {
	return fmt.Sprintf("%d:%s", orgID, fmt.Sprint(resourceID))
}

func splitOSSOrgID(id string) (int64, string) {
	if strings.ContainsRune(id, ':') {
		parts := strings.SplitN(id, ":", 2)
		orgID, _ := strconv.ParseInt(parts[0], 10, 64)
		return orgID, parts[1]
	}

	return 0, id
}

func clientFromOSSOrgID(meta interface{}, id string) (*gapi.Client, int64, string) {
	orgID, restOfID := splitOSSOrgID(id)
	client := meta.(*client).gapi
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

func clientFromOrgIDAttr(meta interface{}, d *schema.ResourceData) (*gapi.Client, int64) {
	orgID, _ := strconv.ParseInt(d.Get("org_id").(string), 10, 64)
	client := meta.(*client).gapi
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}
