package grafana

import (
	"fmt"
	"regexp"
	"strconv"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccTeam_basic(t *testing.T) {
	var team gapi.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "email", "teamEmail@example.com",
					),
					resource.TestMatchResourceAttr(
						"grafana_team.test", "id", regexp.MustCompile(`\d+`),
					),
				),
			},
			{
				Config: testAccTeamConfig_updateName,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "name", "terraform-acc-test-update",
					),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "email", "teamEmailUpdate@example.com",
					),
				),
			},
		},
	})
}

func TestAccTeam_Members(t *testing.T) {
	var team gapi.Team

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccTeamCheckDestroy(&team),
		Steps: []resource.TestStep{
			{
				Config: testAccTeamConfig_memberAdd,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "members.#", "1",
					),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "members.0", "john.doe@example.com",
					),
				),
			},
			{
				Config: testAccTeamConfig_memberRemove,
				Check: resource.ComposeTestCheckFunc(
					testAccTeamCheckExists("grafana_team.test", &team),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "name", "terraform-acc-test",
					),
					resource.TestCheckResourceAttr(
						"grafana_team.test", "members.#", "0",
					),
				),
			},
		},
	})
}

func testAccTeamCheckExists(rn string, a *gapi.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s", rn)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}
		tmp, err := strconv.ParseInt(rs.Primary.ID, 10, 64)
		id := int64(tmp)
		if err != nil {
			return fmt.Errorf("resource id is malformed")
		}

		client := testAccProvider.Meta().(*gapi.Client)
		team, err := client.Team(id)
		if err != nil {
			return fmt.Errorf("error getting data source: %s", err)
		}

		*a = *team

		return nil
	}
}

func testAccTeamCheckDestroy(a *gapi.Team) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*gapi.Client)
		team, err := client.Team(a.ID)
		if err == nil && team.Name != "" {
			return fmt.Errorf("team still exists")
		}
		return nil
	}
}

const testAccTeamConfig_basic = `
resource "grafana_team" "test" {
  name  = "terraform-acc-test"
  email = "teamEmail@example.com"
}
`
const testAccTeamConfig_updateName = `
resource "grafana_team" "test" {
  name    = "terraform-acc-test-update"
  email   = "teamEmailUpdate@example.com"
}
`
const testAccTeamConfig_memberAdd = `
resource "grafana_team" "test" {
  name    = "terraform-acc-test"
  email   = "teamEmail@example.com"
  members = [
    "john.doe@example.com",
  ]
}
`
const testAccTeamConfig_memberRemove = `
resource "grafana_team" "test" {
  name    = "terraform-acc-test"
  email   = "teamEmail@example.com"
  members = [ ]
}
`
