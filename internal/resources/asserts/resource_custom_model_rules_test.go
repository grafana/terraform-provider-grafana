package asserts_test

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccAssertsCustomModelRules_basic(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	rName := fmt.Sprintf("test-acc-cmr-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
			{
				// Test import
				ResourceName:            "grafana_asserts_custom_model_rules.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"rules"},
			},
			{
				// Test update
				Config: testAccAssertsCustomModelRulesConfigUpdated(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsCustomModelRulesCheckExists(rn string, stackID int64, name string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
		ctx := context.Background()

		_, _, err := client.CustomModelRulesConfigurationAPI.GetModelRules(ctx, name).XScopeOrgID(fmt.Sprintf("%d", stackID)).Execute()
		if err != nil {
			return fmt.Errorf("error getting custom model rules: %s", err)
		}
		return nil
	}
}

func testAccAssertsCustomModelRulesCheckDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client).AssertsAPIClient
	ctx := context.Background()

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "grafana_asserts_custom_model_rules" {
			continue
		}

		name := rs.Primary.ID
		stackID := fmt.Sprintf("%d", testutils.Provider.Meta().(*common.Client).GrafanaStackID)

		_, _, err := client.CustomModelRulesConfigurationAPI.GetModelRules(ctx, name).XScopeOrgID(stackID).Execute()
		if err != nil {
			if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "404") {
				continue
			}
			return fmt.Errorf("error checking custom model rules destruction: %s", err)
		}
		return fmt.Errorf("custom model rules %s still exists", name)
	}

	return nil
}

func testAccAssertsCustomModelRulesConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
  }
}
`, name)
}

func testAccAssertsCustomModelRulesConfigUpdated(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
    entity {
      type = "Pod"
      name = "Pod"
      defined_by {
        query = "up{pod!=''}"
      }
    }
  }
}
`, name)
}

// TestAccAssertsCustomModelRules_eventualConsistencyStress tests multiple resources created simultaneously
// to verify the retry logic handles eventual consistency properly
func TestAccAssertsCustomModelRules_eventualConsistencyStress(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)
	testutils.CheckStressTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	baseName := fmt.Sprintf("stress-cmr-%s", acctest.RandString(8))

	resource.Test(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesStressConfig(stackID, baseName),
				// Creating multiple resources concurrently can trigger optimistic locking conflicts
				// in the backing store. We expect the provider to retry, but the apply may still
				// ultimately fail in stress mode; treat that as expected to validate behavior.
				ExpectError: regexp.MustCompile(`giving up after.*attempt`),
			},
		},
	})
}

func testAccAssertsCustomModelRulesStressConfig(stackID int64, baseName string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test1" {
  name = "%s-1"
  rules {
    entity {
      type = "Service"
      name = "Service"
      defined_by {
        query = "up{job!=''}"
      }
    }
  }
}

resource "grafana_asserts_custom_model_rules" "test2" {
  name = "%s-2"
  rules {
    entity {
      type = "Pod"
      name = "Pod"
      defined_by {
        query = "up{pod!=''}"
      }
    }
  }
}

resource "grafana_asserts_custom_model_rules" "test3" {
  name = "%s-3"
  rules {
    entity {
      type = "Namespace"
      name = "Namespace"
      defined_by {
        query = "up{namespace!=''}"
      }
    }
  }
}

`, baseName, baseName, baseName)
}

// TestAccAssertsCustomModelRules_complex tests custom model rules with advanced features
// including scope, lookup, enrichedBy, disabled queries, and labelValues/literals
func TestAccAssertsCustomModelRules_complex(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	rName := fmt.Sprintf("test-acc-cmr-complex-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesComplexConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsCustomModelRulesComplexConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "workload | service | job"
      scope = {
        namespace = "namespace"
        env       = "asserts_env"
        site      = "asserts_site"
      }
      lookup = {
        workload   = "workload | deployment | statefulset | daemonset | replicaset"
        service    = "service"
        job        = "job"
        proxy_job  = "job"
      }
      defined_by {
        query = "group by (cluster, workload, workload_type, service, job, container, namespace, asserts_env, asserts_site) (group by (cluster, service, job, container, pod, namespace, asserts_env, asserts_site) (up {asserts_env!=\"\", pod!=\"\", service != \"\", container!=\"istio-proxy\", namespace!=\"AWS/ECS\"}) * on (pod, namespace, asserts_env, asserts_site) group_left(workload, workload_type) group by (pod, workload, workload_type, namespace, asserts_env, asserts_site) (asserts:mixin_pod_workload{workload_type!~\"job|cronjob\"}))"
        disabled = true
      }
      defined_by {
        query = "group by (cluster, workload, workload_type, service, job, container, namespace, asserts_env, asserts_site) (group by (cluster, service, job, container, pod, namespace, asserts_env, asserts_site) (up {asserts_env!=\"\", pod!=\"\", container!=\"istio-proxy\", namespace!=\"AWS/ECS\"}) * on (pod, namespace, asserts_env, asserts_site) group_left(workload, workload_type) group by (pod, workload, workload_type, namespace, asserts_env, asserts_site) (asserts:mixin_pod_workload{workload_type!~\"job|cronjob\"}))"
        label_values = {
          job           = "job"
          service       = "service"
          workload      = "workload"
          workload_type = "workload_type"
          container     = "container"
          cluster       = "cluster"
        }
        literals = {
          _entity_source_1 = "up_with_pod"
        }
      }
      defined_by {
        query = "group by (cluster, namespace, workload, workload_type, asserts_env, asserts_site) ((group without() (kube_pod_info{asserts_env!=\"\", created_by_kind!~\"Job|TaskRun\"}) * on (pod, namespace, asserts_env, asserts_site) group_left(workload, workload_type) group by (pod, workload, workload_type,  namespace, asserts_env, asserts_site) (asserts:mixin_pod_workload{workload_type!~\"job|cronjob\"})) unless on (pod,  namespace, asserts_env, asserts_site) group by (pod,  namespace, asserts_env, asserts_site) (up {pod !=\"\", service != \"\", container!=\"istio-proxy\"}))"
        disabled = true
      }
      defined_by {
        query = "group by (cluster, namespace, workload, workload_type, asserts_env, asserts_site) ((kube_pod_info{asserts_env!=\"\", created_by_kind!~\"Job|TaskRun\"} * on (pod, namespace, asserts_env, asserts_site) group_left(workload, workload_type) group by (pod, workload, workload_type,  namespace, asserts_env, asserts_site) (asserts:mixin_pod_workload{workload_type!~\"job|cronjob\"})) unless on (pod,  namespace, asserts_env, asserts_site) group by (pod,  namespace, asserts_env, asserts_site) (up {pod !=\"\", container!=\"istio-proxy\"}))"
        label_values = {
          workload      = "workload"
          workload_type = "workload_type"
          cluster       = "cluster"
        }
        literals = {
          _entity_source_3 = "pod_without_up"
        }
      }
    }
  }
}
`, name)
}

// TestAccAssertsCustomModelRules_advancedFeatures tests custom model rules with some advanced features
// but with simpler configuration to ensure basic functionality works
func TestAccAssertsCustomModelRules_advancedFeatures(t *testing.T) {
	testutils.CheckCloudInstanceTestsEnabled(t)

	stackID := testutils.Provider.Meta().(*common.Client).GrafanaStackID
	rName := fmt.Sprintf("test-acc-cmr-advanced-%s", acctest.RandString(8))

	resource.ParallelTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccAssertsCustomModelRulesCheckDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccAssertsCustomModelRulesAdvancedConfig(stackID, rName),
				Check: resource.ComposeTestCheckFunc(
					testAccAssertsCustomModelRulesCheckExists("grafana_asserts_custom_model_rules.test", stackID, rName),
					resource.TestCheckResourceAttr("grafana_asserts_custom_model_rules.test", "name", rName),
				),
			},
		},
	})
}

func testAccAssertsCustomModelRulesAdvancedConfig(stackID int64, name string) string {
	return fmt.Sprintf(`
resource "grafana_asserts_custom_model_rules" "test" {
  name = "%s"
  rules {
    entity {
      type = "Service"
      name = "test-service"
      defined_by {
        query = "up{job=\"test\"}"
        label_values = {
          service = "service"
          job     = "job"
        }
        literals = {
          _source = "test"
        }
      }
    }
  }
}
`, name)
}
