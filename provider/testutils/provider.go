package testutils

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// ProviderFactories is a static map containing only the main provider instance
// It is configured from the main provider package when the test suite is initialized
// but it is used in tests of every package
var ProviderFactories map[string]func() (*schema.Provider, error)

// Provider is the "main" provider instance
//
// This Provider can be used in testing code for API calls without requiring
// the use of saving and referencing specific ProviderFactories instances.
//
// It is configured from the main provider package when the test suite is initialized
// but it is used in tests of every package
var Provider *schema.Provider

// ProviderWaitGroup is a WaitGroup that is used to wait for the provider to be initialized
// The provider is initialized in the main package, but tests are run in every package
var ProviderWaitGroup sync.WaitGroup

func init() {
	ProviderWaitGroup.Add(1)
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
		example = strings.ReplaceAll(example, k, v)
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
func CheckOSSTestsEnabled(t *testing.T) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_OSS") {
		t.Skip("TF_ACC_OSS must be set to a truthy value for OSS acceptance tests")
	}

	CheckEnvVarsSet(t,
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

	if !AccTestsEnabled("TF_ACC_CLOUD_API") {
		t.Skip("TF_ACC_CLOUD_API must be set to a truthy value for Cloud API acceptance tests")
	}

	CheckEnvVarsSet(t, "GRAFANA_CLOUD_API_KEY", "GRAFANA_CLOUD_ORG")
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
		"GRAFANA_ORG_ID",
		"GRAFANA_SM_ACCESS_TOKEN",
		"GRAFANA_ONCALL_ACCESS_TOKEN",
	)
}

// CheckEnterpriseTestsEnabled checks if the enterprise tests are enabled. This should be the first line of any test that tests Grafana Enterprise features
func CheckEnterpriseTestsEnabled(t *testing.T) {
	t.Helper()

	if !AccTestsEnabled("TF_ACC_ENTERPRISE") {
		t.Skip("TF_ACC_ENTERPRISE must be set to a truthy value for Enterprise acceptance tests")
	}

	CheckEnvVarsSet(t,
		"GRAFANA_URL",
		"GRAFANA_AUTH",
		"GRAFANA_ORG_ID",
	)
}
