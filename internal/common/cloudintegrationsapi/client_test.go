package cloudintegrationsapi_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi/models"
)

const (
	integrationsBasePath = "/api/plugin-proxy/grafana-easystart-app/integrations-api-admin"
	rulesConvertPath     = "/api/convert/prometheus/config/v1/rules"
	cloudPromUID         = "grafanacloud-prom"
)

func newTestClient(t *testing.T, svr *httptest.Server) *cloudintegrationsapi.Client {
	t.Helper()
	c, err := cloudintegrationsapi.NewClient(svr.URL, "test-token", svr.Client(), "test-user-agent", map[string]string{"X-Custom": "header-value"})
	require.NoError(t, err)
	return c
}

func TestUnit_NewClient(t *testing.T) {
	t.Parallel()

	t.Run("creates client with provided http.Client", func(t *testing.T) {
		t.Parallel()
		c, err := cloudintegrationsapi.NewClient("https://grafana.example.com", "my-token", &http.Client{}, "my-agent", map[string]string{"X-Foo": "bar"})
		require.NoError(t, err)
		assert.NotNil(t, c)
	})

	t.Run("creates retry client when http.Client is nil", func(t *testing.T) {
		t.Parallel()
		c, err := cloudintegrationsapi.NewClient("https://grafana.example.com", "my-token", nil, "my-agent", nil)
		require.NoError(t, err)
		assert.NotNil(t, c)
	})
}

func TestUnit_GetIntegration(t *testing.T) {
	t.Parallel()

	t.Run("success with response deserialization", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, integrationsBasePath+"/integrations/docker", r.URL.Path)

			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
				Data: models.Integration{
					Name:            "Docker",
					Slug:            "docker",
					Version:         "2.1.0",
					DashboardFolder: "Docker",
					Installation:    &models.Installation{Version: "2.1.0"},
				},
			})
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		resp, err := c.GetIntegration(context.Background(), "docker")
		require.NoError(t, err)
		assert.Equal(t, "Docker", resp.Data.Name)
		assert.Equal(t, "docker", resp.Data.Slug)
		assert.Equal(t, "2.1.0", resp.Data.Version)
		assert.Equal(t, "Docker", resp.Data.DashboardFolder)
		assert.NotNil(t, resp.Data.Installation)
		assert.Equal(t, "2.1.0", resp.Data.Installation.Version)
	})

	t.Run("sets auth, content-type, user-agent, and default headers", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "Bearer test-token", r.Header.Get("Authorization"))
			assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
			assert.Equal(t, "test-user-agent", r.Header.Get("User-Agent"))
			assert.Equal(t, "header-value", r.Header.Get("X-Custom"))
			_, _ = w.Write([]byte(`{"data":{}}`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegration(context.Background(), "docker")
		require.NoError(t, err)
	})

	t.Run("returns ErrNotFound on 404", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegration(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cloudintegrationsapi.ErrNotFound)
	})

	t.Run("returns ErrUnauthorized on 401", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegration(context.Background(), "docker")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cloudintegrationsapi.ErrUnauthorized)
	})

	t.Run("returns error on 500", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal"}`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegration(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestUnit_IsIntegrationInstalled(t *testing.T) {
	t.Parallel()

	t.Run("returns true when installed", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
				Data: models.Integration{
					Slug:         "docker",
					Installation: &models.Installation{Version: "1.0.0"},
				},
			})
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		installed, err := c.IsIntegrationInstalled(context.Background(), "docker")
		require.NoError(t, err)
		assert.True(t, installed)
	})

	t.Run("returns false when not installed", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
				Data: models.Integration{Slug: "docker"},
			})
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		installed, err := c.IsIntegrationInstalled(context.Background(), "docker")
		require.NoError(t, err)
		assert.False(t, installed)
	})

	t.Run("propagates error from GetIntegration", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.IsIntegrationInstalled(context.Background(), "docker")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Integrations & Rules API
// ---------------------------------------------------------------------------

func TestUnit_GetIntegrationRules(t *testing.T) {
	t.Parallel()

	t.Run("success with response deserialization", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, integrationsBasePath+"/integrations/docker/rules", r.URL.Path)

			_ = json.NewEncoder(w).Encode(models.IntegrationRulesResponse{
				Data: models.IntegrationRulesData{
					Namespace: "Docker",
					RecordingRules: []models.RuleGroup{
						{Name: "recording_group", Rules: []models.Rule{{Record: "job:up:sum", Expr: "sum(up)"}}},
					},
					AlertingRules: []models.RuleGroup{
						{Name: "alerting_group", Rules: []models.Rule{{Alert: "HighErrors", Expr: "rate(errors[5m]) > 0.1"}}},
					},
				},
			})
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		rules, err := c.GetIntegrationRules(context.Background(), "docker")
		require.NoError(t, err)
		assert.Equal(t, "Docker", rules.Namespace)
		assert.Len(t, rules.RecordingRules, 1)
		assert.Len(t, rules.AlertingRules, 1)
		assert.Equal(t, "recording_group", rules.RecordingRules[0].Name)
		assert.Equal(t, "alerting_group", rules.AlertingRules[0].Name)
	})

	t.Run("returns ErrNotFound on 404", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegrationRules(context.Background(), "nonexistent")
		assert.Error(t, err)
		assert.ErrorIs(t, err, cloudintegrationsapi.ErrNotFound)
	})

	t.Run("returns error on 500", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"error":"internal"}`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		_, err := c.GetIntegrationRules(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestUnit_UninstallIntegration(t *testing.T) {
	t.Parallel()

	dockerIntegration := models.GetIntegrationResponse{
		Data: models.Integration{
			Slug:            "docker",
			Name:            "Docker",
			DashboardFolder: "Docker",
		},
	}

	t.Run("success", func(t *testing.T) {
		t.Parallel()
		var uninstallCalled bool
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/integrations/docker"):
				_ = json.NewEncoder(w).Encode(dockerIntegration)
			case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, rulesConvertPath):
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/integrations/docker/uninstall"):
				uninstallCalled = true
				w.WriteHeader(http.StatusOK)
			default:
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegration(context.Background(), "docker")
		require.NoError(t, err)
		assert.True(t, uninstallCalled, "uninstall API endpoint should be called")
	})

	t.Run("propagates error from GetIntegration", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`server error`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegration(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get integration details")
	})

	t.Run("propagates error from uninstall API", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(dockerIntegration)
			case r.Method == http.MethodDelete:
				w.WriteHeader(http.StatusOK)
			case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/uninstall"):
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"error":"failed"}`))
			default:
				w.WriteHeader(http.StatusOK)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegration(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to uninstall integration docker")
	})
}

// ---------------------------------------------------------------------------
// Rules Installation - Temporary during migration to Grafana Alerting
// ---------------------------------------------------------------------------

func TestUnit_InstallIntegrationRules(t *testing.T) {
	t.Parallel()

	t.Run("success: fetches rules, resolves namespace, posts to convert API", func(t *testing.T) {
		t.Parallel()
		var convertCalled bool
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/integrations/docker/rules"):
				_ = json.NewEncoder(w).Encode(models.IntegrationRulesResponse{
					Data: models.IntegrationRulesData{
						RecordingRules: []models.RuleGroup{
							{Name: "rec", Rules: []models.Rule{{Record: "rec:metric", Expr: "sum(up)"}}},
						},
						AlertingRules: []models.RuleGroup{
							{Name: "alert", Rules: []models.Rule{{Alert: "TestAlert", Expr: "up == 0"}}},
						},
					},
				})
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/integrations/docker"):
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{
						Name:            "Docker",
						DashboardFolder: "Docker",
					},
				})
			case r.Method == http.MethodPost && r.URL.Path == rulesConvertPath:
				convertCalled = true
				assert.Equal(t, cloudPromUID, r.Header.Get("X-Grafana-Alerting-Datasource-UID"))

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				var payload map[string][]models.RuleGroup
				assert.NoError(t, json.Unmarshal(body, &payload))
				assert.Contains(t, payload, "Docker")
				assert.Len(t, payload["Docker"], 2)
				w.WriteHeader(http.StatusOK)
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				w.WriteHeader(http.StatusInternalServerError)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.InstallIntegrationRules(context.Background(), "docker", nil)
		require.NoError(t, err)
		assert.True(t, convertCalled, "convert API should be called")
	})

	t.Run("skips when alerts disabled", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("no requests expected when alerts are disabled, got: %s %s", r.Method, r.URL.Path)
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		config := &models.InstallationConfig{
			ConfigurableAlerts: &models.ConfigurableAlerts{AlertsDisabled: true},
		}
		err := c.InstallIntegrationRules(context.Background(), "docker", config)
		require.NoError(t, err)
	})

	t.Run("skips when no rule groups returned", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/rules"):
				_ = json.NewEncoder(w).Encode(models.IntegrationRulesResponse{
					Data: models.IntegrationRulesData{},
				})
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{Name: "Docker", DashboardFolder: "Docker"},
				})
			default:
				t.Errorf("unexpected request (no POST expected): %s %s", r.Method, r.URL.Path)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.InstallIntegrationRules(context.Background(), "docker", nil)
		require.NoError(t, err)
	})

	t.Run("skips when namespace resolves to empty", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/rules"):
				_ = json.NewEncoder(w).Encode(models.IntegrationRulesResponse{
					Data: models.IntegrationRulesData{
						RecordingRules: []models.RuleGroup{{Name: "rec", Rules: []models.Rule{{Record: "m", Expr: "1"}}}},
					},
				})
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{Name: "", DashboardFolder: "", RuleNamespace: ""},
				})
			default:
				t.Errorf("unexpected request (no POST expected): %s %s", r.Method, r.URL.Path)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.InstallIntegrationRules(context.Background(), "test", nil)
		require.NoError(t, err)
	})

	t.Run("propagates error from GetIntegrationRules", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`server error`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.InstallIntegrationRules(context.Background(), "docker", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get integration rules")
	})

	t.Run("propagates error from convert API", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/rules"):
				_ = json.NewEncoder(w).Encode(models.IntegrationRulesResponse{
					Data: models.IntegrationRulesData{
						AlertingRules: []models.RuleGroup{{Name: "a", Rules: []models.Rule{{Alert: "X", Expr: "1"}}}},
					},
				})
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{Name: "Docker", DashboardFolder: "Docker"},
				})
			case r.Method == http.MethodPost && r.URL.Path == rulesConvertPath:
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`conversion failed`))
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.InstallIntegrationRules(context.Background(), "docker", nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "500")
	})
}

func TestUnit_UninstallIntegrationRules(t *testing.T) {
	t.Parallel()

	t.Run("success: deletes rule namespace", func(t *testing.T) {
		t.Parallel()
		var deleteCalled bool
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/integrations/docker"):
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{Name: "Docker", DashboardFolder: "Docker"},
				})
			case r.Method == http.MethodDelete && strings.HasPrefix(r.URL.Path, rulesConvertPath):
				deleteCalled = true
				assert.Equal(t, rulesConvertPath+"/Docker", r.URL.Path)
				w.WriteHeader(http.StatusOK)
			default:
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegrationRules(context.Background(), "docker")
		require.NoError(t, err)
		assert.True(t, deleteCalled, "DELETE to rules convert API should be called")
	})

	t.Run("ignores 404 on DELETE", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{Name: "Docker", DashboardFolder: "Docker"},
				})
			case r.Method == http.MethodDelete:
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegrationRules(context.Background(), "docker")
		require.NoError(t, err)
	})

	t.Run("skips when namespace resolves to empty", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{},
				})
			default:
				t.Errorf("unexpected request (no DELETE expected): %s %s", r.Method, r.URL.Path)
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegrationRules(context.Background(), "test")
		require.NoError(t, err)
	})

	t.Run("propagates error from GetIntegration", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`server error`))
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegrationRules(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get integration details")
	})

	t.Run("propagates non-404 error from DELETE", func(t *testing.T) {
		t.Parallel()
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.Method == http.MethodGet:
				_ = json.NewEncoder(w).Encode(models.GetIntegrationResponse{
					Data: models.Integration{DashboardFolder: "Docker"},
				})
			case r.Method == http.MethodDelete:
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`server error`))
			}
		}))
		defer svr.Close()

		c := newTestClient(t, svr)
		err := c.UninstallIntegrationRules(context.Background(), "docker")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete rule namespace")
	})
}
