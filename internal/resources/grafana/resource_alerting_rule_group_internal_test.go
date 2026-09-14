package grafana

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
)

func TestUnitUnpackNotificationSettings(t *testing.T) {
	tests := []struct {
		name        string
		input       map[string]any
		wantErr     string
		wantPolicy  string
		wantContact string
	}{
		{
			name:        "contact point only",
			input:       map[string]any{"contact_point": "my-receiver"},
			wantContact: "my-receiver",
		},
		{
			name:       "policy only",
			input:      map[string]any{"policy": "my-policy"},
			wantPolicy: "my-policy",
		},
		{
			name:    "neither set",
			input:   map[string]any{},
			wantErr: `notification_settings: exactly one of "contact_point" or "policy" must be set`,
		},
		{
			name:    "both set",
			input:   map[string]any{"contact_point": "my-receiver", "policy": "my-policy"},
			wantErr: `notification_settings: "contact_point" and "policy" are mutually exclusive`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			settings, err := unpackNotificationSettings([]any{tt.input})
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("expected error %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if settings.Policy != tt.wantPolicy {
				t.Errorf("Policy = %q, want %q", settings.Policy, tt.wantPolicy)
			}
			gotContact := ""
			if settings.Receiver != nil {
				gotContact = *settings.Receiver
			}
			if gotContact != tt.wantContact {
				t.Errorf("Receiver = %q, want %q", gotContact, tt.wantContact)
			}
		})
	}
}

func TestUnitPackNotificationSettings(t *testing.T) {
	receiver := "my-receiver"

	packed, err := packNotificationSettings(&models.AlertRuleNotificationSettings{Policy: "my-policy"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := packed.([]any)[0].(map[string]any)
	if _, ok := result["contact_point"]; ok {
		t.Errorf("expected no contact_point key when routing by policy, got %v", result["contact_point"])
	}
	if result["policy"] != "my-policy" {
		t.Errorf("policy = %v, want %q", result["policy"], "my-policy")
	}

	packed, err = packNotificationSettings(&models.AlertRuleNotificationSettings{Receiver: &receiver})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result = packed.([]any)[0].(map[string]any)
	if _, ok := result["policy"]; ok {
		t.Errorf("expected no policy key when routing by contact point, got %v", result["policy"])
	}
	if result["contact_point"] != receiver {
		t.Errorf("contact_point = %v, want %q", result["contact_point"], receiver)
	}
}
