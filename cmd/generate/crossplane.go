package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func tfToCrossplane(dir string) error {
	dirFiles, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	state, err := getState(dir)
	if err != nil {
		return err
	}

	for _, dirFile := range dirFiles {
		// This doesn't need to be recursive
		if dirFile.IsDir() {
			continue
		}

		if !strings.HasSuffix(dirFile.Name(), ".tf") {
			continue
		}

		filePath := filepath.Join(dir, dirFile.Name())
		hclFile, err := readHCLFile(filePath)
		if err != nil {
			return err
		}

		for _, block := range hclFile.Body().Blocks() {
			if block.Type() == "provider" {
				// TODO: Create crossplane provider config
				continue
			}
			if block.Type() == "resource" {
				resourceType, resourceName := block.Labels()[0], block.Labels()[1]
				apiVersion := "oss.grafana.crossplane.io/v1alpha1"
				snakeCase := strings.TrimPrefix(resourceType, "grafana_")

				switch {
				case strings.HasPrefix(resourceType, "grafana_cloud"):
					apiVersion = "cloud.grafana.crossplane.io/v1alpha1"
					snakeCase = strings.TrimPrefix(resourceType, "grafana_cloud_")
				case strings.HasPrefix(resourceType, "grafana_synthetic_monitoring"):
					apiVersion = "sm.grafana.crossplane.io/v1alpha1"
					snakeCase = strings.TrimPrefix(resourceType, "grafana_synthetic_monitoring_")
				case strings.HasPrefix(resourceType, "grafana_slo"):
					apiVersion = "slo.grafana.crossplane.io/v1alpha1"
				case resourceType == "grafana_contact_point" ||
					resourceType == "grafana_notification_policy" ||
					resourceType == "grafana_mute_timing" ||
					resourceType == "grafana_message_template" ||
					resourceType == "grafana_rule_group":
					apiVersion = "alerting.grafana.crossplane.io/v1alpha1"
				}

				toCamelCase := func(s string) string {
					camelCase := s
					index := strings.Index(camelCase, "_")
					for index >= 0 {
						camelCase = camelCase[:index] + strings.ToUpper(string(camelCase[index+1])) + camelCase[index+2:]
						index = strings.Index(camelCase, "_")
					}
					return camelCase
				}

				resourceFromState, err := state.getResource(resourceType, resourceName)
				if err != nil {
					return err
				}
				resourceValues := resourceFromState.values()

				providerConfigRef := map[string]interface{}{}
				forProvider := map[string]interface{}{}
				for key, value := range block.Body().Attributes() {
					// TODO handle nested blocks
					vStr := string(value.Expr().BuildTokens(nil).Bytes())
					vStr = strings.TrimPrefix(vStr, " ")

					if key == "provider" {
						providerConfigRef["name"] = strings.TrimPrefix(vStr, "grafana.")
						continue
					}

					if strings.HasPrefix(vStr, "jsonencode") {
						vStr = strings.TrimPrefix(vStr, "jsonencode(")
						vStr = strings.TrimSuffix(vStr, ")")
						forProvider[toCamelCase(key)] = vStr
						continue
					}

					if strings.HasPrefix(vStr, `file("${path.module}/`) {
						vStr = strings.TrimPrefix(vStr, `file("${path.module}/`)
						vStr = strings.TrimSuffix(vStr, `")`)
						content, err := os.ReadFile(filepath.Join(dir, vStr))
						if err != nil {
							return err
						}
						forProvider[toCamelCase(key)] = string(content)
						continue
					}

					var v interface{}
					if err := json.Unmarshal([]byte(vStr), &v); err == nil {
						forProvider[toCamelCase(key)] = v
						continue
					}
				}

				resourceAsMap := map[string]interface{}{
					"apiVersion": apiVersion,
					"kind":       strings.Title(toCamelCase(snakeCase)),
					"metadata": map[string]interface{}{
						"name": strings.ReplaceAll(resourceName, "_", "-"),
						"annotations": map[string]interface{}{
							// external name == ID
							"crossplane.io/external-name": resourceValues["id"].(string),
						},
					},
					"spec": map[string]interface{}{
						"forProvider":       forProvider,
						"providerConfigRef": providerConfigRef,
					},
				}

				// if resourceType == "grafana_synthetic_monitoring_installation" {
				// 	// TODO: Add secret output
				// }
				// if resourceType == "grafana_cloud_access_policy_token" {
				// 	// TODO: Add secret output
				// }
				// if resourceType == "grafana_cloud_stack_service_account_token" {
				// 	// TODO: Add secret output
				// }
				// Create crossplane resource
				crossplaneFile, err := os.Create(filepath.Join(dir, fmt.Sprintf("%s_%s.yaml", resourceType, resourceName)))
				if err != nil {
					return err
				}
				if err := yaml.NewEncoder(crossplaneFile).Encode(resourceAsMap); err != nil {
					return err
				}

				if err := os.Remove(filePath); err != nil {
					return err
				}
			}
		}
	}

	return nil
}
