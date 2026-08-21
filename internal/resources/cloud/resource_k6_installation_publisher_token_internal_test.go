package cloud

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestUnitSyncPublisherToken(t *testing.T) {
	for name, tt := range map[string]struct {
		instanceStatus int
		pluginStatus   int
		pluginBody     string
		wantWarning    bool
		wantDetail     string
		wantPluginCall bool
	}{
		"initialized": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"initialized": true, "publisher_token_present": true}`,
			wantPluginCall: true,
		},
		"not initialized, token missing": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"initialized": false, "publisher_token_present": false}`,
			wantWarning:    true,
			wantDetail:     "no valid k6 publisher token",
			wantPluginCall: true,
		},
		"not initialized, folders pending on an rbac stack": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody: `{"initialized": false, "publisher_token_present": true, ` +
				`"folders_initialized": false, "grafana_rbac_enabled": true}`,
			wantWarning:    true,
			wantDetail:     "Grafana folders are not set up",
			wantPluginCall: true,
		},
		"not initialized, both pending": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody: `{"initialized": false, "publisher_token_present": false, ` +
				`"folders_initialized": false, "grafana_rbac_enabled": true}`,
			wantWarning:    true,
			wantDetail:     "publisher token, and the k6 App's Grafana folders",
			wantPluginCall: true,
		},
		"not initialized, folders pending without rbac is not the folders' fault": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody: `{"initialized": false, "publisher_token_present": false, ` +
				`"folders_initialized": false, "grafana_rbac_enabled": false}`,
			wantWarning:    true,
			wantDetail:     "no valid k6 publisher token",
			wantPluginCall: true,
		},
		"not initialized without a reported cause": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"initialized": false}`,
			wantWarning:    true,
			wantDetail:     "reports this stack as not initialized",
			wantPluginCall: true,
		},
		"k6 cloud api does not report the field": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"publisher_token_present": true}`,
			wantPluginCall: true,
		},
		"plugin route fails": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusInternalServerError,
			pluginBody:     `{}`,
			wantWarning:    true,
			wantPluginCall: true,
		},
		"stack lookup fails": {
			instanceStatus: http.StatusNotFound,
			wantWarning:    true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			var srvURL string
			var pluginCalls int

			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if strings.Contains(r.URL.Path, "grafana-app/initialized") {
					pluginCalls++
					w.WriteHeader(tt.pluginStatus)
					_, _ = w.Write([]byte(tt.pluginBody))
					return
				}
				w.WriteHeader(tt.instanceStatus)
				_, _ = fmt.Fprintf(w, `{"id": 1, "url": %q}`, srvURL)
			}))
			t.Cleanup(srv.Close)
			srvURL = srv.URL

			d := schema.TestResourceDataRaw(t, resourceK6Installation().Schema.Schema, map[string]any{
				"stack_id":         "1",
				"grafana_sa_token": "glsa_token",
			})

			diags := syncPublisherToken(context.Background(), d, newTestGcomAPIClient(t, srv))

			if diags.HasError() {
				t.Fatalf("delivery must never fail the apply, got %v", diags)
			}
			if warnings := len(diags); (warnings > 0) != tt.wantWarning {
				t.Fatalf("warnings = %d, want warning = %v (%v)", warnings, tt.wantWarning, diags)
			}
			for _, dg := range diags {
				if dg.Severity != diag.Warning {
					t.Fatalf("severity = %v, want warning", dg.Severity)
				}
				if tt.wantDetail != "" && !strings.Contains(dg.Detail, tt.wantDetail) {
					t.Fatalf("detail = %q, want it to contain %q", dg.Detail, tt.wantDetail)
				}
			}
			if (pluginCalls > 0) != tt.wantPluginCall {
				t.Fatalf("plugin calls = %d, want called = %v", pluginCalls, tt.wantPluginCall)
			}
		})
	}
}
