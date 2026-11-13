package grafana

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func TestUnitDeleteContactPointDetachmentOptIn(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   map[string]any
		wantRead bool
	}{
		{name: "omitted", config: map[string]any{"name": "remove"}},
		{name: "disabled", config: map[string]any{"name": "remove", "detach_from_notification_policies_on_delete": false}},
		{name: "enabled", config: map[string]any{"name": "remove", "detach_from_notification_policies_on_delete": true}, wantRead: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			policyRead := false
			deleteCalls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/contact-points":
					_, _ = w.Write([]byte(`[{"name":"remove","uid":"uid","type":"email","settings":{"addresses":"test@example.com"}}]`))
				case r.Method == http.MethodDelete && r.URL.Path == "/api/v1/provisioning/contact-points/uid":
					deleteCalls++
					if deleteCalls == 1 {
						w.WriteHeader(http.StatusConflict)
						_, _ = w.Write([]byte(`{"messageId":"alerting.notifications.contact-points.referenced","statusCode":409}`))
					} else {
						// Stop retries after checking whether detachment was attempted.
						w.WriteHeader(http.StatusForbidden)
					}
				case r.Method == http.MethodGet && r.URL.Path == "/api/v1/provisioning/policies":
					policyRead = true
					w.WriteHeader(http.StatusForbidden)
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()

			data := schema.TestResourceDataRaw(t, resourceContactPoint().Schema.Schema, tc.config)
			data.SetId("1:remove")
			client := &common.Client{GrafanaAPI: testGrafanaClient(t, server.URL)}
			diagnostics := deleteContactPoint(context.Background(), data, client)
			require.True(t, diagnostics.HasError(), "the mock API must reject deletion")
			require.Equal(t, tc.wantRead, policyRead, "policy access must require explicit opt-in")
		})
	}
}

func TestUnitIsContactPointReferenced(t *testing.T) {
	referenced := "alerting.notifications.contact-points.referenced"
	other := "alerting.notifications.conflict"
	for _, tc := range []struct {
		name string
		err  error
		want bool
	}{
		{name: "no error"},
		{name: "missing payload", err: &provisioning.DeleteContactpointsConflict{}},
		{name: "missing message ID", err: &provisioning.DeleteContactpointsConflict{Payload: &models.PublicError{}}},
		{name: "concurrent write conflict", err: &provisioning.DeleteContactpointsConflict{Payload: &models.PublicError{MessageID: &other}}},
		{name: "referenced contact point", err: &provisioning.DeleteContactpointsConflict{Payload: &models.PublicError{MessageID: &referenced}}, want: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, isContactPointReferenced(tc.err))
		})
	}
}

func TestUnitDetachContactPointFromPolicy(t *testing.T) {
	for _, provenance := range []models.Provenance{"", "api"} {
		t.Run("provenance="+string(provenance), func(t *testing.T) {
			// Clear references at multiple depths without losing other routing settings.
			policy := models.Route{
				Receiver:   "default",
				Provenance: provenance,
				GroupBy:    []string{"alertname"},
				Routes: []*models.Route{
					{
						Receiver:       "remove",
						Continue:       true,
						GroupWait:      "45s",
						ObjectMatchers: models.ObjectMatchers{{"team", "=", "remove"}},
						Routes: []*models.Route{
							{Receiver: "keep", GroupInterval: "10m"},
							{Receiver: "remove", RepeatInterval: "2h"},
						},
					},
					{Receiver: "keep", GroupWait: "1m"},
				},
			}
			var written models.Route
			writes := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/v1/provisioning/policies" {
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				switch r.Method {
				case http.MethodGet:
					_ = json.NewEncoder(w).Encode(policy)
				case http.MethodPut:
					writes++
					if err := json.NewDecoder(r.Body).Decode(&written); err != nil {
						t.Errorf("decode policy: %v", err)
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					wantHeader := ""
					if provenance == "" {
						wantHeader = provenanceDisabled
					}
					if got := r.Header.Get("X-Disable-Provenance"); got != wantHeader {
						t.Errorf("provenance header = %q, want %q", got, wantHeader)
					}
					w.WriteHeader(http.StatusAccepted)
					_ = json.NewEncoder(w).Encode(written)
				default:
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer server.Close()

			err := detachContactPointFromPolicy(context.Background(), testGrafanaClient(t, server.URL), "remove")
			require.NoError(t, err)
			require.Equal(t, 1, writes)
			// All fields except the matching receiver overrides must be preserved.
			policy.Routes[0].Receiver = ""
			policy.Routes[0].Routes[1].Receiver = ""
			require.Equal(t, policy, written)
		})
	}
}

func TestUnitDetachContactPointFromPolicyNoWrite(t *testing.T) {
	for _, receiver := range []string{"default", "remove"} {
		t.Run("root="+receiver, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Errorf("policy must not be modified, got %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(models.Route{
					Receiver: receiver,
					Routes:   []*models.Route{{Receiver: "keep"}},
				})
			}))
			defer server.Close()

			err := detachContactPointFromPolicy(context.Background(), testGrafanaClient(t, server.URL), "remove")
			if receiver == "remove" {
				require.ErrorContains(t, err, "root notification policy receiver")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
