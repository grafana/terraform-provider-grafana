package grafana_test

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/go-openapi/runtime"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana-openapi-client-go/client/annotations"
	"github.com/grafana/grafana-openapi-client-go/client/provisioning"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// Helpers that check if a resource exists or doesn't. To define a new one, use the newCheckExistsHelper function.
// A function that gets a resource by their Terraform ID is required.
var (
	alertingContactPointCheckExists = newCheckExistsHelper(
		func(p *models.ContactPoints) string { return (*p)[0].Name },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.ContactPoints, error) {
			params := provisioning.NewGetContactpointsParams().WithName(&id)
			resp, err := client.Provisioning.GetContactpoints(params)
			if castErr, ok := err.(*runtime.APIError); strings.HasPrefix(os.Getenv("GRAFANA_VERSION"), "10.4") && ok && castErr.Code == 500 {
				return nil, &runtime.APIError{Code: 404} // There's a bug in 10.4 where the API returns a 500 if no contact points are found
			}
			if err != nil {
				return nil, err
			}
			if len(resp.Payload) == 0 {
				return nil, &runtime.APIError{Code: 404}
			}
			return &resp.Payload, nil
		},
	)
	alertingMessageTemplateCheckExists = newCheckExistsHelper(
		func(t *models.NotificationTemplate) string { return t.Name },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.NotificationTemplate, error) {
			resp, err := client.Provisioning.GetTemplate(id, grafana.ContentTypeNegotiator(http.DefaultTransport))
			return payloadOrError(resp, err)
		},
	)
	alertingMuteTimingCheckExists = newCheckExistsHelper(
		func(t *models.MuteTimeInterval) string { return t.Name },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.MuteTimeInterval, error) {
			resp, err := client.Provisioning.GetMuteTiming(id)
			return payloadOrError(resp, err)
		},
	)
	alertingNotificationPolicyCheckExists = newCheckExistsHelper(
		func(t *models.Route) string { return "Global" }, // It's a singleton. ID doesn't matter.
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Route, error) {
			resp, err := client.Provisioning.GetPolicyTree()
			if err != nil {
				return nil, err
			}
			tree := resp.Payload
			if len(tree.Routes) == 0 || tree.Receiver == "grafana-default-email" {
				return nil, &runtime.APIError{Code: 404, Response: "the default notification policy is set"}
			}
			return tree, nil
		},
	)
	alertingRuleGroupCheckExists = newCheckExistsHelper(
		func(g *models.AlertRuleGroup) string { return g.FolderUID + ":" + g.Title },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.AlertRuleGroup, error) {
			folder, title, _ := strings.Cut(id, ":")
			resp, err := client.Provisioning.GetAlertRuleGroup(title, folder)
			return payloadOrError(resp, err)
		},
	)
	alertingRuleCheckExists = newCheckExistsHelper(
		func(r *models.ProvisionedAlertRule) string { return r.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.ProvisionedAlertRule, error) {
			resp, err := client.Provisioning.GetAlertRule(id)
			return payloadOrError(resp, err)
		},
	)
	annotationsCheckExists = newCheckExistsHelper(
		func(a *models.Annotation) string { return strconv.FormatInt(a.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Annotation, error) {
			resp, err := client.Annotations.GetAnnotations(annotations.NewGetAnnotationsParams())
			if err != nil {
				return nil, err
			}
			for _, a := range resp.Payload {
				if strconv.FormatInt(a.ID, 10) == id {
					return a, nil
				}
			}
			return nil, &runtime.APIError{Code: 404, Response: "annotation not found"}
		},
	)
	dashboardCheckExists = newCheckExistsHelper(
		func(d *models.DashboardFullWithMeta) string {
			if d.Dashboard == nil {
				return ""
			}
			return d.Dashboard.(map[string]interface{})["uid"].(string)
		},
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.DashboardFullWithMeta, error) {
			resp, err := client.Dashboards.GetDashboardByUID(id)
			return payloadOrError(resp, err)
		},
	)
	dashboardPublicCheckExists = newCheckExistsHelper(
		func(d *models.PublicDashboard) string { return d.DashboardUID + ":" + d.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.PublicDashboard, error) {
			dashboardUID, _, _ := strings.Cut(id, ":")
			resp, err := client.DashboardPublic.GetPublicDashboard(dashboardUID)
			return payloadOrError(resp, err)
		},
	)
	datasourceCheckExists = newCheckExistsHelper(
		func(d *models.DataSource) string { return d.UID },
		func(client *goapi.GrafanaHTTPAPI, uid string) (*models.DataSource, error) {
			resp, err := client.Datasources.GetDataSourceByUID(uid)
			return payloadOrError(resp, err)
		},
	)
	datasourcePermissionsCheckExists = newCheckExistsHelper(
		datasourceCheckExists.getIDFunc, // We use the DS as the reference
		func(client *goapi.GrafanaHTTPAPI, uid string) (*models.DataSource, error) {
			ds, err := datasourceCheckExists.getResourceFunc(client, uid)
			if err != nil {
				return nil, err
			}
			resp, err := client.AccessControl.GetResourcePermissions(ds.UID, "datasources")
			if err != nil {
				return nil, err
			}
			// Only managed permissions should be checked
			var managedPermissions []*models.ResourcePermissionDTO
			for _, p := range resp.Payload {
				if p.IsManaged {
					managedPermissions = append(managedPermissions, p)
				}
			}
			if len(managedPermissions) == 0 {
				return nil, &runtime.APIError{Code: 404, Response: "no managed permissions found"}
			}
			return ds, nil
		},
	)
	folderCheckExists = newCheckExistsHelper(
		func(f *models.Folder) string { return f.UID },
		func(client *goapi.GrafanaHTTPAPI, uid string) (*models.Folder, error) {
			resp, err := client.Folders.GetFolderByUID(uid)
			return payloadOrError(resp, err)
		},
	)
	libraryPanelCheckExists = newCheckExistsHelper(
		func(t *models.LibraryElementResponse) string { return t.Result.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.LibraryElementResponse, error) {
			resp, err := client.LibraryElements.GetLibraryElementByUID(id)
			return payloadOrError(resp, err)
		},
	)
	orgCheckExists = newCheckExistsHelper(
		func(o *models.OrgDetailsDTO) string { return strconv.FormatInt(o.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.OrgDetailsDTO, error) {
			resp, err := client.Orgs.GetOrgByID(mustParseInt64(id))
			if err, ok := err.(runtime.ClientResponseStatus); ok && err.IsCode(403) {
				return nil, &runtime.APIError{Code: 404, Response: "forbidden. The org either does not exist or the user does not have access to it"}
			}
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
			resp, err := client.Playlists.GetPlaylist(id)
			return payloadOrError(resp, err)
		},
	)
	roleCheckExists = newCheckExistsHelper(
		func(r *models.RoleDTO) string { return r.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.RoleDTO, error) {
			resp, err := client.AccessControl.GetRole(id)
			return payloadOrError(resp, err)
		},
	)
	roleAssignmentCheckExists = newCheckExistsHelper(
		func(r *models.RoleDTO) string { return r.UID },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.RoleDTO, error) {
			resp, err := client.AccessControl.GetRole(id)
			if err != nil {
				return nil, err
			}
			assignResp, err := client.AccessControl.GetRoleAssignments(id)
			if err != nil {
				return nil, err
			}
			assignments := assignResp.Payload
			if len(assignments.ServiceAccounts) == 0 && len(assignments.Teams) == 0 && len(assignments.Users) == 0 {
				return nil, &runtime.APIError{Code: 404, Response: "no assignments found"}
			}
			return resp.Payload, nil
		},
	)
	serviceAccountCheckExists = newCheckExistsHelper(
		func(t *models.ServiceAccountDTO) string { return strconv.FormatInt(t.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.ServiceAccountDTO, error) {
			resp, err := client.ServiceAccounts.RetrieveServiceAccount(mustParseInt64(id))
			return payloadOrError(resp, err)
		},
	)
	serviceAccountPermissionsCheckExists = newCheckExistsHelper(
		serviceAccountCheckExists.getIDFunc, // We use the SA as the reference
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.ServiceAccountDTO, error) {
			sa, err := serviceAccountCheckExists.getResourceFunc(client, id)
			if err != nil {
				return nil, err
			}
			resp, err := client.AccessControl.GetResourcePermissions(id, "serviceaccounts")
			if err != nil {
				return nil, err
			}
			// Only managed permissions should be checked
			var managedPermissions []*models.ResourcePermissionDTO
			for _, p := range resp.Payload {
				if p.IsManaged {
					managedPermissions = append(managedPermissions, p)
				}
			}
			if len(managedPermissions) == 0 {
				return nil, &runtime.APIError{Code: 404, Response: "no managed permissions found"}
			}
			return sa, nil
		},
	)
	teamCheckExists = newCheckExistsHelper(
		func(t *models.TeamDTO) string { return strconv.FormatInt(t.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.TeamDTO, error) {
			resp, err := client.Teams.GetTeamByID(id)
			return payloadOrError(resp, err)
		},
	)
	userCheckExists = newCheckExistsHelper(
		func(u *models.UserProfileDTO) string { return strconv.FormatInt(u.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.UserProfileDTO, error) {
			resp, err := client.Users.GetUserByID(mustParseInt64(id))
			return payloadOrError(resp, err)
		},
	)

	reportCheckExists = newCheckExistsHelper(
		func(u *models.Report) string { return strconv.FormatInt(u.ID, 10) },
		func(client *goapi.GrafanaHTTPAPI, id string) (*models.Report, error) {
			resp, err := client.Reports.GetReport(mustParseInt64(id))
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
		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(1)
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
		if err := h.destroyed(obj, &models.OrgDetailsDTO{ID: orgID})(s); err == nil {
			return fmt.Errorf("the destroyed check passed but shouldn't for resource %s with ID %q. This is a bug in the test", rn, rs.Primary.ID)
		}

		*v = *obj
		return nil
	}
}

// destroyed checks that the resource doesn't exist in the default org
// For non-default orgs, we should only check that the org was destroyed
func (h *checkExistsHelper[T]) destroyed(v *T, org *models.OrgDetailsDTO) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		var orgID int64 = 1
		if org != nil {
			orgID = org.ID
		}

		client := testutils.Provider.Meta().(*common.Client).GrafanaAPI.WithOrgID(orgID)
		id := h.getIDFunc(v)
		_, err := h.getResourceFunc(client, id)
		if err == nil {
			return fmt.Errorf("%T %s still exists in org %d", v, id, orgID)
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("error checking if resource %s was destroyed in org %d: %s", id, orgID, err)
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
