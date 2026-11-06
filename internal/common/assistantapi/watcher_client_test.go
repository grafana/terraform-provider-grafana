package assistantapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hashicorp/go-retryablehttp"
)

func TestClientWatcherMutationsDoNotReplay(t *testing.T) {
	for _, operation := range []string{"create", "calibration", "start"} {
		t.Run(operation, func(t *testing.T) {
			var calls atomic.Int64
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(http.StatusBadGateway) }))
			defer server.Close()
			retry := retryablehttp.NewClient()
			retry.RetryMax = 3
			retry.RetryWaitMin = time.Millisecond
			retry.RetryWaitMax = time.Millisecond
			client, err := NewClient(server.URL, nil, "token", retry.StandardClient(), "", nil)
			if err != nil {
				t.Fatal(err)
			}
			switch operation {
			case "create":
				_, err = client.CreateWatcher(context.Background(), WatcherCreate{Name: "test", Prompt: "test"})
			case "calibration":
				_, err = client.CalibrateWatcher(context.Background(), "id", WatcherCalibrationRequest{Unattended: true})
			case "start":
				_, err = client.StartWatcher(context.Background(), "id")
			}
			if err == nil || calls.Load() != 1 {
				t.Fatalf("error=%v, requests=%d; unsafe POST must be sent exactly once", err, calls.Load())
			}
		})
	}
}
func TestClientWatcherPatchDoesNotOwnGeneratedDefinition(t *testing.T) {
	patch := WatcherUpdate{Description: ptr("renamed"), ExpectedDefinitionVersion: ptr(int64(7))}
	data, err := json.Marshal(patch)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err = json.Unmarshal(data, &fields); err != nil {
		t.Fatal(err)
	}
	if len(fields) != 2 || fields["queries"] != nil || fields["calibrationContext"] != nil {
		t.Fatalf("unexpected patch: %s", data)
	}
}
func ptr[T any](v T) *T { return &v }

// A create error must never adopt a similarly named Watcher or resend POST.
func TestClientWatcherCreateFailureDoesNotRecover(t *testing.T) {
	for _, code := range []int{400, 401, 403, 409, 422, 429, 502} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			calls := 0
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPost {
					t.Errorf("unexpected recovery request %s", r.Method)
				}
				var body map[string]any
				if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
					t.Error(err)
				}
				if _, ok := body["requestId"]; ok {
					t.Error("unsupported creation key sent")
				}
				if _, ok := body["teamUid"]; ok {
					t.Error("unsupported team assignment sent")
				}
				w.WriteHeader(code)
			})
			_, err := client.CreateWatcher(context.Background(), WatcherCreate{Name: "test", Prompt: "alerts"})
			if err == nil || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

// #15241 accepts the identical PUT after it has committed. Test the real HTTP
// retry transport, including the original guard and combined cadence/settings.
func TestClientWatcherUpdateRetry(t *testing.T) {
	for _, code := range []int{502, 403, 409} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			patch := WatcherUpdate{
				ExpectedDefinitionVersion: ptr(int64(7)), TriggerIntervalSeconds: ptr(int64(1800)),
				Labels:   ptr(map[string]string{"service": "checkout"}),
				Actions:  &WatcherActions{Investigation: &WatcherInvestigationAction{Enabled: true, TeamNames: []string{"Platform"}}},
				AutoStop: &WatcherAutoStop{PauseAt: "2099-01-01T00:00:00Z"},
			}
			original, err := json.Marshal(patch)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				if r.Method != http.MethodPut {
					t.Errorf("unexpected method %s", r.Method)
				}
				body, readErr := io.ReadAll(r.Body)
				if readErr != nil {
					t.Error(readErr)
				}
				if !bytes.Equal(body, original) {
					t.Errorf("retry changed body or version guard: %s", body)
				}
				if calls == 1 {
					w.WriteHeader(code)
					return
				}
				writeJSON(t, w, map[string]any{"data": Watcher{ID: "w", DefinitionVersion: 8, Status: WatcherStatusCalibrating, TriggerIntervalSeconds: 1800, Labels: *patch.Labels, Actions: patch.Actions, AutoStop: patch.AutoStop}})
			}))
			defer server.Close()
			retry := retryablehttp.NewClient()
			retry.RetryMax = 2
			retry.RetryWaitMin = time.Millisecond
			retry.RetryWaitMax = time.Millisecond
			client, err := NewClient(server.URL, nil, "token", retry.StandardClient(), "", nil)
			if err != nil {
				t.Fatal(err)
			}
			got, err := client.UpdateWatcher(context.Background(), "w", patch)
			if code == 502 {
				if err != nil || calls != 2 || got.DefinitionVersion != 8 || got.Status != WatcherStatusCalibrating {
					t.Fatalf("got=%+v err=%v calls=%d", got, err, calls)
				}
			} else if err == nil || calls != 1 {
				t.Fatalf("definitive failure retried: err=%v calls=%d", err, calls)
			}
		})
	}
}
