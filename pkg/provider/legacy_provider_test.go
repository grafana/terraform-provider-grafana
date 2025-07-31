package provider_test

import (
	"context"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestProvider(t *testing.T) {
	testutils.IsUnitTest(t)

	if err := provider.Provider("dev").InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigure(t *testing.T) {
	testutils.IsUnitTest(t)

	// Helper for header tests
	checkHeaders := func(t *testing.T, provider *schema.Provider) {
		gotHeaders := provider.Meta().(*common.Client).GrafanaAPIConfig.HTTPHeaders
		if len(gotHeaders) != 4 {
			t.Errorf("expected 4 HTTP header, got %d", len(gotHeaders))
		}
		if gotHeaders["Authorization"] != "Bearer test" {
			t.Errorf("expected HTTP header Authorization to be \"Bearer test\", got %q", gotHeaders["Authorization"])
		}
		if gotHeaders["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected HTTP header X-Custom-Header to be \"custom-value\", got %q", gotHeaders["X-Custom-Header"])
		}
	}

	// Helper for status codes tests
	checkStatusCodes := func(t *testing.T, provider *schema.Provider) {
		gotStatusCodes := provider.Meta().(*common.Client).GrafanaAPIConfig.RetryStatusCodes
		if len(gotStatusCodes) != 2 {
			t.Errorf("expected 2 status codes, got %d", len(gotStatusCodes))
		}
		if gotStatusCodes[0] != "5xx" {
			t.Errorf("expected status code 500, got %s", gotStatusCodes[0])
		}
		if gotStatusCodes[1] != "123" {
			t.Errorf("expected status code 123, got %s", gotStatusCodes[1])
		}
	}

	envBackup := os.Environ()
	defer func() {
		os.Clearenv()
		for _, v := range envBackup {
			kv := strings.SplitN(v, "=", 2)
			os.Setenv(kv[0], kv[1])
		}
	}()

	cases := []struct {
		name        string
		config      map[string]interface{}
		env         map[string]string
		expectedErr string
		check       func(t *testing.T, provider *schema.Provider)
	}{
		{
			name: "grafana config from env",
			env: map[string]string{
				"GRAFANA_AUTH": "admin:admin",
				"GRAFANA_URL":  "https://test.com",
			},
		},
		{
			name: "grafana status codes from env",
			env: map[string]string{
				"GRAFANA_AUTH":               "admin:admin",
				"GRAFANA_URL":                "https://test.com",
				"GRAFANA_RETRY_STATUS_CODES": "5xx,123",
			},
			check: checkStatusCodes,
		},
		{
			name: "header config",
			env: map[string]string{
				"GRAFANA_AUTH": "admin:admin",
				"GRAFANA_URL":  "https://test.com",
			},
			config: map[string]interface{}{
				"http_headers": map[string]interface{}{
					"Authorization":   "Bearer test",
					"X-Custom-Header": "custom-value",
				},
			},
			check: checkHeaders,
		},
		{
			name: "header config from env",
			env: map[string]string{
				"GRAFANA_AUTH":         "admin:admin",
				"GRAFANA_URL":          "https://test.com",
				"GRAFANA_HTTP_HEADERS": `{"X-Custom-Header": "custom-value", "Authorization": "Bearer test"}`,
			},
			check: checkHeaders,
		},
		{
			name: "invalid header",
			env: map[string]string{
				"GRAFANA_AUTH":         "admin:admin",
				"GRAFANA_URL":          "https://test.com",
				"GRAFANA_HTTP_HEADERS": `blabla`,
			},
			expectedErr: "failed to parse GRAFANA_HTTP_HEADERS: invalid character 'b' looking for beginning of value",
		},
		{
			name: "grafana cloud config from env",
			env: map[string]string{
				"GRAFANA_CLOUD_ACCESS_POLICY_TOKEN": "testtest",
			},
		},
		{
			name: "grafana sm config from env",
			env: map[string]string{
				"GRAFANA_SM_ACCESS_TOKEN": "testtest",
			},
		},
		{
			name: "grafana oncall config from env",
			env: map[string]string{
				"GRAFANA_ONCALL_ACCESS_TOKEN": "testtest",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			os.Clearenv()
			for k, v := range tc.env {
				os.Setenv(k, v)
			}

			test := resource.TestStep{
				// Resource is irrelevant, it's just there to test the provider being configured
				// Terraform will "validate" the provider, but not actually use it when planning
				PlanOnly:           true,
				ExpectNonEmptyPlan: true,
				Config: `resource "grafana_folder" "test" {
					title = "test"
				}`,
			}

			if tc.expectedErr != "" {
				test.ExpectError = regexp.MustCompile(tc.expectedErr)
			}

			// Configure the provider and check it
			provider := provider.Provider("dev")
			provider.Configure(context.Background(), terraform.NewResourceConfigRaw(tc.config))
			if tc.check != nil {
				tc.check(t, provider)
			}
			// Run the plan to check for validation errors
			resource.UnitTest(t, resource.TestCase{
				Providers: map[string]*schema.Provider{
					"grafana": provider,
				},
				Steps: []resource.TestStep{test},
			})
		})
	}
}
