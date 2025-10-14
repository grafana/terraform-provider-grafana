package grafana_test

import (
	"errors"
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

func TestAccServiceAccount_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	var sa models.ServiceAccountDTO
	var updatedSA models.ServiceAccountDTO
	name := acctest.RandString(10)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountCheckExists.destroyed(&updatedSA, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountConfig(name, "Editor", false),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", defaultOrgIDRegexp),
					testutils.CheckLister("grafana_service_account.test"),
				),
			},
			// Change the name. Check that the ID stays the same.
			{
				Config: testServiceAccountConfig(name+"-updated", "Editor", false),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &updatedSA),
					func(s *terraform.State) error {
						if sa.ID != updatedSA.ID {
							return errors.New("ID changed")
						}
						return nil
					},
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name+"-updated"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "Editor"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", defaultOrgIDRegexp),
				),
			},
			// Import test
			{
				ResourceName:      "grafana_service_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceAccount_NoneRole(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=10.2.0")

	name := acctest.RandString(10)
	var sa models.ServiceAccountDTO

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             serviceAccountCheckExists.destroyed(&sa, nil),
		Steps: []resource.TestStep{
			{
				Config: testServiceAccountConfig(name, "None", false),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "None"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "is_disabled", "false"),
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", defaultOrgIDRegexp),
				),
			},
			// Disable the SA
			{
				Config: testServiceAccountConfig(name, "None", true),
				Check: resource.ComposeTestCheckFunc(
					serviceAccountCheckExists.exists("grafana_service_account.test", &sa),
					resource.TestCheckResourceAttr("grafana_service_account.test", "name", name),
					resource.TestCheckResourceAttr("grafana_service_account.test", "org_id", "1"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "role", "None"),
					resource.TestCheckResourceAttr("grafana_service_account.test", "is_disabled", "true"),
					resource.TestMatchResourceAttr("grafana_service_account.test", "id", defaultOrgIDRegexp),
				),
			},
			// Import test
			{
				ResourceName:      "grafana_service_account.test",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccServiceAccount_many_longtest(t *testing.T) {
	if testing.Short() { // Also named "longtest" to allow targeting with -run=.*longtest
		t.Skip("skipping test in short mode")
	}
	testutils.CheckOSSTestsEnabled(t, ">=9.1.0")

	name := acctest.RandString(10)

	// For each SA, check that it exists and has the correct name, then check that it is properly destroyed
	createdServiceAccounts := make([]models.ServiceAccountDTO, 60)
	checks := []resource.TestCheckFunc{}
	destroyedChecks := []resource.TestCheckFunc{}
	for i := range 60 {
		checks = append(checks, serviceAccountCheckExists.exists(fmt.Sprintf("grafana_service_account.test_%d", i), &createdServiceAccounts[i]))
		checks = append(checks, resource.TestCheckResourceAttr(fmt.Sprintf("grafana_service_account.test_%d", i), "name", fmt.Sprintf("%s-%d", name, i)))
		destroyedChecks = append(destroyedChecks, serviceAccountCheckExists.destroyed(&createdServiceAccounts[i], nil))
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             resource.ComposeAggregateTestCheckFunc(destroyedChecks...),
		Steps: []resource.TestStep{
			{
				Config: testManyServiceAccountsConfig(name, 60),
				Check:  resource.ComposeAggregateTestCheckFunc(checks...),
			},
		},
	})
}

func TestAccServiceAccount_invalid_role(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				ExpectError: regexp.MustCompile(`.*expected role to be one of \[.+\], got InvalidRole`),
				Config:      testServiceAccountConfig("any", "InvalidRole", false),
			},
		},
	})
}

func testManyServiceAccountsConfig(prefix string, count int) string {
	config := ``

	for i := range count {
		config += fmt.Sprintf(`
		resource "grafana_service_account" "test_%[2]d" {
			name        = "%[1]s-%[2]d"
			is_disabled = false
			role        = "Viewer"
		}
`, prefix, i)
	}

	return config
}

func testServiceAccountConfig(name, role string, disabled bool) string {
	return fmt.Sprintf(`
resource "grafana_service_account" "test" {
	name        = "%[1]s"
	role        = "%[2]s"
	is_disabled = %[3]t
}`, name, role, disabled)
}
