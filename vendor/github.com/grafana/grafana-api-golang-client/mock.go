package gapi

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

type mockServer struct {
	code   int
	server *httptest.Server
}

func (m *mockServer) Close() {
	m.server.Close()
}

func gapiTestTools(t *testing.T, code int, body string) (*mockServer, *Client) {
	t.Helper()

	mock := &mockServer{
		code: code,
	}

	mock.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(mock.code)
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, body)
	}))

	tr := &http.Transport{
		Proxy: func(req *http.Request) (*url.URL, error) {
			return url.Parse(mock.server.URL)
		},
	}

	httpClient := &http.Client{Transport: tr}

	client, err := New("http://my-grafana.com", Config{APIKey: "my-key", Client: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return mock, client
}
