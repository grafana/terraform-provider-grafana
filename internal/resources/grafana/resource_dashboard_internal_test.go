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
