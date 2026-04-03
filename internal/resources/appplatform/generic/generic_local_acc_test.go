package generic_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// These tests intentionally run the Terraform acceptance harness against an httptest server.
// They cover provider wiring behavior end-to-end without depending on a live Grafana instance.

func checkGenericLocalAcceptanceEnabled(t *testing.T) {
	t.Helper()

	if !testutils.AccTestsEnabled("TF_ACC") {
		t.Skip("TF_ACC must be set to run Terraform acceptance harness checks")
	}
}

func setupGenericLocalProvider(t *testing.T, handlerFactory func(*handlerFailureRecorder) http.Handler) {
	t.Helper()

	t.Setenv("GRAFANA_URL", "")
	t.Setenv("GRAFANA_AUTH", "")
	t.Setenv("GRAFANA_ORG_ID", "")
	t.Setenv("GRAFANA_STACK_ID", "")

	handlerErrors := &handlerFailureRecorder{}
	t.Cleanup(func() {
		handlerErrors.assertClear(t)
	})

	server := httptest.NewServer(handlerFactory(handlerErrors))
	t.Cleanup(server.Close)

	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "test-token")
}

func TestAccGenericResource_orgIDWinsOverStackID(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	var bootdataCalls atomic.Int32
	var orgCreateCalls atomic.Int32
	var stackCreateCalls atomic.Int32
	var orgDeleteCalls atomic.Int32

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bootdata":
				bootdataCalls.Add(1)
				http.Error(w, "bootdata should not be used when org_id is configured", http.StatusInternalServerError)
			case "/apis/folder.grafana.app/v1":
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"resources":[{"name":"folders","kind":"Folder","namespaced":true}]}`))
				if err != nil {
					handlerErrors.recordf("failed to write discovery response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/org-7/folders":
				orgCreateCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-precedence-folder","uid":"uuid-precedence","resourceVersion":"1"},"spec":{"title":"Generic Precedence Folder"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write create response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/org-7/folders/generic-precedence-folder":
				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-precedence-folder","uid":"uuid-precedence","resourceVersion":"1"},"spec":{"title":"Generic Precedence Folder"}}`))
					if err != nil {
						handlerErrors.recordf("failed to write get response: %v", err)
					}
				case http.MethodDelete:
					orgDeleteCalls.Add(1)
					w.WriteHeader(http.StatusOK)
				default:
					handlerErrors.recordf("unexpected org namespace method %q", req.Method)
					http.Error(w, "unexpected method", http.StatusInternalServerError)
				}
			case "/apis/folder.grafana.app/v1/namespaces/stacks-99/folders":
				stackCreateCalls.Add(1)
				http.Error(w, "stack namespace should not be used when org_id is configured", http.StatusInternalServerError)
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	t.Setenv("GRAFANA_ORG_ID", "7")
	t.Setenv("GRAFANA_STACK_ID", "99")

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccGenericOrgPrecedenceFolderConfig(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-precedence-folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Folder"),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
			},
		},
	})

	if bootdataCalls.Load() != 0 {
		t.Fatalf("expected org_id precedence to avoid bootdata, got %d bootdata calls", bootdataCalls.Load())
	}
	if stackCreateCalls.Load() != 0 {
		t.Fatalf("expected org_id precedence to avoid stack namespace, got %d stack create calls", stackCreateCalls.Load())
	}
	if orgCreateCalls.Load() == 0 {
		t.Fatal("expected org namespace create call")
	}
	if orgDeleteCalls.Load() == 0 {
		t.Fatal("expected org namespace delete call")
	}
}

func TestAccGenericResource_autodiscoveryDoesNotCacheAcrossOperations(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	var bootdataCalls atomic.Int32
	var createCalls atomic.Int32
	var getCalls atomic.Int32
	var deleteCalls atomic.Int32

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bootdata":
				bootdataCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"settings":{"namespace":"stacks-21"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write bootdata response: %v", err)
				}
			case "/apis/folder.grafana.app/v1":
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"resources":[{"name":"folders","kind":"Folder","namespaced":true}]}`))
				if err != nil {
					handlerErrors.recordf("failed to write discovery response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/stacks-21/folders":
				createCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-autodiscovery-folder","uid":"uuid-autodiscovery","resourceVersion":"1"},"spec":{"title":"Generic Autodiscovery Folder"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write create response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/stacks-21/folders/generic-autodiscovery-folder":
				switch req.Method {
				case http.MethodGet:
					getCalls.Add(1)
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-autodiscovery-folder","uid":"uuid-autodiscovery","resourceVersion":"1"},"spec":{"title":"Generic Autodiscovery Folder"}}`))
					if err != nil {
						handlerErrors.recordf("failed to write get response: %v", err)
					}
				case http.MethodDelete:
					deleteCalls.Add(1)
					w.WriteHeader(http.StatusOK)
				default:
					handlerErrors.recordf("unexpected autodiscovery method %q", req.Method)
					http.Error(w, "unexpected method", http.StatusInternalServerError)
				}
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccGenericAutodiscoveryFolderConfig(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-autodiscovery-folder"),
				),
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					if states[0].Attributes["metadata.uid"] != "generic-autodiscovery-folder" {
						return fmt.Errorf("expected imported metadata.uid to be generic-autodiscovery-folder, got %q", states[0].Attributes["metadata.uid"])
					}
					return nil
				},
			},
		},
	})

	if bootdataCalls.Load() < 2 {
		t.Fatalf("expected autodiscovery to call /bootdata more than once, got %d", bootdataCalls.Load())
	}
	if createCalls.Load() == 0 {
		t.Fatal("expected autodiscovery test to create the resource")
	}
	if getCalls.Load() == 0 {
		t.Fatal("expected autodiscovery test to read the resource")
	}
	if deleteCalls.Load() == 0 {
		t.Fatal("expected autodiscovery test to delete the resource")
	}
}

func TestAccGenericResource_hybridOverridesManifest(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	var createCalls atomic.Int32
	var deleteCalls atomic.Int32

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/apis/folder.grafana.app/v1":
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"resources":[{"name":"folders","kind":"Folder","namespaced":true}]}`))
				if err != nil {
					handlerErrors.recordf("failed to write discovery response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/org-7/folders":
				createCalls.Add(1)

				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					handlerErrors.recordf("failed to decode create request: %v", err)
					http.Error(w, "invalid request body", http.StatusBadRequest)
					return
				}

				if payload["apiVersion"] != "folder.grafana.app/v1" {
					handlerErrors.recordf("expected overridden apiVersion, got %#v", payload["apiVersion"])
				}
				if payload["kind"] != "Folder" {
					handlerErrors.recordf("expected overridden kind, got %#v", payload["kind"])
				}

				metadata, ok := payload["metadata"].(map[string]any)
				if !ok {
					handlerErrors.recordf("expected metadata map in create request, got %#v", payload["metadata"])
				} else if metadata["name"] != "generic-hybrid-folder" {
					handlerErrors.recordf("expected manifest metadata.name, got %#v", metadata["name"])
				}

				spec, ok := payload["spec"].(map[string]any)
				if !ok {
					handlerErrors.recordf("expected spec map in create request, got %#v", payload["spec"])
				} else if spec["title"] != "Hybrid Override Folder" {
					handlerErrors.recordf("expected overridden spec.title, got %#v", spec["title"])
				}

				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-hybrid-folder","uid":"uuid-hybrid","resourceVersion":"1"},"spec":{"title":"Hybrid Override Folder"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write create response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/org-7/folders/generic-hybrid-folder":
				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-hybrid-folder","uid":"uuid-hybrid","resourceVersion":"1"},"spec":{"title":"Hybrid Override Folder"}}`))
					if err != nil {
						handlerErrors.recordf("failed to write get response: %v", err)
					}
				case http.MethodDelete:
					deleteCalls.Add(1)
					w.WriteHeader(http.StatusOK)
				default:
					handlerErrors.recordf("unexpected hybrid override method %q", req.Method)
					http.Error(w, "unexpected method", http.StatusInternalServerError)
				}
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	t.Setenv("GRAFANA_ORG_ID", "7")

	config := testAccGenericHybridOverrideConfig()
	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-hybrid-folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "spec.title", "Hybrid Override Folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "api_group", "folder.grafana.app"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "version", "v1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "kind", "Folder"),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})

	if createCalls.Load() == 0 {
		t.Fatal("expected hybrid override test to create the resource")
	}
	if deleteCalls.Load() == 0 {
		t.Fatal("expected hybrid override test to delete the resource")
	}
}

func testAccGenericOrgPrecedenceFolderConfig() string {
	return `
provider "grafana" {
  # URL, auth, org_id, and stack_id are provided by the test environment.
}

resource "grafana_apps_generic_resource" "test" {
  metadata = {
    uid = "generic-precedence-folder"
  }

  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    spec = {
      title = "Generic Precedence Folder"
    }
  }
}
`
}

func testAccGenericHybridOverrideConfig() string {
	return `
provider "grafana" {
  # URL, auth, and org_id are provided by the test environment.
}

resource "grafana_apps_generic_resource" "test" {
  api_group = "folder.grafana.app"
  version   = "v1"
  kind      = "Folder"

  spec = {
    title = "Hybrid Override Folder"
  }

  manifest = {
    apiVersion = "example.invalid/v9"
    kind       = "IgnoredFolder"
    metadata = {
      name = "generic-hybrid-folder"
    }
    spec = {
      title = "Ignored Manifest Title"
    }
  }
}
`
}

func testAccGenericAutodiscoveryFolderConfig() string {
	return `
provider "grafana" {
  # URL and auth are provided by the test environment.
}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-autodiscovery-folder"
    }
    spec = {
      title = "Generic Autodiscovery Folder"
    }
  }
}
`
}

type handlerFailureRecorder struct {
	mu  sync.Mutex
	err error
}

func (r *handlerFailureRecorder) recordf(format string, args ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err == nil {
		r.err = fmt.Errorf(format, args...)
	}
}

func (r *handlerFailureRecorder) assertClear(t *testing.T) {
	t.Helper()

	r.mu.Lock()
	defer r.mu.Unlock()

	if r.err != nil {
		t.Fatal(r.err)
	}
}
