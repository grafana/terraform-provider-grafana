package grafana_test

import (
	"strconv"
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceRole_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=9.0.0")

	var role models.RoleDTO
	for _, hidden := range []bool{false, true} {
		formatted := strconv.FormatBool(hidden)
		t.Run("hidden="+formatted, func(t *testing.T) {
			config := testutils.TestAccExample(t, "data-sources/grafana_role/data-source.tf")
			if hidden {
				config = testutils.TestAccExampleWithReplace(t, "data-sources/grafana_role/data-source.tf", map[string]string{
					`hidden      = false`: `hidden      = true`,
				})
			}

			resource.Test(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				CheckDestroy:             roleCheckExists.destroyed(&role, nil),
				Steps: []resource.TestStep{
					{
						Config: config,
						Check: resource.ComposeTestCheckFunc(
							roleCheckExists.exists("grafana_role.test", &role),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "name", "test-role"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "description", "test-role description"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "uid", "test-ds-role-uid"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "version", "1"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "global", "true"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "hidden", formatted),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.#", "3"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.0.action", "org.users:add"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.0.scope", "users:*"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.1.action", "org.users:read"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.1.scope", "users:*"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.2.action", "org.users:write"),
							resource.TestCheckResourceAttr("data.grafana_role.from_name", "permissions.2.scope", "users:*"),
						),
					},
				},
			})
		})
	}
}
