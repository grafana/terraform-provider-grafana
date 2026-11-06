package assistantapi

import (
	"context"
	"encoding/json"
	"fmt"
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

func TestClientWatcherRecoversLostCreateResponse(t *testing.T) {
	const key = "b6ceda5d-91d8-48f9-8cc7-a7873a31c88a"
	creates := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			creates++
			w.WriteHeader(http.StatusBadGateway)
		case r.URL.Path == pathPrefix+"/watcher-agents":
			if r.URL.Query().Get("createdByMe") != "true" {
				t.Error("recovery must be restricted to caller-created watchers")
			}
			writeJSON(t, w, map[string]any{"data": map[string]any{"agents": []Watcher{{ID: "recovered", RequestID: key, Name: "test"}}}})
		case r.URL.Path == pathPrefix+"/watcher-agents/recovered":
			writeJSON(t, w, map[string]any{"data": Watcher{ID: "recovered", RequestID: key, Name: "test"}})
		default:
			t.Errorf("unexpected recovery request %s", r.URL)
		}
	})
	watcher, err := client.CreateWatcher(context.Background(), WatcherCreate{RequestID: key, Name: "test", Prompt: "alerts"})
	if err != nil || watcher.ID != "recovered" || creates != 1 {
		t.Fatalf("watcher=%+v err=%v creates=%d", watcher, err, creates)
	}
}

func TestClientWatcherDoesNotRecoverRejectedCreate(t *testing.T) {
	for _, code := range []int{400, 401, 403, 409, 422, 429} {
		t.Run(fmt.Sprint(code), func(t *testing.T) {
			reads := 0
			client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodGet {
					reads++
					writeJSON(t, w, map[string]any{"data": map[string]any{"agents": []Watcher{{ID: "another-callers-watcher", RequestID: "same-key"}}}})
					return
				}
				w.WriteHeader(code)
			})
			_, err := client.CreateWatcher(context.Background(), WatcherCreate{RequestID: "same-key", Name: "test", Prompt: "alerts"})
			if err == nil || reads != 0 {
				t.Fatalf("rejected create must not adopt a visible Watcher: err=%v reads=%d", err, reads)
			}
		})
	}
}
