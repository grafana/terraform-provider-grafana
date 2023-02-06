package grafana

import (
	"fmt"
	"strconv"
	"strings"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func MakeOSSOrgID(orgID int64, resourceID interface{}) string {
	return fmt.Sprintf("%d:%s", orgID, fmt.Sprint(resourceID))
}

func SplitOSSOrgID(id string) (int64, string) {
	if strings.ContainsRune(id, ':') {
		parts := strings.SplitN(id, ":", 2)
		orgID, _ := strconv.ParseInt(parts[0], 10, 64)
		return orgID, parts[1]
	}

	return 0, id
}

func ClientFromOSSOrgID(meta interface{}, id string) (*gapi.Client, int64, string) {
	orgID, restOfID := SplitOSSOrgID(id)
	client := meta.(*common.Client).GrafanaAPI
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID, restOfID
}

func ClientFromOrgIDAttr(meta interface{}, d *schema.ResourceData) (*gapi.Client, int64) {
	orgID, _ := strconv.ParseInt(d.Get("org_id").(string), 10, 64)
	client := meta.(*common.Client).GrafanaAPI
	if orgID > 0 {
		client = client.WithOrgID(orgID)
	}
	return client, orgID
}
