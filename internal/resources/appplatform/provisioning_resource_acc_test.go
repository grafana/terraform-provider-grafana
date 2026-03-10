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

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningConnectionDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningConnectionConfig(uid, keyV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.type", "github"),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "spec.github.app_id", "12345"),
					terraformresource.TestCheckResourceAttr(provisioningConnectionResourceName, "secure_version", "1"),
					testAccProvisioningConnectionExists(provisioningConnectionResourceName, func(conn *appplatform.ProvisioningConnection) error {
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
						if conn.Secure.ClientSecret.Name != "" {
							return fmt.Errorf("expected client secret to remain unset for github connections")
						}
						return nil
					}),
				),
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
				Config: testAccProvisioningConnectionConfig(uid, keyV1, 1),
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
				Config: testAccProvisioningConnectionConfig(uid, keyV2, 2),
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

	terraformresource.Test(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckProvisioningRepositoryDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccProvisioningRepositoryConfig(uid, token, webhookSecret, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.type", "github"),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "spec.github.branch", "main"),
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryExists(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
						if repo.Spec.Type != provisioningv0alpha1.GitHubRepositoryType {
							return fmt.Errorf("expected github repository type, got %q", repo.Spec.Type)
						}
						if repo.Spec.GitHub == nil {
							return fmt.Errorf("expected github repository config to be present")
						}
						if repo.Spec.GitHub.Path != "examples" {
							return fmt.Errorf("expected path examples, got %q", repo.Spec.GitHub.Path)
						}
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
				Config: testAccProvisioningRepositoryConfig(uid, tokenV1, webhookSecretV1, 1),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "1"),
					testAccProvisioningRepositoryExists(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
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
				Config: testAccProvisioningRepositoryConfig(uid, tokenV2, webhookSecretV2, 2),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(provisioningRepositoryResourceName, "secure_version", "2"),
					testAccProvisioningRepositoryExists(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
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
					testAccProvisioningRepositoryExists(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
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
					testAccProvisioningRepositoryExists(provisioningRepositoryResourceName, func(repo *appplatform.ProvisioningRepository) error {
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

func testAccProvisioningConnectionConfig(uid, privateKeyPath string, secureVersion int) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_connection_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "Acceptance GitHub App connection"
    description = "Acceptance test connection"
    type        = "github"
    url         = "https://github.com"

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

  secure_version = %d
}
`, uid, privateKeyPath, secureVersion)
}

func testAccProvisioningRepositoryConfig(uid, token, webhookSecret string, secureVersion int) string {
	return fmt.Sprintf(`
resource "grafana_apps_provisioning_repository_v0alpha1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title       = "Acceptance Git Sync repository"
    description = "Acceptance test repository"
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
      path                        = "examples"
      generate_dashboard_previews = false
    }
  }

  secure {
    token = {
      create = "%s"
    }
    webhook_secret = {
      create = "%s"
    }
  }

  secure_version = %d
}
`, uid, token, webhookSecret, secureVersion)
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

func testAccProvisioningConnectionExists(resourceName string, check func(*appplatform.ProvisioningConnection) error) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		uid, err := stateResourceUID(s, resourceName)
		if err != nil {
			return err
		}

		client := testutils.Provider.Meta().(*common.Client)
		conn, err := getProvisioningConnection(context.Background(), client, uid)
		if err != nil {
			return err
		}

		if check != nil {
			return check(conn)
		}

		return nil
	}
}

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

		if lastErr != nil {
			return lastErr
		}

		return fmt.Errorf("timed out waiting for provisioning connection %s", uid)
	}
}

func testAccProvisioningRepositoryExists(resourceName string, check func(*appplatform.ProvisioningRepository) error) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		uid, err := stateResourceUID(s, resourceName)
		if err != nil {
			return err
		}

		client := testutils.Provider.Meta().(*common.Client)
		repo, err := getProvisioningRepository(context.Background(), client, uid)
		if err != nil {
			return err
		}

		if check != nil {
			return check(repo)
		}

		return nil
	}
}

func testAccProvisioningRepositoryAbsent(uid string) terraformresource.TestCheckFunc {
	return func(_ *terraform.State) error {
		client := testutils.Provider.Meta().(*common.Client)
		deadline := time.Now().Add(30 * time.Second)

		for time.Now().Before(deadline) {
			_, err := getProvisioningRepository(context.Background(), client, uid)
			if apierrors.IsNotFound(err) {
				return nil
			}
			if err != nil {
				return err
			}
			time.Sleep(1 * time.Second)
		}

		return fmt.Errorf("timed out waiting for provisioning repository %s to be deleted", uid)
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

	for time.Now().Before(deadline) {
		if _, err := getProvisioningConnection(ctx, client, uid); err == nil {
			time.Sleep(1 * time.Second)
			continue
		} else if apierrors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("error checking provisioning connection %s: %w", uid, err)
		}
	}

	return fmt.Errorf("provisioning connection %s still exists", uid)
}

func waitForProvisioningRepositoryDestroy(ctx context.Context, client *common.Client, uid string) error {
	deadline := time.Now().Add(30 * time.Second)

	for time.Now().Before(deadline) {
		if _, err := getProvisioningRepository(ctx, client, uid); err == nil {
			time.Sleep(1 * time.Second)
			continue
		} else if apierrors.IsNotFound(err) {
			return nil
		} else {
			return fmt.Errorf("error checking provisioning repository %s: %w", uid, err)
		}
	}

	return fmt.Errorf("provisioning repository %s still exists", uid)
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

func waitForProvisioningAPI(t *testing.T) {
	t.Helper()

	baseURL := strings.TrimRight(os.Getenv("GRAFANA_URL"), "/")
	if baseURL == "" {
		t.Fatal("GRAFANA_URL must be set")
	}

	reqURL := baseURL + "/apis/provisioning.grafana.app/v0alpha1/namespaces/" + claims.OrgNamespaceFormatter(1) + "/repositories"
	client := &http.Client{Timeout: 5 * time.Second}
	deadline := time.Now().Add(2 * time.Minute)

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
		}

		time.Sleep(2 * time.Second)
	}

	t.Fatalf("timed out waiting for provisioning API at %s", reqURL)
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
