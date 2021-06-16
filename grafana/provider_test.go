package grafana

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"testing"

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

// testAccProviderConfigure ensures testAccProvider is only configured once
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
		"grafana": func() (*schema.Provider, error) {
			return Provider("testacc")(), nil
		},
	}
}

func TestProvider(t *testing.T) {
	if err := Provider("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

// testAccPreCheck verifies required provider testing configuration
//
// This PreCheck function should be present in every acceptance test. It allows
// test configurations to omit a provider configuration with region and ensures
// testing functions that attempt to call AWS APIs are previously configured.
//
// These verifications and configuration are preferred at this level to prevent
// provider developers from experiencing less clear errors for every test.
func testAccPreCheck(t *testing.T) {
	testAccProviderConfigure.Do(func() {
		if v := os.Getenv("GRAFANA_URL"); v == "" {
			t.Fatal("GRAFANA_URL must be set for acceptance tests")
		}
		if v := os.Getenv("GRAFANA_AUTH"); v == "" {
			t.Fatal("GRAFANA_AUTH must be set for acceptance tests")
		}
		if v := os.Getenv("GRAFANA_ORG_ID"); v == "" {
			t.Fatal("GRAFANA_ORG_ID must be set for acceptance tests")
		}
		// Since we are outside the scope of the Terraform configuration we must
		// call Configure() to properly initialize the provider configuration.
		err := testAccProvider.Configure(context.Background(), terraform.NewResourceConfigRaw(nil))
		if err != nil {
			t.Fatal(err)
		}
	})
}

// testAccExample returns an example config from the examples directory.
// Examples are used for both documentation and acceptance tests.
func testAccExample(t *testing.T, path string) string {
	path = fmt.Sprintf("../examples/%s", path)
	example, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(example)
}
