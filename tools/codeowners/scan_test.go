package main

import "testing"

func TestExtractAppPlatformKind(t *testing.T) {
	tests := []struct {
		tfName string
		want   string
	}{
		{"grafana_apps_alerting_alertrule_v0alpha1", "alertrule"},
		{"grafana_apps_notifications_inhibitionrule_v1beta1", "inhibitionrule"},
		{"grafana_apps_rules_recordingrule_v0alpha1", "recordingrule"},
		{"grafana_apps_secret_keeper_v1beta1", "keeper"},
		{"grafana_apps_secret_keeper_activation_v1beta1", "keeper_activation"},
		{"grafana_apps_dashboard_dashboard_v1beta1", "dashboard"},
		{"grafana_apps_productactivation_appo11yconfig_v1alpha1", "appo11yconfig"},
		{"grafana_apps_provisioning_connection_v0alpha1", "connection"},
		{"grafana_apps_generic_resource", ""}, // no version part
		{"grafana_dashboard", ""},             // not an appplatform resource
	}

	for _, tt := range tests {
		got := extractAppPlatformKind(tt.tfName)
		if got != tt.want {
			t.Errorf("extractAppPlatformKind(%q) = %q, want %q", tt.tfName, got, tt.want)
		}
	}
}
