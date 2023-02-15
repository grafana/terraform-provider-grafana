package grafana_test

import (
	"fmt"
	"strings"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const paylistResource = "grafana_playlist.test"

func TestAccPlaylist_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestMatchResourceAttr(paylistResource, "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr(paylistResource, "name", rName),
					resource.TestCheckResourceAttr(paylistResource, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By Tag",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "2",
						"title": "Terraform Dashboard By ID",
					}),
				),
			},
			{
				ResourceName:      paylistResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_update(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")
	updatedName := "updated name"

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestCheckResourceAttr(paylistResource, "interval", "5m"),
				),
			},
			{
				Config: testAccPlaylistConfigBasic(rName, "10m"),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestCheckResourceAttr(paylistResource, "interval", "10m"),
				),
			},
			{
				Config: testAccPlaylistConfigUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					resource.TestMatchResourceAttr(paylistResource, "id", defaultOrgIDRegexp),
					resource.TestCheckResourceAttr(paylistResource, "name", updatedName),
					resource.TestCheckResourceAttr(paylistResource, "item.#", "1"),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By ID",
						"type":  "dashboard_by_id",
						"value": "3",
					}),
				),
			},
			{
				ResourceName:      paylistResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccPlaylist_disappears(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					testAccPlaylistCheckExists(),
					testAccPlaylistDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccPlaylist_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t)

	rName := acctest.RandomWithPrefix("tf-acc-test")
	var org gapi.Org

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      testAccPlaylistDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigInOrg(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					// Check that the playlist is in the correct organization
					resource.TestMatchResourceAttr(paylistResource, "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg(paylistResource, "grafana_organization.test"),

					testAccPlaylistCheckExists(),
					resource.TestCheckResourceAttr(paylistResource, "name", rName),
					resource.TestCheckResourceAttr(paylistResource, "item.#", "2"),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "1",
						"title": "Terraform Dashboard By Tag",
					}),
					resource.TestCheckTypeSetElemNestedAttrs(paylistResource, "item.*", map[string]string{
						"order": "2",
						"title": "Terraform Dashboard By ID",
					}),
				),
			},
			{
				ResourceName:      paylistResource,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccPlaylistCheckExists() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[paylistResource]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", paylistResource, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI

		orgID, playlistID := grafana.SplitOSSOrgID(rs.Primary.ID)

		// If the org ID is set, check that the playlist doesn't exist in the default org
		if orgID > 0 {
			playlist, err := client.Playlist(playlistID)
			if err == nil || playlist != nil {
				return fmt.Errorf("expected no playlist with ID %s in default org but found one", playlistID)
			}
			client = client.WithOrgID(orgID)
		}

		_, err := client.Playlist(playlistID)
		if err != nil {
			return fmt.Errorf("error getting playlist: %w", err)
		}

		return nil
	}
}

func testAccPlaylistDisappears() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[paylistResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", paylistResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client, _, playlistID := grafana.ClientFromOSSOrgID(testutils.Provider.Meta(), rs.Primary.ID)

		return client.DeletePlaylist(playlistID)
	}
}

func testAccPlaylistDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_playlist" {
			continue
		}

		client, _, playlistID := grafana.ClientFromOSSOrgID(testutils.Provider.Meta(), rs.Primary.ID)
		playlist, err := client.Playlist(playlistID)

		if err != nil {
			if strings.HasPrefix(err.Error(), "status: 404") {
				continue
			}
			return err
		}

		if playlist != nil && playlist.ID != 0 {
			return fmt.Errorf("Playlist still exists: %+v", playlist)
		}
	}

	return nil
}

func testAccPlaylistConfigBasic(name, interval string) string {
	return fmt.Sprintf(`
resource "grafana_playlist" "test" {
	name     = %[1]q
	interval = %[2]q

	item {
		order = 2
		title = "Terraform Dashboard By ID"
	}

	item {
		order = 1
		title = "Terraform Dashboard By Tag"
	}

}
`, name, interval)
}

func testAccPlaylistConfigInOrg(name, interval string) string {
	return fmt.Sprintf(`
resource "grafana_organization" "test" {
	name = %[1]q
}

resource "grafana_playlist" "test" {
	org_id   = grafana_organization.test.id
	name     = %[1]q
	interval = %[2]q

	item {
		order = 2
		title = "Terraform Dashboard By ID"
	}

	item {
		order = 1
		title = "Terraform Dashboard By Tag"
	}

}
`, name, interval)
}

func testAccPlaylistConfigUpdate(name string) string {
	return fmt.Sprintf(`
resource "grafana_playlist" "test" {
	name     = %[1]q
	interval = "5m"
	
	item {
		order = 1
		title = "Terraform Dashboard By ID"
		type = "dashboard_by_id"
		value = "3"
	}
}
`, name)
}
