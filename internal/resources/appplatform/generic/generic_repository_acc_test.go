package generic_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

type genericRepositoryConfig struct {
	Title         string
	Path          string
	TokenCreate   string
	TokenName     string
	WebhookCreate string
	WebhookName   string
	SecureVersion *int
}

func TestAccGenericResource_repositorySecure(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	suffix := strings.ToLower(acctest.RandStringFromCharSet(8, acctest.CharSetAlphaNum))
	tokenV1 := acctest.RandString(24)
	tokenV2 := acctest.RandString(24)
	webhookSecretV1 := acctest.RandString(24)
	webhookSecretV2 := acctest.RandString(24)
	var tokenNameV1 string
	var webhookSecretNameV1 string

	// Step 1: create with secure version 1 — both token and webhook use create.
	configV1 := renderGenericRepositoryConfig(t, suffix, genericRepositoryConfig{
		Title:         "Generic Repository " + suffix,
		Path:          "examples",
		TokenCreate:   tokenV1,
		WebhookCreate: webhookSecretV1,
		SecureVersion: genericSecureVersion(1),
	})

	// Step 2: change secure values but keep secure_version=1 — provider must NOT re-send.
	configSecureChangedNoVersion := renderGenericRepositoryConfig(t, suffix, genericRepositoryConfig{
		Title:         "Generic Repository " + suffix,
		Path:          "examples",
		TokenCreate:   tokenV2,
		WebhookCreate: webhookSecretV2,
		SecureVersion: genericSecureVersion(1),
	})

	// Step 3: bump secure_version to 2 with new create values — secrets must rotate.
	configV2 := renderGenericRepositoryConfig(t, suffix, genericRepositoryConfig{
		Title:         "Generic Repository " + suffix + " Updated",
		Path:          "docs",
		TokenCreate:   tokenV2,
		WebhookCreate: webhookSecretV2,
		SecureVersion: genericSecureVersion(2),
	})

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckGenericProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			// Step 1: initial create with secure fields.
			{
				Config: configV1,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttrSet(genericResourceName, "id"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.metadata.name", "generic-repository-"+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.title", "Generic Repository "+suffix),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.github.path", "examples"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.apiVersion", "provisioning.grafana.app/v1beta1"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.kind", "Repository"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "secure_version", "1"),
					genericCheckNoStateAttributePrefix(genericResourceName, "secure"),
					genericEventually(genericResourceName, getProvisioningRepository, func(repository *appplatform.ProvisioningRepository) error {
						if repository.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if repository.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to be populated")
						}
						tokenNameV1 = repository.Secure.Token.Name
						webhookSecretNameV1 = repository.Secure.WebhookSecret.Name
						return nil
					}),
				),
			},
			// Step 2: secure values changed but version NOT bumped — secrets must NOT rotate.
			{
				Config: configSecureChangedNoVersion,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "secure_version", "1"),
					genericCheckNoStateAttributePrefix(genericResourceName, "secure"),
					genericEventually(genericResourceName, getProvisioningRepository, func(repository *appplatform.ProvisioningRepository) error {
						if tokenNameV1 == "" || webhookSecretNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if repository.Secure.Token.Name != tokenNameV1 {
							return fmt.Errorf("expected token secure reference to remain unchanged without a secure_version bump, got %q", repository.Secure.Token.Name)
						}
						if repository.Secure.WebhookSecret.Name != webhookSecretNameV1 {
							return fmt.Errorf("expected webhook secret secure reference to remain unchanged without a secure_version bump, got %q", repository.Secure.WebhookSecret.Name)
						}
						return nil
					}),
				),
			},
			// Step 3: bump secure_version to 2 — secrets must rotate via create.
			{
				Config: configV2,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.title", "Generic Repository "+suffix+" Updated"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "manifest.spec.github.path", "docs"),
					terraformresource.TestCheckResourceAttr(genericResourceName, "secure_version", "2"),
					genericCheckNoStateAttributePrefix(genericResourceName, "secure"),
					genericEventually(genericResourceName, getProvisioningRepository, func(repository *appplatform.ProvisioningRepository) error {
						if tokenNameV1 == "" {
							return fmt.Errorf("missing baseline token secure reference from initial apply")
						}
						if repository.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to remain populated after secure rotation")
						}
						if repository.Secure.Token.Name == tokenNameV1 {
							return fmt.Errorf("expected token secure reference to rotate after secure_version bump, still %q", repository.Secure.Token.Name)
						}
						return nil
					}),
				),
			},
			// Step 4: idempotent — no changes expected.
			{
				Config:             configV2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			// Step 5: import — secure fields are write-only so they're ignored.
			{
				ResourceName:      genericResourceName,
				ImportState:       true,
				ImportStateIdFunc: genericResourceImportIDFunc(genericResourceName),
				ImportStateCheck: func(states []*terraform.InstanceState) error {
					if len(states) != 1 {
						return fmt.Errorf("expected one imported state, got %d", len(states))
					}
					if states[0].Attributes["manifest.metadata.name"] != "generic-repository-"+suffix {
						return fmt.Errorf("expected imported manifest.metadata.name, got %q", states[0].Attributes["manifest.metadata.name"])
					}
					return nil
				},
			},
			// Step 6: re-apply after import — secure fields re-sent via create.
			{
				Config: configV2,
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(genericResourceName, "secure_version", "2"),
					genericCheckNoStateAttributePrefix(genericResourceName, "secure"),
					genericEventually(genericResourceName, getProvisioningRepository, func(repository *appplatform.ProvisioningRepository) error {
						if repository.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be rebound after import")
						}
						return nil
					}),
				),
			},
			// Step 7: idempotent after import re-apply.
			{
				Config:             configV2,
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func renderGenericRepositoryConfig(t *testing.T, suffix string, cfg genericRepositoryConfig) string {
	t.Helper()

	tokenConfig := renderGenericSecureField(t, "token", cfg.TokenCreate, cfg.TokenName)
	webhookConfig := renderGenericSecureField(t, "webhookSecret", cfg.WebhookCreate, cfg.WebhookName)
	secureVersion := ""
	if cfg.SecureVersion != nil {
		secureVersion = fmt.Sprintf("\n  secure_version = %d\n", *cfg.SecureVersion)
	}

	return fmt.Sprintf(`
%s

resource "grafana_apps_generic_resource" "test" {
  manifest = {
    apiVersion = "provisioning.grafana.app/v1beta1"
    kind       = "Repository"
    metadata = {
      name = "generic-repository-%s"
    }
    spec = {
      title       = %q
      description = "Acceptance repository managed through the generic resource"
      type        = "github"
      workflows   = ["write"]
      sync = {
        enabled         = false
        target          = "folder"
        intervalSeconds = 300
      }
      github = {
        url                       = "https://github.com/grafana/terraform-provider-grafana"
        branch                    = "main"
        path                      = %q
        generateDashboardPreviews = false
      }
    }
  }

  secure = {
    token = {
      %s
    }
    webhookSecret = {
      %s
    }
  }%s
}
`, genericProviderConfig(t), suffix, cfg.Title, cfg.Path, tokenConfig, webhookConfig, secureVersion)
}

func renderGenericSecureField(t *testing.T, fieldName, createValue, nameValue string) string {
	t.Helper()

	createValue = strings.TrimSpace(createValue)
	nameValue = strings.TrimSpace(nameValue)

	switch {
	case createValue != "" && nameValue != "":
		t.Fatalf("secure field %q must set exactly one of create or name", fieldName)
	case createValue == "" && nameValue == "":
		t.Fatalf("secure field %q must set exactly one of create or name", fieldName)
	case createValue != "":
		return fmt.Sprintf("create = %q", createValue)
	}

	return fmt.Sprintf("name = %q", nameValue)
}

func genericSecureVersion(version int) *int {
	return &version
}

func testAccCheckGenericProvisioningRepositoryDestroy(s *terraform.State) error {
	return genericCheckDestroy(s, "grafana_apps_generic_resource", "repository", getProvisioningRepository)
}

func getProvisioningRepository(ctx context.Context, client *common.Client, uid string) (*appplatform.ProvisioningRepository, error) {
	return getProvisioningResource[*appplatform.ProvisioningRepository, *appplatform.ProvisioningRepositoryList](
		ctx,
		client,
		uid,
		appplatform.RepositoryKind(),
	)
}

func getProvisioningResource[T sdkresource.Object, L sdkresource.ListObject](
	ctx context.Context,
	client *common.Client,
	uid string,
	kind sdkresource.Kind,
) (T, error) {
	var zero T

	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(kind)
	if err != nil {
		return zero, fmt.Errorf("failed to create provisioning client: %w", err)
	}

	ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	namespacedClient := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[T, L](rcli, kind),
		ns,
	)

	return namespacedClient.Get(ctx, uid)
}
