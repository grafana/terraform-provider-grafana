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

func TestAccGenericResource_orgIDFallbackWhenBootdataHasNoStack(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	var bootdataCalls atomic.Int32
	var orgCreateCalls atomic.Int32
	var orgDeleteCalls atomic.Int32

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bootdata":
				bootdataCalls.Add(1)
				// Bootdata returns no stack — simulates a local/OSS instance.
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"settings":{"namespace":"default"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write bootdata response: %v", err)
				}
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
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	t.Setenv("GRAFANA_ORG_ID", "7")

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccGenericOrgPrecedenceFolderConfig(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", "generic-precedence-folder"),
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

	if bootdataCalls.Load() == 0 {
		t.Fatal("expected bootdata to be called for namespace discovery")
	}
	if orgCreateCalls.Load() == 0 {
		t.Fatal("expected org namespace create call as fallback")
	}
	if orgDeleteCalls.Load() == 0 {
		t.Fatal("expected org namespace delete call")
	}
}

func TestAccGenericResource_bootdataCloudWinsOverOrgID(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	var bootdataCalls atomic.Int32
	var stackCreateCalls atomic.Int32
	var orgCreateCalls atomic.Int32
	var stackDeleteCalls atomic.Int32

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bootdata":
				bootdataCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"settings":{"namespace":"stacks-42"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write bootdata response: %v", err)
				}
			case "/apis/folder.grafana.app/v1":
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"resources":[{"name":"folders","kind":"Folder","namespaced":true}]}`))
				if err != nil {
					handlerErrors.recordf("failed to write discovery response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/stacks-42/folders":
				stackCreateCalls.Add(1)
				w.Header().Set("Content-Type", "application/json")
				_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-precedence-folder","uid":"uuid-cloud","resourceVersion":"1"},"spec":{"title":"Generic Precedence Folder"}}`))
				if err != nil {
					handlerErrors.recordf("failed to write create response: %v", err)
				}
			case "/apis/folder.grafana.app/v1/namespaces/stacks-42/folders/generic-precedence-folder":
				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					_, err := w.Write([]byte(`{"apiVersion":"folder.grafana.app/v1","kind":"Folder","metadata":{"name":"generic-precedence-folder","uid":"uuid-cloud","resourceVersion":"1"},"spec":{"title":"Generic Precedence Folder"}}`))
					if err != nil {
						handlerErrors.recordf("failed to write get response: %v", err)
					}
				case http.MethodDelete:
					stackDeleteCalls.Add(1)
					w.WriteHeader(http.StatusOK)
				default:
					handlerErrors.recordf("unexpected method %q", req.Method)
					http.Error(w, "unexpected method", http.StatusInternalServerError)
				}
			case "/apis/folder.grafana.app/v1/namespaces/org-7/folders":
				orgCreateCalls.Add(1)
				http.Error(w, "org namespace should not be used when bootdata returns a cloud stack", http.StatusInternalServerError)
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	// org_id is set, but bootdata returns a cloud stack — bootdata wins.
	t.Setenv("GRAFANA_ORG_ID", "7")

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccGenericOrgPrecedenceFolderConfig(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", "generic-precedence-folder"),
				),
			},
		},
	})

	if bootdataCalls.Load() == 0 {
		t.Fatal("expected bootdata to be called")
	}
	if stackCreateCalls.Load() == 0 {
		t.Fatal("expected cloud stack namespace to be used from bootdata")
	}
	if orgCreateCalls.Load() != 0 {
		t.Fatalf("expected org namespace NOT to be used when bootdata returns a cloud stack, got %d org create calls", orgCreateCalls.Load())
	}
	if stackDeleteCalls.Load() == 0 {
		t.Fatal("expected cloud stack namespace delete call")
	}
}

func TestAccGenericResource_secureCreateSendsRawValue(t *testing.T) {
	checkGenericLocalAcceptanceEnabled(t)

	secretValue := "test-secret-value-not-a-real-credential" //nolint:gosec
	var capturedSecureCreate string

	setupGenericLocalProvider(t, func(handlerErrors *handlerFailureRecorder) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			switch req.URL.Path {
			case "/bootdata":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"settings":{"namespace":"default"}}`))
			case "/apis/provisioning.grafana.app/v1beta1":
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"resources":[{"name":"repositories","kind":"Repository","namespaced":true}]}`))
			case "/apis/provisioning.grafana.app/v1beta1/namespaces/org-5/repositories":
				var payload map[string]any
				if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
					handlerErrors.recordf("failed to decode create request: %v", err)
					http.Error(w, "bad request", http.StatusBadRequest)
					return
				}

				// Capture the secure.token.create value from the wire payload.
				if secure, ok := payload["secure"].(map[string]any); ok {
					if token, ok := secure["token"].(map[string]any); ok {
						if create, ok := token["create"].(string); ok {
							capturedSecureCreate = create
						}
					}
				}

				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`{"apiVersion":"provisioning.grafana.app/v1beta1","kind":"Repository","metadata":{"name":"generic-secure-test","uid":"uuid-secure","resourceVersion":"1"},"spec":{"title":"Secure Test","type":"github"},"secure":{"token":{"name":"inline-stored"}}}`))
			case "/apis/provisioning.grafana.app/v1beta1/namespaces/org-5/repositories/generic-secure-test":
				switch req.Method {
				case http.MethodGet:
					w.Header().Set("Content-Type", "application/json")
					_, _ = w.Write([]byte(`{"apiVersion":"provisioning.grafana.app/v1beta1","kind":"Repository","metadata":{"name":"generic-secure-test","uid":"uuid-secure","resourceVersion":"1"},"spec":{"title":"Secure Test","type":"github"},"secure":{"token":{"name":"inline-stored"}}}`))
				case http.MethodDelete:
					w.WriteHeader(http.StatusOK)
				default:
					handlerErrors.recordf("unexpected method %q", req.Method)
					http.Error(w, "unexpected method", http.StatusInternalServerError)
				}
			default:
				handlerErrors.recordf("unexpected request path %q", req.URL.Path)
				http.Error(w, "unexpected request path", http.StatusInternalServerError)
			}
		})
	})

	t.Setenv("GRAFANA_ORG_ID", "5")

	config := fmt.Sprintf(`
provider "grafana" {}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "provisioning.grafana.app/v1beta1"
    kind       = "Repository"
    metadata = {
      name = "generic-secure-test"
    }
    spec = {
      title = "Secure Test"
      type  = "github"
    }
  }

  secure = {
    token = {
      create = %q
    }
  }

  secure_version = 1
}
`, secretValue)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
				),
			},
		},
	})

	if capturedSecureCreate != secretValue {
		t.Fatalf("expected secure.token.create on the wire to be %q, got %q", secretValue, capturedSecureCreate)
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
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", "generic-autodiscovery-folder"),
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
					if states[0].Attributes["manifest.metadata.name"] != "generic-autodiscovery-folder" {
						return fmt.Errorf("expected imported manifest.metadata.name to be generic-autodiscovery-folder, got %q", states[0].Attributes["manifest.metadata.name"])
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

func testAccGenericOrgPrecedenceFolderConfig() string {
	return `
provider "grafana" {
  # URL, auth, org_id, and stack_id are provided by the test environment.
}

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-precedence-folder"
    }
    spec = {
      title = "Generic Precedence Folder"
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
