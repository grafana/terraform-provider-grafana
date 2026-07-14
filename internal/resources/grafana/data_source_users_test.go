package grafana_test

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceUsers_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	checks := []resource.TestCheckFunc{
		// Let's not test the number of users found as due to the test concurrency we might not find the expected value
		// We know we have at least two users: admin and the one created by the example
		resource.TestCheckTypeSetElemNestedAttrs(
			"data.grafana_users.all_users", "users.*", map[string]string{
				"login": "admin",
			}),
		resource.TestCheckTypeSetElemNestedAttrs(
			"data.grafana_users.all_users", "users.*", map[string]string{
				"login": "test-grafana-users",
				"email": "all_users@example.com",
			}),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_users/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
