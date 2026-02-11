package grafana

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestAMConfigClient_Get(t *testing.T) {
	t.Run("successfully fetches config", func(t *testing.T) {
		expectedConfig := map[string]any{
			"route": map[string]any{
				"receiver": "default",
			},
			"receivers": []any{
				map[string]any{"name": "default"},
			},
		}

		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Errorf("expected GET, got %s", r.Method)
			}
			if r.URL.Path != "/api/alertmanager/grafana/config/api/v1/alerts" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
			}
			if r.Header.Get(orgIDHeader) != "1" {
				t.Errorf("unexpected org ID header: %s", r.Header.Get(orgIDHeader))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(expectedConfig)
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		config, err := client.Get(context.Background(), 1, "grafana")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		route, ok := config["route"].(map[string]any)
		if !ok {
			t.Fatalf("expected route in config, got %v", config)
		}
		if route["receiver"] != "default" {
			t.Errorf("expected receiver 'default', got %v", route["receiver"])
		}
	})

	t.Run("returns error on 404", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		_, err := client.Get(context.Background(), 1, "unknown-am")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != `alertmanager "unknown-am" not found` {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("returns error on server error", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("internal error"))
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		_, err := client.Get(context.Background(), 1, "grafana")
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "failed to get alertmanager config, status 500: internal error" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("uses basic auth when no API key", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok {
				t.Error("expected basic auth")
			}
			if user != "admin" || pass != "password" {
				t.Errorf("unexpected credentials: %s:%s", user, pass)
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:   svr.Client(),
			baseURL:  *svrURL,
			username: "admin",
			password: "password",
		}

		_, err := client.Get(context.Background(), 1, "grafana")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("omits org header when orgID is 0", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get(orgIDHeader) != "" {
				t.Error("expected no org ID header when orgID is 0")
			}

			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		_, err := client.Get(context.Background(), 0, "grafana")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestAMConfigClient_Post(t *testing.T) {
	t.Run("successfully posts config", func(t *testing.T) {
		configToPost := map[string]any{
			"route": map[string]any{
				"receiver": "updated",
			},
		}

		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if r.URL.Path != "/api/alertmanager/grafana/config/api/v1/alerts" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if r.Header.Get("Content-Type") != "application/json" {
				t.Errorf("unexpected Content-Type: %s", r.Header.Get("Content-Type"))
			}
			if r.Header.Get("Authorization") != "Bearer test-api-key" {
				t.Errorf("unexpected Authorization header: %s", r.Header.Get("Authorization"))
			}

			body, _ := io.ReadAll(r.Body)
			var received map[string]any
			json.Unmarshal(body, &received)

			route, ok := received["route"].(map[string]any)
			if !ok || route["receiver"] != "updated" {
				t.Errorf("unexpected request body: %s", string(body))
			}

			w.WriteHeader(http.StatusAccepted)
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		err := client.Post(context.Background(), 1, "grafana", configToPost)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("returns error on server error", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("invalid config"))
		}))
		defer svr.Close()

		svrURL, _ := url.Parse(svr.URL)
		client := &AMConfigClient{
			client:  svr.Client(),
			baseURL: *svrURL,
			apiKey:  "test-api-key",
		}

		err := client.Post(context.Background(), 1, "grafana", map[string]any{})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if err.Error() != "failed to post alertmanager config, status 400: invalid config" {
			t.Errorf("unexpected error message: %v", err)
		}
	})

	t.Run("accepts 200, 201, and 202 status codes", func(t *testing.T) {
		statusCodes := []int{http.StatusOK, http.StatusCreated, http.StatusAccepted}

		for _, statusCode := range statusCodes {
			t.Run(http.StatusText(statusCode), func(t *testing.T) {
				svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(statusCode)
				}))
				defer svr.Close()

				svrURL, _ := url.Parse(svr.URL)
				client := &AMConfigClient{
					client:  svr.Client(),
					baseURL: *svrURL,
					apiKey:  "test-api-key",
				}

				err := client.Post(context.Background(), 1, "grafana", map[string]any{})
				if err != nil {
					t.Errorf("unexpected error for status %d: %v", statusCode, err)
				}
			})
		}
	})
}
