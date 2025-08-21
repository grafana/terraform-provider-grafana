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

func TestAccDatasourceOrganizationUser_disambiguation(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	var user1, user2 models.UserProfileDTO
	checks := []resource.TestCheckFunc{
		userCheckExists.exists("grafana_user.user1", &user1),
		userCheckExists.exists("grafana_user.user2", &user2),
		resource.TestCheckResourceAttr("data.grafana_organization_user.from_email", "login", "login1"),
		resource.TestCheckResourceAttr("data.grafana_organization_user.from_email", "email", "test@example.com"),
		resource.TestCheckResourceAttr("data.grafana_organization_user.from_login", "login", "log"),
		resource.TestCheckResourceAttr("data.grafana_organization_user.from_login", "email", "test@example.com~"),
	}

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy: resource.ComposeTestCheckFunc(
			userCheckExists.destroyed(&user1, nil),
			userCheckExists.destroyed(&user2, nil),
		),
		Steps: []resource.TestStep{
			{
				Config: testAccDatasourceOrganizationUserDisambiguation,
				Check:  resource.ComposeTestCheckFunc(checks...),
			},
		},
	})
}

var testAccDatasourceOrganizationUserDisambiguation = `
resource "grafana_user" "user1" {
  email = "test@example.com"
  name = "Test User 1"
  login = "login1"
  password = "my-password"
}

resource "grafana_user" "user2" {
  email = "test@example.com~"
  name = "Test User 1a"
  login = "log"
  password = "my-password"
}

data "grafana_organization_user" "from_email" {
  email = grafana_user.user1.email
}

data "grafana_organization_user" "from_login" {
  login = grafana_user.user2.login
}
`
