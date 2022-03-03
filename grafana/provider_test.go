package grafana

import (
	"context"
	"io/ioutil"
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
// testAccPreCheck(t) must be called before using this provider instance.
var testAccProvider *schema.Provider

// testAccProviderConfigure ensures that testAccProvider is only configured once.
//
// The testAccPreCheck(t) function is invoked for every test and this prevents
// extraneous reconfiguration to the same values each time. However, this does
// not prevent reconfiguration that may happen should the address of
// testAccProvider be errantly reused in ProviderFactories.
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

// testAccPreCheckEnv contains all environment variables that must be present
// for acceptance tests to run. These are checked in testAccPreCheck.
var testAccPreCheckEnv = []string{
	"GRAFANA_URL",
	"GRAFANA_AUTH",
	"GRAFANA_ORG_ID",
}

// testAccPreCheck verifies required provider testing configuration. It should
// be present in every acceptance test.
//
// These verifications and configuration are preferred at this level to prevent
// provider developers from experiencing less clear errors for every test.
func testAccPreCheck(t *testing.T) {
	for _, e := range testAccPreCheckEnv {
		if v := os.Getenv(e); v == "" {
			t.Fatal(e + " must be set for acceptance tests")
		}
	}
	testAccProviderConfigure.Do(func() {
		// Since we are outside the scope of the Terraform configuration we must
		// call Configure() to properly initialize the provider configuration.
		err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		if err != nil {
			t.Fatal(err)
		}
	})
}

// testAccPreCheckCloud should be called by cloud acceptance tests
func testAccPreCheckCloud(t *testing.T) {
	testAccPreCheckEnv = append(testAccPreCheckEnv, "GRAFANA_SM_ACCESS_TOKEN")
	testAccPreCheck(t)
}

// testAccPreCheckCloudStack should be called by cloud stack acceptance tests
func testAccPreCheckCloudStack(t *testing.T) {
	testAccPreCheckEnv = append(testAccPreCheckEnv, "GRAFANA_CLOUD_API_KEY")
	testAccPreCheck(t)
}

// testAccExample returns an example config from the examples directory.
// Examples are used for both documentation and acceptance tests.
func testAccExample(t *testing.T, path string) string {
	example, err := ioutil.ReadFile("../examples/" + path)
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
	return enabled
}

func IsUnitTest(t *testing.T) {
	t.Helper()

	if accTestsEnabled(t, "TF_ACC") {
		t.Skip("Skipping acceptance tests")
	}
}

func CheckOSSTestsEnabled(t *testing.T) {
	t.Helper()
	if !accTestsEnabled(t, "TF_ACC_OSS") {
		t.Skip("TF_ACC_OSS must be set to a truthy value for OSS acceptance tests")
	}
}

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

func CheckCloudTestsEnabled(t *testing.T) {
	t.Helper()
	if !accTestsEnabled(t, "TF_ACC_CLOUD") {
		t.Skip("TF_ACC_CLOUD must be set to a truthy value for Cloud acceptance tests")
	}
}

func CheckEnterpriseTestsEnabled(t *testing.T) {
	t.Helper()
	if !accTestsEnabled(t, "TF_ACC_ENTERPRISE") {
		t.Skip("TF_ACC_ENTERPRISE must be set to a truthy value for Enterprise acceptance tests")
	}
}
