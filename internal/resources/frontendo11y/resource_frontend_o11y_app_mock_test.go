package frontendo11y_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// fakeFrontendO11yAPI serves the Grafana Cloud stack lookup and the Frontend
// Observability app endpoints used by grafana_frontend_o11y_app.
type fakeFrontendO11yAPI struct {
	mu     sync.Mutex
	apps   map[int64]frontendo11yapi.App
	nextID int64
}

func newFakeFrontendO11yAPI(t *testing.T) *fakeFrontendO11yAPI {
	t.Helper()
	f := &fakeFrontendO11yAPI{apps: map[int64]frontendo11yapi.App{}, nextID: 1}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/instances/1", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "slug": "mystack", "regionSlug": "prod-us-east-0", "createdAt": "2024-01-01T00:00:00Z",
		})
	})
	mux.HandleFunc("/faro/api/v1/app", f.handleApps)
	mux.HandleFunc("/faro/api/v1/app/", f.handleApp)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	t.Setenv("GRAFANA_CLOUD_API_URL", server.URL)
	t.Setenv("GRAFANA_CLOUD_ACCESS_POLICY_TOKEN", "token")
	t.Setenv("GRAFANA_FRONTEND_O11Y_API_URL", server.URL+"/faro")
	t.Setenv("GRAFANA_FRONTEND_O11Y_API_ACCESS_TOKEN", "token")
	return f
}

func (f *fakeFrontendO11yAPI) store(app frontendo11yapi.App) frontendo11yapi.App {
	app.Key = "appkey"
	app.CollectEndpointURL = "https://faro-collector.example.com/collect"
	f.apps[app.ID] = app
	return app
}

func (f *fakeFrontendO11yAPI) handleApps(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		apps := []frontendo11yapi.App{}
		for _, app := range f.apps {
			apps = append(apps, app)
		}
		_ = json.NewEncoder(w).Encode(apps)
	case http.MethodPost:
		var app frontendo11yapi.App
		_ = json.NewDecoder(r.Body).Decode(&app)
		app.ID = f.nextID
		f.nextID++
		_ = json.NewEncoder(w).Encode(f.store(app))
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeFrontendO11yAPI) handleApp(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	var id int64
	if err := json.Unmarshal([]byte(strings.TrimPrefix(r.URL.Path, "/faro/api/v1/app/")), &id); err != nil {
		http.NotFound(w, r)
		return
	}
	if _, ok := f.apps[id]; !ok {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		_ = json.NewEncoder(w).Encode(f.apps[id])
	case http.MethodPut:
		var app frontendo11yapi.App
		_ = json.NewDecoder(r.Body).Decode(&app)
		app.ID = id
		_ = json.NewEncoder(w).Encode(f.store(app))
	case http.MethodDelete:
		delete(f.apps, id)
		w.WriteHeader(http.StatusNoContent)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeFrontendO11yAPI) deleteAll() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.apps = map[int64]frontendo11yapi.App{}
}

const frontendO11yAppMockConfig = `
resource "grafana_frontend_o11y_app" "test" {
  stack_id        = 1
  name            = "mock-app"
  allowed_origins = ["https://app.example.com"]

  extra_log_attributes = {}
  settings             = {}
}
`

// An app deleted outside of Terraform must be dropped from the state on
// refresh so that the next plan recreates it.
func TestUnitFrontendO11yApp_DeletedOutsideTerraform_Mock(t *testing.T) {
	api := newFakeFrontendO11yAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: frontendO11yAppMockConfig,
				Check:  resource.TestCheckResourceAttr("grafana_frontend_o11y_app.test", "id", "1"),
			},
			{
				PreConfig:          api.deleteAll,
				Config:             frontendO11yAppMockConfig,
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
			},
		},
	})
}
