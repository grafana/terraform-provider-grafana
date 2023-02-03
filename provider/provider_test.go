package provider

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/provider/common"
	"github.com/grafana/terraform-provider-grafana/provider/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func init() {
	testutils.Provider = Provider("testacc")()

	// Always allocate a new provider instance each invocation, otherwise gRPC
	// ProviderConfigure() can overwrite configuration during concurrent testing.
	testutils.ProviderFactories = map[string]func() (*schema.Provider, error){
		"grafana": func() (*schema.Provider, error) {
			return Provider("testacc")(), nil
		},
	}

	// If any acceptance tests are enabled, the test provider must be configured
	if testutils.AccTestsEnabled("TF_ACC") {
		// Since we are outside the scope of the Terraform configuration we must
		// call Configure() to properly initialize the provider configuration.
		err := testutils.Provider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		if err != nil {
			panic(fmt.Sprintf("failed to configure provider: %v", err))
		}
	}
}

func TestProvider(t *testing.T) {
	testutils.IsUnitTest(t)

	if err := Provider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigure(t *testing.T) {
	testutils.IsUnitTest(t)

	// Helper for header tests
	checkHeaders := func(t *testing.T, provider *schema.Provider) {
		gotHeaders := provider.Meta().(*common.Client).GrafanaAPIConfig.HTTPHeaders
		if len(gotHeaders) != 2 {
			t.Errorf("expected 2 HTTP header, got %d", len(gotHeaders))
		}
		if gotHeaders["Authorization"] != "Bearer test" {
			t.Errorf("expected HTTP header Authorization to be \"Bearer test\", got %q", gotHeaders["Authorization"])
		}
		if gotHeaders["X-Custom-Header"] != "custom-value" {
			t.Errorf("expected HTTP header X-Custom-Header to be \"custom-value\", got %q", gotHeaders["X-Custom-Header"])
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
			name:        "no config",
			env:         map[string]string{},
			expectedErr: "\"auth\": one of `auth,cloud_api_key,oncall_access_token,sm_access_token` must\nbe specified",
		},
		{
			name: "grafana config from env",
			env: map[string]string{
				"GRAFANA_AUTH": "admin:admin",
				"GRAFANA_URL":  "https://test.com",
			},
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
			expectedErr: "invalid http_headers config: invalid character 'b' looking for beginning of value",
		},
		{
			name: "grafana cloud config from env",
			env: map[string]string{
				"GRAFANA_CLOUD_API_KEY": "testtest",
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
			provider := Provider("dev")()
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
