package grafana_test

import (
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/grafana/terraform-provider-grafana/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Helpers that check if a resource exists or doesn't. To define a new one, use the newCheckExistsHelper function.
// A function that gets a resource by their Terraform ID is required.
var (
	folderCheckExists = newCheckExistsHelper(
		func(f *models.Folder) string { return f.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Folder, error) {
			return grafana.GetFolderByIDorUID(client.Folders, id)
		},
	)
	teamCheckExists = newCheckExistsHelper(
		func(t *models.TeamDTO) string { return strconv.FormatInt(t.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.TeamDTO, error) {
			params := teams.NewGetTeamByIDParams().WithTeamID(id)
			team, err := client.Teams.GetTeamByID(params, nil)
			if err != nil {
				return nil, err
			}
			return team.GetPayload(), nil
		},
	)
)

type checkExistsGetResourceFunc[T interface{}] func(client *goapi.GrafanaHTTPAPI, id string) (*T, error)
type checkExistsGetIDFunc[T interface{}] func(*T) string

type checkExistsHelper[T interface{}] struct {
	getIDFunc       func(*T) string
	getResourceFunc checkExistsGetResourceFunc[T]
}

func newCheckExistsHelper[T interface{}](getIDFunc checkExistsGetIDFunc[T], getResourceFunc checkExistsGetResourceFunc[T]) checkExistsHelper[T] {
	return checkExistsHelper[T]{getIDFunc: getIDFunc, getResourceFunc: getResourceFunc}
}

// exists checks that the resource exists in the correct org.
// If the org is not the default one, it also checks that the resource doesn't exist in the default org.
func (h *checkExistsHelper[T]) exists(rn string, v *T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		orgID, idStr := grafana.SplitOrgResourceID(rs.Primary.ID)

		// If the org ID is set, check that the resource doesn't exist in the default org
		client := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.WithOrgID(1)
		if orgID > 1 {
			_, err := h.getResourceFunc(client, idStr)
			if err == nil {
				return fmt.Errorf("resource %s with ID %q exists in org %d, but should not", rn, rs.Primary.ID, orgID)
			} else if !common.IsNotFoundError(err) {
				return fmt.Errorf("error checking if resource %s with ID %q exists in org %d: %s", rn, rs.Primary.ID, orgID, err)
			}
			client = client.WithOrgID(orgID)
		}

		obj, err := h.getResourceFunc(client, idStr)
		if err != nil {
			return fmt.Errorf("error getting resource %s with ID %q: %s", rn, rs.Primary.ID, err)
		}

		*v = *obj
		return nil
	}
}

// destroyed checks that the resource doesn't exist in the default org
// For non-default orgs, we should only check that the org was destroyed
func (h *checkExistsHelper[T]) destroyed(v *T, org *gapi.Org) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var orgID int64 = 1
		if org != nil {
			orgID = org.ID
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaOAPI.WithOrgID(orgID)
		id := h.getIDFunc(v)
		_, err := h.getResourceFunc(client, id)
		if err == nil {
			return fmt.Errorf("resource %s still exists in org %d", id, orgID)
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("error checking if resource %s exists in org %d: %s", id, orgID, err)
		}
		return nil
	}
}
