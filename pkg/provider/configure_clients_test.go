package provider

import (
	"os"
	"testing"

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
				assert.Nil(t, c.OnCallClient)
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
			name: "Stack URL and auth to be set, empty strings; OnCall URL set (it has a default)",
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
