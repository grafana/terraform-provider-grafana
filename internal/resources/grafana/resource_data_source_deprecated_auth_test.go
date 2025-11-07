package grafana

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestCheckDeprecatedPrometheusAuth_SigV4(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"sigV4Auth":true,"sigV4Region":"us-east-1"}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 1 {
		t.Errorf("Expected 1 diagnostic, got %d", len(diags))
	}

	if len(diags) > 0 {
		if diags[0].Severity != 1 { // Warning
			t.Errorf("Expected warning severity, got %d", diags[0].Severity)
		}
		if diags[0].Summary != "Deprecated authentication method" {
			t.Errorf("Unexpected summary: %s", diags[0].Summary)
		}
	}
}

func TestCheckDeprecatedPrometheusAuth_Azure(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"azureAuth":true}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 1 {
		t.Errorf("Expected 1 diagnostic, got %d", len(diags))
	}

	if len(diags) > 0 {
		if diags[0].Severity != 1 { // Warning
			t.Errorf("Expected warning severity, got %d", diags[0].Severity)
		}
		if diags[0].Summary != "Deprecated authentication method" {
			t.Errorf("Unexpected summary: %s", diags[0].Summary)
		}
	}
}

func TestCheckDeprecatedPrometheusAuth_NoDeprecatedAuth(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"httpMethod":"POST"}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 0 {
		t.Errorf("Expected 0 diagnostics, got %d", len(diags))
	}
}

func TestCheckDeprecatedPrometheusAuth_NonPrometheusDataSource(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-loki",
		"type":              "loki",
		"url":               "http://localhost:3100",
		"json_data_encoded": `{"sigV4Auth":true}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 0 {
		t.Errorf("Expected 0 diagnostics for non-Prometheus datasource, got %d", len(diags))
	}
}

func TestCheckDeprecatedPrometheusAuth_EmptyJSONData(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name": "test-prometheus",
		"type": "prometheus",
		"url":  "http://localhost:9090",
	})

	diags := checkDeprecatedPrometheusAuth(d)

	if len(diags) != 0 {
		t.Errorf("Expected 0 diagnostics for empty JSON data, got %d", len(diags))
	}
}

func TestCheckDeprecatedPrometheusAuth_BothAuthMethods(t *testing.T) {
	d := schema.TestResourceDataRaw(t, resourceDataSource().Schema.Schema, map[string]interface{}{
		"name":              "test-prometheus",
		"type":              "prometheus",
		"url":               "http://localhost:9090",
		"json_data_encoded": `{"sigV4Auth":true,"azureAuth":true}`,
	})

	diags := checkDeprecatedPrometheusAuth(d)

	// Should get warnings for both auth methods
	if len(diags) != 2 {
		t.Errorf("Expected 2 diagnostics, got %d", len(diags))
	}
}
