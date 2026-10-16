package oncall

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnitOnCallEscalation_scheduleNotifications(t *testing.T) {
	for _, stepType := range []string{
		"notify_on_call_from_schedule",
		"notify_next_on_call_from_schedule",
		"notify_previous_on_call_from_schedule",
	} {
		for _, method := range []string{http.MethodPost, http.MethodPut} {
			for _, important := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/important=%t", stepType, method, important), func(t *testing.T) {
					resource := resourceEscalation().Schema
					_, errs := resource.Schema["type"].ValidateFunc(stepType, "type")
					require.Empty(t, errs)

					requests := make(chan map[string]interface{}, 1)
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						if r.Method != http.MethodGet {
							assert.Equal(t, method, r.Method)
							var body map[string]interface{}
							if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
								t.Errorf("decode escalation request: %v", err)
								w.WriteHeader(http.StatusBadRequest)
								return
							}
							requests <- body
						}
						w.Header().Set("Content-Type", "application/json")
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]interface{}{
							"id": "step-id", "escalation_chain_id": "chain-id", "position": 0,
							"type": stepType, "notify_on_call_from_schedule": "schedule-id", "important": important,
						}))
					}))
					defer server.Close()
					client, err := onCallAPI.New(server.URL, "test-token")
					require.NoError(t, err)
					data := schema.TestResourceDataRaw(t, resource.Schema, map[string]interface{}{
						"escalation_chain_id": "chain-id", "position": 0, "type": stepType,
						"notify_on_call_from_schedule": "schedule-id", "important": important,
					})
					if method == http.MethodPost {
						require.Empty(t, resourceEscalationCreate(context.Background(), data, client))
					} else {
						data.SetId("step-id")
						require.Empty(t, resourceEscalationUpdate(context.Background(), data, client))
					}
					select {
					case body := <-requests:
						assert.Equal(t, stepType, body["type"])
						assert.Equal(t, "schedule-id", body["notify_on_call_from_schedule"])
						assert.Equal(t, important, body["important"])
					default:
						t.Fatal("expected an escalation write request")
					}
					assert.Equal(t, "step-id", data.Id())
					assert.Equal(t, "schedule-id", data.Get("notify_on_call_from_schedule"))
					assert.Equal(t, important, data.Get("important"))
				})
			}
		}
	}
}
