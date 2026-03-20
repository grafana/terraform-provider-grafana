package grafana

import "testing"

func TestIsKubernetesStyleDashboard(t *testing.T) {
	t.Run("legacy dashboard", func(t *testing.T) {
		if isKubernetesStyleDashboard(map[string]any{"title": "legacy"}) {
			t.Fatal("expected legacy dashboard shape to be treated as non-kubernetes")
		}
	})

	t.Run("kubernetes style dashboard", func(t *testing.T) {
		if !isKubernetesStyleDashboard(map[string]any{
			"apiVersion": "dashboard.grafana.app/v2beta1",
			"kind":       "Dashboard",
			"metadata": map[string]any{
				"name": "test-dashboard",
			},
			"spec": map[string]any{
				"title": "test dashboard",
			},
		}) {
			t.Fatal("expected kubernetes dashboard shape to be detected")
		}
	})
}

func TestNormalizeDashboardConfigJSONForState(t *testing.T) {
	t.Run("preserves kubernetes dashboard shape when remote body matches local spec", func(t *testing.T) {
		configJSON := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"test dashboard"}}`
		remoteDashJSON := map[string]any{
			"title":   "test dashboard",
			"id":      7,
			"uid":     "test-dashboard",
			"version": 3,
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != NormalizeDashboardConfigJSON(configJSON) {
			t.Fatalf("expected kubernetes-shaped config to be preserved, got %s", got)
		}
	})

	t.Run("stores remote dashboard body under spec when kubernetes dashboard drifts", func(t *testing.T) {
		configJSON := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"local dashboard"}}`
		remoteDashJSON := map[string]any{
			"title":   "remote dashboard",
			"id":      11,
			"uid":     "test-dashboard",
			"version": 5,
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := `{"apiVersion":"dashboard.grafana.app/v2beta1","kind":"Dashboard","metadata":{"name":"test-dashboard"},"spec":{"title":"remote dashboard"}}`
		if got != want {
			t.Fatalf("expected remote dashboard body to be stored under spec, got %s", got)
		}
	})

	t.Run("drops generated uid for legacy dashboard config without uid", func(t *testing.T) {
		configJSON := `{"title":"legacy dashboard"}`
		remoteDashJSON := map[string]any{
			"title": "legacy dashboard",
			"uid":   "generated-uid",
		}

		got, err := normalizeDashboardConfigJSONForState(configJSON, remoteDashJSON)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != `{"title":"legacy dashboard"}` {
			t.Fatalf("expected generated uid to be removed, got %s", got)
		}
	})
}
