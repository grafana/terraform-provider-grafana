package grafana

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
)

const testReceiverOpsgenie = "opsgenie"

func TestParseMatcherString(t *testing.T) {
	tests := []struct {
		input    string
		expected models.ObjectMatcher
	}{
		{
			input:    "alertname=~.+",
			expected: models.ObjectMatcher{"alertname", "=~", ".+"},
		},
		{
			input:    "severity!=critical",
			expected: models.ObjectMatcher{"severity", "!=", "critical"},
		},
		{
			input:    "env=production",
			expected: models.ObjectMatcher{"env", "=", "production"},
		},
		{
			input:    "name!~test.*",
			expected: models.ObjectMatcher{"name", "!~", "test.*"},
		},
		{
			input:    "label=value=with=equals",
			expected: models.ObjectMatcher{"label", "=", "value=with=equals"},
		},
		{
			input:    "label=",
			expected: models.ObjectMatcher{"label", "=", ""},
		},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseMatcherString(tc.input)
			if result[0] != tc.expected[0] || result[1] != tc.expected[1] || result[2] != tc.expected[2] {
				t.Errorf("parseMatcherString(%q) = %v, want %v", tc.input, result, tc.expected)
			}
		})
	}
}

func newTestRoute() *models.Route {
	return &models.Route{
		Receiver:       testReceiverOpsgenie,
		GroupBy:        []string{"cluster", "region", "alertname"},
		GroupWait:      "10s",
		GroupInterval:  "1m",
		RepeatInterval: "5m",
		Routes: []*models.Route{
			{
				Receiver: "grafana-oncall",
				Continue: true,
				ObjectMatchers: models.ObjectMatchers{
					{"alertname", "=~", ".+"},
				},
				MuteTimeIntervals:   []string{"weekends"},
				ActiveTimeIntervals: []string{"business-hours"},
			},
		},
	}
}

func TestRouteModelToAMConfig(t *testing.T) {
	original := newTestRoute()
	amConfig, err := routeModelToAMConfig(original)
	if err != nil {
		t.Fatalf("routeModelToAMConfig failed: %v", err)
	}

	if amConfig["receiver"] != testReceiverOpsgenie {
		t.Errorf("expected receiver %q, got %v", testReceiverOpsgenie, amConfig["receiver"])
	}
	groupBy, ok := amConfig["group_by"].([]any)
	if !ok || len(groupBy) != 3 {
		t.Errorf("expected group_by with 3 items, got %v (type %T)", amConfig["group_by"], amConfig["group_by"])
	}
	routes, ok := amConfig["routes"].([]any)
	if !ok || len(routes) != 1 {
		t.Fatalf("expected 1 child route, got %v", amConfig["routes"])
	}
	childRoute := routes[0].(map[string]any)
	if childRoute["receiver"] != "grafana-oncall" {
		t.Errorf("expected child receiver 'grafana-oncall', got %v", childRoute["receiver"])
	}
	if childRoute["continue"] != true {
		t.Errorf("expected continue=true, got %v", childRoute["continue"])
	}
	matchers, ok := childRoute["matchers"].([]string)
	if !ok || len(matchers) != 1 || matchers[0] != "alertname=~.+" {
		t.Errorf("expected matchers ['alertname=~.+'], got %v", childRoute["matchers"])
	}
}

func TestRouteModelToAMConfigRoundTrip(t *testing.T) {
	original := newTestRoute()
	amConfig, err := routeModelToAMConfig(original)
	if err != nil {
		t.Fatalf("routeModelToAMConfig failed: %v", err)
	}

	roundTripped, err := amConfigToRouteModel(amConfig)
	if err != nil {
		t.Fatalf("amConfigToRouteModel failed: %v", err)
	}

	if roundTripped.Receiver != original.Receiver {
		t.Errorf("receiver: got %q, want %q", roundTripped.Receiver, original.Receiver)
	}
	if len(roundTripped.GroupBy) != len(original.GroupBy) {
		t.Errorf("group_by length: got %d, want %d", len(roundTripped.GroupBy), len(original.GroupBy))
	}
	if roundTripped.GroupWait != original.GroupWait {
		t.Errorf("group_wait: got %q, want %q", roundTripped.GroupWait, original.GroupWait)
	}
	if len(roundTripped.Routes) != 1 {
		t.Fatalf("routes length: got %d, want 1", len(roundTripped.Routes))
	}

	child := roundTripped.Routes[0]
	if child.Receiver != "grafana-oncall" {
		t.Errorf("child receiver: got %q, want %q", child.Receiver, "grafana-oncall")
	}
	if !child.Continue {
		t.Error("child continue: got false, want true")
	}
	if len(child.ObjectMatchers) != 1 {
		t.Fatalf("child matchers length: got %d, want 1", len(child.ObjectMatchers))
	}
}

func TestParseContactPointAMConfigID(t *testing.T) {
	tests := []struct {
		id          string
		expectOrgID int64
		expectAMUID string
		expectName  string
	}{
		{
			id:          "1:grafana/my-receiver",
			expectOrgID: 1,
			expectAMUID: "grafana",
			expectName:  "my-receiver",
		},
		{
			id:          "42:grafanacloud-ngalertmanager/opsgenie",
			expectOrgID: 42,
			expectAMUID: "grafanacloud-ngalertmanager",
			expectName:  "opsgenie",
		},
		{
			id:          "0:grafana/test",
			expectOrgID: 0,
			expectAMUID: "grafana",
			expectName:  "test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			orgID, amUID, name := parseContactPointAMConfigID(tc.id)
			if orgID != tc.expectOrgID {
				t.Errorf("orgID: got %d, want %d", orgID, tc.expectOrgID)
			}
			if amUID != tc.expectAMUID {
				t.Errorf("amUID: got %q, want %q", amUID, tc.expectAMUID)
			}
			if name != tc.expectName {
				t.Errorf("name: got %q, want %q", name, tc.expectName)
			}
		})
	}
}

func TestIsGrafanaManagedAM(t *testing.T) {
	tests := []struct {
		name     string
		amUID    string
		amConfig map[string]any
		expected bool
	}{
		{
			name:     "built-in grafana AM is always managed",
			amUID:    "grafana",
			amConfig: map[string]any{},
			expected: true,
		},
		{
			name:     "grafanacloud-ngalertmanager without grafana_managed_receiver_configs is not managed",
			amUID:    "grafanacloud-ngalertmanager",
			amConfig: map[string]any{},
			expected: false,
		},
		{
			name:  "AM with grafana_managed_receiver_configs is managed",
			amUID: "custom-grafana-am",
			amConfig: map[string]any{
				"receivers": []any{
					map[string]any{
						"name":                             "default",
						"grafana_managed_receiver_configs": []any{},
					},
				},
			},
			expected: true,
		},
		{
			name:  "AM with native configs is not managed",
			amUID: "my-external-am",
			amConfig: map[string]any{
				"receivers": []any{
					map[string]any{
						"name": "default",
						"opsgenie_configs": []any{
							map[string]any{"api_key": "xxx"},
						},
					},
				},
			},
			expected: false,
		},
		{
			name:     "AM with no receivers and unknown UID is not managed",
			amUID:    "some-external-am",
			amConfig: map[string]any{},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := isGrafanaManagedAM(tc.amUID, tc.amConfig)
			if result != tc.expected {
				t.Errorf("isGrafanaManagedAM(%q, ...) = %v, want %v", tc.amUID, result, tc.expected)
			}
		})
	}
}

func TestEmbeddedContactPointToNativeConfig(t *testing.T) {
	t.Run(testReceiverOpsgenie, func(t *testing.T) {
		typ := testReceiverOpsgenie
		p := &models.EmbeddedContactPoint{
			UID:                   "uid1",
			Name:                  "test",
			Type:                  &typ,
			DisableResolveMessage: false,
			Settings: map[string]any{
				"apiKey":  "secret",
				"apiUrl":  "https://api.eu.opsgenie.com/",
				"message": "{{ .CommonAnnotations.summary }}",
			},
		}

		configKey, native, err := embeddedContactPointToNativeConfig(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if configKey != "opsgenie_configs" {
			t.Errorf("configKey: got %q, want %q", configKey, "opsgenie_configs")
		}
		if native["api_key"] != "secret" {
			t.Errorf("api_key: got %v, want %q", native["api_key"], "secret")
		}
		if native["api_url"] != "https://api.eu.opsgenie.com/" {
			t.Errorf("api_url: got %v, want %q", native["api_url"], "https://api.eu.opsgenie.com/")
		}
		if native["message"] != "{{ .CommonAnnotations.summary }}" {
			t.Errorf("message: got %v", native["message"])
		}
		if native["send_resolved"] != true {
			t.Errorf("send_resolved: got %v, want true", native["send_resolved"])
		}
	})

	t.Run("skips false booleans", func(t *testing.T) {
		typ := testReceiverOpsgenie
		p := &models.EmbeddedContactPoint{
			Type: &typ,
			Settings: map[string]any{
				"apiKey":    "secret",
				"autoClose": false,
			},
		}

		_, native, err := embeddedContactPointToNativeConfig(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if _, ok := native["auto_close"]; ok {
			t.Error("auto_close should be omitted when false")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		typ := "oncall"
		p := &models.EmbeddedContactPoint{
			Type:     &typ,
			Settings: map[string]any{},
		}

		_, _, err := embeddedContactPointToNativeConfig(p)
		if err == nil {
			t.Error("expected error for unsupported type")
		}
	})

	t.Run("filters grafana-only settings", func(t *testing.T) {
		typ := testReceiverOpsgenie
		p := &models.EmbeddedContactPoint{
			Type: &typ,
			Settings: map[string]any{
				"apiKey":           "secret",
				"message":          "test",
				"overridePriority": true,
				"sendTagsAs":       "tags",
			},
		}

		_, native, err := embeddedContactPointToNativeConfig(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Valid native fields should be present
		if native["api_key"] != "secret" {
			t.Errorf("api_key: got %v, want %q", native["api_key"], "secret")
		}
		if native["message"] != "test" {
			t.Errorf("message: got %v, want %q", native["message"], "test")
		}
		// Grafana-only fields should be filtered out
		if _, ok := native["override_priority"]; ok {
			t.Error("override_priority should be filtered out")
		}
		if _, ok := native["send_tags_as"]; ok {
			t.Error("send_tags_as should be filtered out")
		}
	})

	t.Run("maps og_priority to priority for native AM", func(t *testing.T) {
		typ := testReceiverOpsgenie
		p := &models.EmbeddedContactPoint{
			Type: &typ,
			Settings: map[string]any{
				"apiKey":      "secret",
				"og_priority": "{{ .CommonAnnotations.priority }}",
			},
		}

		_, native, err := embeddedContactPointToNativeConfig(p)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// og_priority should be mapped to priority
		if native["priority"] != "{{ .CommonAnnotations.priority }}" {
			t.Errorf("priority: got %v, want %q", native["priority"], "{{ .CommonAnnotations.priority }}")
		}
		// og_priority should not exist
		if _, ok := native["og_priority"]; ok {
			t.Error("og_priority should be mapped to priority, not kept")
		}
	})
}

func TestNativeConfigToEmbeddedContactPoint(t *testing.T) {
	t.Run("basic conversion", func(t *testing.T) {
		config := map[string]any{
			"api_key":       "secret",
			"api_url":       "https://api.eu.opsgenie.com/",
			"message":       "test message",
			"send_resolved": false,
		}

		p := nativeConfigToEmbeddedContactPoint(testReceiverOpsgenie, "test-receiver", config)

		if *p.Type != testReceiverOpsgenie {
			t.Errorf("Type: got %q, want %q", *p.Type, testReceiverOpsgenie)
		}
		if p.Name != "test-receiver" {
			t.Errorf("Name: got %q, want %q", p.Name, "test-receiver")
		}
		if !p.DisableResolveMessage {
			t.Error("DisableResolveMessage: got false, want true (send_resolved was false)")
		}

		settings := p.Settings.(map[string]any)
		if settings["apiKey"] != "secret" {
			t.Errorf("apiKey: got %v, want %q", settings["apiKey"], "secret")
		}
		if settings["apiUrl"] != "https://api.eu.opsgenie.com/" {
			t.Errorf("apiUrl: got %v, want %q", settings["apiUrl"], "https://api.eu.opsgenie.com/")
		}
		if settings["message"] != "test message" {
			t.Errorf("message: got %v, want %q", settings["message"], "test message")
		}
		if _, ok := settings["sendResolved"]; ok {
			t.Error("sendResolved should not be in settings (extracted to DisableResolveMessage)")
		}
	})

	t.Run("maps priority to og_priority for Grafana", func(t *testing.T) {
		config := map[string]any{
			"api_key":  "secret",
			"priority": "{{ .CommonAnnotations.priority }}",
		}

		p := nativeConfigToEmbeddedContactPoint(testReceiverOpsgenie, "test-receiver", config)

		settings := p.Settings.(map[string]any)
		// priority should be mapped to og_priority
		if settings["og_priority"] != "{{ .CommonAnnotations.priority }}" {
			t.Errorf("og_priority: got %v, want %q", settings["og_priority"], "{{ .CommonAnnotations.priority }}")
		}
		// priority should not exist (mapped to og_priority)
		if _, ok := settings["priority"]; ok {
			t.Error("priority should be mapped to og_priority, not kept")
		}
	})
}

func TestParseNotificationPolicyAMConfigID(t *testing.T) {
	tests := []struct {
		id          string
		expectOrgID int64
		expectAMUID string
	}{
		{
			id:          "1:grafana/policy",
			expectOrgID: 1,
			expectAMUID: "grafana",
		},
		{
			id:          "42:grafanacloud-ngalertmanager/policy",
			expectOrgID: 42,
			expectAMUID: "grafanacloud-ngalertmanager",
		},
	}

	for _, tc := range tests {
		t.Run(tc.id, func(t *testing.T) {
			orgID, amUID := parseNotificationPolicyAMConfigID(tc.id)
			if orgID != tc.expectOrgID {
				t.Errorf("orgID: got %d, want %d", orgID, tc.expectOrgID)
			}
			if amUID != tc.expectAMUID {
				t.Errorf("amUID: got %q, want %q", amUID, tc.expectAMUID)
			}
		})
	}
}

// TestNonEmptyNotifier verifies that the nonEmptyNotifier function correctly
// identifies empty vs non-empty notifier data. This is critical for the AM Config
// path where empty notifier entries without UIDs should be skipped.
func TestNonEmptyNotifier(t *testing.T) {
	notifier := opsGenieNotifier{}

	tests := []struct {
		name     string
		data     map[string]any
		expected bool
	}{
		{
			name:     "empty data is not non-empty",
			data:     map[string]any{},
			expected: false,
		},
		{
			name: "data with only optional fields is not non-empty",
			data: map[string]any{
				"auto_close":        false,
				"override_priority": false,
				"responders":        []any{},
			},
			expected: false,
		},
		{
			name: "data with required field (api_key) is non-empty",
			data: map[string]any{
				"api_key": "secret-key",
			},
			expected: true,
		},
		{
			name: "data with empty string for required field is not non-empty",
			data: map[string]any{
				"api_key": "",
			},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := nonEmptyNotifier(notifier, tc.data)
			if result != tc.expected {
				t.Errorf("nonEmptyNotifier(...) = %v, want %v", result, tc.expected)
			}
		})
	}
}
