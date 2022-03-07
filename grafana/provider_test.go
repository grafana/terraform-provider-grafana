package grafana

import (
	"context"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// testAccProviderFactories is a static map containing only the main provider instance
var testAccProviderFactories map[string]func() (*schema.Provider, error)

// testAccProvider is the "main" provider instance
//
// This Provider can be used in testing code for API calls without requiring
// the use of saving and referencing specific ProviderFactories instances.
//
// It is configured within the accTestsEnabled function (if acceptance tests are enabled)
// testAccProviderConfigure is used to make sure that we only configure the provider once
var testAccProvider *schema.Provider
var testAccProviderConfigure sync.Once

func init() {
	testAccProvider = Provider("testacc")()

	// Always allocate a new provider instance each invocation, otherwise gRPC
	// ProviderConfigure() can overwrite configuration during concurrent testing.
	testAccProviderFactories = map[string]func() (*schema.Provider, error){
		//nolint:unparam // error is always nil
		"grafana": func() (*schema.Provider, error) {
			return Provider("testacc")(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	IsUnitTest(t)

	if err := Provider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProviderConfigure(t *testing.T) {
	IsUnitTest(t)

	// Helper for header tests
	checkHeaders := func(t *testing.T, provider *schema.Provider) {
		gotHeaders := provider.Meta().(*client).gapiConfig.HTTPHeaders
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
			expectedErr: "\"auth\": one of `auth,cloud_api_key,sm_access_token` must be specified",
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

// testAccExample returns an example config from the examples directory.
// Examples are used for both documentation and acceptance tests.
func testAccExample(t *testing.T, path string) string {
	example, err := os.ReadFile("../examples/" + path)
	if err != nil {
		t.Fatal(err)
	}
	return string(example)
}

func accTestsEnabled(t *testing.T, envVarName string) bool {
	v, ok := os.LookupEnv(envVarName)
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		t.Fatalf("%s must be set to a boolean value", envVarName)
	}

	// If any acceptance tests are enabled, the test provider must be configured
	if enabled {
		testAccProviderConfigure.Do(func() {
			// Since we are outside the scope of the Terraform configuration we must
			// call Configure() to properly initialize the provider configuration.
			err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
			if err != nil {
				t.Fatalf("failed to configure provider: %v", err)
			}
		})
	}

	return enabled
}

func checkEnvVarsSet(t *testing.T, envVars ...string) {
	t.Helper()

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			t.Fatalf("%s must be set", envVar)
		}
	}
}

func IsUnitTest(t *testing.T) {
	t.Helper()

	if accTestsEnabled(t, "TF_ACC") {
		t.Skip("Skipping acceptance tests")
	}
}

// CheckOSSTestsEnabled checks if the OSS acceptance tests are enabled. This should be the first line of any test that uses Grafana OSS features only
func CheckOSSTestsEnabled(t *testing.T) {
	t.Helper()

	if !accTestsEnabled(t, "TF_ACC_OSS") {
		t.Skip("TF_ACC_OSS must be set to a truthy value for OSS acceptance tests")
	}

	checkEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_ORG_ID",
		"GRAFANA_VERSION",
	)
}

// CheckOSSTestsSemver allows to skip tests that are not supported by the Grafana OSS version
func CheckOSSTestsSemver(t *testing.T, semverConstraint string) {
	t.Helper()

	versionStr := os.Getenv("GRAFANA_VERSION")
	if semverConstraint != "" && versionStr != "" {
		version := semver.MustParse(versionStr)
		c, err := semver.NewConstraint(semverConstraint)
		if err != nil {
			t.Fatalf("invalid constraint %s: %v", semverConstraint, err)
		}
		if !c.Check(version) {
			t.Skipf("skipping test for Grafana version `%s`, constraint `%s`", versionStr, semverConstraint)
		}
	}
}

// CheckCloudTestsEnabled checks if the cloud tests are enabled. This should be the first line of any test that tests Cloud API features
func CheckCloudAPITestsEnabled(t *testing.T) {
	t.Helper()

	if !accTestsEnabled(t, "TF_ACC_CLOUD_API") {
		t.Skip("TF_ACC_CLOUD_API must be set to a truthy value for Cloud API acceptance tests")
	}

	checkEnvVarsSet(t, "GRAFANA_CLOUD_API_KEY")
}

// CheckCloudInstanceTestsEnabled checks if tests that run on cloud instances are enabled. This should be the first line of any test that tests Grafana Cloud Pro features
func CheckCloudInstanceTestsEnabled(t *testing.T) {
	t.Helper()

	if !accTestsEnabled(t, "TF_ACC_CLOUD_INSTANCE") {
		t.Skip("TF_ACC_CLOUD_INSTANCE must be set to a truthy value for Cloud instance acceptance tests")
	}

	checkEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_ORG_ID",
		"GRAFANA_SM_ACCESS_TOKEN",
	)
}

// CheckEnterpriseTestsEnabled checks if the enterprise tests are enabled. This should be the first line of any test that tests Grafana Enterprise features
func CheckEnterpriseTestsEnabled(t *testing.T) {
	t.Helper()

	if !accTestsEnabled(t, "TF_ACC_ENTERPRISE") {
		t.Skip("TF_ACC_ENTERPRISE must be set to a truthy value for Enterprise acceptance tests")
	}

	checkEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_ORG_ID",
	)
}
