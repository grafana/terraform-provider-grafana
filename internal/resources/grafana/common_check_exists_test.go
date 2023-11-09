package grafana_test

import (
	"fmt"
	"strconv"

	gapi "github.com/grafana/grafana-api-golang-client"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/access_control"
	"github.com/grafana/grafana-openapi-client-go/client/annotations"
	"github.com/grafana/grafana-openapi-client-go/client/datasources"
	"github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/client/library_elements"
	"github.com/grafana/grafana-openapi-client-go/client/playlists"
	"github.com/grafana/grafana-openapi-client-go/client/service_accounts"
	"github.com/grafana/grafana-openapi-client-go/client/teams"
	"github.com/grafana/grafana-openapi-client-go/client/users"
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
	annotationsCheckExists = newCheckExistsHelper(
		func(a *models.Annotation) string { return strconv.FormatInt(a.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Annotation, error) {
			params := annotations.NewGetAnnotationByIDParams().WithAnnotationID(id)
			resp, err := client.Annotations.GetAnnotationByID(params, nil)
			return payloadOrError(resp, err)
		},
	)
	datasourceCheckExists = newCheckExistsHelper(
		func(d *models.DataSource) string { return strconv.FormatInt(d.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.DataSource, error) {
			params := datasources.NewGetDataSourceByIDParams().WithID(id)
			resp, err := client.Datasources.GetDataSourceByID(params, nil)
			return payloadOrError(resp, err)
		},
	)
	folderCheckExists = newCheckExistsHelper(
		func(f *models.Folder) string { return strconv.FormatInt(f.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Folder, error) {
			params := folders.NewGetFolderByIDParams().WithFolderID(mustParseInt64(id))
			resp, err := client.Folders.GetFolderByID(params, nil)
			return payloadOrError(resp, err)
		},
	)
	libraryPanelCheckExists = newCheckExistsHelper(
		func(t *models.LibraryElementResponse) string { return t.Result.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.LibraryElementResponse, error) {
			params := library_elements.NewGetLibraryElementByUIDParams().WithLibraryElementUID(id)
			resp, err := client.LibraryElements.GetLibraryElementByUID(params, nil)
			return payloadOrError(resp, err)
		},
	)
	playlistCheckExists = newCheckExistsHelper(
		func(p *models.Playlist) string {
			if p.UID == "" {
				return strconv.FormatInt(p.ID, 10)
			}
			return p.UID
		},
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Playlist, error) {
			params := playlists.NewGetPlaylistParams().WithUID(id)
			resp, err := client.Playlists.GetPlaylist(params, nil)
			return payloadOrError(resp, err)
		},
	)
	roleCheckExists = newCheckExistsHelper(
		func(r *models.RoleDTO) string { return r.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.RoleDTO, error) {
			params := access_control.NewGetRoleParams().WithRoleUID(id)
			resp, err := client.AccessControl.GetRole(params, nil)
			return payloadOrError(resp, err)
		},
	)
	serviceAccountCheckExists = newCheckExistsHelper(
		func(t *models.ServiceAccountDTO) string { return strconv.FormatInt(t.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.ServiceAccountDTO, error) {
			params := service_accounts.NewRetrieveServiceAccountParams().WithServiceAccountID(mustParseInt64(id))
			resp, err := client.ServiceAccounts.RetrieveServiceAccount(params, nil)
			return payloadOrError(resp, err)
		},
	)
	teamCheckExists = newCheckExistsHelper(
		func(t *models.TeamDTO) string { return strconv.FormatInt(t.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.TeamDTO, error) {
			params := teams.NewGetTeamByIDParams().WithTeamID(id)
			resp, err := client.Teams.GetTeamByID(params, nil)
			return payloadOrError(resp, err)
		},
	)
	userCheckExists = newCheckExistsHelper(
		func(u *models.UserProfileDTO) string { return strconv.FormatInt(u.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.UserProfileDTO, error) {
			params := users.NewGetUserByIDParams().WithUserID(mustParseInt64(id))
			resp, err := client.Users.GetUserByID(params, nil)
			return payloadOrError(resp, err)
		},
	)
)

type checkExistsGetResourceFunc[T interface{}] func(client *goapi.GrafanaHTTPAPI, id string) (*T, error)
type checkExistsGetIDFunc[T interface{}] func(*T) string

type checkExistsHelper[T interface{}] struct {
	getIDFunc       func(*T) string
	getResourceFunc checkExistsGetResourceFunc[T]
}

// newCheckExistsHelper creates a test helper that checks if a resource exists or not.
// The getIDFunc function should return the ID of the resource.
// The getResourceFunc function should return the resource from the given ID.
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

		// Sanity check: The "destroyed" function should fail here because the resource exists
		if err := h.destroyed(obj, &gapi.Org{ID: orgID})(s); err == nil {
			return fmt.Errorf("the destroyed check passed but shouldn't for resource %s with ID %q. This is a bug in the test", rn, rs.Primary.ID)
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
			return fmt.Errorf("%T %s still exists in org %d", v, id, orgID)
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("error checking if resource %s exists in org %d: %s", id, orgID, err)
		}
		return nil
	}
}

func mustParseInt64(s string) int64 {
	i, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return i
}

// payloadOrError returns the error if not nil, or the payload otherwise. This saves 4 lines of code on each helper.
func payloadOrError[T interface{}, R interface{ GetPayload() *T }](resp R, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}
