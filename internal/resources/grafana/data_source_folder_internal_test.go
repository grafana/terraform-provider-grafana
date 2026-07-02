package grafana

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/go-openapi/strfmt"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
)

// testGrafanaClient builds a Grafana API client pointed at the given test server.
func testGrafanaClient(t *testing.T, serverURL string) *goapi.GrafanaHTTPAPI {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatalf("parse server url: %v", err)
	}
	return goapi.NewHTTPClientWithConfig(strfmt.Default, &goapi.TransportConfig{
		Host:     parsed.Host,
		BasePath: "/api",
		Schemes:  []string{parsed.Scheme},
	})
}

// When only the UID is known, the folder is resolved with a direct
// GET /api/folders/:uid lookup and /api/search is never called. This is what
// makes the data source reliable for stacks with more folders than a single
// /api/search page returns.
func TestFindFolderWithTitleAndUID_UIDOnly_UsesDirectLookup(t *testing.T) {
	var searchCalled bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/search":
			searchCalled = true
			http.Error(w, "search should not be called", http.StatusInternalServerError)
		case r.URL.Path == "/api/folders/existing-folder":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"uid":"existing-folder","title":"Existing"}`)
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer server.Close()

	uid, err := findFolderWithTitleAndUID(testGrafanaClient(t, server.URL), "", "existing-folder")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "existing-folder" {
		t.Fatalf("got uid %q, want %q", uid, "existing-folder")
	}
	if searchCalled {
		t.Fatal("/api/search was called; UID-only lookup must not use search")
	}
}

// A missing UID still surfaces the FolderWithUIDNotFound error via the direct
// lookup, matching the behaviour callers and acceptance tests rely on.
func TestFindFolderWithTitleAndUID_UIDOnly_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"folder not found"}`)
	}))
	defer server.Close()

	_, err := findFolderWithTitleAndUID(testGrafanaClient(t, server.URL), "", "missing-folder")
	want := fmt.Sprintf(FolderWithUIDNotFound, "missing-folder")
	if err == nil || err.Error() != want {
		t.Fatalf("got error %v, want %q", err, want)
	}
}

// When searching by title the request is sorted so that the paginated listing
// is ordered consistently across page requests.
func TestFindFolderWithTitleAndUID_ByTitle_SendsSort(t *testing.T) {
	var gotSort string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/search" {
			http.Error(w, "unexpected path "+r.URL.Path, http.StatusInternalServerError)
			return
		}
		gotSort = r.URL.Query().Get("sort")
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `[{"uid":"folder-uid","title":"My Folder"}]`)
	}))
	defer server.Close()

	uid, err := findFolderWithTitleAndUID(testGrafanaClient(t, server.URL), "My Folder", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if uid != "folder-uid" {
		t.Fatalf("got uid %q, want %q", uid, "folder-uid")
	}
	if gotSort != "alpha-asc" {
		t.Fatalf("got sort %q, want %q", gotSort, "alpha-asc")
	}
}
