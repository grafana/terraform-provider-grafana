package grafana_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestUnitFolderPermission_MixedUserAndRole(t *testing.T) {
	type permissionPayload struct {
		BuiltInRole string `json:"builtInRole"`
		Permission  string `json:"permission"`
		TeamID      int64  `json:"teamId"`
		UserID      int64  `json:"userId"`
	}
	type setPermissionsBody struct {
		Permissions []permissionPayload `json:"permissions"`
	}

	var applied []permissionPayload

	mux := http.NewServeMux()
	mux.HandleFunc("/api/folders/mixed-folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": 1, "uid": "mixed-folder", "title": "mixed",
		})
	})
	mux.HandleFunc("/api/access-control/folders/mixed-folder", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var payload setPermissionsBody
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, perm := range payload.Permissions {
				if perm.UserID == 0 && perm.TeamID == 0 && perm.BuiltInRole == "" {
					w.WriteHeader(http.StatusBadRequest)
					json.NewEncoder(w).Encode(map[string]any{
						"message": "built-in role [] is not valid",
					})
					return
				}
			}
			applied = append([]permissionPayload{}, payload.Permissions...)
			json.NewEncoder(w).Encode(map[string]any{"message": "Permissions updated"})
			return
		}

		var current []map[string]any
		hasExplicit := false
		for _, perm := range applied {
			if perm.Permission == "" {
				continue
			}
			hasExplicit = true
			item := map[string]any{
				"isManaged":  true,
				"permission": perm.Permission,
			}
			if perm.BuiltInRole != "" {
				item["builtInRole"] = perm.BuiltInRole
			}
			if perm.UserID > 0 {
				item["userId"] = perm.UserID
			}
			if perm.TeamID > 0 {
				item["teamId"] = perm.TeamID
			}
			current = append(current, item)
		}
		if !hasExplicit {
			current = []map[string]any{
				{
					"isManaged":  true,
					"permission": "View",
					"roleName":   "managed:builtins:viewer:permissions",
				},
				{
					"isManaged":  true,
					"permission": "Admin",
					"userId":     1,
				},
			}
		}
		json.NewEncoder(w).Encode(current)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")

	config := `
resource "grafana_folder_permission" "test" {
  folder_uid = "mixed-folder"
  permissions {
    user_id    = "10"
    permission = "Admin"
  }
  permissions {
    role       = "Editor"
    permission = "View"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_folder_permission.test", "folder_uid", "mixed-folder"),
					resource.TestCheckResourceAttr("grafana_folder_permission.test", "permissions.#", "2"),
				),
			},
		},
	})
}

func TestUnitDashboardPermission_MixedUserAndRole(t *testing.T) {
	type permissionPayload struct {
		BuiltInRole string `json:"builtInRole"`
		Permission  string `json:"permission"`
		TeamID      int64  `json:"teamId"`
		UserID      int64  `json:"userId"`
	}
	type setPermissionsBody struct {
		Permissions []permissionPayload `json:"permissions"`
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/dashboards/uid/mixed-dash", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"dashboard": map[string]any{"uid": "mixed-dash", "title": "mixed"},
			"meta":      map[string]any{},
		})
	})
	mux.HandleFunc("/api/access-control/dashboards/mixed-dash", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			var payload setPermissionsBody
			if err := json.Unmarshal(body, &payload); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			for _, perm := range payload.Permissions {
				if perm.UserID == 0 && perm.TeamID == 0 && perm.BuiltInRole == "" {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprint(w, `{"message":"built-in role [] is not valid"}`)
					return
				}
			}
			json.NewEncoder(w).Encode(map[string]any{"message": "Permissions updated"})
			return
		}

		json.NewEncoder(w).Encode([]map[string]any{
			{"isManaged": true, "permission": "Admin", "userId": 10},
			{"isManaged": true, "permission": "View", "builtInRole": "Editor"},
		})
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")

	config := `
resource "grafana_dashboard_permission" "test" {
  dashboard_uid = "mixed-dash"
  permissions {
    user_id    = "10"
    permission = "Admin"
  }
  permissions {
    role       = "Editor"
    permission = "View"
  }
}
`

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("grafana_dashboard_permission.test", "dashboard_uid", "mixed-dash"),
					resource.TestCheckResourceAttr("grafana_dashboard_permission.test", "permissions.#", "2"),
				),
			},
		},
	})
}
