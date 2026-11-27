package adaptivetracesapi

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAuth       = "12345:glc_abc"
	testAuthHeader = "Basic MTIzNDU6Z2xjX2FiYw=="
)

func newTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()

	svr := httptest.NewServer(handler)
	t.Cleanup(svr.Close)

	return NewClient(testAuth, svr.URL, svr.Client(), "test-user-agent", nil)
}

func writeJSON(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, err := w.Write([]byte(body))
	assert.NoError(t, err)
}

func readBody(t *testing.T, r *http.Request) []byte {
	t.Helper()

	bs, err := io.ReadAll(r.Body)
	require.NoError(t, err)
	return bs
}

// policyOperations calls every client method that returns a policy, with an arbitrary policy.
var policyOperations = []struct {
	name   string
	method string
	path   string
	call   func(c *Client) (*Policy, error)
}{
	{"GetPolicy", http.MethodGet, "/adaptive-traces/api/v1/policies/p1", func(c *Client) (*Policy, error) {
		return c.GetPolicy(context.Background(), "p1")
	}},
	{"CreatePolicy", http.MethodPost, "/adaptive-traces/api/v1/policies", func(c *Client) (*Policy, error) {
		return c.CreatePolicy(context.Background(), Policy{})
	}},
	{"UpdatePolicy", http.MethodPut, "/adaptive-traces/api/v1/policies/p1", func(c *Client) (*Policy, error) {
		return c.UpdatePolicy(context.Background(), "p1", Policy{})
	}},
}

type operation struct {
	name   string
	method string
	path   string
	call   func(c *Client) error
}

// allOperations calls every client method with an arbitrary policy.
var allOperations = func() []operation {
	var ops []operation
	for _, op := range policyOperations {
		ops = append(ops, operation{op.name, op.method, op.path, func(c *Client) error { _, err := op.call(c); return err }})
	}
	return append(ops, operation{"DeletePolicy", http.MethodDelete, "/adaptive-traces/api/v1/policies/p1", func(c *Client) error {
		return c.DeletePolicy(context.Background(), "p1")
	}})
}()

// writeTruncated declares a longer body than it sends, so the client fails to read the body.
func writeTruncated(t *testing.T, w http.ResponseWriter, status int, body string) {
	t.Helper()

	w.Header().Set("Content-Length", fmt.Sprint(len(body)+100))
	w.WriteHeader(status)
	_, err := w.Write([]byte(body))
	assert.NoError(t, err)
}

func Test_NewClient(t *testing.T) {
	t.Run("builds URLs from the base URL with or without a trailing slash", func(t *testing.T) {
		for _, suffix := range []string{"", "/", "/tempo", "/tempo/"} {
			t.Run("suffix "+suffix, func(t *testing.T) {
				gotPath := make(chan string, 1)
				svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					gotPath <- r.URL.Path
					writeJSON(t, w, http.StatusOK, `{"id": "p1"}`)
				}))
				defer svr.Close()

				c := NewClient(testAuth, svr.URL+suffix, svr.Client(), "test-user-agent", nil)
				_, err := c.GetPolicy(context.Background(), "p1")

				require.NoError(t, err)
				assert.Equal(t, strings.TrimSuffix(suffix, "/")+"/adaptive-traces/api/v1/policies/p1", <-gotPath)
			})
		}
	})

	t.Run("accepts a nil http client and sets a timeout", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id": "p1"}`)
		}))
		defer svr.Close()

		c := NewClient(testAuth, svr.URL, nil, "test-user-agent", nil)
		_, err := c.GetPolicy(context.Background(), "p1")

		require.NoError(t, err)
		assert.Equal(t, defaultTimeout, c.client.Timeout)
	})
}

func TestClient_RequestHeaders(t *testing.T) {
	for _, op := range allOperations {
		t.Run(op.name, func(t *testing.T) {
			var requests atomic.Int32
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				assert.Equal(t, testAuthHeader, r.Header.Get("Authorization"))
				assert.Equal(t, "test-user-agent", r.Header.Get("User-Agent"))
				assert.Equal(t, "value", r.Header.Get("Grafana-Terraform-Provider"))
				assert.Equal(t, "application/json", r.Header.Get("Accept"))

				if r.Method == http.MethodPost || r.Method == http.MethodPut {
					assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
				} else {
					assert.Empty(t, r.Header.Get("Content-Type"))
				}

				writeJSON(t, w, http.StatusOK, `{"id": "p1"}`)
			}))
			defer svr.Close()

			c := NewClient(testAuth, svr.URL, svr.Client(), "test-user-agent", map[string]string{"Grafana-Terraform-Provider": "value"})

			require.NoError(t, op.call(c))
			assert.Equal(t, int32(1), requests.Load())
		})
	}

	t.Run("keeps the default user agent when empty", func(t *testing.T) {
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			assert.True(t, strings.HasPrefix(r.Header.Get("User-Agent"), "Go-http-client"))
			writeJSON(t, w, http.StatusOK, `{"id": "p1"}`)
		}))
		defer svr.Close()

		c := NewClient(testAuth, svr.URL, svr.Client(), "", nil)
		_, err := c.GetPolicy(context.Background(), "p1")

		require.NoError(t, err)
	})
}

func TestClient_GetPolicy(t *testing.T) {
	t.Run("decodes a policy including the read-only fields", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodGet, r.Method)
			assert.Equal(t, "/adaptive-traces/api/v1/policies/p1", r.URL.Path)
			assert.Empty(t, r.URL.RawQuery)
			assert.Empty(t, readBody(t, r))

			// labels is not decoded: policies created from alerts carry a nested source_labels object.
			writeJSON(t, w, http.StatusOK, `{
				"id": "p1",
				"type": "status_code",
				"name": "Errors",
				"body": {"status_codes": ["ERROR"]},
				"expires_at": "2026-12-01T00:00:00Z",
				"labels": {"type": "base", "source_labels": {"service": "api"}},
				"version_recommendation_id": "r1",
				"version_created_at": "2026-11-01T10:00:00Z",
				"version_created_by": "alice"
			}`)
		})

		policy, err := c.GetPolicy(context.Background(), "p1")

		require.NoError(t, err)
		require.NotNil(t, policy.ExpiresAt)
		expiresAt := "2026-12-01T00:00:00Z"
		assert.Equal(t, &Policy{
			ID:               "p1",
			Type:             "status_code",
			Name:             "Errors",
			Body:             json.RawMessage(`{"status_codes": ["ERROR"]}`),
			ExpiresAt:        &expiresAt,
			VersionCreatedAt: "2026-11-01T10:00:00Z",
			VersionCreatedBy: "alice",
		}, policy)
	})

	t.Run("escapes the policy ID in the path", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, "/adaptive-traces/api/v1/policies/a%2Fb%20c", r.URL.EscapedPath())
			writeJSON(t, w, http.StatusOK, `{"id": "a/b c"}`)
		})

		_, err := c.GetPolicy(context.Background(), "a/b c")

		require.NoError(t, err)
	})
}

func TestClient_CreatePolicy(t *testing.T) {
	t.Run("sends the writable fields and decodes the created policy", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPost, r.Method)
			assert.Equal(t, "/adaptive-traces/api/v1/policies", r.URL.Path)
			assert.JSONEq(t, `{
				"type": "volumetric",
				"name": "Volume",
				"body": {"sampling_percentage": 5},
				"expires_at": "2026-12-01T00:00:00Z"
			}`, string(readBody(t, r)))

			writeJSON(t, w, http.StatusCreated, `{
				"id": "p1",
				"type": "volumetric",
				"name": "Volume",
				"body": {"sampling_percentage": 5},
				"expires_at": "2026-12-01T00:00:00Z",
				"version_created_at": "2026-11-01T10:00:00Z",
				"version_created_by": "alice"
			}`)
		})
		expiresAt := "2026-12-01T00:00:00Z"

		policy, err := c.CreatePolicy(context.Background(), Policy{
			Type:      "volumetric",
			Name:      "Volume",
			Body:      json.RawMessage(`{"sampling_percentage":5}`),
			ExpiresAt: &expiresAt,
		})

		require.NoError(t, err)
		assert.Equal(t, "p1", policy.ID)
		assert.Equal(t, "volumetric", policy.Type)
		assert.JSONEq(t, `{"sampling_percentage": 5}`, string(policy.Body))
		assert.Equal(t, &expiresAt, policy.ExpiresAt)
		assert.Equal(t, "2026-11-01T10:00:00Z", policy.VersionCreatedAt)
		assert.Equal(t, "alice", policy.VersionCreatedBy)
	})

	t.Run("does not send read-only fields", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			var sent map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(readBody(t, r), &sent))
			assert.Len(t, sent, 3)
			assert.Contains(t, sent, "type")
			assert.Contains(t, sent, "name")
			assert.Contains(t, sent, "body")
			writeJSON(t, w, http.StatusCreated, `{"id": "p1"}`)
		})

		_, err := c.CreatePolicy(context.Background(), Policy{
			ID:               "p0",
			Type:             "diversity",
			Name:             "Unique",
			Body:             json.RawMessage(`{}`),
			VersionCreatedAt: "2026-11-01T10:00:00Z",
			VersionCreatedBy: "alice",
		})

		require.NoError(t, err)
	})

	t.Run("accepts any 2xx status", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id": "p1"}`)
		})

		policy, err := c.CreatePolicy(context.Background(), Policy{Body: json.RawMessage(`{}`)})

		require.NoError(t, err)
		assert.Equal(t, "p1", policy.ID)
	})
}

func TestClient_PolicyResponseWithoutPolicy(t *testing.T) {
	tests := []struct {
		name          string
		status        int
		body          string
		expectedError string
	}{
		{"204 without a body", http.StatusNoContent, "", "failed to unmarshal response body"},
		{"201 without a body", http.StatusCreated, "", "failed to unmarshal response body"},
		{"200 with JSON null", http.StatusOK, "null", "adaptive traces API returned a response without a policy ID"},
		{"200 with an empty object", http.StatusOK, "{}", "adaptive traces API returned a response without a policy ID"},
		{"200 with an empty ID", http.StatusOK, `{"id": "", "name": "Errors"}`, "adaptive traces API returned a response without a policy ID"},
	}
	for _, op := range policyOperations {
		for _, tt := range tests {
			t.Run(op.name+" "+tt.name, func(t *testing.T) {
				c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(tt.status)
					_, err := w.Write([]byte(tt.body))
					assert.NoError(t, err)
				})

				policy, err := op.call(c)

				assert.Nil(t, policy)
				assert.ErrorContains(t, err, tt.expectedError)
			})
		}
	}
}

func TestClient_UpdatePolicy(t *testing.T) {
	t.Run("sends the whole policy in a PUT", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			assert.Equal(t, http.MethodPut, r.Method)
			assert.Equal(t, "/adaptive-traces/api/v1/policies/p1", r.URL.Path)
			assert.JSONEq(t, `{
				"type": "latency",
				"name": "Slow",
				"body": {"threshold_ms": 5000},
				"expires_at": "2026-12-01T00:00:00Z"
			}`, string(readBody(t, r)))

			writeJSON(t, w, http.StatusOK, `{
				"id": "p1",
				"type": "latency",
				"name": "Slow",
				"body": {"threshold_ms": 5000},
				"expires_at": "2026-12-01T00:00:00Z",
				"version_created_at": "2026-11-02T10:00:00Z",
				"version_created_by": "bob"
			}`)
		})
		expiresAt := "2026-12-01T00:00:00Z"

		policy, err := c.UpdatePolicy(context.Background(), "p1", Policy{
			ID:        "p1",
			Type:      "latency",
			Name:      "Slow",
			Body:      json.RawMessage(`{"threshold_ms":5000}`),
			ExpiresAt: &expiresAt,
		})

		require.NoError(t, err)
		assert.Equal(t, "p1", policy.ID)
		assert.Equal(t, "bob", policy.VersionCreatedBy)
		assert.Equal(t, "2026-11-02T10:00:00Z", policy.VersionCreatedAt)
	})

	t.Run("omits expires_at when it is nil", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw := readBody(t, r)
			assert.NotContains(t, string(raw), "expires_at")

			var sent map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(raw, &sent))
			assert.Len(t, sent, 3)
			writeJSON(t, w, http.StatusOK, `{"id": "p1", "type": "latency", "name": "Slow", "body": {}}`)
		})

		policy, err := c.UpdatePolicy(context.Background(), "p1", Policy{
			Type: "latency",
			Name: "Slow",
			Body: json.RawMessage(`{}`),
		})

		require.NoError(t, err)
		assert.Nil(t, policy.ExpiresAt)
	})
}

func TestClient_DeletePolicy(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusOK} {
		t.Run(fmt.Sprintf("succeeds on %d", status), func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodDelete, r.Method)
				assert.Equal(t, "/adaptive-traces/api/v1/policies/p1", r.URL.Path)
				assert.Empty(t, readBody(t, r))
				w.WriteHeader(status)
			})

			require.NoError(t, c.DeletePolicy(context.Background(), "p1"))
		})
	}
}

func TestClient_NotFound(t *testing.T) {
	for _, op := range allOperations {
		t.Run(op.name+" names the request", func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, http.StatusNotFound, `{"error": "policy not found"}`)
			})

			err := op.call(c)

			require.ErrorIs(t, err, ErrNotFound)
			assert.Regexp(t, `^adaptive traces: not found: `+op.method+` http://127\.0\.0\.1:\d+`+op.path+`$`, err.Error())
		})

		t.Run(op.name+" when the body cannot be read", func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				writeTruncated(t, w, http.StatusNotFound, `{"error": "pol`)
			})

			require.ErrorIs(t, op.call(c), ErrNotFound)
		})
	}
}

func TestClient_ResponseBodyErrors(t *testing.T) {
	t.Run("keeps the status when an error body cannot be read", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeTruncated(t, w, http.StatusServiceUnavailable, "service unav")
		})

		err := c.DeletePolicy(context.Background(), "p1")

		assert.ErrorContains(t, err, "adaptive traces API returned status 503")
	})

	t.Run("reports a success body that cannot be read", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeTruncated(t, w, http.StatusOK, `{"id": "p1"`)
		})

		policy, err := c.GetPolicy(context.Background(), "p1")

		assert.Nil(t, policy)
		assert.ErrorContains(t, err, "failed to read response body")
	})

	t.Run("rejects a success body over the size limit", func(t *testing.T) {
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, `{"id": "p1", "name": "`+strings.Repeat("x", maxResponseBytes)+`"}`)
		})

		policy, err := c.GetPolicy(context.Background(), "p1")

		assert.Nil(t, policy)
		assert.EqualError(t, err, fmt.Sprintf("adaptive traces API response body exceeds %d bytes", maxResponseBytes))
	})

	t.Run("accepts a success body at the size limit", func(t *testing.T) {
		prefix, suffix := `{"id": "p1", "name": "`, `"}`
		c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			writeJSON(t, w, http.StatusOK, prefix+strings.Repeat("x", maxResponseBytes-len(prefix)-len(suffix))+suffix)
		})

		policy, err := c.GetPolicy(context.Background(), "p1")

		require.NoError(t, err)
		assert.Equal(t, "p1", policy.ID)
	})
}

func TestClient_ErrorMapping(t *testing.T) {
	tests := []struct {
		name            string
		status          int
		body            string
		expectedMessage string
	}{
		{"400 with error body", http.StatusBadRequest, `{"error": "invalid policy: name is required"}`, "invalid policy: name is required"},
		{"403 with error body", http.StatusForbidden, `{"error": "forbidden"}`, "forbidden"},
		{"413 with error body", http.StatusRequestEntityTooLarge, `{"error": "request body too large"}`, "request body too large"},
		{"500 with error body", http.StatusInternalServerError, `{"error": "internal error"}`, "internal error"},
		{"503 with error body", http.StatusServiceUnavailable, `{"error": "service unavailable"}`, "service unavailable"},
		{"401 with plain text", http.StatusUnauthorized, "  invalid credentials\n", "invalid credentials"},
		{"413 with plain text", http.StatusRequestEntityTooLarge, "request body too large\n", "request body too large"},
		{"502 with HTML", http.StatusBadGateway, "<html>Bad Gateway</html>", "<html>Bad Gateway</html>"},
		{"400 with JSON without error", http.StatusBadRequest, `{"message": "nope"}`, `{"message": "nope"}`},
		{"400 with empty error", http.StatusBadRequest, `{"error": ""}`, `{"error": ""}`},
		{"500 with empty body", http.StatusInternalServerError, "", "Internal Server Error"},
		{"502 with a long body", http.StatusBadGateway, strings.Repeat("x", 2*maxErrorMessageBytes), strings.Repeat("x", maxErrorMessageBytes) + " (truncated)"},
		{
			"502 with a long body cut inside a character", http.StatusBadGateway,
			strings.Repeat("x", maxErrorMessageBytes-1) + "é" + strings.Repeat("x", 10),
			strings.Repeat("x", maxErrorMessageBytes-1) + " (truncated)",
		},
		{"400 with a long error field", http.StatusBadRequest, `{"error": "` + strings.Repeat("x", 2*maxErrorMessageBytes) + `"}`, strings.Repeat("x", maxErrorMessageBytes) + " (truncated)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, err := w.Write([]byte(tt.body))
				assert.NoError(t, err)
			})

			_, err := c.CreatePolicy(context.Background(), Policy{Body: json.RawMessage(`{}`)})

			assert.EqualError(t, err, fmt.Sprintf("adaptive traces API returned status %d: %s", tt.status, tt.expectedMessage))
			assert.NotErrorIs(t, err, ErrNotFound)
		})
	}

	for _, op := range allOperations {
		t.Run(op.name+" returns the status and message", func(t *testing.T) {
			c := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
				writeJSON(t, w, http.StatusServiceUnavailable, `{"error": "service unavailable"}`)
			})

			err := op.call(c)

			assert.EqualError(t, err, "adaptive traces API returned status 503: service unavailable")
		})
	}
}
