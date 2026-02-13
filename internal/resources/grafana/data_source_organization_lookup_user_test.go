package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationLookupUser_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user models.UserProfileDTO
	checks := []resource.TestCheckFunc{
		userCheckExists.exists("grafana_user.test", &user),
	}

	checks = append(checks,
		resource.TestMatchResourceAttr(
			"data.grafana_organization_lookup_user.test", "user_id", common.IDRegexp,
		),
		resource.TestCheckResourceAttr(
			"data.grafana_organization_lookup_user.test", "login", "test-datasource",
		),
	)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_organization_lookup_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
