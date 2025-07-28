package grafana_test

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDatasourceOrganizationUser_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user models.UserProfileDTO
	checks := []resource.TestCheckFunc{
		userCheckExists.exists("grafana_user.test", &user),
	}
	for _, rName := range []string{"from_email", "from_login"} {
		checks = append(checks,
			resource.TestMatchResourceAttr(
				"data.grafana_organization_user."+rName, "user_id", common.IDRegexp,
			),
			resource.TestCheckResourceAttr(
				"data.grafana_organization_user."+rName, "login", "test-datasource",
			),
		)
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             userCheckExists.destroyed(&user, nil),
		Steps: []resource.TestStep{
			{
				Config: testutils.TestAccExample(t, "data-sources/grafana_organization_user/data-source.tf"),
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}
