package grafana

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/stretchr/testify/require"
)

func TestBuildSetResourcePermissionCommands_SkipsSubjectlessManagedPermission(t *testing.T) {
	current := []*models.ResourcePermissionDTO{
		{
			// Defensive regression case: managed permission without a subject should be ignored
			// instead of being turned into a "subjectless" delete command.
			IsManaged:   true,
			IsInherited: false,
			Permission:  "Query",
		},
	}
	desired := []*models.SetResourcePermissionCommand{}

	areEqual := func(a *models.ResourcePermissionDTO, b *models.SetResourcePermissionCommand) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltInRole
	}

	cmds, err := buildSetResourcePermissionCommands(current, desired, areEqual)
	require.NoError(t, err)
	require.Empty(t, cmds)
}

func TestBuildSetResourcePermissionCommands_RejectsDesiredWithoutSubject(t *testing.T) {
	current := []*models.ResourcePermissionDTO{}
	desired := []*models.SetResourcePermissionCommand{
		{
			Permission: "Query",
		},
	}

	areEqual := func(a *models.ResourcePermissionDTO, b *models.SetResourcePermissionCommand) bool {
		return a.Permission == b.Permission && a.TeamID == b.TeamID && a.UserID == b.UserID && a.BuiltInRole == b.BuiltInRole
	}

	_, err := buildSetResourcePermissionCommands(current, desired, areEqual)
	require.Error(t, err)
}
