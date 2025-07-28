package generate

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"gopkg.in/yaml.v2"
)

func convertToCrossplane(cfg *Config) error {
	ctx := context.Background()
	resourcesMap := provider.ResourcesMap()

	state, err := getPlannedState(ctx, cfg)
	if err != nil {
		return err
	}

	// Convert to JSON, this representation is pretty much what Crossplane expects (in snake_case rather than camelCase)
	if err := convertToTFJSON(cfg.OutputDir); err != nil {
		return err
	}

	dirFiles, err := os.ReadDir(cfg.OutputDir)
	if err != nil {
		return err
	}

	resourceConfigs := map[string]map[string][]interface{}{}
	for _, dirFile := range dirFiles {
		if strings.HasSuffix(dirFile.Name(), ".json") {
			file, err := os.Open(filepath.Join(cfg.OutputDir, dirFile.Name()))
			if err != nil {
				return err
			}
			defer file.Close()

			var fileJSON map[string]interface{}
			if err := json.NewDecoder(file).Decode(&fileJSON); err != nil {
				return err
			}

			if fileResources, ok := fileJSON["resource"]; ok {
				for kind, kindResources := range fileResources.(map[string]interface{}) {
					if _, ok := resourceConfigs[kind]; !ok {
						resourceConfigs[kind] = map[string][]interface{}{}
					}
					for rName, r := range kindResources.(map[string]interface{}) {
						resourceConfigs[kind][rName] = r.([]interface{})
					}
				}
			}
		}

		// Remove (Terraform) files from the output directory
		if err := os.RemoveAll(filepath.Join(cfg.OutputDir, dirFile.Name())); err != nil {
			return err
		}
	}

	// Write the provider file
	providerFile, err := os.Create(filepath.Join(cfg.OutputDir, "provider.yaml"))
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(providerFile).Encode(yaml.MapSlice{
		{Key: "apiVersion", Value: "grafana.crossplane.io/v1beta1"},
		{Key: "kind", Value: "ProviderConfig"},
		{Key: "metadata", Value: yaml.MapSlice{
			{Key: "name", Value: "grafana-provider"},
		}},
		{Key: "spec", Value: yaml.MapSlice{
			{Key: "credentials", Value: yaml.MapSlice{
				{Key: "source", Value: "Secret"},
				{Key: "secretRef", Value: yaml.MapSlice{
					{Key: "namespace", Value: "crossplane"},
					{Key: "name", Value: "grafana-provider"},
					{Key: "key", Value: "credentials"},
				}},
			}},
		}},
	}); err != nil {
		return err
	}
	defer providerFile.Close()

	for _, r := range state.PlannedValues.RootModule.Resources {
		apiVersion := "oss.grafana.crossplane.io/v1alpha1"
		snakeCaseType := strings.TrimPrefix(r.Type, "grafana_")
		name := strings.ReplaceAll(strings.TrimPrefix(r.Name, "_"), "_", "-")
		resourceInfo := resourcesMap[r.Type]

		switch resourceInfo.Category {
		case common.CategoryCloud:
			apiVersion = "cloud.grafana.crossplane.io/v1alpha1"
			snakeCaseType = strings.TrimPrefix(r.Type, "grafana_cloud_")
		case common.CategorySyntheticMonitoring:
			apiVersion = "sm.grafana.crossplane.io/v1alpha1"
			snakeCaseType = strings.TrimPrefix(r.Type, "grafana_synthetic_monitoring_")
		case common.CategorySLO:
			apiVersion = "slo.grafana.crossplane.io/v1alpha1"
		case common.CategoryAlerting:
			apiVersion = "alerting.grafana.crossplane.io/v1alpha1"
		case common.CategoryMachineLearning:
			apiVersion = "ml.grafana.crossplane.io/v1alpha1"
		case common.CategoryOnCall:
			apiVersion = "oncall.grafana.crossplane.io/v1alpha1"
		case common.CategoryGrafanaEnterprise:
			apiVersion = "enterprise.grafana.crossplane.io/v1alpha1"
		}

		kind := toCamelCase(snakeCaseType)
		kind = strings.ToUpper(string(kind[0])) + kind[1:]

		id := r.AttributeValues["id"].(string)
		forProvider := forProviderMap(resourceConfigs[r.Type][r.Name][0].(map[string]interface{}), r.AttributeValues)
		providerConfigRef := map[string]interface{}{
			"name": "grafana-provider",
		}
		resourceAsMap := yaml.MapSlice{
			{Key: "apiVersion", Value: apiVersion},
			{Key: "kind", Value: kind},
			{Key: "metadata", Value: yaml.MapSlice{
				{Key: "name", Value: name},
				{Key: "annotations", Value: map[string]string{
					"crossplane.io/external-name": id,
				}},
			}},
			{Key: "spec", Value: yaml.MapSlice{
				{Key: "forProvider", Value: forProvider},
				{Key: "providerConfigRef", Value: providerConfigRef},
			}},
		}

		// Create crossplane resource
		crossplaneFile, err := os.Create(filepath.Join(cfg.OutputDir, fmt.Sprintf("%s-%s.yaml", strings.ReplaceAll(snakeCaseType, "_", "-"), name)))
		if err != nil {
			return err
		}
		defer crossplaneFile.Close()
		if err := yaml.NewEncoder(crossplaneFile).Encode(resourceAsMap); err != nil {
			return err
		}
	}

	return nil
}

func forProviderMap(m map[string]interface{}, plannedAttributeValues map[string]interface{}) map[string]interface{} {
	result := map[string]interface{}{}
	for k, v := range m {
		if mapValue, ok := v.(map[string]interface{}); ok {
			result[toCamelCase(k)] = forProviderMap(mapValue, plannedAttributeValues[k].(map[string]interface{}))
		} else if stringValue, ok := v.(string); ok && strings.Contains(stringValue, "${") {
			result[toCamelCase(k)] = plannedAttributeValues[k]
		} else {
			result[toCamelCase(k)] = v
		}
	}
	return result
}

func toCamelCase(s string) string {
	camelCase := s
	index := strings.Index(camelCase, "_")
	for index >= 0 {
		camelCase = camelCase[:index] + strings.ToUpper(string(camelCase[index+1])) + camelCase[index+2:]
		index = strings.Index(camelCase, "_")
	}
	return camelCase
}
