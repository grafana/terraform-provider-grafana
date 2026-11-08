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

// listSeverityIDs lists the IDs of all incident severities in the organization,
// archived ones and the system-managed Pending severity included. Pending can
// be imported to manage its label, description, icon and colors.
func listSeverityIDs(ctx context.Context, client *common.Client, _ any) ([]string, error) {
	if client.IncidentClient == nil {
		return nil, nil
	}
	severities, err := incident.NewSeveritiesService(client.IncidentClient).GetOrgSeverities(ctx, incident.GetOrgSeveritiesRequest{})
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(severities.Severities))
	for _, severity := range severities.Severities {
		ids = append(ids, severity.SeverityID)
	}
	return ids, nil
}
