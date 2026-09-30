package incident_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"

	incident "github.com/grafana/incident-go"
)

// incidentAPIBasePath is where the provider expects the Incident API to live
// under the Grafana URL, mirroring createIncidentClient in
// pkg/provider/configure_clients.go.
const incidentAPIBasePath = "/api/plugins/grafana-irm-app/resources/api/v1/"

// The active-role bounds the real roles service enforces:
//
//   - CreateRole and UnarchiveRole refuse when the organization already has
//     maxActiveRoles active roles.
//   - DeleteRole and ArchiveRole refuse when it has minActiveRoles or fewer.
//   - UpdateRole checks neither, even though it writes the archived column.
//     That is why the resource routes archived transitions through
//     ArchiveRole/UnarchiveRole instead.
//
// A fresh organization is seeded with three roles (commander, investigator,
// observer), so it has two free slots and is one archive away from the floor.
const (
	maxActiveRoles = 5
	minActiveRoles = 2
)

// The details and hints the real service returns when a bound is hit. Tests
// match on these so a change in the API's wording shows up as a test failure
// rather than as a silently weaker assertion.
//
// Both ceilings (create and unarchive) share one error, as do both floors
// (archive and delete): the API distinguishes them by code, not by message.
var (
	errActiveCeiling = fmt.Sprintf("A maximum of %d active roles is allowed", maxActiveRoles)
	errActiveFloor   = fmt.Sprintf("At least %d active roles are required", minActiveRoles)
)

// The problem details the fake replies with, keyed the way the API keys them.
const (
	codeRoleNotFound     = "roles.not_found"
	codeActiveCeiling    = "roles.maximum_active_reached"
	codeActiveFloor      = "roles.minimum_active_required"
	codeValidationFailed = "common.validation_failed"
	codeInternalError    = "common.internal_error"

	hintRoleNotFound  = "Check the role ID. Use GetRoles to list available roles."
	hintActiveCeiling = "Archive or delete an active role before creating or unarchiving another role."
	hintActiveFloor   = "Create or unarchive another role before deleting or archiving this role."
)

// fakeRolesAPI is an in-memory stand-in for the Incident roles API that
// enforces the same active-role bounds as the real service. It exists because
// both bounds depend on how many roles an organization already has, which a
// live stack cannot be put into without archiving roles the test does not own.
type fakeRolesAPI struct {
	mu     sync.Mutex
	roles  []incident.Role
	nextID int
	calls  []string

	// deleteNotFound makes DeleteRole answer 404 while GetRoles keeps listing
	// the role. See setDeleteNotFound.
	deleteNotFound bool
}

func newFakeRolesAPI(seed ...incident.Role) *fakeRolesAPI {
	f := &fakeRolesAPI{nextID: 1}
	for _, role := range seed {
		f.add(role)
	}
	return f
}

// activeRoles builds a set of active roles to seed an organization with.
func activeRoles(names ...string) []incident.Role {
	roles := make([]incident.Role, 0, len(names))
	for _, name := range names {
		roles = append(roles, incident.Role{Name: name})
	}
	return roles
}

// start serves the fake API and points the provider at it for the duration of
// the test.
func (f *fakeRolesAPI) start(t *testing.T) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(incidentAPIBasePath, f.handle)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")
}

func (f *fakeRolesAPI) handle(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimPrefix(r.URL.Path, incidentAPIBasePath)

	var req struct {
		Role   incident.Role `json:"role"`
		RoleID int           `json:"roleID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, codeValidationFailed, "Validation Failed", fmt.Sprintf("%s: decode request: %s", method, err), "")
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, method)

	switch method {
	case "RolesService.GetRoles":
		writeJSON(w, map[string]any{"roles": f.roles})

	case "RolesService.CreateRole":
		if f.activeCountLocked() >= maxActiveRoles {
			writeActiveCeiling(w)
			return
		}
		writeJSON(w, map[string]any{"role": f.addLocked(req.Role)})

	// UpdateRole replaces the whole role, including the archived column, and
	// deliberately checks no bounds. A provider that used it to archive would
	// pass this handler and fail the archive tests' state assertions.
	case "RolesService.UpdateRole":
		i := f.indexOfLocked(req.Role.RoleID)
		if i < 0 {
			writeRoleNotFound(w)
			return
		}
		req.Role.OrgID = f.roles[i].OrgID
		f.roles[i] = req.Role
		writeJSON(w, map[string]any{"role": f.roles[i]})

	case "RolesService.DeleteRole":
		// The real service looks the role up before it checks the floor, so a
		// role that is already gone answers 404 rather than the bound error.
		if f.deleteNotFound {
			writeRoleNotFound(w)
			return
		}
		if f.activeCountLocked() <= minActiveRoles {
			writeActiveFloor(w)
			return
		}
		i := f.indexOfLocked(req.RoleID)
		if i < 0 {
			writeRoleNotFound(w)
			return
		}
		f.roles = slices.Delete(f.roles, i, i+1)
		writeJSON(w, map[string]any{})

	case "RolesService.ArchiveRole":
		if f.activeCountLocked() <= minActiveRoles {
			writeActiveFloor(w)
			return
		}
		if !f.setArchivedLocked(req.RoleID, true) {
			writeRoleNotFound(w)
			return
		}
		writeJSON(w, map[string]any{})

	case "RolesService.UnarchiveRole":
		if f.activeCountLocked() >= maxActiveRoles {
			writeActiveCeiling(w)
			return
		}
		if !f.setArchivedLocked(req.RoleID, false) {
			writeRoleNotFound(w)
			return
		}
		writeJSON(w, map[string]any{})

	default:
		writeProblem(w, http.StatusInternalServerError, codeInternalError, "Internal Server Error", fmt.Sprintf("%s: not implemented by the fake roles API", method), "")
	}
}

// add inserts a role without any bound check, the way a role that predates the
// test would exist. Tests also use it mid-run to change the organization out of
// band.
func (f *fakeRolesAPI) add(role incident.Role) incident.Role {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.addLocked(role)
}

// setDeleteNotFound makes DeleteRole answer 404 while GetRoles keeps listing
// the role. That is how a role removed outside Terraform between the refresh
// and the apply presents itself, and it is the only way to reach the delete
// path for a missing role: if the refresh saw it gone, Terraform would drop it
// from state and never call Delete at all.
func (f *fakeRolesAPI) setDeleteNotFound(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteNotFound = v
}

// snapshot returns a copy of the organization's roles.
func (f *fakeRolesAPI) snapshot() []incident.Role {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.roles)
}

// role looks a role up by name, or nil when no such role exists.
func (f *fakeRolesAPI) role(name string) *incident.Role {
	for _, role := range f.snapshot() {
		if role.Name == name {
			return &role
		}
	}
	return nil
}

// activeCount is the number of non-archived roles, which is what both bounds
// are measured against.
func (f *fakeRolesAPI) activeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.activeCountLocked()
}

// called reports whether the fake ever served the given RPC, for example
// "RolesService.ArchiveRole".
func (f *fakeRolesAPI) called(method string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Contains(f.calls, method)
}

// firstCall is the index of the first call to the given RPC, or -1. Tests use
// it to pin the order of the calls an update makes.
func (f *fakeRolesAPI) firstCall(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Index(f.calls, method)
}

func (f *fakeRolesAPI) addLocked(role incident.Role) incident.Role {
	role.RoleID = f.nextID
	role.OrgID = "1"
	f.nextID++
	f.roles = append(f.roles, role)
	return role
}

func (f *fakeRolesAPI) activeCountLocked() int {
	count := 0
	for _, role := range f.roles {
		if !role.Archived {
			count++
		}
	}
	return count
}

func (f *fakeRolesAPI) indexOfLocked(roleID int) int {
	return slices.IndexFunc(f.roles, func(role incident.Role) bool { return role.RoleID == roleID })
}

// setArchivedLocked reports whether the role existed and was updated.
func (f *fakeRolesAPI) setArchivedLocked(roleID int, archived bool) bool {
	i := f.indexOfLocked(roleID)
	if i < 0 {
		return false
	}
	f.roles[i].Archived = archived
	return true
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}

func writeRoleNotFound(w http.ResponseWriter) {
	writeProblem(w, http.StatusNotFound, codeRoleNotFound, "Role Not Found", "Role not found", hintRoleNotFound)
}

func writeActiveCeiling(w http.ResponseWriter) {
	writeProblem(w, http.StatusConflict, codeActiveCeiling, "Maximum Active Roles Reached", errActiveCeiling, hintActiveCeiling)
}

func writeActiveFloor(w http.ResponseWriter) {
	writeProblem(w, http.StatusConflict, codeActiveFloor, "Minimum Active Roles Required", errActiveFloor, hintActiveFloor)
}

// writeProblem replies the way the API's RPC layer does for a structured error:
// a problem body under the mapped status, carrying a machine-readable code and
// hint, with the legacy "error" field preserved for older consumers.
//
// The fake models a server that returns structured errors. A server that does
// not answers 500 with a bare {"error": "..."} body and no code, which the
// generated client still surfaces as an *incident.APIError, but with Code and
// Hint empty and HTTPStatusCode 500.
func writeProblem(w http.ResponseWriter, status int, code, title, detail, hint string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	body := map[string]any{
		"type":      "https://grafana.com/docs/grafana-cloud/alerting-and-irm/irm/reference/errors#" + code,
		"title":     title,
		"status":    status,
		"detail":    detail,
		"code":      code,
		"retryable": false,
		"error":     detail,
	}
	if hint != "" {
		body["hint"] = hint
	}
	_ = json.NewEncoder(w).Encode(body)
}
