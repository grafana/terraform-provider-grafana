package fleetmanagementapi

import (
	"crypto/tls"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_NewClient(t *testing.T) {
	t.Run("successfully creates a new client", func(t *testing.T) {
		actualClient := NewClient(
			"test:auth",
			"https://test.url",
			&http.Client{},
			"test-user-agent",
			map[string]string{"key": "value"},
		)

		assert.NotNil(t, actualClient)
		assert.NotNil(t, actualClient.CollectorServiceClient)
		assert.NotNil(t, actualClient.PipelineServiceClient)
	})
}

func Test_newHTTPClient(t *testing.T) {
	t.Run("creates a new client with default transport", func(t *testing.T) {
		client := &http.Client{}
		auth := "test:auth"
		userAgent := "test-user-agent"
		headers := map[string]string{"key": "value"}

		actualClient := newHTTPClient(client, auth, userAgent, headers)

		expectedClient := &http.Client{
			Transport: &transport{
				auth:          auth,
				headers:       headers,
				userAgent:     userAgent,
				baseTransport: http.DefaultTransport,
			},
			CheckRedirect: client.CheckRedirect,
			Jar:           client.Jar,
			Timeout:       client.Timeout,
		}

		assert.NotNil(t, actualClient)
		assert.Equal(t, expectedClient, actualClient)
		assert.Same(t, http.DefaultTransport, actualClient.Transport.(*transport).baseTransport)
	})

	t.Run("uses existing transport if provided", func(t *testing.T) {
		existingTransport := &http.Transport{
			DisableCompression: true,
		}
		client := &http.Client{Transport: existingTransport}
		auth := "test:auth"
		userAgent := "test-user-agent"
		headers := map[string]string{"key": "value"}

		actualClient := newHTTPClient(client, auth, userAgent, headers)

		expectedClient := &http.Client{
			Transport: &transport{
				auth:          auth,
				userAgent:     userAgent,
				headers:       headers,
				baseTransport: existingTransport,
			},
			CheckRedirect: client.CheckRedirect,
			Jar:           client.Jar,
			Timeout:       client.Timeout,
		}

		assert.NotNil(t, actualClient)
		assert.Equal(t, expectedClient, actualClient)
		assert.Same(t, existingTransport, actualClient.Transport.(*transport).baseTransport)
	})
}

func Test_transport_RoundTrip(t *testing.T) {
	t.Run("sets headers correctly", func(t *testing.T) {
		svr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			encoded := base64.StdEncoding.EncodeToString([]byte("test:auth"))
			assert.Equal(t, "Basic "+encoded, r.Header.Get("Authorization"))
			assert.Equal(t, "test-user-agent", r.Header.Get("User-Agent"))
			assert.Equal(t, "value1", r.Header.Get("key1"))
			assert.Equal(t, "value2", r.Header.Get("key2"))
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(svr.Close)

		client := &http.Client{
			Transport: &transport{
				auth:      "test:auth",
				userAgent: "test-user-agent",
				headers: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				baseTransport: &http.Transport{
					TLSClientConfig: &tls.Config{
						InsecureSkipVerify: true, // #nosec G402
					},
				},
			},
		}

		req, err := http.NewRequest(http.MethodGet, svr.URL, nil)
		require.NoError(t, err)

		resp, err := client.Do(req)
		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
