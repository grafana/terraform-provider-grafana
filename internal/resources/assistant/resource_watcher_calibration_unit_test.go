package assistant

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnitWatcherCalibrationRecovery(t *testing.T) {
	for _, scenario := range []string{"completed", "lost_response", "running", "failed", "superseded", "timeout", "forbidden"} {
		t.Run(scenario, func(t *testing.T) {
			posts, gets := 0, 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				var data any
				switch {
				case strings.HasSuffix(req.URL.Path, "/initial-calibration"):
					if req.Method == "POST" {
						posts++
						if scenario == "lost_response" {
							w.WriteHeader(http.StatusBadGateway)
							return
						}
						data = map[string]any{"status": "running", "unattended": true, "startWhenReady": false, "startedAt": "first"}
					} else {
						gets++
						status := "idle"
						attempt := "first"
						if scenario == "running" || scenario == "superseded" || scenario == "timeout" {
							status = "running"
						}
						if scenario == "completed" || scenario == "lost_response" && posts > 0 || scenario == "running" && gets > 1 {
							status = "completed"
						}
						if scenario == "failed" && gets > 1 {
							status = "failed"
						}
						if scenario == "superseded" && gets > 1 {
							attempt = "second"
						}
						data = map[string]any{"status": status, "unattended": true, "startWhenReady": false, "startedAt": attempt, "failureReason": "no_checks"}
					}
				default:
					agent := map[string]any{"id": "w", "status": "draft"}
					if scenario == "completed" {
						agent["status"] = "ready"
						agent["calibratedAt"] = "2026-11-05T12:00:00Z"
					}
					data = agent
				}
				_ = json.NewEncoder(w).Encode(map[string]any{"data": data})
			}))
			defer server.Close()
			client, err := assistantapi.NewClient(server.URL, nil, "token", server.Client(), "", nil)
			if err != nil {
				t.Fatal(err)
			}
			r := watcherCalibrationResource{client: client, pollInterval: time.Millisecond}
			ctx := context.Background()
			if scenario == "timeout" {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, 10*time.Millisecond)
				defer cancel()
			}
			status, err := r.ensureCalibration(ctx, watcherCalibrationModel{WatcherID: types.StringValue("w"), Started: types.BoolValue(false), TimeoutSeconds: types.Int64Value(2)})
			wantError := scenario == "failed" || scenario == "superseded" || scenario == "timeout" || scenario == "forbidden"
			if (err != nil) != wantError {
				t.Fatalf("status=%+v error=%v", status, err)
			}
			expectedPosts := 0
			if scenario == "lost_response" || scenario == "failed" || scenario == "forbidden" {
				expectedPosts = 1
			}
			if posts != expectedPosts {
				t.Fatalf("POSTs=%d, want %d; do not replay or restart existing calibration", posts, expectedPosts)
			}
		})
	}
}

func TestUnitWatcherActionUpdatePreservesUnmodeledSettings(t *testing.T) {
	live := &assistantapi.WatcherActions{
		MSTeams: json.RawMessage(`{"enabled":true,"minSeverity":"critical","target":{"teamId":"team","channelId":"channel"}}`),
		Slack:   &assistantapi.WatcherSlackAction{Enabled: true, MinSeverity: "critical", Target: &assistantapi.WatcherSlackTarget{Type: "channel", ChannelID: "channel", ThreadTS: "thread"}},
	}
	wanted := &assistantapi.WatcherActions{Slack: &assistantapi.WatcherSlackAction{Enabled: false, Target: &assistantapi.WatcherSlackTarget{Type: "channel", ChannelID: "channel"}}}
	got := mergeWatcherActions(wanted, live)
	if string(got.MSTeams) != string(live.MSTeams) || got.Slack.MinSeverity != "critical" || got.Slack.Target.ThreadTS != "thread" {
		t.Fatalf("lost unmodeled settings: %+v", got)
	}
	removed := mergeWatcherActions(nil, live)
	data, _ := json.Marshal(removed)
	if removed.Slack != nil || removed.Investigation != nil || string(removed.MSTeams) != string(live.MSTeams) {
		t.Fatalf("incorrect removal patch %s", data)
	}
}
