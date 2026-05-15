package appplatform_test

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/grafana/authlib/claims"
	"github.com/grafana/grafana-app-sdk/resource"
	"github.com/grafana/grafana/apps/alerting/alertenrichment/pkg/apis/alertenrichment/v1beta1"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	terraformresource "github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const alertEnrichmentResourceName = "grafana_apps_alertenrichment_alertenrichment_v1beta1.test"

// importStateIDFunc returns a function that extracts the UID from metadata for import tests.
// They need an id to fetch the resource, and by default they use ID which is set to UUID in our case,
// but to get the response we need the UID.
func importStateIDFunc(resourceName string) terraformresource.ImportStateIdFunc {
	return func(s *terraform.State) (string, error) {
		rs, ok := s.RootModule().Resources[resourceName]
		if !ok {
			return "", fmt.Errorf("resource not found: %s", resourceName)
		}
		// Use the UID from metadata for import instead of the UUID ID
		uid := rs.Primary.Attributes["metadata.uid"]
		if uid == "" {
			return "", fmt.Errorf("UID is empty in resource %s", resourceName)
		}
		return uid, nil
	}
}

// alertEnrichmentConfigBuilder helps build HCL for alert enrichments
type alertEnrichmentConfigBuilder struct {
	uid                         string
	title                       string
	description                 string
	alertRuleUIDs               []string
	receivers                   []string
	labelMatchers               []matcherConfig
	annotMatchers               []matcherConfig
	assignSteps                 []assignStepConfig
	assistantInvestigationsStep *assistantInvestigationsStepConfig
	externalStep                *externalStepConfig
	assertsStep                 *assertsStepConfig
	explainStep                 *explainStepConfig
	siftStep                    *siftStepConfig
	dataSourceSteps             []dataSourceStepConfig
	conditionalStep             *conditionalStepConfig
	disableProvenance           *bool
}

type matcherConfig struct {
	matchType string
	name      string
	value     string
}

type assignStepConfig struct {
	timeout     string
	annotations map[string]string
}

type assistantInvestigationsStepConfig struct {
	timeout string
}

type externalStepConfig struct {
	timeout string
	url     string
}

type assertsStepConfig struct {
	timeout string
}

type explainStepConfig struct {
	timeout    string
	annotation string
}

type siftStepConfig struct {
	timeout string
}

type dataSourceStepConfig struct {
	timeout   string
	logsQuery *logsQueryConfig
	rawQuery  *rawQueryConfig
}

type logsQueryConfig struct {
	dataSourceType string
	dataSourceUID  string
	expr           string
	maxLines       int
}

type rawQueryConfig struct {
	refID   string
	request string
}

type conditionalStepConfig struct {
	timeout string
}

func newAlertEnrichmentConfig(uid, title string) *alertEnrichmentConfigBuilder {
	return &alertEnrichmentConfigBuilder{
		uid:   uid,
		title: title,
	}
}

func (b *alertEnrichmentConfigBuilder) withDescription(desc string) *alertEnrichmentConfigBuilder {
	b.description = desc
	return b
}

func (b *alertEnrichmentConfigBuilder) withDisableProvenance(disable bool) *alertEnrichmentConfigBuilder {
	b.disableProvenance = &disable
	return b
}

func (b *alertEnrichmentConfigBuilder) withAlertRuleUIDs(uids ...string) *alertEnrichmentConfigBuilder {
	b.alertRuleUIDs = uids
	return b
}

func (b *alertEnrichmentConfigBuilder) withReceivers(receivers ...string) *alertEnrichmentConfigBuilder {
	b.receivers = receivers
	return b
}

func (b *alertEnrichmentConfigBuilder) withLabelMatcher(matchType, name, value string) *alertEnrichmentConfigBuilder {
	b.labelMatchers = append(b.labelMatchers, matcherConfig{matchType, name, value})
	return b
}

func (b *alertEnrichmentConfigBuilder) withAnnotationMatcher(matchType, name, value string) *alertEnrichmentConfigBuilder {
	b.annotMatchers = append(b.annotMatchers, matcherConfig{matchType, name, value})
	return b
}

func (b *alertEnrichmentConfigBuilder) withAssignStep(timeout string, annotations map[string]string) *alertEnrichmentConfigBuilder {
	b.assignSteps = append(b.assignSteps, assignStepConfig{
		timeout:     timeout,
		annotations: annotations,
	})
	return b
}

func (b *alertEnrichmentConfigBuilder) withAssistantInvestigationsStep(timeout string) *alertEnrichmentConfigBuilder {
	b.assistantInvestigationsStep = &assistantInvestigationsStepConfig{
		timeout: timeout,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) withExternalStep(timeout string, url string) *alertEnrichmentConfigBuilder {
	b.externalStep = &externalStepConfig{
		timeout: timeout,
		url:     url,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) withAssertsStep(timeout string) *alertEnrichmentConfigBuilder {
	b.assertsStep = &assertsStepConfig{
		timeout: timeout,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) withExplainStep(timeout string, annotation string) *alertEnrichmentConfigBuilder {
	b.explainStep = &explainStepConfig{
		timeout:    timeout,
		annotation: annotation,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) withSiftStep(timeout string) *alertEnrichmentConfigBuilder {
	b.siftStep = &siftStepConfig{
		timeout: timeout,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) withDataSourceLogsStep(timeout, dataSourceType, dataSourceUID, expr string, maxLines int) *alertEnrichmentConfigBuilder {
	b.dataSourceSteps = append(b.dataSourceSteps, dataSourceStepConfig{
		timeout: timeout,
		logsQuery: &logsQueryConfig{
			dataSourceType: dataSourceType,
			dataSourceUID:  dataSourceUID,
			expr:           expr,
			maxLines:       maxLines,
		},
	})
	return b
}

func (b *alertEnrichmentConfigBuilder) withDataSourceRawStep(timeout, refID, request string) *alertEnrichmentConfigBuilder {
	b.dataSourceSteps = append(b.dataSourceSteps, dataSourceStepConfig{
		timeout: timeout,
		rawQuery: &rawQueryConfig{
			refID:   refID,
			request: request,
		},
	})
	return b
}

func (b *alertEnrichmentConfigBuilder) withConditionalBasic(timeout string) *alertEnrichmentConfigBuilder {
	b.conditionalStep = &conditionalStepConfig{
		timeout: timeout,
	}
	return b
}

func (b *alertEnrichmentConfigBuilder) build() string {
	config := b.buildHeader()
	config += b.buildArrayFields()
	config += b.buildMatchers()
	config += b.buildSteps()
	config += b.buildFooter()
	return config
}

func (b *alertEnrichmentConfigBuilder) buildHeader() string {
	config := fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
	metadata {
		uid = "%s"
	}

	spec {
		title = "%s"
		description = "%s"`, b.uid, b.title, b.description)

	if b.disableProvenance != nil {
		config += fmt.Sprintf(`
		disable_provenance = %t`, *b.disableProvenance)
	}

	return config
}

func (b *alertEnrichmentConfigBuilder) buildArrayFields() string {
	var config string

	if len(b.alertRuleUIDs) > 0 {
		config += `
		alert_rule_uids = [`
		for i, uid := range b.alertRuleUIDs {
			if i > 0 {
				config += ", "
			}
			config += fmt.Sprintf(`"%s"`, uid)
		}
		config += "]"
	}

	if len(b.receivers) > 0 {
		config += `
		receivers = [`
		for i, receiver := range b.receivers {
			if i > 0 {
				config += ", "
			}
			config += fmt.Sprintf(`"%s"`, receiver)
		}
		config += "]"
	}

	return config
}

func (b *alertEnrichmentConfigBuilder) buildMatchers() string {
	var config string

	if len(b.labelMatchers) > 0 {
		config += "\n\t\tlabel_matchers = ["
		for i, m := range b.labelMatchers {
			if i > 0 {
				config += ","
			}
			config += fmt.Sprintf(`{ type = "%s", name = "%s", value = "%s" }`, m.matchType, m.name, m.value)
		}
		config += "]"
	}

	if len(b.annotMatchers) > 0 {
		config += "\n\t\tannotation_matchers = ["
		for i, m := range b.annotMatchers {
			if i > 0 {
				config += ","
			}
			config += fmt.Sprintf(`{ type = "%s", name = "%s", value = "%s" }`, m.matchType, m.name, m.value)
		}
		config += "]"
	}

	return config
}

func (b *alertEnrichmentConfigBuilder) buildSteps() string {
	var config string

	// Maintain a consistent order to simplify test assertions
	config += b.buildAssignStep()
	config += b.buildAssistantInvestigationStep()
	config += b.buildExternalStep()
	config += b.buildAssertsStep()
	config += b.buildExplainStep()
	config += b.buildSiftStep()
	config += b.buildDataSourceStep()
	config += b.buildConditionalStep()

	return config
}

func (b *alertEnrichmentConfigBuilder) buildAssignStep() string {
	if len(b.assignSteps) == 0 {
		return ""
	}

	var config string
	for _, step := range b.assignSteps {
		if len(step.annotations) == 0 {
			continue
		}
		config += `
        step {
          assign {`
		timeout := step.timeout
		if timeout != "" && timeout != "0s" {
			config += fmt.Sprintf(`
            timeout = "%s"`, timeout)
		}
		config += `
            annotations = {`
		for name, value := range step.annotations {
			config += fmt.Sprintf(`
                %s = "%s"`, name, value)
		}
		config += `
            }`
		config += `
          }
        }`
	}
	return config
}

func (b *alertEnrichmentConfigBuilder) buildAssistantInvestigationStep() string {
	if b.assistantInvestigationsStep == nil {
		return ""
	}

	timeout := b.assistantInvestigationsStep.timeout
	if timeout == "" {
		timeout = "30s"
	}
	return fmt.Sprintf(`
        step {
          assistant_investigations {
            timeout = "%s"
          }
        }`, timeout)
}

func (b *alertEnrichmentConfigBuilder) buildExternalStep() string {
	if b.externalStep == nil || b.externalStep.url == "" {
		return ""
	}

	timeout := b.externalStep.timeout
	if timeout == "" {
		timeout = "30s"
	}
	return fmt.Sprintf(`
        step {
          external {
            timeout = "%s"
            url = "%s"
          }
        }`, timeout, b.externalStep.url)
}

func (b *alertEnrichmentConfigBuilder) buildAssertsStep() string {
	if b.assertsStep == nil {
		return ""
	}

	timeout := b.assertsStep.timeout
	if timeout == "" {
		timeout = "30s"
	}
	return fmt.Sprintf(`
        step {
          asserts {
            timeout = "%s"
          }
        }`, timeout)
}

func (b *alertEnrichmentConfigBuilder) buildExplainStep() string {
	if b.explainStep == nil {
		return ""
	}

	timeout := b.explainStep.timeout
	if timeout == "" {
		timeout = "30s"
	}
	config := fmt.Sprintf(`
        step {
          explain {
            timeout = "%s"`, timeout)
	if b.explainStep.annotation != "" {
		config += fmt.Sprintf(`
            annotation = "%s"`, b.explainStep.annotation)
	}
	config += `
          }
        }`
	return config
}

func (b *alertEnrichmentConfigBuilder) buildSiftStep() string {
	if b.siftStep == nil {
		return ""
	}

	timeout := b.siftStep.timeout
	if timeout == "" {
		timeout = "30s"
	}
	return fmt.Sprintf(`
        step {
          sift {
            timeout = "%s"
          }
        }`, timeout)
}

func (b *alertEnrichmentConfigBuilder) buildDataSourceStep() string {
	if len(b.dataSourceSteps) == 0 {
		return ""
	}
	var config string
	for _, step := range b.dataSourceSteps {
		timeout := step.timeout
		if timeout == "" {
			timeout = "30s"
		}
		block := fmt.Sprintf(`
        step {
          data_source {
            timeout = "%s"`, timeout)

		if step.logsQuery != nil {
			block += `
            logs_query {`
			block += fmt.Sprintf(`
                data_source_type = "%s"
                expr = "%s"`, step.logsQuery.dataSourceType, step.logsQuery.expr)
			if step.logsQuery.dataSourceUID != "" {
				block += fmt.Sprintf(`
                data_source_uid = "%s"`, step.logsQuery.dataSourceUID)
			}
			if step.logsQuery.maxLines > 0 {
				block += fmt.Sprintf(`
                max_lines = %d`, step.logsQuery.maxLines)
			}
			block += `
            }`
		} else if step.rawQuery != nil {
			block += `
            raw_query {`
			block += fmt.Sprintf(`
                request = %s`, step.rawQuery.request)
			if step.rawQuery.refID != "" {
				block += fmt.Sprintf(`
                ref_id = "%s"`, step.rawQuery.refID)
			}
			block += `
            }`
		}
		block += `
          }
        }`
		config += block
	}
	return config
}

func (b *alertEnrichmentConfigBuilder) buildConditionalStep() string {
	if b.conditionalStep == nil {
		return ""
	}

	timeout := b.conditionalStep.timeout
	if timeout == "" {
		timeout = "30s"
	}

	config := fmt.Sprintf(`
        step {
          conditional {
            timeout = "%s"
            
            if {
              label_matchers = [{
                type  = "="
                name  = "severity"
                value = "critical"
              }]
            }
            
            then {
              step {
                assign {
                  annotations = {
                    priority = "P1"
                  }
                }
              }
            }
            
            else {
              step {
                assign {
                  annotations = {
                    priority = "P3"
                  }
                }
              }
            }
          }
        }`, timeout)

	return config
}

func (b *alertEnrichmentConfigBuilder) buildFooter() string {
	return `
	}

	options {
		overwrite = true
	}
}
`
}

func TestAccAlertEnrichment(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	t.Run("full", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				{
					Config: newAlertEnrichmentConfig(
						fmt.Sprintf("comprehensive-%s", randSuffix),
						"comprehensive-alert-enrichment",
					).withDescription("description-1").
						withAlertRuleUIDs("critical-api-alerts", "critical-db-alerts").
						withReceivers("pagerduty-critical", "slack-alerting", "email-oncall").
						withLabelMatcher("=", "severity", "critical").
						withLabelMatcher("!=", "environment", "test").
						withAssignStep("30s", map[string]string{
							"priority":        "P1",
							"escalation_time": "5m",
							"team_contact":    "alerting-{{ $labels.service }}@grafana.com",
							"runbook":         "https://runbooks.grafana.com/{{ $labels.alert_name }}",
						}).
						withAssignStep("25s", map[string]string{
							"second_assign": "true",
							"sequence":      "2",
						}).
						withAssistantInvestigationsStep("31s").
						withExternalStep("32s", "https://example.com/enrich").
						withAssertsStep("33s").
						withExplainStep("34s", "explanation").
						withSiftStep("35s").
						withDataSourceLogsStep("36s", "loki", "test-loki-uid", `{job=\"my-app\"} | json | level=\"error\"`, 5).
						withDataSourceRawStep("37s", "A", `"{\"datasource\":{\"type\":\"prometheus\",\"uid\":\"test-uid\"},\"expr\":\"up\",\"refId\":\"A\"}"`).
						withConditionalBasic("38s").
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "comprehensive-alert-enrichment"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "description-1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.#", "3"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "2"),

						// Verify steps and ordering
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "10"),
						// First assign step
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.annotations.%", "4"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.timeout", "30s"),
						// Second assign step
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.1.assign.annotations.%", "2"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.1.assign.timeout", "25s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.1.assign.annotations.second_assign", "true"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.1.assign.annotations.sequence", "2"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.2.assistant_investigations.timeout", "31s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.3.external.timeout", "32s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.3.external.url", "https://example.com/enrich"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.4.asserts.timeout", "33s"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.5.explain.timeout", "34s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.5.explain.annotation", "explanation"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.6.sift.timeout", "35s"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.7.data_source.logs_query.data_source_type", "loki"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.7.data_source.logs_query.data_source_uid", "test-loki-uid"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.7.data_source.logs_query.expr", `{job="my-app"} | json | level="error"`),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.7.data_source.logs_query.max_lines", "5"),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.8.data_source.raw_query.ref_id", "A"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.8.data_source.raw_query.request", `{"datasource":{"type":"prometheus","uid":"test-uid"},"expr":"up","refId":"A"}`),

						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.timeout", "38s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.if.label_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.then.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.then.step.0.assign.annotations.priority", "P1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.else.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.9.conditional.else.step.0.assign.annotations.priority", "P3"),

						terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				{
					ResourceName:      alertEnrichmentResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				},
			},
		})
	})

	t.Run("update", func(t *testing.T) {
		randSuffix := acctest.RandString(6)
		uid := fmt.Sprintf("update-test-%s", randSuffix)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				// Create basic enrichment
				{
					Config: newAlertEnrichmentConfig(uid, "test-alert-enrichment").
						withDescription("description-1").
						withAssignStep("10s", map[string]string{
							"enriched_by": "terraform-test",
						}).
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "test-alert-enrichment"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.annotations.%", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.timeout", "10s"),
						terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Update with annotations and timeout
				{
					Config: newAlertEnrichmentConfig(uid, "updated-alert-enrichment").
						withDescription("description-2").
						withAssignStep("15s", map[string]string{
							"enriched_by":   "updated-terraform-test",
							"updated_field": "new-annotation",
						}).
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "updated-alert-enrichment"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "description-2"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.annotations.%", "2"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.timeout", "15s"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assign.annotations.enriched_by", "updated-terraform-test"),
						terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Add matchers and alert rules
				{
					Config: newAlertEnrichmentConfig(uid, "updated-alert-enrichment").
						withDescription("description-3").
						withAlertRuleUIDs("rule-1", "rule-2").
						withLabelMatcher("=", "severity", "critical").
						withAssignStep("15s", map[string]string{
							"enriched_by":   "updated-terraform-test",
							"updated_field": "new-annotation",
						}).
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.0.type", "="),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.0.name", "severity"),
						terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				// Remove all fields to minimal configuration (just title, no steps)
				{
					Config: testAccAlertEnrichmentMinimal(uid),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "minimal-alert-enrichment"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", ""),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "0"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.#", "0"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", "0"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.annotation_matchers.#", "0"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "0"),
						terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
					),
				},
				{
					ResourceName:      alertEnrichmentResourceName,
					ImportState:       true,
					ImportStateVerify: true,
					ImportStateVerifyIgnore: []string{
						"options.%",
						"options.overwrite",
					},
					ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				},
			},
		})
	})

	t.Run("invalid_assign_empty_annotations", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				{
					Config:      testAccAlertEnrichmentAssignEmptyAnnotations(fmt.Sprintf("invalid-assign-%s", randSuffix)),
					ExpectError: regexp.MustCompile("Missing Required Attribute"),
				},
			},
		})
	})

	t.Run("invalid_data_source_both_queries", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				{
					Config:      testAccAlertEnrichmentDataSourceBothQueries(fmt.Sprintf("invalid-ds-both-%s", randSuffix)),
					ExpectError: regexp.MustCompile("Invalid number of attributes"),
				},
			},
		})
	})

	t.Run("invalid_data_source_no_queries", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				{
					Config:      testAccAlertEnrichmentDataSourceNoQueries(fmt.Sprintf("invalid-ds-none-%s", randSuffix)),
					ExpectError: regexp.MustCompile("Invalid number of attributes"),
				},
			},
		})
	})

	t.Run("invalid_timeout_format", func(t *testing.T) {
		randSuffix := acctest.RandString(6)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				{
					Config:      testAccAlertEnrichmentInvalidTimeout(fmt.Sprintf("invalid-timeout-%s", randSuffix)),
					ExpectError: regexp.MustCompile("invalid timeout"),
				},
			},
		})
	})

	t.Run("data_source_query_type_switch", func(t *testing.T) {
		randSuffix := acctest.RandString(6)
		uid := fmt.Sprintf("switch-test-%s", randSuffix)

		terraformresource.ParallelTest(t, terraformresource.TestCase{
			ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
			Steps: []terraformresource.TestStep{
				// Start with logs query
				{
					Config: newAlertEnrichmentConfig(uid, "Query Type Switch Test").
						withDataSourceLogsStep("30s", "loki", "test-uid", `{job=\"test\"}`, 3).
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.logs_query.data_source_type", "loki"),
					),
				},
				// Switch to raw query
				{
					Config: newAlertEnrichmentConfig(uid, "Query Type Switch Test").
						withDataSourceRawStep("30s", "B", `"{\"datasource\":{\"type\":\"prometheus\",\"uid\":\"prom-uid\"},\"expr\":\"up\"}"`).
						build(),
					Check: terraformresource.ComposeTestCheckFunc(
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
						terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.raw_query.ref_id", "B"),
					),
				},
			},
		})
	})
}

func TestAccAlertEnrichment_matchers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	testCases := []struct {
		name          string
		labelMatchers []matcherConfig
		annotMatchers []matcherConfig
		expectedTitle string
	}{
		{
			name: "single_equal_matcher",
			labelMatchers: []matcherConfig{
				{"=", "severity", "critical"},
			},
			expectedTitle: "Single Equal Matcher",
		},
		{
			name: "regex_matchers",
			labelMatchers: []matcherConfig{
				{"=~", "service", "api-.*"},
			},
			expectedTitle: "Regex Matchers",
		},
		{
			name: "negative_matchers",
			labelMatchers: []matcherConfig{
				{"!=", "environment", "test"},
				{"!~", "service", "temp-.*"},
			},
			expectedTitle: "Negative Matchers",
		},
		{
			name: "multiple_same_type_matchers",
			labelMatchers: []matcherConfig{
				{"=", "severity", "critical"},
				{"=", "team", "alerting"},
			},
			expectedTitle: "Multiple Same Type Matchers",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			randSuffix := acctest.RandString(6)

			builder := newAlertEnrichmentConfig(
				fmt.Sprintf("matcher-%s-%s", tc.name, randSuffix),
				tc.expectedTitle,
			).withDescription("").withAssignStep("5s", map[string]string{
				"matched_by": tc.name,
			})

			for _, matcher := range tc.labelMatchers {
				builder = builder.withLabelMatcher(matcher.matchType, matcher.name, matcher.value)
			}

			for _, matcher := range tc.annotMatchers {
				builder = builder.withAnnotationMatcher(matcher.matchType, matcher.name, matcher.value)
			}

			terraformresource.ParallelTest(t, terraformresource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []terraformresource.TestStep{
					{
						Config: builder.build(),
						Check: terraformresource.ComposeTestCheckFunc(
							terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", tc.expectedTitle),
							terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.label_matchers.#", fmt.Sprintf("%d", len(tc.labelMatchers))),
							terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.annotation_matchers.#", fmt.Sprintf("%d", len(tc.annotMatchers))),
							terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
						),
					},
					{
						ResourceName:      alertEnrichmentResourceName,
						ImportState:       true,
						ImportStateVerify: true,
						ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
					},
				},
			})
		})
	}
}

func TestAccAlertEnrichment_alertRulesAndReceivers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("rules-receivers-%s", randSuffix),
					"Rules and Receivers Enrichment",
				).withDescription("Enrichment for specific alert rules and receivers").
					withAlertRuleUIDs("rule-uid-1", "rule-uid-2").
					withReceivers("pagerduty", "slack-critical").
					withAssignStep("0s", map[string]string{
						"filtered": "true",
					}).
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Rules and Receivers Enrichment"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.#", "2"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.0", "rule-uid-1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.alert_rule_uids.1", "rule-uid-2"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.#", "2"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.0", "pagerduty"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.receivers.1", "slack-critical"),
					terraformresource.TestCheckResourceAttrSet(alertEnrichmentResourceName, "id"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_invalidMatchers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(6)

	testCases := []struct {
		name          string
		matcherType   string
		matcherName   string
		matcherValue  string
		expectedError string
	}{
		{
			name:          "invalid_matcher_type",
			matcherType:   "invalid",
			matcherName:   "severity",
			matcherValue:  "critical",
			expectedError: "invalid matcher type",
		},
		{
			name:          "empty_matcher_name",
			matcherType:   "=",
			matcherName:   "",
			matcherValue:  "critical",
			expectedError: "matcher 'type' and 'name' must be set",
		},
		{
			name:          "empty_matcher_type",
			matcherType:   "",
			matcherName:   "severity",
			matcherValue:  "critical",
			expectedError: "matcher 'type' and 'name' must be set",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			terraformresource.ParallelTest(t, terraformresource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []terraformresource.TestStep{
					{
						Config: newAlertEnrichmentConfig(
							fmt.Sprintf("invalid-matcher-%s-%s", tc.name, randSuffix),
							"Invalid Matcher Test",
						).withLabelMatcher(tc.matcherType, tc.matcherName, tc.matcherValue).
							withAssignStep("10s", map[string]string{
								"test": "invalid-matcher",
							}).
							build(),
						ExpectError: regexp.MustCompile(tc.expectedError),
					},
				},
			})
		})
	}
}

func TestAccAlertEnrichment_assistantInvestigationsStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("assistant-investigation-%s", randSuffix),
					"Assistant Investigation Test",
				).withDescription("Tests assistant investigation").
					withAssistantInvestigationsStep("50s").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Assistant Investigation Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests assistant investigation"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.assistant_investigations.timeout", "50s"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_externalStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("external-enricher-%s", randSuffix),
					"External Enricher Test",
				).withDescription("Tests external HTTP enricher").
					withExternalStep("45s", "https://grafana.com/enrich-alert").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "External Enricher Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests external HTTP enricher"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.external.timeout", "45s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.external.url", "https://grafana.com/enrich-alert"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_assertsStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("asserts-enricher-%s", randSuffix),
					"Asserts Enricher Test",
				).withDescription("Tests Asserts service integration").
					withAssertsStep("50s").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Asserts Enricher Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests Asserts service integration"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.asserts.timeout", "50s"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_explainStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			// Test with default annotation
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("explain-%s", randSuffix),
					"Explain Enricher Test",
				).withDescription("Tests AI explain enricher with default annotation").
					withExplainStep("55s", "").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Explain Enricher Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests AI explain enricher with default annotation"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.explain.timeout", "55s"),
				),
			},
			// Update to custom annotation
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("explain-%s", randSuffix),
					"Explain Enricher Custom Test",
				).withDescription("Tests AI explain enricher with custom annotation").
					withExplainStep("45s", "custom_explanation_field").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Explain Enricher Custom Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests AI explain enricher with custom annotation"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.explain.timeout", "45s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.explain.annotation", "custom_explanation_field"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_siftStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("sift-enricher-%s", randSuffix),
					"Sift Enricher Test",
				).withDescription("Tests Sift service integration").
					withSiftStep("40s").
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Sift Enricher Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests Sift service integration"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.sift.timeout", "40s"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_dataSourceLogsStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("datasource-logs-%s", randSuffix),
					"Data Source Logs Query Test",
				).withDescription("Tests data source logs query enricher").
					withDataSourceLogsStep("35s", "loki", "test-loki-uid", `{job=\"my-app\"} | json | severity=\"error\"`, 5).
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Data Source Logs Query Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests data source logs query enricher"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.timeout", "35s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.logs_query.data_source_type", "loki"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.logs_query.data_source_uid", "test-loki-uid"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.logs_query.expr", `{job="my-app"} | json | severity="error"`),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.logs_query.max_lines", "5"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func TestAccAlertEnrichment_dataSourceRawStep(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	randSuffix := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("datasource-raw-%s", randSuffix),
					"Data Source Raw Query Test",
				).withDescription("Tests data source raw query enricher").
					withDataSourceRawStep("25s", "A", `"{\"datasource\":{\"type\":\"prometheus\",\"uid\":\"test-uid\"},\"expr\":\"up\",\"refId\":\"A\"}"`).
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Data Source Raw Query Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests data source raw query enricher"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.timeout", "25s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.raw_query.ref_id", "A"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.raw_query.request", `{"datasource":{"type":"prometheus","uid":"test-uid"},"expr":"up","refId":"A"}`),
				),
			},
			// Test without refId
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("datasource-raw-%s", randSuffix),
					"Data Source Raw Query No RefId Test",
				).withDescription("Tests raw query without refId").
					withDataSourceRawStep("15s", "", `"{\"datasource\":{\"type\":\"loki\",\"uid\":\"loki-uid\"},\"expr\":\"{job=\\\"test\\\"}\"}" `).
					build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Data Source Raw Query No RefId Test"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.description", "Tests raw query without refId"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.timeout", "15s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.raw_query.request", `{"datasource":{"type":"loki","uid":"loki-uid"},"expr":"{job=\"test\"}"}`),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.data_source.raw_query.ref_id", ""),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func testAccCheckAlertEnrichmentResourceDestroy(s *terraform.State) error {
	client := testutils.Provider.Meta().(*common.Client)

	for _, r := range s.RootModule().Resources {
		if r.Type != alertEnrichmentResourceName {
			continue
		}

		rcli, err := client.GrafanaAppPlatformAPI.ClientFor(v1beta1.AlertEnrichmentKind())
		if err != nil {
			return fmt.Errorf("failed to create app platform client: %w", err)
		}

		ns := claims.OrgNamespaceFormatter(client.GrafanaOrgID)
		namespacedClient := resource.NewNamespaced(
			resource.NewTypedClient[*v1beta1.AlertEnrichment, *v1beta1.AlertEnrichmentList](rcli, v1beta1.AlertEnrichmentKind()),
			ns,
		)

		if _, err := namespacedClient.Get(context.Background(), r.Primary.ID); err == nil {
			return fmt.Errorf("AlertEnrichment %s still exists", r.Primary.ID)
		} else if !apierrors.IsNotFound(err) {
			return fmt.Errorf("error checking if AlertEnrichment %s exists: %w", r.Primary.ID, err)
		}
	}
	return nil
}

func TestAccAlertEnrichment_emptySteps(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []terraformresource.TestStep{
			{
				Config:      testAccAlertEnrichmentEmptySteps(uid),
				ExpectError: regexp.MustCompile("Invalid step configuration"),
			},
		},
	})
}

func testAccAlertEnrichmentMinimal(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "minimal-alert-enrichment"
  }
}
`, uid)
}

func testAccAlertEnrichmentEmptySteps(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Empty Steps Enrichment"
    description = "This should fail validation"

    # Empty steps block - this should cause validation error
    step {
    }
  }
}
`, uid)
}

func testAccAlertEnrichmentAssignEmptyAnnotations(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Invalid Assign Step"
    description = "This should fail validation"

    step {
      assign {
        timeout = "30s"
        # Missing annotations - should fail requireAttrsWhenPresent validation
      }
    }
  }
}
`, uid)
}

func testAccAlertEnrichmentDataSourceBothQueries(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Invalid Data Source Step"
    description = "This should fail validation"

    step {
      data_source {
        timeout = "30s"
        # Both queries configured - should fail attributeCountExactly(1) validation
        logs_query {
          data_source_type = "loki"
          expr = "{job=\"test\"}"
        }
        raw_query {
          request = "{\"expr\":\"up\"}"
        }
      }
    }
  }
}
`, uid)
}

func testAccAlertEnrichmentDataSourceNoQueries(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Invalid Data Source Step"
    description = "This should fail validation"

    step {
      data_source {
        timeout = "30s"
        # Neither query configured - should fail attributeCountExactly(1) validation
      }
    }
  }
}
`, uid)
}

func testAccAlertEnrichmentInvalidTimeout(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Invalid Timeout Step"
    description = "This should fail validation"

    step {
      assign {
        timeout = "invalid-timeout-format"
        annotations = {
          test = "value"
        }
      }
    }
  }
}
`, uid)
}

func TestAccAlertEnrichment_conditional(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccAlertEnrichmentConditionalBasic(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Test Conditional Enrichment"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.timeout", "45s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.label_matchers.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.label_matchers.0.type", "="),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.label_matchers.0.name", "severity"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.label_matchers.0.value", "critical"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.0.assign.annotations.priority", "P1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.0.assign.annotations.priority", "P3"),
				),
			},
		},
	})
}

func testAccAlertEnrichmentConditionalBasic(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Test Conditional Enrichment"

    step {
      conditional {
        timeout = "45s"
        
        if {
          label_matchers = [{
            type  = "="
            name  = "severity"
            value = "critical"
          }]
        }
        
        then {
          step {
            assign {
              annotations = {
                priority = "P1"
              }
            }
          }
        }
        
        else {
          step {
            assign {
              annotations = {
                priority = "P3"
              }
            }
          }
        }
      }
    }
  }
}
`, uid)
}

func TestAccAlertEnrichment_conditionalWithDataSourceCondition(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccAlertEnrichmentConditionalWithDataSource(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Test Conditional With Data Source"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.data_source_condition.request", `{"datasource":{"type":"prometheus","uid":"test-uid"},"expr":"up == 0","refId":"A"}`),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.0.assign.annotations.severity", "critical"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.0.assign.annotations.severity", "warning"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func testAccAlertEnrichmentConditionalWithDataSource(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Test Conditional With Data Source"
    description = "Conditional with data source request"

    step {
      conditional {
        if {
          data_source_condition {
            request = jsonencode({
              datasource = {
                type = "prometheus"
                uid  = "test-uid"
              }
              expr  = "up == 0"
              refId = "A"
            })
          }
        }
        
        then {
          step {
            assign {
              annotations = {
                severity = "critical"
              }
            }
          }
        }
        
        else {
          step {
            assign {
              annotations = {
                severity = "warning"
              }
            }
          }
        }
      }
    }
  }
}
`, uid)
}

func TestAccAlertEnrichment_conditionalWithAnnotationMatchers(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccAlertEnrichmentConditionalWithAnnotationMatchers(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Test Conditional With Annotation Matchers"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.annotation_matchers.#", "2"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.annotation_matchers.0.type", "="),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.annotation_matchers.0.name", "runbook_url"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.annotation_matchers.1.type", "!="),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.if.annotation_matchers.1.name", "suppress"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func testAccAlertEnrichmentConditionalWithAnnotationMatchers(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Test Conditional With Annotation Matchers"
    description = "Conditional with annotation matchers"

    step {
      conditional {
        if {
          annotation_matchers = [
            {
              type  = "="
              name  = "runbook_url"
              value = "https://runbook.example.com"
            },
            {
              type  = "!="
              name  = "suppress"
              value = "true"
            }
          ]
        }
        
        then {
          step {
            assign {
              annotations = {
                escalation = "high"
              }
            }
          }
        }
        
        else {
          step {
            assign {
              annotations = {
                escalation = "low"
              }
            }
          }
        }
      }
    }
  }
}
`, uid)
}

func TestAccAlertEnrichment_conditionalWithMultipleEnricherTypes(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(10)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			{
				Config: testAccAlertEnrichmentConditionalWithMultipleEnrichers(uid),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "metadata.uid", uid),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.title", "Test Conditional With Multiple Enrichers"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.#", "1"),
					// then branch enrichers in order
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.#", "3"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.0.assign.annotations.escalation", "immediate"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.1.external.url", "https://pager.example.com/create-incident"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.then.step.2.explain.annotation", "ai_analysis"),
					// else branch enrichers in order
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.#", "3"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.0.assign.annotations.escalation", "standard"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.1.sift.timeout", "30s"),
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.step.0.conditional.else.step.2.asserts.timeout", "25s"),
				),
			},
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateVerifyIgnore: []string{
					"options.%",
					"options.overwrite",
				},
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
			},
		},
	})
}

func testAccAlertEnrichmentConditionalWithMultipleEnrichers(uid string) string {
	return fmt.Sprintf(`
resource "grafana_apps_alertenrichment_alertenrichment_v1beta1" "test" {
  metadata {
    uid = "%s"
  }

  spec {
    title = "Test Conditional With Multiple Enrichers"
    description = "Conditional with multiple enricher types"

    step {
      conditional {
        timeout = "55s"
        
        if {
          label_matchers = [{
            type  = "="
            name  = "severity"
            value = "critical"
          }]
        }
        
        then {
          step {
            assign {
              annotations = {
                escalation = "immediate"
              }
            }
          }
          step {
            external {
              url = "https://pager.example.com/create-incident"
            }
          }
          step {
            explain {
              annotation = "ai_analysis"
            }
          }
        }
        
        else {
          step {
            assign {
              annotations = {
                escalation = "standard"
              }
            }
          }
          step {
            sift {
              timeout = "30s"
            }
          }
          step {
            asserts {
              timeout = "25s"
            }
          }
        }
      }
    }
  }
}
`, uid)
}

func checkProvenanceAnnotation(expectedValue string) terraformresource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[alertEnrichmentResourceName]
		if !ok {
			return fmt.Errorf("Not found: %s", alertEnrichmentResourceName)
		}
		provenance := rs.Primary.Attributes["metadata.annotations.grafana.com/provenance"]
		if provenance != expectedValue {
			return fmt.Errorf("Expected provenance annotation to be '%s', got '%s'", expectedValue, provenance)
		}
		return nil
	}
}

func TestAccAlertEnrichment_disableProvenance(t *testing.T) {
	testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0, <12.3.0") // TODO: alert enrichment API returns 404 on Grafana 12.3+

	uid := acctest.RandString(6)

	terraformresource.ParallelTest(t, terraformresource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		CheckDestroy:             testAccCheckAlertEnrichmentResourceDestroy,
		Steps: []terraformresource.TestStep{
			// Create without disable_provenance (default false)
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("provenance-test-%s", uid),
					"test-provenance-default",
				).withDescription("Test provenance behavior").build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.disable_provenance", "false"),
					checkProvenanceAnnotation("api"),
				),
			},
			// Import and verify
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update to disable_provenance = true
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("provenance-test-%s", uid),
					"test-provenance-disabled",
				).withDescription("Test provenance disabled").withDisableProvenance(true).build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.disable_provenance", "true"),
					checkProvenanceAnnotation(""),
				),
			},
			// Import and verify
			{
				ResourceName:      alertEnrichmentResourceName,
				ImportStateIdFunc: importStateIDFunc(alertEnrichmentResourceName),
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update back to disable_provenance = false
			{
				Config: newAlertEnrichmentConfig(
					fmt.Sprintf("provenance-test-%s", uid),
					"test-provenance-enabled",
				).withDescription("Test provenance re-enabled").withDisableProvenance(false).build(),
				Check: terraformresource.ComposeTestCheckFunc(
					terraformresource.TestCheckResourceAttr(alertEnrichmentResourceName, "spec.disable_provenance", "false"),
					checkProvenanceAnnotation("api"),
				),
			},
		},
	})
}
