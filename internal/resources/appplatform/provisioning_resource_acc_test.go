package appplatform_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	provisioningv0alpha1 "github.com/grafana/grafana/apps/provisioning/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	provisioningConnectionResourceName = "grafana_apps_provisioning_connection_v0alpha1.test"
	provisioningRepositoryResourceName = "grafana_apps_provisioning_repository_v0alpha1.test"
	provisioningLocalRepositoryPath    = "conf/provisioning"
)

func TestAccProvisioningConnection_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-conn-" + strings.ToLower(acctest.RandString(8))
	keyV1 := provisioningFixturePath(t, "github-app-private-key-v1.pem")
	titleV1 := "Acceptance GitHub App connection"
	titleV2 := "Acceptance GitHub App connection updated"
	descriptionV1 := "Acceptance test connection"
	descriptionV2 := "Acceptance test connection updated"
	var privateKeyNameV1 string
	var tokenNameV1 string

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningConnectionDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningConnectionConfig(uid, titleV1, descriptionV1, keyV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.title", titleV1),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.description", descriptionV1),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.type", "github"),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.github.app_id", "12345"),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "secure_version", "1"),
					testAccProvisioningConnectionEventually(provisioningConnectionResourceName, func(conn *appplatform.ProvisioningConnection) error {
						if conn.Spec.Type != provisioningv0alpha1.GithubConnectionType {
							return fmt.Errorf("expected github connection type, got %q", conn.Spec.Type)
						}
						if conn.Spec.GitHub == nil {
							return fmt.Errorf("expected github config to be present")
						}
						if conn.Spec.GitHub.AppID != "12345" {
							return fmt.Errorf("expected appID 12345, got %q", conn.Spec.GitHub.AppID)
						}
						if conn.Secure.PrivateKey.Name == "" {
							return fmt.Errorf("expected private key secure reference to be populated")
						}
						if conn.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if conn.Secure.ClientSecret.Name != "" {
							return fmt.Errorf("expected client secret to remain unset for github connections")
						}
						privateKeyNameV1 = conn.Secure.PrivateKey.Name
						tokenNameV1 = conn.Secure.Token.Name
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningConnectionConfig(uid, titleV2, descriptionV2, keyV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.title", titleV2),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.description", descriptionV2),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "secure_version", "1"),
					testAccProvisioningConnectionEventually(provisioningConnectionResourceName, func(conn *appplatform.ProvisioningConnection) error {
						if conn.Spec.Title != titleV2 {
							return fmt.Errorf("expected updated connection title %q, got %q", titleV2, conn.Spec.Title)
						}
						if conn.Spec.Description != descriptionV2 {
							return fmt.Errorf("expected updated connection description %q, got %q", descriptionV2, conn.Spec.Description)
						}
						if privateKeyNameV1 == "" || tokenNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if conn.Secure.PrivateKey.Name == "" {
							return fmt.Errorf("expected private key secure reference to remain populated after spec update")
						}
						if conn.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to remain populated after spec update")
						}
						if conn.Secure.PrivateKey.Name != privateKeyNameV1 {
							return fmt.Errorf("expected private key secure reference to remain unchanged after spec update, got %q", conn.Secure.PrivateKey.Name)
						}
						if conn.Secure.Token.Name != tokenNameV1 {
							return fmt.Errorf("expected token secure reference to remain unchanged after spec update, got %q", conn.Secure.Token.Name)
						}
						if conn.Secure.ClientSecret.Name != "" {
							return fmt.Errorf("expected client secret to remain unset for github connections")
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      provisioningConnectionResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				// Grafana's GitHub connection mutator rewrites spec.url to the installation URL
				// on read, so import returns a canonicalized value rather than the configured base URL.
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
					"secure",
					"secure_version",
					"spec.url",
				},
				ImportStateIdFunc: importStateUIDFunc(provisioningConnectionResourceName),
			},
		},
	})
}

func TestAccProvisioningConnection_secureRotation(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-conn-rotate-" + strings.ToLower(acctest.RandString(8))
	keyV1 := provisioningFixturePath(t, "github-app-private-key-v1.pem")
	keyV2 := provisioningFixturePath(t, "github-app-private-key-v2.pem")

	var privateKeyNameV1 string
	var tokenNameV1 string

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningConnectionDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningConnectionConfig(uid, "Acceptance GitHub App connection", "Acceptance test connection", keyV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "secure_version", "1"),
					testAccProvisioningConnectionEventually(provisioningConnectionResourceName, func(conn *appplatform.ProvisioningConnection) error {
						if conn.Secure.PrivateKey.Name == "" {
							return fmt.Errorf("expected private key secure reference to be populated")
						}
						if conn.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						privateKeyNameV1 = conn.Secure.PrivateKey.Name
						tokenNameV1 = conn.Secure.Token.Name
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningConnectionConfig(uid, "Acceptance GitHub App connection", "Acceptance test connection", keyV2, 2),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "secure_version", "2"),
					testAccProvisioningConnectionEventually(provisioningConnectionResourceName, func(conn *appplatform.ProvisioningConnection) error {
						if privateKeyNameV1 == "" || tokenNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if conn.Secure.PrivateKey.Name == "" {
							return fmt.Errorf("expected rotated private key secure reference to be populated")
						}
						if conn.Secure.Token.Name == "" {
							return fmt.Errorf("expected rotated token secure reference to be populated")
						}
						if conn.Secure.PrivateKey.Name == privateKeyNameV1 {
							return fmt.Errorf("expected private key secure reference to rotate, still %q", conn.Secure.PrivateKey.Name)
						}
						if conn.Secure.Token.Name == tokenNameV1 {
							return fmt.Errorf("expected token secure reference to rotate, still %q", conn.Secure.Token.Name)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccProvisioningRepository_basic(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-repo-basic-" + strings.ToLower(acctest.RandString(8))
	token := acctest.RandString(24)
	webhookSecret := acctest.RandString(24)
	titleV1 := "Acceptance Git Sync repository"
	titleV2 := "Acceptance Git Sync repository updated"
	descriptionV1 := "Acceptance test repository"
	descriptionV2 := "Acceptance test repository updated"
	pathV1 := "examples"
	pathV2 := "docs"
	var tokenNameV1 string
	var webhookSecretNameV1 string

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryConfig(uid, titleV1, descriptionV1, pathV1, token, webhookSecret, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.title", titleV1),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.description", descriptionV1),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.type", "github"),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.github.branch", "main"),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.github.path", pathV1),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Spec.Type != provisioningv0alpha1.GitHubRepositoryType {
							return fmt.Errorf("expected github repository type, got %q", repo.Spec.Type)
						}
						if repo.Spec.GitHub == nil {
							return fmt.Errorf("expected github repository config to be present")
						}
						if repo.Spec.GitHub.Path != pathV1 {
							return fmt.Errorf("expected path %q, got %q", pathV1, repo.Spec.GitHub.Path)
						}
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to be populated")
						}
						tokenNameV1 = repo.Secure.Token.Name
						webhookSecretNameV1 = repo.Secure.WebhookSecret.Name
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningRepositoryConfig(uid, titleV2, descriptionV2, pathV2, token, webhookSecret, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.title", titleV2),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.description", descriptionV2),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.github.path", pathV2),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Spec.Title != titleV2 {
							return fmt.Errorf("expected updated repository title %q, got %q", titleV2, repo.Spec.Title)
						}
						if repo.Spec.Description != descriptionV2 {
							return fmt.Errorf("expected updated repository description %q, got %q", descriptionV2, repo.Spec.Description)
						}
						if repo.Spec.GitHub == nil {
							return fmt.Errorf("expected github repository config to be present")
						}
						if repo.Spec.GitHub.Path != pathV2 {
							return fmt.Errorf("expected updated path %q, got %q", pathV2, repo.Spec.GitHub.Path)
						}
						if tokenNameV1 == "" || webhookSecretNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to remain populated after spec update")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to remain populated after spec update")
						}
						if repo.Secure.Token.Name != tokenNameV1 {
							return fmt.Errorf("expected token secure reference to remain unchanged after spec update, got %q", repo.Secure.Token.Name)
						}
						if repo.Secure.WebhookSecret.Name != webhookSecretNameV1 {
							return fmt.Errorf("expected webhook secret secure reference to remain unchanged after spec update, got %q", repo.Secure.WebhookSecret.Name)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      provisioningRepositoryResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
					"secure",
					"secure_version",
				},
				ImportStateIdFunc: importStateUIDFunc(provisioningRepositoryResourceName),
			},
		},
	})
}

func TestAccProvisioningRepository_secureRotation(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-repo-rotate-" + strings.ToLower(acctest.RandString(8))
	tokenV1 := acctest.RandString(24)
	tokenV2 := acctest.RandString(24)
	webhookSecretV1 := acctest.RandString(24)
	webhookSecretV2 := acctest.RandString(24)

	var tokenNameV1 string
	var webhookSecretNameV1 string

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryConfig(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", tokenV1, webhookSecretV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to be populated")
						}
						tokenNameV1 = repo.Secure.Token.Name
						webhookSecretNameV1 = repo.Secure.WebhookSecret.Name
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningRepositoryConfig(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", tokenV2, webhookSecretV2, 2),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "2"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if tokenNameV1 == "" || webhookSecretNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected rotated token secure reference to be populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected rotated webhook secret secure reference to be populated")
						}
						if repo.Secure.Token.Name == tokenNameV1 {
							return fmt.Errorf("expected token secure reference to rotate, still %q", repo.Secure.Token.Name)
						}
						if repo.Secure.WebhookSecret.Name == webhookSecretNameV1 {
							return fmt.Errorf("expected webhook secret secure reference to rotate, still %q", repo.Secure.WebhookSecret.Name)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccProvisioningRepository_secureChangeIgnoredWithoutVersionChange(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-repo-same-version-" + strings.ToLower(acctest.RandString(8))
	tokenV1 := acctest.RandString(24)
	tokenV2 := acctest.RandString(24)
	webhookSecretV1 := acctest.RandString(24)
	webhookSecretV2 := acctest.RandString(24)

	var tokenNameV1 string
	var webhookSecretNameV1 string
	var metadataVersionV1 string

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryConfig(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", tokenV1, webhookSecretV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to be populated")
						}
						tokenNameV1 = repo.Secure.Token.Name
						webhookSecretNameV1 = repo.Secure.WebhookSecret.Name
						return nil
					}),
					testAccCaptureStateAttribute(provisioningRepositoryResourceName, "metadata.version", &metadataVersionV1),
				),
			},
			{
				Config: testAccProvisioningRepositoryConfig(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", tokenV2, webhookSecretV2, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if tokenNameV1 == "" || webhookSecretNameV1 == "" {
							return fmt.Errorf("missing baseline secure references from initial apply")
						}
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to remain populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to remain populated")
						}
						if repo.Secure.Token.Name != tokenNameV1 {
							return fmt.Errorf("expected token secure reference to remain unchanged without secure_version bump, got %q", repo.Secure.Token.Name)
						}
						if repo.Secure.WebhookSecret.Name != webhookSecretNameV1 {
							return fmt.Errorf("expected webhook secret secure reference to remain unchanged without secure_version bump, got %q", repo.Secure.WebhookSecret.Name)
						}
						return nil
					}),
					testAccCheckStateAttributeEquals(provisioningRepositoryResourceName, "metadata.version", &metadataVersionV1),
				),
			},
		},
	})
}

func TestAccProvisioningRepository_secureRemoval(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-repo-remove-" + strings.ToLower(acctest.RandString(8))
	token := acctest.RandString(24)
	webhookSecret := acctest.RandString(24)

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryConfig(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", token, webhookSecret, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						if repo.Secure.WebhookSecret.Name == "" {
							return fmt.Errorf("expected webhook secret secure reference to be populated")
						}
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningRepositoryConfigWithoutWebhookSecret(uid, "Acceptance Git Sync repository", "Acceptance test repository", "examples", token, 2),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "2"),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						// secure_version bumps intentionally re-apply configured secure values, so
						// the surviving token reference may rotate even though only webhook_secret
						// was removed in config.
						if repo.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to remain populated")
						}
						if repo.Secure.WebhookSecret.Name != "" {
							return fmt.Errorf("expected webhook secret secure reference to be removed, got %q", repo.Secure.WebhookSecret.Name)
						}
						return nil
					}),
				),
			},
		},
	})
}

func TestAccProvisioningRepository_viaConnection(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	connectionUID := "git-sync-conn-ref-" + strings.ToLower(acctest.RandString(8))
	repositoryUID := "git-sync-repo-via-conn-" + strings.ToLower(acctest.RandString(8))
	keyV1 := provisioningFixturePath(t, "github-app-private-key-v1.pem")

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningConnectionAndRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryViaConnectionConfig(connectionUID, repositoryUID, keyV1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr("grafana_apps_provisioning_connection_v0alpha1.github_app", "metadata.uid", connectionUID),
					terraformresource.TestCheckResourceAttr("grafana_apps_provisioning_repository_v0alpha1.test", "metadata.uid", repositoryUID),
					terraformresource.TestCheckResourceAttr("grafana_apps_provisioning_repository_v0alpha1.test", "spec.connection.name", connectionUID),
					terraformresource.TestCheckResourceAttr("grafana_apps_provisioning_repository_v0alpha1.test", "spec.type", "github"),
					terraformresource.TestCheckResourceAttr("grafana_apps_provisioning_repository_v0alpha1.test", "spec.github.path", "docs"),
					testAccProvisioningConnectionEventually("grafana_apps_provisioning_connection_v0alpha1.github_app", func(conn *appplatform.ProvisioningConnection) error {
						if conn.Secure.PrivateKey.Name == "" {
							return fmt.Errorf("expected private key secure reference to be populated")
						}
						if conn.Secure.Token.Name == "" {
							return fmt.Errorf("expected token secure reference to be populated")
						}
						return nil
					}),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Spec.Connection == nil {
							return fmt.Errorf("expected repository connection reference to be present")
						}
						if repo.Spec.Connection.Name != connectionUID {
							return fmt.Errorf("expected repository connection name %q, got %q", connectionUID, repo.Spec.Connection.Name)
						}
						return nil
					}),
				),
			},
			{
				Config: testAccProvisioningReferencedConnectionConfig(connectionUID, keyV1),
				Check: terraformresource.ComposeTestCheckFunc(
					testAccProvisioningConnectionEventually("grafana_apps_provisioning_connection_v0alpha1.github_app", func(conn *appplatform.ProvisioningConnection) error {
						if conn.Name != connectionUID {
							return fmt.Errorf("expected connection %q to remain after repository removal, got %q", connectionUID, conn.Name)
						}
						return nil
					}),
					testAccProvisioningRepositoryAbsent(repositoryUID),
				),
			},
		},
	})
}

func TestAccProvisioningRepository_local(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-repo-local-" + strings.ToLower(acctest.RandString(8))

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryLocalConfig(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.type", "local"),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.local.path", provisioningLocalRepositoryPath),
					testAccProvisioningRepositoryEventually(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Spec.Type != provisioningv0alpha1.LocalRepositoryType {
							return fmt.Errorf("expected local repository type, got %q", repo.Spec.Type)
						}
						if repo.Spec.Local == nil {
							return fmt.Errorf("expected local repository config to be present")
						}
						if repo.Spec.Local.Path != provisioningLocalRepositoryPath {
							return fmt.Errorf("expected local repository path %q, got %q", provisioningLocalRepositoryPath, repo.Spec.Local.Path)
						}
						return nil
					}),
				),
			},
			{
				ResourceName:      provisioningRepositoryResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"metadata.version",
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateUIDFunc(provisioningRepositoryResourceName),
			},
		},
	})
}

func TestAccProvisioningRepository_missingSecureVersion(t *testing.T) {
	testutils.CheckOSSTestsEnabled(t, ">=13.0.0")
	waitForProvisioningAPI(t)

	uid := "git-sync-missing-secure-version-" + strings.ToLower(acctest.RandString(8))

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      testAccProvisioningRepositoryMissingSecureVersionConfig(uid),
				ExpectError: regexp.MustCompile("Missing secure version"),
			},
		},
	})
}

func testAccProvisioningConnectionConfig(uid, title, description, privateKeyPath string, secureVersion int) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_connection_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "%s"
    description = "%s"
    type        = "github"
    url         = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  # GitHub App connections only take the private key as input; Grafana generates
  # the connection token asynchronously from that key, and the acc tests verify
  # that full provider + Grafana behavior via the live API.
  secure {
    private_key = {
      create = filebase64(%q)
    }
  }

  secure_version = %d
}
`, uid, title, description, privateKeyPath, secureVersion)
}

func testAccProvisioningRepositoryConfig(uid, title, description, path, token, webhookSecret string, secureVersion int) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "%s"
    description = "%s"
    type        = "github"
    workflows   = ["write"]

    sync {
      enabled          = false
      target           = "instance"
      interval_seconds = 300
    }

    github {
      url                         = "https://github.com/grafana/terraform-provider-grafana"
      branch                      = "main"
      path                        = "%s"
      generate_dashboard_previews = false
    }
  }

  secure {
    token = {
      create = %q
    }
    webhook_secret = {
      create = %q
    }
  }

  secure_version = %d
}
`, uid, title, description, path, token, webhookSecret, secureVersion)
}

func testAccProvisioningRepositoryConfigWithoutWebhookSecret(uid, title, description, path, token string, secureVersion int) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "%s"
    description = "%s"
    type        = "github"
    workflows   = ["write"]

    sync {
      enabled          = false
      target           = "instance"
      interval_seconds = 300
    }

    github {
      url                         = "https://github.com/grafana/terraform-provider-grafana"
      branch                      = "main"
      path                        = "%s"
      generate_dashboard_previews = false
    }
  }

  secure {
    token = {
      create = %q
    }
  }

  secure_version = %d
}
`, uid, title, description, path, token, secureVersion)
}

func testAccProvisioningRepositoryViaConnectionConfig(connectionUID, repositoryUID, privateKeyPath string) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_connection_v0alpha1" "github_app" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Referenced GitHub App connection"
    type  = "github"
    url   = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = filebase64(%q)
    }
  }

  secure_version = 1
}

resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  depends_on = [grafana_apps_provisioning_connection_v0alpha1.github_app]

  metadata {
    uid = "%s"
  }

  spec {
    title       = "Repository via connection"
    description = "Repository referencing a connection resource"
    type        = "github"
    workflows   = ["branch"]

    sync {
      enabled          = false
      target           = "instance"
      interval_seconds = 300
    }

    github {
      url                         = "https://github.com/grafana/terraform-provider-grafana"
      branch                      = "main"
      path                        = "docs"
      generate_dashboard_previews = false
    }

    connection {
      name = "%s"
    }
  }
}
`, connectionUID, privateKeyPath, repositoryUID, connectionUID)
}

func testAccProvisioningReferencedConnectionConfig(connectionUID, privateKeyPath string) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_connection_v0alpha1" "github_app" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Referenced GitHub App connection"
    type  = "github"
    url   = "https://github.com"

    github {
      app_id          = "12345"
      installation_id = "67890"
    }
  }

  secure {
    private_key = {
      create = filebase64(%q)
    }
  }

  secure_version = 1
}
`, connectionUID, privateKeyPath)
}

func testAccProvisioningRepositoryLocalConfig(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "Local repository"
    description = "Local repository fixture mounted via Docker Compose"
    type        = "local"
    workflows   = ["write"]

    sync {
      enabled          = false
      target           = "instance"
      interval_seconds = 300
    }

    local {
      path = "%s"
    }
  }
}
`, uid, provisioningLocalRepositoryPath)
}

func testAccProvisioningRepositoryMissingSecureVersionConfig(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "Missing secure version"
    description = "Acceptance test validation"
    type        = "github"
    workflows   = ["write"]

    sync {
      enabled          = false
      target           = "instance"
      interval_seconds = 300
    }

    github {
      url    = "https://github.com/grafana/terraform-provider-grafana"
      branch = "main"
      path   = "examples"
    }
  }

  secure {
    token = {
      create = "replace-me"
    }
  }
}
`, uid)
}

// Provisioning live API checks poll because Grafana can mutate these resources
// asynchronously after apply (for example canonicalizing GitHub connection URLs
// and generating connection tokens from private keys).
func testAccProvisioningConnectionEventually(resourceName string, check func(*appplatform.ProvisioningConnection) error) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		uid, err := stateResourceUID(s, resourceName)
		if err != nil {
			return err
		}

		client := testutils.Provider.Meta().(*common.Client)
		deadline := time.Now().Add(30 * time.Second)
		var lastErr error

		for time.Now().Before(deadline) {
			conn, err := getProvisioningConnection(context.Background(), client, uid)
			if err != nil {
				lastErr = err
			} else if check == nil {
				return nil
			} else if err := check(conn); err == nil {
				return nil
			} else {
				lastErr = err
			}

			time.Sleep(1 * time.Second)
		}

		if lastErr == nil {
			lastErr = fmt.Errorf("timed out waiting for provisioning connection %s", uid)
		}

		return lastErr
	}
}

// Repository live API checks use the same polling pattern as connections because
// provisioning writes can be followed by asynchronous server-side mutations or
// transient read errors while the resource settles.
func testAccProvisioningRepositoryEventually(resourceName string, check func(*appplatform.ProvisioningRepository) error) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		uid, err := stateResourceUID(s, resourceName)
		if err != nil {
			return err
		}

		client := testutils.Provider.Meta().(*common.Client)
		deadline := time.Now().Add(30 * time.Second)
		var lastErr error

		for time.Now().Before(deadline) {
			repo, err := getProvisioningRepository(context.Background(), client, uid)
			if err != nil {
				lastErr = err
			} else if check == nil {
				return nil
			} else if err := check(repo); err == nil {
				return nil
			} else {
				lastErr = err
			}

			time.Sleep(1 * time.Second)
		}

		if lastErr == nil {
			lastErr = fmt.Errorf("timed out waiting for provisioning repository %s", uid)
		}

		return lastErr
	}
}

func testAccProvisioningRepositoryAbsent(uid string) terraformresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		deadline := time.Now().Add(30 * time.Second)
		var lastErr error

		for time.Now().Before(deadline) {
			_, err := getProvisioningRepository(context.Background(), client, uid)
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				lastErr = err
				time.Sleep(1 * time.Second)
				continue
			}
			lastErr = fmt.Errorf("provisioning repository %s still exists", uid)
			time.Sleep(1 * time.Second)
		}

		if lastErr == nil {
			lastErr = fmt.Errorf("timed out waiting for provisioning repository %s to be deleted", uid)
		}

		return lastErr
	}
}

func testAccCheckProvisioningConnectionDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_apps_provisioning_connection_v0alpha1" {
			continue
		}

		uid := r.Primary.Attributes["metadata.uid"]
		if uid == "" {
			continue
		}

		if err := waitForProvisioningConnectionDestroy(context.Background(), client, uid); err != nil {
			return err
		}
	}

	return nil
}

func testAccCheckProvisioningConnectionAndRepositoryDestroy(s *terraform.State) error {
	if err := testAccCheckProvisioningRepositoryDestroy(s); err != nil {
		return err
	}

	return testAccCheckProvisioningConnectionDestroy(s)
}

func testAccCheckProvisioningRepositoryDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != "grafana_apps_provisioning_repository_v0alpha1" {
			continue
		}

		uid := r.Primary.Attributes["metadata.uid"]
		if uid == "" {
			continue
		}

		if err := waitForProvisioningRepositoryDestroy(context.Background(), client, uid); err != nil {
			return err
		}
	}

	return nil
}

func waitForProvisioningConnectionDestroy(ctx context.Context, client *common.Client, uid string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := getProvisioningConnection(ctx, client, uid); err == nil {
			lastErr = fmt.Errorf("provisioning connection %s still exists", uid)
			time.Sleep(1 * time.Second)
			continue
		} else if apierrors.IsNotFound(err) {
			return nil
		} else {
			lastErr = fmt.Errorf("error checking provisioning connection %s: %w", uid, err)
			time.Sleep(1 * time.Second)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for provisioning connection %s to be deleted", uid)
	}

	return lastErr
}

func waitForProvisioningRepositoryDestroy(ctx context.Context, client *common.Client, uid string) error {
	deadline := time.Now().Add(30 * time.Second)
	var lastErr error

	for time.Now().Before(deadline) {
		if _, err := getProvisioningRepository(ctx, client, uid); err == nil {
			lastErr = fmt.Errorf("provisioning repository %s still exists", uid)
			time.Sleep(1 * time.Second)
			continue
		} else if apierrors.IsNotFound(err) {
			return nil
		} else {
			lastErr = fmt.Errorf("error checking provisioning repository %s: %w", uid, err)
			time.Sleep(1 * time.Second)
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("timed out waiting for provisioning repository %s to be deleted", uid)
	}

	return lastErr
}

func getProvisioningConnection(ctx context.Context, client *common.Client, uid string) (*appplatform.ProvisioningConnection, error) {
	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.ConnectionKind())
	if err != nil {
		return nil, fmt.Errorf("failed to create provisioning connection client: %w", err)
	}

	ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	namespacedClient := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*appplatform.ProvisioningConnection, *appplatform.ProvisioningConnectionList](rcli, appplatform.ConnectionKind()),
		ns,
	)

	return namespacedClient.Get(ctx, uid)
}

func getProvisioningRepository(ctx context.Context, client *common.Client, uid string) (*appplatform.ProvisioningRepository, error) {
	rcli, err := client.GrafanaAppPlatformAPI.ClientFor(appplatform.RepositoryKind())
	if err != nil {
		return nil, fmt.Errorf("failed to create provisioning repository client: %w", err)
	}

	ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
	namespacedClient := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*appplatform.ProvisioningRepository, *appplatform.ProvisioningRepositoryList](rcli, appplatform.RepositoryKind()),
		ns,
	)

	return namespacedClient.Get(ctx, uid)
}

func stateResourceUID(s *terraform.State, resourceName string) (string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource not found in state: %s", resourceName)
	}

	uid := rs.Primary.Attributes["metadata.uid"]
	if uid == "" {
		return "", fmt.Errorf("metadata.uid is empty for resource %s", resourceName)
	}

	return uid, nil
}

func importStateUIDFunc(resourceName string) terraformresource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		return stateResourceUID(s, resourceName)
	}
}

func testAccCaptureStateAttribute(resourceName, attribute string, out *string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		value, err := stateResourceAttribute(s, resourceName, attribute)
		if err != nil {
			return err
		}
		*out = value
		return nil
	}
}

func testAccCheckStateAttributeEquals(resourceName, attribute string, expected *string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		if expected == nil || *expected == "" {
			return fmt.Errorf("missing expected state value for %s", attribute)
		}

		value, err := stateResourceAttribute(s, resourceName, attribute)
		if err != nil {
			return err
		}
		if value != *expected {
			return fmt.Errorf("expected %s of %s to remain %q, got %q", attribute, resourceName, *expected, value)
		}
		return nil
	}
}

func waitForProvisioningAPI(t *testing.T) {
	t.Helper()

	baseURL := strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")
	if baseURL == "" {
		t.Fatal("GRAFANA_URL must be set")
	}

	reqURL := baseURL + "/apis/provisioning.grafana.app/v0alpha1/namespaces/" + claims.OrgNamespaceFormatter(grafanaOrgID(t)) + "/repositories"
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(2 * time.Minute)
	start := time.Now()
	nextLog := 10 * time.Second
	lastResult := "no response yet"

	for time.Now().Before(deadline) {
		req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, reqURL, nil)
		if err != nil {
			t.Fatalf("failed to create provisioning readiness request: %v", err)
		}

		setGrafanaAuth(req)

		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastResult = fmt.Sprintf("status %d", resp.StatusCode)
		} else {
			lastResult = err.Error()
		}

		if elapsed := time.Since(start); elapsed >= nextLog {
			t.Logf("waiting for provisioning API at %s (%s elapsed, last result: %s)", reqURL, elapsed.Round(time.Second), lastResult)
			nextLog += 10 * time.Second
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("timed out waiting for provisioning API at %s (last result: %s)", reqURL, lastResult)
}

func grafanaOrgID(t *testing.T) int64 {
	t.Helper()

	orgIDStr := strings.TrimSpace(os.Getenv("GRAFANA_ORG_ID"))
	if orgIDStr == "" {
		return 1
	}

	orgID, err := strconv.ParseInt(orgIDStr, 10, 64)
	if err != nil {
		t.Fatalf("failed to parse GRAFANA_ORG_ID %q: %v", orgIDStr, err)
	}

	return orgID
}

func stateResourceAttribute(s *terraform.State, resourceName, attribute string) (string, error) {
	rs, ok := s.RootModule().Resources[resourceName]
	if !ok {
		return "", fmt.Errorf("resource not found in state: %s", resourceName)
	}

	value, ok := rs.Primary.Attributes[attribute]
	if !ok {
		return "", fmt.Errorf("attribute %s not found for resource %s", attribute, resourceName)
	}

	return value, nil
}

func setGrafanaAuth(req *http.Request) {
	auth := os.Getenv("GRAFANA_AUTH")
	if auth == "" {
		return
	}

	if username, password, ok := strings.Cut(auth, ":"); ok {
		req.SetBasicAuth(username, password)
		return
	}

	req.Header.Set("Authorization", "Bearer "+auth)
}

func provisioningFixturePath(t *testing.T, name string) string {
	t.Helper()

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not resolve current file path")
	}

	return filepath.Join(
		filepath.Dir(currentFile),
		"..",
		"..",
		"..",
		"testdata",
		"appplatform",
		"provisioning",
		name,
	)
}
