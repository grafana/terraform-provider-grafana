package grafana

import (
	"testing"

	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestBuiltInRoleFromRoleName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		roleName string
		want     string
	}{
		{roleName: "managed:builtins:viewer:permissions", want: "Viewer"},
		{roleName: "managed:builtins:editor:permissions", want: "Editor"},
		{roleName: "managed:builtins:admin:permissions", want: "Admin"},
		{roleName: "managed:builtins:Viewer:permissions", want: "Viewer"},
		{roleName: "managed:users:10:permissions", want: ""},
		{roleName: "Editor", want: ""},
		{roleName: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.roleName, func(t *testing.T) {
			t.Parallel()
			if got := builtInRoleFromRoleName(tt.roleName); got != tt.want {
				t.Fatalf("builtInRoleFromRoleName(%q) = %q, want %q", tt.roleName, got, tt.want)
			}
		})
	}
}

func TestParsePermissionIdentityID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name                      string
		id                        string
		allowServiceAccountPrefix bool
		want                      int64
		wantErr                   bool
	}{
		{name: "empty", id: "", want: 0},
		{name: "zero", id: "0", want: 0},
		{name: "numeric", id: "12", want: 12},
		{name: "org scoped", id: "1:12", want: 12},
		{name: "cloud service account", id: "mystack:12", allowServiceAccountPrefix: true, want: 12},
		{name: "cloud prefix rejected for team", id: "mystack:12", allowServiceAccountPrefix: false, wantErr: true},
		{name: "non numeric", id: "abc", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := parsePermissionIdentityID(tt.id, tt.allowServiceAccountPrefix)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parsePermissionIdentityID(%q) error = nil, want error", tt.id)
				}
				return
			}
			if err != nil {
				t.Fatalf("parsePermissionIdentityID(%q) unexpected error: %v", tt.id, err)
			}
			if got != tt.want {
				t.Fatalf("parsePermissionIdentityID(%q) = %d, want %d", tt.id, got, tt.want)
			}
		})
	}
}

func TestPermissionCommandFromBulkItemMixedUserAndRole(t *testing.T) {
	t.Parallel()

	userCmd, err := permissionCommandFromBulkItem(bulkPermissionItemModel{
		UserID:     types.StringValue("10"),
		Role:       types.StringNull(),
		TeamID:     types.StringNull(),
		Permission: types.StringValue("Admin"),
	})
	if err != nil {
		t.Fatalf("user command: %v", err)
	}
	if userCmd.UserID != 10 || userCmd.BuiltInRole != "" || userCmd.TeamID != 0 {
		t.Fatalf("user command = %+v, want user 10 without role", userCmd)
	}

	roleCmd, err := permissionCommandFromBulkItem(bulkPermissionItemModel{
		Role:       types.StringValue("Editor"),
		UserID:     types.StringNull(),
		TeamID:     types.StringNull(),
		Permission: types.StringValue("View"),
	})
	if err != nil {
		t.Fatalf("role command: %v", err)
	}
	if roleCmd.BuiltInRole != "Editor" || roleCmd.UserID != 0 || roleCmd.TeamID != 0 {
		t.Fatalf("role command = %+v, want Editor without identity", roleCmd)
	}

	if _, err := permissionCommandFromBulkItem(bulkPermissionItemModel{
		Permission: types.StringValue("Admin"),
		Role:       types.StringNull(),
		UserID:     types.StringNull(),
		TeamID:     types.StringNull(),
	}); err == nil {
		t.Fatal("expected error for permission item with no assignment target")
	}
}

func TestBuildResourcePermissionCommandsMixedUserAndRole(t *testing.T) {
	t.Parallel()

	current := []*models.ResourcePermissionDTO{
		{
			IsManaged:   true,
			Permission:  "View",
			RoleName:    "managed:builtins:viewer:permissions",
			BuiltInRole: "",
		},
		{
			IsManaged:  true,
			Permission: "Admin",
			UserID:     1,
		},
	}
	desired := []*models.SetResourcePermissionCommand{
		{UserID: 10, Permission: "Admin"},
		{BuiltInRole: "Editor", Permission: "View"},
	}

	got := buildResourcePermissionCommands(current, desired)
	for _, cmd := range got {
		if !hasPermissionAssignment(cmd) {
			t.Fatalf("built command has no assignment target: %+v", cmd)
		}
	}

	wantDeletes := map[string]bool{"Viewer": false, "user:1": false}
	wantAdds := map[string]bool{"user:10": false, "Editor": false}
	for _, cmd := range got {
		switch {
		case cmd.Permission == "" && cmd.BuiltInRole == "Viewer":
			wantDeletes["Viewer"] = true
		case cmd.Permission == "" && cmd.UserID == 1:
			wantDeletes["user:1"] = true
		case cmd.Permission == "Admin" && cmd.UserID == 10:
			wantAdds["user:10"] = true
		case cmd.Permission == "View" && cmd.BuiltInRole == "Editor":
			wantAdds["Editor"] = true
		default:
			t.Fatalf("unexpected command: %+v", cmd)
		}
	}
	for name, found := range wantDeletes {
		if !found {
			t.Fatalf("missing delete for %s in %+v", name, got)
		}
	}
	for name, found := range wantAdds {
		if !found {
			t.Fatalf("missing add for %s in %+v", name, got)
		}
	}
}
