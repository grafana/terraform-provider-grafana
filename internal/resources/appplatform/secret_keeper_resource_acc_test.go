package appplatform_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/authlib/claims"
	sdkresource "github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/secret/pkg/apis/secret/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

func TestAccResourceKeeper_basic(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	name := fmt.Sprintf("tf-keeper-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	const resourceName = "grafana_apps_secret_keeper_v1beta1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperConfig(name, "Initial description"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.uid", name),
					resource.TestCheckResourceAttr(resourceName, "spec.description", "Initial description"),
					resource.TestCheckResourceAttr(resourceName, "spec.aws.region", "us-east-1"),
					resource.TestCheckResourceAttr(resourceName, "spec.aws.assume_role.assume_role_arn", "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"),
					resource.TestCheckResourceAttr(resourceName, "spec.aws.assume_role.external_id", "grafana-unique-external-id"),
				),
			},
			{
				Config:             testAccKeeperConfig(name, "Initial description"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				Config: testAccKeeperConfig(name, "Updated description"),
				Check:  resource.TestCheckResourceAttr(resourceName, "spec.description", "Updated description"),
			},
			{
				Config:             testAccKeeperConfig(name, "Updated description"),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: func(s *terraform.State) (string, error) {
					rs, ok := s.RootModule().Resources[resourceName]
					if !ok {
						return "", fmt.Errorf("resource %q not found in state", resourceName)
					}
					uid := rs.Primary.Attributes["metadata.uid"]
					if uid == "" {
						return "", fmt.Errorf("resource %q has no metadata.uid", resourceName)
					}
					return uid, nil
				},
			},
		},
	})
}

func TestAccResourceKeeper_deleteIdempotent(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	name := fmt.Sprintf("tf-keeper-delete-idem-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories:  testutils.ProtoV5ProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccKeeperCheckDestroyIdempotent,
		ErrorCheck:                testAccIgnoreNotFound,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperConfig(name, "Delete idempotent test"),
			},
		},
	})
}

func TestAccResourceKeeper_validation(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	longDescription := strings.Repeat("a", 254)

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      testAccKeeperConfigWithName("Invalid_Name", "Valid description"),
				ExpectError: regexp.MustCompile("(?i)lower case|dns-1123|subdomain"),
			},
			{
				Config:      testAccKeeperConfigWithName("valid-name", longDescription),
				ExpectError: regexp.MustCompile("(?i)between 1 and 253|length"),
			},
			{
				Config:      testAccKeeperConfigMissingAWSRegion("missing-region"),
				ExpectError: regexp.MustCompile("(?i)region|missing required attribute|required"),
			},
			{
				Config:      testAccKeeperConfigMissingAssumeRoleARN("missing-assume-role-arn"),
				ExpectError: regexp.MustCompile("(?i)assume_role_arn"),
			},
			{
				Config:      testAccKeeperConfigMissingAssumeRoleExternalID("missing-external-id"),
				ExpectError: regexp.MustCompile("(?i)external_id"),
			},
		},
	})
}

func TestAccResourceKeeper_delete(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	name := fmt.Sprintf("tf-keeper-delete-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories:  testutils.ProtoV5ProviderFactories,
		PreventPostDestroyRefresh: true,
		CheckDestroy:              testAccKeeperCheckDestroy,
		ErrorCheck:                testAccIgnoreNotFound,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperConfig(name, "Delete test"),
			},
		},
	})
}

func TestAccResourceKeeperActivation_lastWriteWins(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	keeperA := fmt.Sprintf("tf-keeper-a-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	keeperB := fmt.Sprintf("tf-keeper-b-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	valueName := fmt.Sprintf("tf-ref-activation-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperActivationConfigLastWriteWins(keeperA, keeperB, valueName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.uid", valueName),
					testAccCheckSecureValueKeeper(resourceName, keeperB),
				),
			},
		},
	})
}

func TestAccResourceKeeperActivation_deleteSetsSystem(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	keeperName := fmt.Sprintf("tf-keeper-delete-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
	valueName := fmt.Sprintf("tf-value-system-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			// Create keeper + activation.
			{
				Config: testAccKeeperActivationConfig(keeperName),
			},
			// Remove activation by switching to keeper-only config.
			{
				Config: testAccKeeperConfig(keeperName, "Keeper for activation delete test"),
				Check:  testAccCheckResourceGone("grafana_apps_secret_keeper_activation_v1beta1.test"),
			},
			// Ensure secure values are created with the system keeper.
			{
				Config: testAccSecureValueConfigValue(valueName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(resourceName, "metadata.uid", valueName),
					testAccCheckSecureValueKeeper(resourceName, appplatform.SystemKeeperName),
				),
			},
		},
	})
}

func TestAccResourceKeeperActivation_updateIdempotent(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	keeperName := fmt.Sprintf("tf-keeper-activate-idem-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperActivationConfig(keeperName),
			},
			{
				Config:             testAccKeeperActivationConfig(keeperName),
				PlanOnly:           true,
				ExpectNonEmptyPlan: false,
			},
		},
	})
}

func TestAccResourceKeeperActivation_deleteIdempotent(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	keeperName := fmt.Sprintf("tf-keeper-delete-idem-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccKeeperActivationDeleteIdempotent,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperActivationConfig(keeperName),
			},
		},
	})
}

func TestAccResourceKeeperActivation_import(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	keeperName := fmt.Sprintf("tf-keeper-import-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

	const resourceName = "grafana_apps_secret_keeper_activation_v1beta1.test"

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccKeeperActivationConfig(keeperName),
			},
			{
				ResourceName:      resourceName,
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func testAccKeeperConfig(name, description string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "%s"
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}
`, name, description)
}

func testAccKeeperConfigWithName(name, description string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = %q
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}
`, name, description)
}

func testAccKeeperActivationConfigLastWriteWins(keeperA, keeperB, valueName string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "a" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "Keeper A for activation test"
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}

resource "grafana_apps_secret_keeper_v1beta1" "b" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "Keeper B for activation test"
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}

resource "grafana_apps_secret_keeper_activation_v1beta1" "a" {
  metadata {
    uid = grafana_apps_secret_keeper_v1beta1.a.metadata.uid
  }
}

resource "grafana_apps_secret_keeper_activation_v1beta1" "b" {
  metadata {
    uid = grafana_apps_secret_keeper_v1beta1.b.metadata.uid
  }
  depends_on = [grafana_apps_secret_keeper_activation_v1beta1.a]
}

resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "External API key"
    ref         = "path/to/existing/secret"
    decrypters  = ["grafana"]
  }
  depends_on = [grafana_apps_secret_keeper_activation_v1beta1.b]
}
`, keeperA, keeperB, valueName)
}

func testAccKeeperActivationConfig(keeperName string) string {
	return fmt.Sprintf(`
%s

resource "grafana_apps_secret_keeper_activation_v1beta1" "test" {
  metadata {
    uid = grafana_apps_secret_keeper_v1beta1.test.metadata.uid
  }
}
`, testAccKeeperConfig(keeperName, "Keeper for activation delete test"))
}

func testAccKeeperConfigMissingAWSRegion(name string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = "Missing region"
    aws {
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}
`, name)
}

func testAccKeeperConfigMissingAssumeRoleARN(name string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = "Missing assume_role_arn"
    aws {
      region = "us-east-1"
      assume_role {
        external_id = "grafana-unique-external-id"
      }
    }
  }
}
`, name)
}

func testAccKeeperConfigMissingAssumeRoleExternalID(name string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = "Missing external_id"
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
      }
    }
  }
}
`, name)
}

func testAccCheckSecureValueKeeper(resourceName, expectedKeeper string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource %q has no ID set", resourceName)
		}

		commonClient := testutils.Provider.Meta().(*common.Client)
		secureValuesClient, namespace, err := testAccSecureValueClient(commonClient)
		if err != nil {
			return err
		}

		name := rs.Primary.Attributes["metadata.uid"]
		if name == "" {
			return fmt.Errorf("secure value %q has no metadata.uid", rs.Primary.ID)
		}
		secureValue, err := secureValuesClient.Get(context.Background(), name)
		if err != nil {
			return err
		}

		if secureValue.Status.Keeper != expectedKeeper {
			return fmt.Errorf("expected active keeper %q, got %q (namespace %q)", expectedKeeper, secureValue.Status.Keeper, namespace)
		}

		return nil
	}
}

func testAccCheckResourceGone(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if _, ok := s.RootModule().Resources[resourceName]; ok {
			return fmt.Errorf("resource %q still present in state", resourceName)
		}
		return nil
	}
}

func testAccKeeperActivationDeleteIdempotent(_ *terraform.State) error {
	commonClient := testutils.Provider.Meta().(*common.Client)
	keepersClient, _, err := testAccKeeperClient(commonClient)
	if err != nil {
		return err
	}

	body := io.NopCloser(strings.NewReader("{}"))
	opts := sdkresource.CustomRouteRequestOptions{
		Path: "activate",
		Verb: "POST",
		Body: body,
	}

	if _, err := keepersClient.SubresourceRequest(context.Background(), appplatform.SystemKeeperName, opts); err != nil {
		return fmt.Errorf("failed to activate system keeper (first): %w", err)
	}

	body = io.NopCloser(strings.NewReader("{}"))
	opts.Body = body
	if _, err := keepersClient.SubresourceRequest(context.Background(), appplatform.SystemKeeperName, opts); err != nil {
		return fmt.Errorf("failed to activate system keeper (second): %w", err)
	}

	return nil
}

func testAccKeeperCheckDestroy(s *terraform.State) error {
	commonClient := testutils.Provider.Meta().(*common.Client)
	keepersClient, _, err := testAccKeeperClient(commonClient)
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_apps_secret_keeper_v1beta1" {
			continue
		}

		if rs.Primary.ID == "" {
			continue
		}

		name := rs.Primary.Attributes["metadata.uid"]
		if name == "" {
			return fmt.Errorf("keeper %q has no metadata.uid", rs.Primary.ID)
		}
		_, err := keepersClient.Get(context.Background(), name)
		if err == nil {
			return fmt.Errorf("keeper %q still exists", rs.Primary.ID)
		}
		if !testAccIsNotFound(err) {
			return err
		}
	}

	return nil
}

func testAccKeeperCheckDestroyIdempotent(s *terraform.State) error {
	commonClient := testutils.Provider.Meta().(*common.Client)
	keepersClient, _, err := testAccKeeperClient(commonClient)
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_apps_secret_keeper_v1beta1" {
			continue
		}

		name := rs.Primary.Attributes["metadata.uid"]
		if name == "" {
			continue
		}

		_, err := keepersClient.Get(context.Background(), name)
		if err == nil {
			return fmt.Errorf("keeper %q still exists", rs.Primary.ID)
		}
		if !testAccIsNotFound(err) {
			return err
		}
	}

	return nil
}

func testAccSecureValueClient(commonClient *common.Client) (*sdkresource.NamespacedClient[*v1beta1.SecureValue, *v1beta1.SecureValueList], string, error) {
	namespace, err := testAccNamespace(commonClient)
	if err != nil {
		return nil, "", err
	}

	rcli, err := commonClient.GrafanaAppPlatformAPI.ClientFor(v1beta1.SecureValueKind())
	if err != nil {
		return nil, "", err
	}

	client := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*v1beta1.SecureValue, *v1beta1.SecureValueList](rcli, v1beta1.SecureValueKind()),
		namespace,
	)

	return client, namespace, nil
}

func testAccKeeperClient(commonClient *common.Client) (*sdkresource.NamespacedClient[*v1beta1.Keeper, *v1beta1.KeeperList], string, error) {
	namespace, err := testAccNamespace(commonClient)
	if err != nil {
		return nil, "", err
	}

	rcli, err := commonClient.GrafanaAppPlatformAPI.ClientFor(v1beta1.KeeperKind())
	if err != nil {
		return nil, "", err
	}

	client := sdkresource.NewNamespaced(
		sdkresource.NewTypedClient[*v1beta1.Keeper, *v1beta1.KeeperList](rcli, v1beta1.KeeperKind()),
		namespace,
	)

	return client, namespace, nil
}

func testAccNamespace(commonClient *common.Client) (string, error) {
	switch {
	case commonClient.GrafanaStackID > 0:
		return claims.CloudNamespaceFormatter(commonClient.GrafanaStackID), nil
	case commonClient.GrafanaOrgID > 0:
		return claims.OrgNamespaceFormatter(commonClient.GrafanaOrgID), nil
	default:
		return "", fmt.Errorf("missing Grafana org or stack ID")
	}
}

func testAccIgnoreNotFound(err error) error {
	if err == nil {
		return nil
	}
	if testAccIsNotFound(err) {
		return nil
	}
	return err
}

func testAccIsNotFound(err error) bool {
	if err == nil {
		return false
	}
	if apierrors.IsNotFound(err) {
		return true
	}

	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "not found") || strings.Contains(msg, "status: 404")
}
