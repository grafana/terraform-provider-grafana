package grafana

import (
	"testing"

	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
)

func TestUnitClientFromExistingOrgResourceReusesTransport(t *testing.T) {
	cases := []struct {
		name          string
		providerOrgID int64
		id            string
		wantOrgID     int64
		wantSameTrans bool
	}{
		{name: "token auth, org 0 in ID", providerOrgID: 0, id: "0:folder-uid", wantOrgID: 0, wantSameTrans: true},
		{name: "basic auth, provider org in ID", providerOrgID: 1, id: "1:folder-uid", wantOrgID: 1, wantSameTrans: true},
		{name: "ID without org", providerOrgID: 1, id: "folder-uid", wantOrgID: 1, wantSameTrans: true},
		{name: "different org in ID", providerOrgID: 1, id: "2:folder-uid", wantOrgID: 2, wantSameTrans: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			base := goapi.NewHTTPClientWithConfig(strfmt.Default, &goapi.TransportConfig{
				Host:     "localhost:3000",
				BasePath: "/api",
				Schemes:  []string{"http"},
				OrgID:    tc.providerOrgID,
			})
			r := &basePluginFrameworkResource{client: base}

			client, orgID, split, err := r.clientFromExistingOrgResource(resourceFolderPermissionID, tc.id)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if orgID != tc.wantOrgID {
				t.Errorf("orgID = %d, want %d", orgID, tc.wantOrgID)
			}
			if client.OrgID() != tc.wantOrgID {
				t.Errorf("client.OrgID() = %d, want %d", client.OrgID(), tc.wantOrgID)
			}
			if len(split) != 1 || split[0] != "folder-uid" {
				t.Errorf("split = %v, want [folder-uid]", split)
			}
			if same := client.Transport == base.Transport; same != tc.wantSameTrans {
				t.Errorf("transport reused = %v, want %v", same, tc.wantSameTrans)
			}
		})
	}
}
