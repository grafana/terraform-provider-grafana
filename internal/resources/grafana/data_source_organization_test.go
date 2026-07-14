package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganization_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var org models.OrgDetailsDTO
	checks := []resource.TestCheckFunc{
		orgCheckExists.exists("grafana_organization.test", &org),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "name", "test-org",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "admins.#", "1",
		),
		resource.TestCheckResourceAttr(
			// Grafana automatically adds the admin user to all orgs
			"data.grafana_organization.from_name", "admins.0", "admin@localhost",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "viewers.#", "2",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "viewers.0", "viewer-01@example.com",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "viewers.1", "viewer-02@example.com",
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization.from_name", "editors.#", "0",
		),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             orgCheckExists.destroyed(&org, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_organization/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
