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
		wantPluginCall bool
	}{
		"token stored": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"publisher_token_present": true}`,
			wantPluginCall: true,
		},
		"token not stored": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"publisher_token_present": false}`,
			wantWarning:    true,
			wantPluginCall: true,
		},
		"k6 cloud api does not report the field": {
			instanceStatus: http.StatusOK,
			pluginStatus:   http.StatusOK,
			pluginBody:     `{"initialized": true}`,
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
			}
			if (pluginCalls > 0) != tt.wantPluginCall {
				t.Fatalf("plugin calls = %d, want called = %v", pluginCalls, tt.wantPluginCall)
			}
		})
	}
}
