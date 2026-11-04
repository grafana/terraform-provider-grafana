package grafana

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// newOrgListerTestServer serves two orgs (1 and 2). Requests scoped to org 2 are forbidden,
// which is how Grafana responds once an org has been deleted. GET /api/orgs/2 returns
// org2Status, so tests can simulate org 2 being deleted (404) or still existing (200).
func newOrgListerTestServer(t *testing.T, org2Status int) *common.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/orgs":
			if r.URL.Query().Get("page") == "0" {
				fmt.Fprint(w, `[{"id":1,"name":"Main Org."},{"id":2,"name":"other"}]`)
				return
			}
			fmt.Fprint(w, `[]`)
		case "/api/orgs/2":
			w.WriteHeader(org2Status)
			fmt.Fprint(w, `{"id":2,"name":"other"}`)
		case "/api/v1/provisioning/templates":
			if r.Header.Get("X-Grafana-Org-Id") == "2" {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{}`)
				return
			}
			fmt.Fprint(w, `[{"name":"tmpl"}]`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	t.Cleanup(server.Close)
	return &common.Client{GrafanaAPI: testGrafanaClient(t, server.URL)}
}

func TestUnitListerFunctionOrgResource_SkipsDeletedOrg(t *testing.T) {
	client := newOrgListerTestServer(t, http.StatusNotFound)

	ids, err := listerFunctionOrgResource(listMessageTemplate)(context.Background(), client, NewListerData(false, false))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(ids, []string{"1:tmpl"}) {
		t.Fatalf("got ids %v, want [1:tmpl]", ids)
	}
}

func TestUnitListerFunctionOrgResource_ForbiddenInExistingOrg(t *testing.T) {
	client := newOrgListerTestServer(t, http.StatusOK)

	_, err := listerFunctionOrgResource(listMessageTemplate)(context.Background(), client, NewListerData(false, false))
	if err == nil {
		t.Fatal("expected a forbidden error for an org that still exists")
	}
}
