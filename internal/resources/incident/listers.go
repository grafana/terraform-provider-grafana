package incident

import (
	"context"
	"strconv"

	incident "github.com/grafana/incident-go"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

// listRoleIDs lists the IDs of all incident roles in the organization.
func listRoleIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.IncidentClient == nil {
		return nil, nil
	}
	roles, err := incident.NewRolesService(client.IncidentClient).GetRoles(ctx, incident.GetRolesRequest{})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(roles.Roles))
	for _, role := range roles.Roles {
		ids = append(ids, strconv.Itoa(role.RoleID))
	}
	return ids, nil
}
