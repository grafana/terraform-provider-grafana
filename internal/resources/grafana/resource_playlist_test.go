package grafana_test

import (
	"fmt"
	"testing"

	gapi "github.com/grafana/grafana-api-golang-client"
	"github.com/grafana/grafana-openapi-client-go/client/playlists"
	"github.com/grafana/grafana-openapi-client-go/models"
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
	var playlist models.Playlist

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      playlistCheckExists.destroyed(&playlist, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					playlistCheckExists.exists(paylistResource, &playlist),
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
	var playlist models.Playlist

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      playlistCheckExists.destroyed(&playlist, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					playlistCheckExists.exists(paylistResource, &playlist),
					resource.TestCheckResourceAttr(paylistResource, "interval", "5m"),
				),
			},
			{
				Config: testAccPlaylistConfigBasic(rName, "10m"),
				Check: resource.ComposeTestCheckFunc(
					playlistCheckExists.exists(paylistResource, &playlist),
					resource.TestCheckResourceAttr(paylistResource, "interval", "10m"),
				),
			},
			{
				Config: testAccPlaylistConfigUpdate(updatedName),
				Check: resource.ComposeTestCheckFunc(
					playlistCheckExists.exists(paylistResource, &playlist),
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
	var playlist models.Playlist

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      playlistCheckExists.destroyed(&playlist, nil),
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigBasic(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					playlistCheckExists.exists(paylistResource, &playlist),
					testAccPlaylistDisappears(),
				),
				ExpectNonEmptyPlan: true,
			},
		},
	})
}

func TestAccPlaylist_inOrg(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=9.0.0") // Querying org-specific playlists is broken pre-9

	rName := acctest.RandomWithPrefix("tf-acc-test")
	var org gapi.Org
	var playlist models.Playlist

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testutils.ProviderFactories,
		CheckDestroy:      playlistCheckExists.destroyed(&playlist, &org),
		Steps: []resource.TestStep{
			{
				Config: testAccPlaylistConfigInOrg(rName, "5m"),
				Check: resource.ComposeTestCheckFunc(
					// Check that the playlist is in the correct organization
					resource.TestMatchResourceAttr(paylistResource, "id", nonDefaultOrgIDRegexp),
					testAccOrganizationCheckExists("grafana_organization.test", &org),
					checkResourceIsInOrg(paylistResource, "grafana_organization.test"),

					playlistCheckExists.exists(paylistResource, &playlist),
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

func testAccPlaylistDisappears() resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[paylistResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", paylistResource)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client, _, playlistID := grafana.OAPIClientFromExistingOrgResource(testutils.Provider.Meta(), rs.Primary.ID)
		_, err := client.Playlists.DeletePlaylist(playlists.NewDeletePlaylistParams().WithUID(playlistID), nil)
		return err
	}
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
