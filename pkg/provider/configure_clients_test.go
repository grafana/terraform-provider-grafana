package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	incident "github.com/grafana/incident-go"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateTempFileIfLiteral(t *testing.T) {
	t.Run("Test with empty string value returns empty and does not create temp file", func(t *testing.T) {
		path, tempFile, err := createTempFileIfLiteral("")
		require.NoError(t, err)
		require.False(t, tempFile, "Expected temp file to not be created")
		require.Empty(t, path)
	})
	t.Run("Test file path returns given path and does not create a temp file", func(t *testing.T) {
		// Create a temporary file to simulate an existing file
		tmp, err := os.CreateTemp(t.TempDir(), "existing-file")
		require.NoError(t, err)

		path, tempFile, err := createTempFileIfLiteral(tmp.Name())
		require.NoError(t, err)
		require.False(t, tempFile, "Expected temp file to not be created")
		require.Equal(t, tmp.Name(), path)
	})

	t.Run("Test with short literal creates temp file and path", func(t *testing.T) {
		caCert := "certTest"

		path, tempFile, err := createTempFileIfLiteral(caCert)
		require.NoError(t, err)
		require.True(t, tempFile, "Expected temp file to be created")
		require.NotEmpty(t, path)

		// Validate the file was created and has the correct content
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, caCert, string(content))

		// Clean up the temporary file
		require.NoError(t, os.Remove(path))
	})

	t.Run("Test with a certificate literal creates temp file and path", func(t *testing.T) {
		caCert := ` -----BEGIN CERTIFICATE-----
	MIIDXTCCAkWgAwIBAgIJAMW9UJtz1MoNMA0GCSqGSIb3DQEBCwUAMEUxCzAJBgNV
	BAYTAkFVMRMwEQYDVQQIDApxdWVlbnNsYW5kMRAwDgYDVQQHDAdicmlzYmFuZTEN
	MAsGA1UECgwEVGVzdDAeFw0xODA2MTAwNzU1NDJaFw0xOTA2MTAwNzU1NDJaMEUx
	CzAJBgNVBAYTAkFVMRMwEQYDVQQIDApxdWVlbnNsYW5kMRAwDgYDVQQHDAdicmlz
	YmFuZTENMAsGA1UECgwEVGVzdDCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoC
	ggEBAK1lpt+lPZJbG7yMYYWzjk8FwGbM3vlUJlC2aQHJ18T2aTtsOaZC1deKtwGR
	qBZMyel3hG0XayZmFQO2DAnOScgn4j+jPEFLWswg+U4MgH80+PA4wHzm+E0v68qD
	S+cA9If1D2I0gtT6jKPm3WYwZ/r0GUn8/JjgiIhCZGfXArH39V2D2KNhJ3W0b7T6
	isfbsHvSKWs/49q8w5J/yN8GOh/n/rThBfhM3FQ2eDdVR1QfvvX5KT69aXhtJlD9
	Z5H8z9DnD8BZxBrzE5hEO74KK13CvAFeKbVp7KvXf6NOy4W31lUd6lmzZ+lR+IxO
	NHElgJoaJ2F2y4XcFXY1cQFhKjkCAwEAAaNQME4wHQYDVR0OBBYEFLPkkSMxs/PR
	1E7VwDhRu5DTHwrNMB8GA1UdIwQYMBaAFLPkkSMxs/PR1E7VwDhRu5DTHwrNMAwG
	A1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAK1lpt+lPZJbG7yMYYWzjk8F
	wGbM3vlUJlC2aQHJ18T2aTtsOaZC1deKtwGRqBZMyel3hG0XayZmFQO2DAnOScgn
	4j+jPEFLWswg+U4MgH80+PA4wHzm+E0v68qDS+cA9If1D2I0gtT6jKPm3WYwZ/r0
	GUn8/JjgiIhCZGfXArH39V2D2KNhJ3W0b7T6isfbsHvSKWs/49q8w5J/yN8GOh/n
	/rThBfhM3FQ2eDdVR1QfvvX5KT69aXhtJlD9Z5H8z9DnD8BZxBrzE5hEO74KK13C
	vAFeKbVp7KvXf6NOy4W31lUd6lmzZ+lR+IxONHElgJoaJ2F2y4XcFXY1cQFhKjkC
	AwEAAaNQME4wHQYDVR0OBBYEFLPkkSMxs/PR1E7VwDhRu5DTHwrNMB8GA1UdIwQY
	MBaAFLPkkSMxs/PR1E7VwDhRu5DTHwrNMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcN
	AQELBQADggEBAK1lpt+lPZJbG7yMYYWzjk8FwGbM3vlUJlC2aQHJ18T2aTtsOaZC
	1deKtwGRqBZMyel3hG0XayZmFQO2DAnOScgn4j+jPEFLWswg+U4MgH80+PA4wHzm
	+E0v68qDS+cA9If1D2I0gtT6jKPm3WYwZ/r0GUn8/JjgiIhCZGfXArH39V2D2KNh
	J3W0b7T6isfbsHvSKWs/49q8w5J/yN8GOh/n/rThBfhM3FQ2eDdVR1QfvvX5KT69
	aXhtJlD9Z5H8z9DnD8BZxBrzE5hEO74KK13CvAFeKbVp7KvXf6NOy4W31lUd6lmz
	Z+lR+IxONHElgJoaJ2F2y4XcFXY1cQFhKjkCAwEAAaNQME4wHQYDVR0OBBYEFLPk
	kSMxs/PR1E7VwDhRu5DTHwrNMB8GA1UdIwQYMBaAFLPkkSMxs/PR1E7VwDhRu5DT
	HwrNMAwGA1UdEwQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBAK1lpt+lPZJbG7yM
	YYWzjk8FwGbM3vlUJl=
    -----END CERTIFICATE-----`

		path, tempFile, err := createTempFileIfLiteral(caCert)
		require.NoError(t, err)
		require.True(t, tempFile, "Expected temp file to be created")
		require.NotEmpty(t, path)

		// Check if the file exists and has the correct content
		content, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, caCert, string(content))

		// Clean up the temporary file
		require.NoError(t, os.Remove(path))
	})
}

func TestGrafanaHTTPRoundTripperSendsBearerAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer my-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "42", r.Header.Get("X-Grafana-Org-Id"))
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom-Header"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	apiConfig := &goapi.TransportConfig{
		OrgID:       42,
		APIKey:      "my-api-key",
		HTTPHeaders: map[string]string{"X-Custom-Header": "custom-value"},
	}
	client := newGrafanaHTTPClient(nil, nil, "my-api-key", apiConfig)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGrafanaHTTPRoundTripperSendsBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		assert.True(t, ok, "expected basic auth")
		assert.Equal(t, "admin", username)
		assert.Equal(t, "secret", password)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	userInfo := url.UserPassword("admin", "secret")
	apiConfig := &goapi.TransportConfig{}
	client := newGrafanaHTTPClient(nil, userInfo, "", apiConfig)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGrafanaHTTPRoundTripperNoAuthWhenNoneConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("Authorization"))
		_, _, ok := r.BasicAuth()
		assert.False(t, ok, "expected no basic auth")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	apiConfig := &goapi.TransportConfig{}
	client := newGrafanaHTTPClient(nil, nil, "", apiConfig)

	resp, err := client.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()
	require.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestCreateClients(t *testing.T) {
	testCases := []struct {
		name     string
		config   ProviderConfig
		expected func(c *common.Client, err error)
	}{
		{
			name: "http with Grafana Cloud",
			config: ProviderConfig{
				URL:  types.StringValue("http://myinstance.grafana.net"),
				Auth: types.StringValue("myapikey"),
			},
			expected: func(c *common.Client, err error) {
				assert.EqualError(t, err, "http not supported in Grafana Cloud. Use the https scheme")
			},
		},
		{
			name: "https with Grafana Cloud",
			config: ProviderConfig{
				URL:  types.StringValue("https://myinstance.grafana.net"),
				Auth: types.StringValue("myapikey"),
			},
			expected: func(c *common.Client, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, c.GrafanaAPI)
				assert.NotNil(t, c.MLAPI)
				assert.NotNil(t, c.SLOClient)
				assert.NotNil(t, c.IncidentClient)
				assert.NotNil(t, c.OnCallClient)
			},
		},
		{
			name: "http with Grafana OSS",
			config: ProviderConfig{
				URL:  types.StringValue("http://localhost:3000"),
				Auth: types.StringValue("admin:admin"),
			},
			expected: func(c *common.Client, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, c.GrafanaAPI)
			},
		},
		{
			name: "Stack URL and auth set to empty strings, OnCall URL set",
			config: ProviderConfig{
				URL:       types.StringValue(""),
				Auth:      types.StringValue(""),
				OncallURL: types.StringValue("http://oncall.url"),
			},
			expected: func(c *common.Client, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, c.GrafanaAPI)
			},
		},
		{
			name: "OnCall client using original config (not setting Grafana URL)",
			config: ProviderConfig{
				OncallAccessToken: types.StringValue("oncall-token"),
				OncallURL:         types.StringValue("http://oncall.url"),
			},
			expected: func(c *common.Client, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, c.OnCallClient)
				assert.Nil(t, c.OnCallClient.GrafanaURL())
			},
		},
		{
			name: "OnCall client setting Grafana URL (using Grafana URL and auth)",
			config: ProviderConfig{
				URL:       types.StringValue("http://localhost:3000"),
				Auth:      types.StringValue("service-account-token"),
				OncallURL: types.StringValue("http://oncall.url"),
			},
			expected: func(c *common.Client, err error) {
				assert.Nil(t, err)
				assert.NotNil(t, c.OnCallClient)
				assert.Equal(t, "http://localhost:3000", c.OnCallClient.GrafanaURL().String())
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := CreateClients(tc.config)
			tc.expected(c, err)
		})
	}
}

func TestCreateClientsOnCallURL(t *testing.T) {
	const (
		euOnCallURL      = "https://oncall-prod-eu-west-0.grafana.net/oncall"
		usOnCallURL      = "https://oncall-prod-us-central-0.grafana.net/oncall"
		pluginSettingsOK = `{"jsonData":{"onCallApiUrl":"` + euOnCallURL + `"}}`
	)

	testCases := []struct {
		name             string
		pluginStatus     int
		oncallURL        string
		oncallToken      string
		expectedBaseURL  string
		expectedToken    string
		expectedLookups  int32
		expectedErrorMsg string
	}{
		{
			name:            "URL derived from plugin settings",
			pluginStatus:    http.StatusOK,
			expectedBaseURL: euOnCallURL + "/api/v1/",
			expectedToken:   "service-account-token",
			expectedLookups: 1,
		},
		{
			name:            "oncall_access_token is used for OnCall calls",
			pluginStatus:    http.StatusOK,
			oncallToken:     "oncall-token",
			expectedBaseURL: euOnCallURL + "/api/v1/",
			expectedToken:   "oncall-token",
			expectedLookups: 1,
		},
		{
			name:            "explicit oncall_url is used without a lookup",
			pluginStatus:    http.StatusOK,
			oncallURL:       usOnCallURL,
			expectedBaseURL: usOnCallURL + "/api/v1/",
			expectedToken:   "service-account-token",
			expectedLookups: 0,
		},
		{
			name:             "lookup fails without oncall_url",
			pluginStatus:     http.StatusForbidden,
			expectedLookups:  1,
			expectedErrorMsg: "status 403",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var lookups atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				lookups.Add(1)
				assert.Equal(t, "/api/plugins/grafana-irm-app/settings", r.URL.Path)
				assert.Equal(t, "Bearer service-account-token", r.Header.Get("Authorization"))
				w.WriteHeader(tc.pluginStatus)
				if tc.pluginStatus == http.StatusOK {
					_, _ = w.Write([]byte(pluginSettingsOK))
				}
			}))
			defer server.Close()

			config := ProviderConfig{
				URL:  types.StringValue(server.URL),
				Auth: types.StringValue("service-account-token"),
			}
			if tc.oncallURL != "" {
				config.OncallURL = types.StringValue(tc.oncallURL)
			}
			if tc.oncallToken != "" {
				config.OncallAccessToken = types.StringValue(tc.oncallToken)
			}

			c, err := CreateClients(config)
			require.NoError(t, err)
			require.NotNil(t, c.OnCallClient)

			err = c.OnCallClient.EnsureBaseURL(t.Context())
			assert.Equal(t, tc.expectedLookups, lookups.Load())
			if tc.expectedErrorMsg != "" {
				require.ErrorContains(t, err, tc.expectedErrorMsg)
				assert.Nil(t, c.OnCallClient.BaseURL())
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.expectedBaseURL, c.OnCallClient.BaseURL().String())
			assert.Empty(t, c.OnCallClient.Warnings())

			req, err := c.OnCallClient.NewRequest(http.MethodGet, "users/", nil)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedToken, req.Header.Get("Authorization"))
			assert.Equal(t, server.URL, req.Header.Get("X-Grafana-URL"))
		})
	}
}

func TestCreateClientsOnCallTokenOnly(t *testing.T) {
	for _, envVar := range []string{"GRAFANA_URL", "GRAFANA_AUTH", "GRAFANA_ONCALL_URL"} {
		t.Setenv(envVar, "")
	}
	config := ProviderConfig{
		OncallAccessToken: types.StringValue("oncall-token"),
	}
	require.NoError(t, config.SetDefaults())
	assert.True(t, config.OncallURL.IsNull(), "oncall_url must not have a default")

	c, err := CreateClients(config)
	require.NoError(t, err)
	require.NotNil(t, c.OnCallClient)
	require.NoError(t, c.OnCallClient.EnsureBaseURL(t.Context()))
	assert.Equal(t, "https://oncall-prod-us-central-0.grafana.net/oncall/api/v1/", c.OnCallClient.BaseURL().String())
	assert.Nil(t, c.OnCallClient.GrafanaURL())
}

func TestCreateIncidentClientRemoteHost(t *testing.T) {
	testCases := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "plain host",
			url:      "https://myinstance.grafana.net",
			expected: "https://myinstance.grafana.net/api/plugins/grafana-irm-app/resources/api/v1/",
		},
		{
			name:     "host with trailing slash",
			url:      "https://myinstance.grafana.net/",
			expected: "https://myinstance.grafana.net/api/plugins/grafana-irm-app/resources/api/v1/",
		},
		{
			// Grafana hosted under a subpath must keep that prefix.
			name:     "host with subpath",
			url:      "https://example.com/grafana",
			expected: "https://example.com/grafana/api/plugins/grafana-irm-app/resources/api/v1/",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			c, err := CreateClients(ProviderConfig{
				URL:  types.StringValue(tc.url),
				Auth: types.StringValue("my-api-key"),
			})
			require.NoError(t, err)
			require.NotNil(t, c.IncidentClient)
			// The trailing slash matters: the generated client concatenates
			// RemoteHost with "<Service>.<Method>" without a separator.
			assert.Equal(t, tc.expected, c.IncidentClient.RemoteHost)
		})
	}
}

func TestIncidentClientRequest(t *testing.T) {
	var gotMethod, gotPath, gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotAuth = r.Method, r.URL.Path, r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"Roles":[]}`))
	}))
	defer server.Close()

	c, err := CreateClients(ProviderConfig{
		URL:  types.StringValue(server.URL),
		Auth: types.StringValue("my-api-key"),
	})
	require.NoError(t, err)
	require.NotNil(t, c.IncidentClient)

	// Reaching a response at all also proves the client's Debug hook is
	// non-nil; the generated code calls it unconditionally on every request.
	resp, err := incident.NewRolesService(c.IncidentClient).GetRoles(context.Background(), incident.GetRolesRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.MethodPost, gotMethod)
	assert.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/RolesService.GetRoles", gotPath)
	assert.Equal(t, "Bearer my-api-key", gotAuth)
}
