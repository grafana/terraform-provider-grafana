package generic_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/grafana/authlib/claims"
	folderclient "github.com/grafana/grafana-openapi-client-go/client/folders"
	"github.com/grafana/grafana-openapi-client-go/models"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccGenericResource_folderRepairsDrift(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccGenericFolderConfig(t, suffix)

	expectedTitle := "Generic Folder " + suffix
	driftedTitle := "Generic Folder Drifted " + suffix

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-folder-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.apiVersion", "folder.grafana.app/v1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "spec.title", expectedTitle),
				),
			},
			{
				Config:             config,
				Check:              testAccMutateGenericFolderTitle(genericResourceName, driftedTitle),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "spec.title", expectedTitle),
					genericEventually(genericResourceName, getGenericFolder, func(folder *models.Folder) error {
						if folder.Title != expectedTitle {
							return fmt.Errorf("expected folder title %q after drift repair, got %q", expectedTitle, folder.Title)
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_folderDetectsReplacementOutsideTerraform(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccGenericFolderConfig(t, suffix)
	replacementTitle := "Generic Folder Replacement " + suffix
	var replacedUID string

	t.Cleanup(func() {
		if replacedUID == "" || testutils.Provider == nil || testutils.Provider.Meta() == nil {
			return
		}

		client := testutils.Provider.Meta().(*common.Client)
		_, err := client.GrafanaAPI.Folders.DeleteFolder(folderclient.NewDeleteFolderParams().WithFolderUID(replacedUID))
		if err != nil && !folderNotFound(err) {
			t.Logf("cleanup failed for replaced folder %q: %v", replacedUID, err)
		}
	})

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		// The test intentionally replaces the resource outside Terraform, so
		// the standard destroy will fail with "replaced outside Terraform".
		// Use a no-op CheckDestroy; the t.Cleanup above handles the real cleanup.
		CheckDestroy: func(_ *terraform.State) error { return nil },
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-folder-"+suffix),
				),
			},
			{
				PreConfig: func() {
					client := testutils.Provider.Meta().(*common.Client)
					uid := "generic-folder-" + suffix

					_, err := client.GrafanaAPI.Folders.DeleteFolder(folderclient.NewDeleteFolderParams().WithFolderUID(uid))
					if err != nil {
						t.Fatalf("failed to delete folder %q for replacement: %v", uid, err)
					}
					replacedUID = uid

					_, err = client.GrafanaAPI.Folders.CreateFolder(&models.CreateFolderCommand{
						Title: replacementTitle,
						UID:   uid,
					})
					if err != nil {
						t.Fatalf("failed to recreate folder %q for replacement: %v", uid, err)
					}
				},
				Config:      config,
				ExpectError: regexp.MustCompile("Resource replaced outside Terraform"),
			},
			{
				// After the replacement error, delete the replacement so destroy
				// doesn't fail, then re-create from clean state.
				PreConfig: func() {
					if replacedUID == "" {
						return
					}
					client := testutils.Provider.Meta().(*common.Client)
					_, _ = client.GrafanaAPI.Folders.DeleteFolder(folderclient.NewDeleteFolderParams().WithFolderUID(replacedUID))
					replacedUID = ""
				},
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
				),
			},
		},
	})
}

func TestAccGenericResource_folderRepairsConfiguredMetadataDrift(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	config := testAccGenericFolderAnnotationLabelConfig(t, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-folder-meta-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.annotations.test.grafana.app/owner", "terraform"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.labels.test.grafana.app/env", "acceptance"),
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
					if states[0].Attributes["manifest.metadata.annotations.test.grafana.app/owner"] != "terraform" {
						return fmt.Errorf("expected imported annotation to round-trip, got %q", states[0].Attributes["manifest.metadata.annotations.test.grafana.app/owner"])
					}
					if states[0].Attributes["manifest.metadata.labels.test.grafana.app/env"] != "acceptance" {
						return fmt.Errorf("expected imported label to round-trip, got %q", states[0].Attributes["manifest.metadata.labels.test.grafana.app/env"])
					}
					return nil
				},
			},
			{
				Config:             config,
				Check:              testAccMutateGenericFolderAnnotationLabel(genericResourceName, "drifted", "ui"),
				ExpectNonEmptyPlan: true,
			},
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.annotations.test.grafana.app/owner", "terraform"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.labels.test.grafana.app/env", "acceptance"),
					genericEventually(genericResourceName, getGenericFolderAnnotationLabel, func(actual [2]string) error {
						if actual[0] != "terraform" {
							return fmt.Errorf("expected repaired annotation %q, got %q", "terraform", actual[0])
						}
						if actual[1] != "acceptance" {
							return fmt.Errorf("expected repaired label %q, got %q", "acceptance", actual[1])
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_folderHybridOverrides(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	expectedTitle := "Generic Hybrid Folder " + suffix

	config := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  api_group = "folder.grafana.app"
  version   = "v1beta1"
  kind      = "Folder"

  metadata = {
    uid = "generic-hybrid-folder-%s"
  }

  spec = {
    title = %q
  }

  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-hybrid-folder-%s"
    }
    spec = {
      title       = "Manifest Title Should Be Overridden"
      description = "Manifest Description Should Survive"
    }
  }
}
`, genericProviderConfig(t), suffix, expectedTitle, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-hybrid-folder-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "api_group", "folder.grafana.app"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "version", "v1beta1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "kind", "Folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "spec.title", expectedTitle),
					genericEventually(genericResourceName, getGenericFolder, func(folder *models.Folder) error {
						if folder.Title != expectedTitle {
							return fmt.Errorf("expected folder title %q from override, got %q", expectedTitle, folder.Title)
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					if states[0].Attributes["metadata.uid"] != "generic-hybrid-folder-"+suffix {
						return fmt.Errorf("expected imported metadata.uid to be generic-hybrid-folder-%s, got %q", suffix, states[0].Attributes["metadata.uid"])
					}
					return nil
				},
			},
		},
	})
}

func TestAccGenericResource_folderEmptySpecOverrideClearsManifestField(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	expectedTitle := "Generic Clear Folder " + suffix

	config := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  spec = {
    title = %q
  }

  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-clear-folder-%s"
    }
    spec = {
      title       = "Manifest Title Should Be Overridden"
      description = "This description should be cleared by the empty override"
    }
  }
}
`, genericProviderConfig(t), expectedTitle, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-clear-folder-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "spec.title", expectedTitle),
					genericEventually(genericResourceName, getGenericFolder, func(folder *models.Folder) error {
						if folder.Title != expectedTitle {
							return fmt.Errorf("expected folder title %q, got %q", expectedTitle, folder.Title)
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_folderRejectsNamespaceOutsideProviderContext(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	// Step 1: namespace mismatch inside manifest.metadata.namespace
	configManifestNamespaceMismatch := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name      = "generic-ns-mismatch-%s"
      namespace = "org-999"
    }
    spec = {
      title = "Namespace Mismatch Folder %s"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	// Step 2: namespace mismatch via top-level metadata
	configTopLevelNamespaceMismatch := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  metadata = {
    uid       = "generic-ns-mismatch-top-%s"
    namespace = "org-999"
  }

  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    spec = {
      title = "Namespace Mismatch Top Level %s"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      configManifestNamespaceMismatch,
				ExpectError: regexp.MustCompile("Namespace does not match provider context"),
			},
			{
				Config:      configTopLevelNamespaceMismatch,
				ExpectError: regexp.MustCompile("Namespace does not match provider context"),
			},
		},
	})
}

func TestAccGenericResource_folderManifestFieldValidation(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	// Step 1: secure inside manifest should be rejected
	configSecureInManifest := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-validation-%s"
    }
    spec = {
      title = "Validation Folder %s"
    }
    secure = {
      token = {
        create = "secret"
      }
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	// Step 2: unsupported manifest field should be rejected
	configUnsupportedField := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-validation-unsupported-%s"
    }
    spec = {
      title = "Unsupported Field Folder %s"
    }
    data = {
      unexpected = true
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	// Step 3: status in manifest should be silently accepted
	configStatusInManifest := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-validation-status-%s"
    }
    spec = {
      title = "Status Field Folder %s"
    }
    status = {
      phase = "ready"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config:      configSecureInManifest,
				ExpectError: regexp.MustCompile("(?i)secure.*must not be set inside.*manifest"),
			},
			{
				Config:      configUnsupportedField,
				ExpectError: regexp.MustCompile("(?i)unsupported.*manifest"),
			},
			{
				Config: configStatusInManifest,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-validation-status-"+suffix),
				),
			},
		},
	})
}

func TestAccGenericResource_folderManifestOnlyRoundTrip(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	expectedTitle := "Generic Manifest Only Folder " + suffix

	// Pure manifest config — no api_group, version, kind, metadata, or spec overrides.
	config := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-manifest-only-%s"
    }
    spec = {
      title = %q
    }
  }
}
`, genericProviderConfig(t), suffix, expectedTitle)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-manifest-only-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.apiVersion", "folder.grafana.app/v1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Folder"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.title", expectedTitle),
					genericEventually(genericResourceName, getGenericFolder, func(folder *models.Folder) error {
						if folder.Title != expectedTitle {
							return fmt.Errorf("expected folder title %q, got %q", expectedTitle, folder.Title)
						}
						return nil
					}),
				),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					if states[0].Attributes["metadata.uid"] != "generic-manifest-only-"+suffix {
						return fmt.Errorf("expected imported metadata.uid, got %q", states[0].Attributes["metadata.uid"])
					}
					if states[0].Attributes["manifest.spec.title"] != expectedTitle {
						return fmt.Errorf("expected imported manifest.spec.title, got %q", states[0].Attributes["manifest.spec.title"])
					}
					return nil
				},
			},
		},
	})
}

func TestAccGenericResource_folderSpecDriftDetectsServerAddedFields(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	expectedTitle := "Generic Spec Drift Folder " + suffix

	// Config has only title in spec — no description.
	config := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-spec-drift-%s"
    }
    spec = {
      title = %q
    }
  }
}
`, genericProviderConfig(t), suffix, expectedTitle)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericFolderDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: config,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "metadata.uid", "generic-spec-drift-"+suffix),
				),
			},
			// Mutate via API: add a description field that wasn't in config.
			{
				Config: config,
				Check:  testAccMutateGenericFolderAddDescription(genericResourceName, "UI-added description"),
				// The mutation adds description to spec; refresh should detect this
				// as drift per goal.md: "anything added compared to config should cause drift".
				ExpectNonEmptyPlan: true,
			},
			// Apply should restore the spec to config state (just title).
			{
				Config: config,
				Check: genericEventually(genericResourceName, getGenericFolder, func(folder *models.Folder) error {
					if folder.Title != expectedTitle {
						return fmt.Errorf("expected folder title %q, got %q", expectedTitle, folder.Title)
					}
					return nil
				}),
			},
			{
				Config:             config,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccGenericResource_secureValidationErrors(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))

	// secure without secure_version
	configNoVersion := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "provisioning.grafana.app/v1beta1"
    kind       = "Repository"
    metadata = {
      name = "generic-secure-val-%s"
    }
    spec = {
      title = "Secure Validation %s"
      type  = "github"
    }
  }

  secure = {
    token = {
      create = "secret-value"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)

	// secure with empty object
	configEmptyObject := fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "provisioning.grafana.app/v1beta1"
    kind       = "Repository"
    metadata = {
      name = "generic-secure-val-empty-%s"
    }
    spec = {
      title = "Secure Validation Empty %s"
      type  = "github"
    }
  }

  secure = {
    token = {}
  }

  secure_version = 1
}
`, genericProviderConfig(t), suffix, suffix)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      configNoVersion,
				ExpectError: regexp.MustCompile("(?i)(secure.version|missing.*secure)"),
			},
			{
				Config:      configEmptyObject,
				ExpectError: regexp.MustCompile("(?i)(must set.*one of|create.*name)"),
			},
		},
	})
}

func testAccMutateGenericFolderAddDescription(resourceName, description string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		uid, err := stateResourceAttribute(s, resourceName, "metadata.uid")
		if err != nil {
			return err
		}

		manifest, err := getGenericFolderManifest(context.Background(), client, uid)
		if err != nil {
			return err
		}

		spec, ok := manifest["spec"].(map[string]any)
		if !ok {
			return fmt.Errorf("folder manifest spec missing or invalid: %#v", manifest["spec"])
		}
		spec["description"] = description
		manifest["spec"] = spec

		if _, err := genericFolderAppPlatformRequest(context.Background(), client, http.MethodPut, uid, manifest); err != nil {
			return err
		}

		return genericEventually(resourceName, getGenericFolderManifestSpec, func(actual map[string]any) error {
			if actual["description"] != description {
				return fmt.Errorf("expected drifted description %q, got %v", description, actual["description"])
			}
			return nil
		})(s)
	}
}

func getGenericFolderManifestSpec(ctx context.Context, client *common.Client, uid string) (map[string]any, error) {
	manifest, err := getGenericFolderManifest(ctx, client, uid)
	if err != nil {
		return nil, err
	}

	spec, _ := manifest["spec"].(map[string]any)
	return spec, nil
}

func testAccGenericFolderConfig(t *testing.T, suffix string) string {
	t.Helper()

	return fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = yamldecode(<<-YAML
    apiVersion: folder.grafana.app/v1
    kind: Folder
    metadata:
      name: generic-folder-%s
      namespace: %s
    spec:
      title: Manifest Folder %s
      description: Generic Folder %s Description
  YAML
  )

  spec = {
    title = "Generic Folder %s"
  }
}
`, genericProviderConfig(t), suffix, claims.OrgNamespaceFormatter(grafanaOrgID(t)), suffix, suffix, suffix)
}

func testAccGenericFolderAnnotationLabelConfig(t *testing.T, suffix string) string {
	t.Helper()

	return fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "folder.grafana.app/v1"
    kind       = "Folder"
    metadata = {
      name = "generic-folder-meta-%s"
      annotations = {
        "test.grafana.app/owner" = "terraform"
      }
      labels = {
        "test.grafana.app/env" = "acceptance"
      }
    }
    spec = {
      title = "Generic Folder Metadata %s"
    }
  }
}
`, genericProviderConfig(t), suffix, suffix)
}

func getGenericFolder(_ context.Context, client *common.Client, uid string) (*models.Folder, error) {
	resp, err := client.GrafanaAPI.Folders.GetFolderByUID(uid)
	if err != nil {
		return nil, err
	}
	return resp.GetPayload(), nil
}

func getGenericFolderAnnotationLabel(ctx context.Context, client *common.Client, uid string) ([2]string, error) {
	manifest, err := getGenericFolderManifest(ctx, client, uid)
	if err != nil {
		return [2]string{}, err
	}

	metadata, _ := manifest["metadata"].(map[string]any)
	annotations, _ := metadata["annotations"].(map[string]any)
	labels, _ := metadata["labels"].(map[string]any)
	annotation, _ := annotations["test.grafana.app/owner"].(string)
	label, _ := labels["test.grafana.app/env"].(string)
	return [2]string{annotation, label}, nil
}

func testAccMutateGenericFolderTitle(resourceName, title string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		uid, err := stateResourceAttribute(s, resourceName, "metadata.uid")
		if err != nil {
			return err
		}

		_, err = client.GrafanaAPI.Folders.UpdateFolder(uid, &models.UpdateFolderCommand{
			Overwrite: true,
			Title:     title,
		})
		if err != nil {
			return err
		}

		return genericEventually(resourceName, getGenericFolder, func(folder *models.Folder) error {
			if folder.Title != title {
				return fmt.Errorf("expected drifted folder title %q, got %q", title, folder.Title)
			}
			return nil
		})(s)
	}
}

func testAccMutateGenericFolderAnnotationLabel(resourceName, annotationValue, labelValue string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		uid, err := stateResourceAttribute(s, resourceName, "metadata.uid")
		if err != nil {
			return err
		}

		manifest, err := getGenericFolderManifest(context.Background(), client, uid)
		if err != nil {
			return err
		}

		metadata, ok := manifest["metadata"].(map[string]any)
		if !ok {
			return fmt.Errorf("folder manifest metadata missing or invalid: %#v", manifest["metadata"])
		}
		annotations, _ := metadata["annotations"].(map[string]any)
		if annotations == nil {
			annotations = map[string]any{}
		}
		annotations["test.grafana.app/owner"] = annotationValue
		metadata["annotations"] = annotations

		labels, _ := metadata["labels"].(map[string]any)
		if labels == nil {
			labels = map[string]any{}
		}
		labels["test.grafana.app/env"] = labelValue
		metadata["labels"] = labels
		manifest["metadata"] = metadata

		if _, err := genericFolderAppPlatformRequest(context.Background(), client, http.MethodPut, uid, manifest); err != nil {
			return err
		}

		return genericEventually(resourceName, getGenericFolderAnnotationLabel, func(actual [2]string) error {
			if actual[0] != annotationValue {
				return fmt.Errorf("expected drifted annotation %q, got %q", annotationValue, actual[0])
			}
			if actual[1] != labelValue {
				return fmt.Errorf("expected drifted label %q, got %q", labelValue, actual[1])
			}
			return nil
		})(s)
	}
}

func testAccCheckGenericFolderDestroy(s *terraform.State) error {
	return genericCheckDestroyWithNotFound(s, "grafana_apps_generic_resource", "folder", getGenericFolder, folderNotFound)
}

func folderNotFound(err error) bool {
	if err == nil {
		return false
	}

	errText := strings.ToLower(err.Error())
	return strings.Contains(errText, "not found") || strings.Contains(errText, "404")
}

func getGenericFolderManifest(ctx context.Context, client *common.Client, uid string) (map[string]any, error) {
	body, err := genericFolderAppPlatformRequest(ctx, client, http.MethodGet, uid, nil)
	if err != nil {
		return nil, err
	}

	var manifest map[string]any
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, fmt.Errorf("failed to decode folder manifest: %w", err)
	}

	return manifest, nil
}

func genericFolderAppPlatformRequest(ctx context.Context, client *common.Client, method, uid string, payload map[string]any) ([]byte, error) {
	namespace, err := testAccNamespace(client)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/apis/folder.grafana.app/v1/namespaces/%s/folders/%s", namespace, uid)

	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to encode folder manifest request: %w", err)
		}
		body = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, client.GrafanaSubpath(path), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	for key, value := range client.GrafanaAPIConfig.HTTPHeaders {
		req.Header.Set(key, value)
	}
	if client.GrafanaAPIConfig.OrgID > 0 {
		req.Header.Set("X-Grafana-Org-Id", strconv.FormatInt(client.GrafanaAPIConfig.OrgID, 10))
	}
	if client.GrafanaAPIConfig.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+client.GrafanaAPIConfig.APIKey)
	} else if client.GrafanaAPIConfig.BasicAuth != nil {
		password, _ := client.GrafanaAPIConfig.BasicAuth.Password()
		req.SetBasicAuth(client.GrafanaAPIConfig.BasicAuth.Username(), password)
	}

	transport := http.DefaultTransport.(*http.Transport).Clone()
	if client.GrafanaAPIConfig.TLSConfig != nil {
		transport.TLSClientConfig = client.GrafanaAPIConfig.TLSConfig.Clone()
	}

	resp, err := (&http.Client{Transport: transport}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("folder app platform %s %s failed with status %d: %s", method, path, resp.StatusCode, strings.TrimSpace(string(responseBody)))
	}

	return responseBody, nil
}
