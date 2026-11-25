package grafana

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/require"
)

// https://github.com/grafana/terraform-provider-grafana/issues/2864
func TestUnitPackContactPoints_webhookHMACConfigWithoutState(t *testing.T) {
	tests := []struct {
		name       string
		hmacConfig map[string]any
		want       []any
	}{
		{
			name:       "only redacted secret",
			hmacConfig: map[string]any{"secret": "[REDACTED]"},
			want:       []any{},
		},
		{
			name:       "empty",
			hmacConfig: map[string]any{},
			want:       []any{},
		},
		{
			name: "secret and headers",
			hmacConfig: map[string]any{
				"secret":          "[REDACTED]",
				"header":          "X-Grafana-Alerting-Signature",
				"timestampHeader": "X-Grafana-Alerting-Timestamp",
			},
			want: []any{map[string]any{
				"secret":           "",
				"header":           "X-Grafana-Alerting-Signature",
				"timestamp_header": "X-Grafana-Alerting-Timestamp",
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := resourceContactPoint().Schema.TestResourceData()
			p := &models.EmbeddedContactPoint{
				UID:  "webhook-uid",
				Name: "hmac-repro",
				Type: common.Ref("webhook"),
				Settings: map[string]any{
					"url":        "https://example.com/hook",
					"hmacConfig": tt.hmacConfig,
				},
			}

			require.NotPanics(t, func() {
				require.NoError(t, packContactPoints([]*models.EmbeddedContactPoint{p}, data))
			})

			webhooks := data.Get("webhook").(*schema.Set).List()
			require.Len(t, webhooks, 1)
			require.Equal(t, tt.want, webhooks[0].(map[string]any)["hmac_config"].(*schema.Set).List())
		})
	}
}
