package testutils

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/grafana/terraform-provider-grafana/v3/pkg/provider"
	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	// ProtoV5ProviderFactories is a static map containing the grafana provider instance
	// It is used to configure the provider in acceptance tests
	ProtoV5ProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
		"grafana": func() (tfprotov5.ProviderServer, error) {
			// Create a provider server
			ctx := context.Background()
			server, err := provider.MakeProviderServer(ctx, "testacc")
			if err != nil {
				return nil, err
			}

			// Get the provider schema and create a provider configuration
			// The config is empty because we'll use environment variables to configure the provider
			schemaResp, err := server.GetProviderSchema(ctx, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to get provider schema: %v", err)
			}
			fields := map[string]tftypes.Value{}
			for _, v := range schemaResp.Provider.Block.Attributes {
				fields[v.Name] = tftypes.NewValue(v.Type, nil)
			}
			testValue := tftypes.NewValue(schemaResp.Provider.ValueType(), fields)
			testDynamicValue, err := tfprotov5.NewDynamicValue(schemaResp.Provider.ValueType(), testValue)
			if err != nil {
				return nil, err
			}

			// Configure the provider
			configureResp, err := server.ConfigureProvider(context.Background(), &tfprotov5.ConfigureProviderRequest{Config: &testDynamicValue})
			if err != nil || len(configureResp.Diagnostics) > 0 {
				if err == nil {
					errs := []error{}
					for _, diag := range configureResp.Diagnostics {
						errs = append(errs, fmt.Errorf("%s %s: %s", diag.Severity, diag.Summary, diag.Detail))
					}
					err = errors.Join(errs...)
				}
				return nil, fmt.Errorf("failed to configure provider: %v", err)
			}
			return server, nil
		},
	}

	// Provider is the "main" provider instance
	//
	// This Provider can be used in testing code for API calls without requiring
	// the use of saving and referencing specific ProviderFactories instances.
	//
	// It is configured from the main provider package when the test suite is initialized
	// but it is used in tests of every package
	Provider *schema.Provider
)

func init() {
	Provider = provider.Provider("testacc")

	// If any acceptance tests are enabled, the test provider must be configured
	if AccTestsEnabled("TF_ACC") {
		// Since we are outside the scope of the Terraform configuration we must
		// call Configure() to properly initialize the provider configuration.
		err := Provider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		if err != nil {
			panic(fmt.Sprintf("failed to configure provider: %v", err))
		}
	}
}

// TestAccExample returns an example config from the examples directory.
// Examples are used for both documentation and acceptance tests.
func TestAccExample(t *testing.T, path string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not get current file")
	}
	example, err := os.ReadFile(filepath.Join(filepath.Dir(currentFile), "..", "..", "examples", path))
	if err != nil {
		t.Fatal(err)
	}
	return string(example)
}

// TestAccExampleWithReplace works like testAccExample, but replaces strings in the example.
func TestAccExampleWithReplace(t *testing.T, path string, replaceMap map[string]string) string {
	t.Helper()

	example := TestAccExample(t, path)
	for k, v := range replaceMap {
		beforeReplace := example
		example = strings.ReplaceAll(example, k, v)
		if example == beforeReplace {
			t.Fatalf("%q not found to replace in example %s", k, path)
		}
	}

	return example
}

func AccTestsEnabled(envVarName string) bool {
	v, ok := os.LookupEnv(envVarName)
	if !ok {
		return false
	}
	enabled, err := strconv.ParseBool(v)
	if err != nil {
		panic(fmt.Sprintf("%s must be set to a boolean value", envVarName))
	}

	return enabled
}

func CheckEnvVarsSet(t *testing.T, envVars ...string) {
	t.Helper()

	for _, envVar := range envVars {
		if os.Getenv(envVar) == "" {
			t.Fatalf("%s must be set", envVar)
		}
	}
}

func IsUnitTest(t *testing.T) {
	t.Helper()

	if AccTestsEnabled("TF_ACC") {
		t.Skip("Skipping acceptance tests")
	}
}

// CheckOSSTestsEnabled checks if the OSS acceptance tests are enabled. This should be the first line of any test that uses Grafana OSS features only
func CheckOSSTestsEnabled(t *testing.T, semverConstraintOptional ...string) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_OSS") {
		t.Skip("TF_ACC_OSS must be set to a truthy value for OSS acceptance tests")
	}

	CheckEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_VERSION",
	)
	checkSemverConstraint(t, semverConstraintOptional...)
}

// CheckCloudTestsEnabled checks if the cloud tests are enabled. This should be the first line of any test that tests Cloud API features
func CheckCloudAPITestsEnabled(t *testing.T) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_CLOUD_API") {
		t.Skip("TF_ACC_CLOUD_API must be set to a truthy value for Cloud API acceptance tests")
	}

	CheckEnvVarsSet(t, "GRAFANA_CLOUD_ACCESS_POLICY_TOKEN", "GRAFANA_CLOUD_ORG")
}

// CheckCloudInstanceTestsEnabled checks if tests that run on cloud instances are enabled. This should be the first line of any test that tests Grafana Cloud Pro features
func CheckCloudInstanceTestsEnabled(t *testing.T) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_CLOUD_INSTANCE") {
		t.Skip("TF_ACC_CLOUD_INSTANCE must be set to a truthy value for Cloud instance acceptance tests")
	}

	CheckEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_SM_ACCESS_TOKEN",
		"GRAFANA_ONCALL_ACCESS_TOKEN",
		"GRAFANA_CLOUD_PROVIDER_URL",
		"GRAFANA_CLOUD_PROVIDER_ACCESS_TOKEN",
		"GRAFANA_CLOUD_PROVIDER_AWS_ROLE_ARN",
		"GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID",
		"GRAFANA_FLEET_MANAGEMENT_AUTH",
		"GRAFANA_FLEET_MANAGEMENT_URL",
	)
}

// CheckEnterpriseTestsEnabled checks if the enterprise tests are enabled. This should be the first line of any test that tests Grafana Enterprise features
func CheckEnterpriseTestsEnabled(t *testing.T, semverConstraintOptional ...string) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_ENTERPRISE") {
		t.Skip("TF_ACC_ENTERPRISE must be set to a truthy value for Enterprise acceptance tests")
	}

	CheckEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
	)
	checkSemverConstraint(t, semverConstraintOptional...)
}

func checkSemverConstraint(t *testing.T, semverConstraintOptional ...string) {
	t.Helper()

	if len(semverConstraintOptional) > 1 {
		panic("checkSemverConstraint accepts at most one argument")
	}
	if len(semverConstraintOptional) == 0 {
		return
	}

	semverConstraint := semverConstraintOptional[0]
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
