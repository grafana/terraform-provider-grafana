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

// The rules the real severities service enforces, after grafana/irm#11894 and
// grafana/irm#11945:
//
//   - CreateOrgSeverity accepts levels 1 to 5. Labels and levels are unique
//     across every severity, archived ones included.
//   - CreateOrgSeverity and UnarchiveOrgSeverity refuse when the organization
//     already has maxActiveSeverities active severities, Pending included.
//     With 6 levels and unique levels, no organization can reach it, but the
//     fake enforces it as the real service does.
//   - UpdateOrgSeverity writes only the fields that changed, and never the level
//     or archivedAt.
//   - The Pending severity (level -1) cannot be archived or deleted.
//   - A severity any incident uses cannot be deleted.
//
// There is no lower bound: archiving or deleting down to Pending alone is
// allowed.
const (
	maxActiveSeverities = 6
	pendingLevel        = -1
)

// The details the real service returns. Tests match on these so a change in
// the API's wording shows up as a test failure.
var (
	errSeverityActiveLimit   = fmt.Sprintf("A maximum of %d active severities is allowed", maxActiveSeverities)
	errSeverityLabelConflict = "A severity with this display label already exists"
	errSeverityLevelConflict = "A severity with this level already exists"
	errSeverityPending       = "The Pending severity is system-managed and cannot be archived or deleted"
	errSeverityInUse         = "The severity is used by incidents and cannot be deleted"
)

const (
	codeSeverityNotFound      = "severities.not_found"
	codeSeverityActiveLimit   = "severities.maximum_active_reached"
	codeSeverityLabelConflict = "severities.label_conflict"
	codeSeverityLevelConflict = "severities.level_conflict"
	codeSeveritySystemManaged = "severities.system_managed"
	codeSeverityInUse         = "severities.in_use"
)

// fakeSeveritiesAPI is an in-memory stand-in for the Incident severities API.
// It exists because the rules above depend on what the organization already
// holds, which a live stack cannot be put into without changing severities the
// test does not own, and because the in-use rule needs incidents.
type fakeSeveritiesAPI struct {
	mu         sync.Mutex
	severities []incident.Severity
	nextID     int
	calls      []string

	// inUse maps a severity ID to the number of incidents using it.
	inUse map[string]int
	// deleteNotFound makes DeleteOrgSeverity answer 404 while the severity can
	// still be read. See fakeRolesAPI.setDeleteNotFound for why.
	deleteNotFound bool
	// refuseArchive makes ArchiveOrgSeverity fail.
	refuseArchive bool
	// refuseUnarchive makes UnarchiveOrgSeverity fail.
	refuseUnarchive bool
	// updateOmitsArchivedAt makes UpdateOrgSeverity answer without archivedAt.
	updateOmitsArchivedAt bool
}

// newFakeSeveritiesAPI seeds an organization with the Pending severity plus
// the given severities, the way every organization starts with Pending.
func newFakeSeveritiesAPI(seed ...incident.Severity) *fakeSeveritiesAPI {
	f := &fakeSeveritiesAPI{nextID: 1, inUse: map[string]int{}}
	f.add(incident.Severity{DisplayLabel: "Pending", Level: pendingLevel, IconName: "question-circle"})
	for _, severity := range seed {
		f.add(severity)
	}
	return f
}

// activeSeverities builds active severities at levels 1, 2, ... to seed an
// organization with.
func activeSeverities(labels ...string) []incident.Severity {
	severities := make([]incident.Severity, 0, len(labels))
	for i, label := range labels {
		severities = append(severities, incident.Severity{DisplayLabel: label, Level: i + 1})
	}
	return severities
}

func (f *fakeSeveritiesAPI) start(t *testing.T) {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc(incidentAPIBasePath, f.handle)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")
}

func (f *fakeSeveritiesAPI) handle(w http.ResponseWriter, r *http.Request) {
	method := strings.TrimPrefix(r.URL.Path, incidentAPIBasePath)

	var req struct {
		Severity   incident.Severity `json:"severity"`
		SeverityID string            `json:"severityID"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeProblem(w, http.StatusBadRequest, codeValidationFailed, "Validation Failed", fmt.Sprintf("%s: decode request: %s", method, err), "")
		return
	}

	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, method)

	switch method {
	case "SeveritiesService.GetOrgSeverities":
		writeJSON(w, map[string]any{"severities": f.severities})

	case "SeveritiesService.GetSeverityByID":
		i := f.indexOfLocked(req.SeverityID)
		if i < 0 {
			writeSeverityNotFound(w)
			return
		}
		writeJSON(w, map[string]any{"severity": f.severities[i]})

	case "SeveritiesService.CreateOrgSeverity":
		if req.Severity.Level < 1 || req.Severity.Level > 5 {
			writeProblem(w, http.StatusBadRequest, codeValidationFailed, "Validation Failed", "Severity level must be between 1 and 5", "")
			return
		}
		for _, severity := range f.severities {
			if severity.Level == req.Severity.Level {
				writeProblem(w, http.StatusConflict, codeSeverityLevelConflict, "Severity Level Conflict", errSeverityLevelConflict, "")
				return
			}
			if severity.DisplayLabel == req.Severity.DisplayLabel {
				writeProblem(w, http.StatusConflict, codeSeverityLabelConflict, "Severity Label Conflict", errSeverityLabelConflict, "")
				return
			}
		}
		if f.activeCountLocked() >= maxActiveSeverities {
			writeSeverityActiveLimit(w)
			return
		}
		writeJSON(w, map[string]any{"severity": f.addLocked(req.Severity)})

	case "SeveritiesService.UpdateOrgSeverity":
		i := f.indexOfLocked(req.Severity.SeverityID)
		if i < 0 {
			writeSeverityNotFound(w)
			return
		}
		for j, severity := range f.severities {
			if j != i && severity.DisplayLabel == req.Severity.DisplayLabel {
				writeProblem(w, http.StatusConflict, codeSeverityLabelConflict, "Severity Label Conflict", errSeverityLabelConflict, "")
				return
			}
		}
		// Only the editable fields are written. The level and archivedAt are
		// kept whatever the request says, as the real update does.
		current := &f.severities[i]
		current.DisplayLabel = req.Severity.DisplayLabel
		current.Description = req.Severity.Description
		current.IconName = req.Severity.IconName
		current.DarkColor = req.Severity.DarkColor
		current.LightColor = req.Severity.LightColor
		updated := *current
		if f.updateOmitsArchivedAt {
			updated.ArchivedAt = ""
		}
		writeJSON(w, map[string]any{"severity": updated})

	case "SeveritiesService.ArchiveOrgSeverity":
		i, ok := f.removableLocked(w, req.SeverityID)
		if !ok {
			return
		}
		if f.refuseArchive {
			writeProblem(w, http.StatusInternalServerError, codeInternalError, "Internal Server Error", "archive refused by the test", "")
			return
		}
		f.severities[i].ArchivedAt = "2026-01-01T00:00:00Z"
		writeJSON(w, map[string]any{})

	case "SeveritiesService.UnarchiveOrgSeverity":
		i := f.indexOfLocked(req.SeverityID)
		if i < 0 {
			writeSeverityNotFound(w)
			return
		}
		if f.refuseUnarchive {
			writeProblem(w, http.StatusInternalServerError, codeInternalError, "Internal Server Error", "unarchive refused by the test", "")
			return
		}
		if f.activeCountLocked() >= maxActiveSeverities {
			writeSeverityActiveLimit(w)
			return
		}
		f.severities[i].ArchivedAt = ""
		writeJSON(w, map[string]any{})

	case "SeveritiesService.DeleteOrgSeverity":
		if f.deleteNotFound {
			writeSeverityNotFound(w)
			return
		}
		i, ok := f.removableLocked(w, req.SeverityID)
		if !ok {
			return
		}
		if count := f.inUse[req.SeverityID]; count > 0 {
			writeProblemWithContext(w, http.StatusConflict, codeSeverityInUse, "Severity In Use", errSeverityInUse,
				"Archive the severity instead of deleting it.",
				map[string]any{"severity_id": req.SeverityID, "incidents_count": count})
			return
		}
		f.severities = slices.Delete(f.severities, i, i+1)
		writeJSON(w, map[string]any{})

	default:
		writeProblem(w, http.StatusInternalServerError, codeInternalError, "Internal Server Error", fmt.Sprintf("%s: not implemented by the fake severities API", method), "")
	}
}

// removableLocked looks up a severity to archive or delete, refusing Pending
// the way the real service does. It writes the error itself when it refuses.
func (f *fakeSeveritiesAPI) removableLocked(w http.ResponseWriter, severityID string) (int, bool) {
	i := f.indexOfLocked(severityID)
	if i < 0 {
		writeSeverityNotFound(w)
		return -1, false
	}
	if f.severities[i].Level == pendingLevel {
		writeProblem(w, http.StatusForbidden, codeSeveritySystemManaged, "System-Managed Severity", errSeverityPending, "")
		return -1, false
	}
	return i, true
}

// add inserts a severity without any check, the way one that predates the test
// would exist.
func (f *fakeSeveritiesAPI) add(severity incident.Severity) incident.Severity {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.addLocked(severity)
}

// setInUse marks the severity with the given label as used by count incidents.
func (f *fakeSeveritiesAPI) setInUse(label string, count int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, severity := range f.severities {
		if severity.DisplayLabel == label {
			f.inUse[severity.SeverityID] = count
		}
	}
}

func (f *fakeSeveritiesAPI) setDeleteNotFound(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.deleteNotFound = v
}

func (f *fakeSeveritiesAPI) setRefuseUnarchive(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refuseUnarchive = v
}

func (f *fakeSeveritiesAPI) setRefuseArchive(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refuseArchive = v
}

func (f *fakeSeveritiesAPI) setUpdateOmitsArchivedAt(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.updateOmitsArchivedAt = v
}

func (f *fakeSeveritiesAPI) snapshot() []incident.Severity {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Clone(f.severities)
}

// severity looks a severity up by label, or nil when no such severity exists.
func (f *fakeSeveritiesAPI) severity(label string) *incident.Severity {
	for _, severity := range f.snapshot() {
		if severity.DisplayLabel == label {
			return &severity
		}
	}
	return nil
}

// pending returns the Pending severity, or nil if it was deleted.
func (f *fakeSeveritiesAPI) pending() *incident.Severity {
	for _, severity := range f.snapshot() {
		if severity.Level == pendingLevel {
			return &severity
		}
	}
	return nil
}

func (f *fakeSeveritiesAPI) activeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.activeCountLocked()
}

func (f *fakeSeveritiesAPI) called(method string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Contains(f.calls, method)
}

func (f *fakeSeveritiesAPI) firstCall(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return slices.Index(f.calls, method)
}

func (f *fakeSeveritiesAPI) addLocked(severity incident.Severity) incident.Severity {
	severity.SeverityID = fmt.Sprintf("severity-%d", f.nextID)
	severity.OrgID = "1"
	f.nextID++
	f.severities = append(f.severities, severity)
	return severity
}

func (f *fakeSeveritiesAPI) activeCountLocked() int {
	count := 0
	for _, severity := range f.severities {
		if severity.ArchivedAt == "" {
			count++
		}
	}
	return count
}

func (f *fakeSeveritiesAPI) indexOfLocked(severityID string) int {
	return slices.IndexFunc(f.severities, func(severity incident.Severity) bool { return severity.SeverityID == severityID })
}

func writeSeverityNotFound(w http.ResponseWriter) {
	writeProblem(w, http.StatusNotFound, codeSeverityNotFound, "Severity Not Found", "Severity not found", "")
}

func writeSeverityActiveLimit(w http.ResponseWriter) {
	writeProblem(w, http.StatusConflict, codeSeverityActiveLimit, "Maximum Active Severities Reached", errSeverityActiveLimit, "")
}
