package appplatform_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

var (
	secureValueHashMu   sync.Mutex
	secureValueHashByID = map[string]string{}
)

func testAccCheckSecureValueValueHashChanged(resourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return fmt.Errorf("resource %q not found in state", resourceName)
		}

		hash := rs.Primary.Attributes["spec.value_hash"]
		if hash == "" {
			return fmt.Errorf("resource %q has no spec.value_hash", resourceName)
		}

		secureValueHashMu.Lock()
		prev, ok := secureValueHashByID[resourceName]
		if ok && prev == hash {
			secureValueHashMu.Unlock()
			return fmt.Errorf("expected spec.value_hash to change, got %q", hash)
		}
		secureValueHashByID[resourceName] = hash
		secureValueHashMu.Unlock()

		return nil
	}
}

func TestAccResourceSecureValue_value(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("creating a secure value with a secret", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-value-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigValue(valueName),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "metadata.uid", valueName),
						resource.TestCheckResourceAttr(resourceName, "spec.description", "Database password"),
						resource.TestCheckResourceAttr(resourceName, "spec.decrypters.#", "2"),
						resource.TestCheckResourceAttrSet(resourceName, "spec.value_hash"),
						resource.TestCheckNoResourceAttr(resourceName, "spec.value"),
					),
				},
				{
					Config:             testAccSecureValueConfigValue(valueName),
					PlanOnly:           true,
					ExpectNonEmptyPlan: false,
				},
				{
					ResourceName:      resourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"spec.value",
						"spec.value_hash",
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(resourceName),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_ref(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("creating a secure value with a ref", func(t *testing.T) {
		keeperName := fmt.Sprintf("tf-keeper-ref-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		valueName := fmt.Sprintf("tf-ref-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigRef(keeperName, valueName, "path/to/existing/secret", "External API key", []string{"grafana"}),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "metadata.uid", valueName),
						resource.TestCheckResourceAttr(resourceName, "spec.ref", "path/to/existing/secret"),
						resource.TestCheckResourceAttr(resourceName, "spec.decrypters.#", "1"),
					),
				},
				{
					Config: testAccSecureValueConfigRef(keeperName, valueName, "path/to/another/secret", "External API key", []string{"grafana"}),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "spec.ref", "path/to/another/secret"),
					),
				},
				{
					Config: testAccSecureValueConfigRef(keeperName, valueName, "path/to/another/secret", "Updated API key", []string{"grafana", "k6"}),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "spec.description", "Updated API key"),
						resource.TestCheckResourceAttr(resourceName, "spec.decrypters.#", "2"),
					),
				},
				{
					Config:             testAccSecureValueConfigRef(keeperName, valueName, "path/to/another/secret", "Updated API key", []string{"grafana", "k6"}),
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
					ImportStateIdFunc: importStateIDFunc(resourceName),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_validation(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("secure value validation", func(t *testing.T) {
		longDescription := strings.Repeat("a", 26)
		longValue := strings.Repeat("v", 24577)
		longRef := strings.Repeat("r", 1025)
		tooManyDecrypters := testAccSecureValueDecryptersList(65)

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccSecureValueConfigInvalid(),
					ExpectError: regexp.MustCompile("(?i)one.*of"),
				},
				{
					Config:      testAccSecureValueConfigWithValueAndDescription("tf-invalid-desc", "value", longDescription, []string{"grafana"}),
					ExpectError: regexp.MustCompile("(?i)between 1 and 25|length"),
				},
				{
					Config:      testAccSecureValueConfigWithValueAndDescription("tf-invalid-value", longValue, "desc", []string{"grafana"}),
					ExpectError: regexp.MustCompile("(?i)between 1 and 24576|length"),
				},
				{
					Config:      testAccSecureValueConfigWithRef("tf-invalid-ref", longRef, "desc", []string{"grafana"}),
					ExpectError: regexp.MustCompile("(?i)between 1 and 1024|length"),
				},
				{
					Config:      testAccSecureValueConfigWithValueAndDescription("tf-invalid-decrypters", "value", "desc", tooManyDecrypters),
					ExpectError: regexp.MustCompile("(?i)at most 64"),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_refRequiresNonSystemKeeper(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("creating a secure value that references a secret on 3rd party secret store requires a keeper to be active that's not the system keeper", func(t *testing.T) {
		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccSecureValueConfigRefWithSystemActive("tf-ref-system"),
					ExpectError: regexp.MustCompile("(?i)system keeper|reference"),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_updateDescriptionDecrypters(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("updating description of a secure value", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-update-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigWithValueAndDescription(valueName, "change-me", "Initial description", []string{"grafana", "k6"}),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "spec.description", "Initial description"),
						resource.TestCheckResourceAttr(resourceName, "spec.decrypters.#", "2"),
					),
				},
				{
					Config: testAccSecureValueConfigWithValueAndDescription(valueName, "change-me", "Updated description", []string{"grafana"}),
					Check: resource.ComposeTestCheckFunc(
						resource.TestCheckResourceAttr(resourceName, "spec.description", "Updated description"),
						resource.TestCheckResourceAttr(resourceName, "spec.decrypters.#", "1"),
					),
				},
				{
					Config:             testAccSecureValueConfigWithValueAndDescription(valueName, "change-me", "Updated description", []string{"grafana"}),
					PlanOnly:           true,
					ExpectNonEmptyPlan: false,
				},
			},
		})
	})
}

func TestAccResourceSecureValue_updateValueRotatesHash(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("updating value rotates hash", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-rotate-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigWithValueAndDescription(valueName, "value-1", "Rotate value", []string{"grafana"}),
					Check:  resource.ComposeTestCheckFunc(resource.TestCheckResourceAttrSet(resourceName, "spec.value_hash"), testAccCheckSecureValueValueHashChanged(resourceName)),
				},
				{
					Config: testAccSecureValueConfigWithValueAndDescription(valueName, "value-2", "Rotate value", []string{"grafana"}),
					Check:  resource.ComposeTestCheckFunc(resource.TestCheckResourceAttrSet(resourceName, "spec.value_hash"), testAccCheckSecureValueValueHashChanged(resourceName)),
				},
				{
					Config:             testAccSecureValueConfigWithValueAndDescription(valueName, "value-2", "Rotate value", []string{"grafana"}),
					PlanOnly:           true,
					ExpectNonEmptyPlan: false,
				},
			},
		})
	})
}

func TestAccResourceSecureValue_decryptersOrderNoDiff(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("decrypters order is preserved", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-order-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigWithValueAndDescription(valueName, "change-me", "Order test", []string{"grafana", "k6"}),
				},
				{
					Config:             testAccSecureValueConfigWithValueAndDescription(valueName, "change-me", "Order test", []string{"k6", "grafana"}),
					PlanOnly:           true,
					ExpectNonEmptyPlan: true,
				},
			},
		})
	})
}

func TestAccResourceSecureValue_decryptersUnique(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("decrypters cannot have duplicated values", func(t *testing.T) {
		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []resource.TestStep{
				{
					Config:      testAccSecureValueConfigWithValueAndDescription("tf-decrypters-unique", "value", "desc", []string{"grafana", "grafana"}),
					ExpectError: regexp.MustCompile("(?i)Duplicate List Value"),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_delete(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("deleting a secure value", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-delete-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		config := testAccSecureValueConfigValue(valueName)

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccSecureValueCheckDestroy,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check:  resource.TestCheckResourceAttrSet(resourceName, "id"),
				},
				{
					ResourceName: resourceName,
					Destroy:      true,
					Config:       config,
				},
			},
		})
	})
}

func TestAccResourceSecureValue_deleteIdempotent(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("deleting a secure value twice is safe", func(t *testing.T) {
		valueName := fmt.Sprintf("tf-delete-idem-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccSecureValueCheckDestroyIdempotent,
			Steps: []resource.TestStep{
				{
					Config: testAccSecureValueConfigValue(valueName),
				},
			},
		})
	})
}

func TestAccResourceSecureValue_deleteRef(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t)
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")

	t.Run("deleting a secure value with a ref", func(t *testing.T) {
		keeperName := fmt.Sprintf("tf-keeper-delete-ref-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		valueName := fmt.Sprintf("tf-delete-ref-%s", acctest.RandStringFromCharSet(6, acctest.CharSetAlphaNum))
		const resourceName = "grafana_apps_secret_securevalue_v1beta1.test"

		config := testAccSecureValueConfigRef(keeperName, valueName, "path/to/existing/secret", "External API key", []string{"grafana"})

		resource.ParallelTest(t, resource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			CheckDestroy:             testAccSecureValueCheckDestroy,
			Steps: []resource.TestStep{
				{
					Config: config,
					Check:  resource.TestCheckResourceAttrSet(resourceName, "id"),
				},
				{
					ResourceName: resourceName,
					Destroy:      true,
					Config:       config,
				},
			},
		})
	})
}

func testAccSecureValueConfigValue(valueName string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "Database password"
    value       = "change-me"
    decrypters  = ["grafana", "k6"]
  }
}
`, valueName)
}

func testAccSecureValueConfigRef(keeperName, valueName, ref, description string, decrypters []string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_keeper_v1beta1" "test" {
  metadata {
    uid = "%s"
  }
  spec {
    description = "Keeper for secure value ref test"
    aws {
      region = "us-east-1"
      assume_role {
        assume_role_arn = "arn:aws:iam::123456789012:role/GrafanaSecretsAccess"
        external_id     = "grafana-unique-external-id"
      }
    }
  }
}

resource "grafana_apps_secret_keeper_activation_v1beta1" "test" {
  metadata {
    uid = grafana_apps_secret_keeper_v1beta1.test.metadata.uid
  }
}

resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = "%s"
  }
  spec {
    description = %q
    ref         = %q
    decrypters  = [%s]
  }
  depends_on = [grafana_apps_secret_keeper_activation_v1beta1.test]
}
`, keeperName, valueName, description, ref, testAccSecureValueDecrypters(decrypters))
}

func testAccSecureValueConfigInvalid() string {
	return `
resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = "invalid-secure-value"
  }
  spec {
    value = "one"
    ref   = "two"
  }
}
`
}

func testAccSecureValueConfigRefWithSystemActive(name string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = "External API key"
    ref         = "path/to/existing/secret"
    decrypters  = ["grafana"]
  }
}
`, name)
}

func testAccSecureValueConfigWithValueAndDescription(name, value, description string, decrypters []string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = %q
    value       = %q
    decrypters  = [%s]
  }
}
`, name, description, value, testAccSecureValueDecrypters(decrypters))
}

func testAccSecureValueConfigWithRef(name, ref, description string, decrypters []string) string {
	return fmt.Sprintf(`
resource "grafana_apps_secret_securevalue_v1beta1" "test" {
  metadata {
    uid = %q
  }
  spec {
    description = %q
    ref         = %q
    decrypters  = [%s]
  }
}
`, name, description, ref, testAccSecureValueDecrypters(decrypters))
}

func testAccSecureValueDecrypters(values []string) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, fmt.Sprintf("%q", value))
	}
	return strings.Join(parts, ", ")
}

func testAccSecureValueDecryptersList(count int) []string {
	values := make([]string, 0, count)
	for i := range count {
		values = append(values, fmt.Sprintf("decrypt-%02d", i))
	}
	return values
}

func testAccSecureValueCheckDestroy(s *terraform.State) error {
	commonClient := testutils.Provider.Meta().(*common.Client)
	secureValuesClient, _, err := testAccSecureValueClient(commonClient)
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_apps_secret_securevalue_v1beta1" {
			continue
		}

		if rs.Primary.ID == "" {
			continue
		}

		name := rs.Primary.Attributes["metadata.uid"]
		if name == "" {
			return fmt.Errorf("secure value %q has no metadata.uid", rs.Primary.ID)
		}

		_, err := secureValuesClient.Get(context.Background(), name)
		if err == nil {
			return fmt.Errorf("secure value %q still exists", rs.Primary.ID)
		}
		if !testAccIsNotFound(err) {
			return err
		}
	}

	return nil
}

func testAccSecureValueCheckDestroyIdempotent(s *terraform.State) error {
	commonClient := testutils.Provider.Meta().(*common.Client)
	secureValuesClient, _, err := testAccSecureValueClient(commonClient)
	if err != nil {
		return err
	}

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_apps_secret_securevalue_v1beta1" {
			continue
		}

		name := rs.Primary.Attributes["metadata.uid"]
		if name == "" {
			continue
		}

		_, err := secureValuesClient.Get(context.Background(), name)
		if err == nil {
			return fmt.Errorf("secure value %q still exists", rs.Primary.ID)
		}
		if !testAccIsNotFound(err) {
			return err
		}
	}

	return nil
}
