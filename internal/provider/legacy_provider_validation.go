package provider

import (
	"context"
	"fmt"
	"log"

	"github.com/grafana/terraform-provider-grafana/internal/common"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file contains validations functions that can be added to a map of resources.
// These validations are added to the Create and Read functions of all resources,
// because they are entrypoints (code that will be run in all cases).

type metadataValidation func(resourceName string, d *schema.ResourceData, m interface{}) error

func readGrafanaClientValidation(resourceName string, d *schema.ResourceData, m interface{}) error {
	if m.(*common.Client).GrafanaOAPI == nil {
		return fmt.Errorf("the Grafana client is required for `%s`. Set the auth and url provider attributes", resourceName)
	}
	return nil
}

func createGrafanaClientValidation(resourceName string, d *schema.ResourceData, m interface{}) error {
	if err := readGrafanaClientValidation(resourceName, d, m); err != nil {
		return err
	}
	orgID, ok := d.GetOk("org_id")
	orgIDStr, orgIDOk := orgID.(string)
	if ok && orgIDOk && orgIDStr != "" && orgIDStr != "0" && m.(*common.Client).GrafanaAPIConfig.APIKey != "" {
		return fmt.Errorf("org_id is only supported with basic auth. API keys are already org-scoped")
	}
	return nil
}

func smClientPresent(resourceName string, d *schema.ResourceData, m interface{}) error {
	if m.(*common.Client).SMAPI == nil {
		return fmt.Errorf("the Synthetic Monitoring client is required for `%s`. Set the sm_access_token provider attribute", resourceName)
	}
	return nil
}

func addResourcesMetadataValidation(validateFunc metadataValidation, resources map[string]*schema.Resource) map[string]*schema.Resource {
	return addCreateReadResourcesMetadataValidation(validateFunc, validateFunc, resources)
}

func addCreateReadResourcesMetadataValidation(readValidateFunc, createValidateFunc metadataValidation, resources map[string]*schema.Resource) map[string]*schema.Resource {
	for name, r := range resources {
		name := name
		//nolint:staticcheck
		if r.Read != nil {
			log.Fatalf("%s: Read function is not supported", name)
		}
		if r.ReadContext != nil {
			prev := r.ReadContext
			r.ReadContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := readValidateFunc(name, d, m); err != nil {
					return diag.FromErr(err)
				}
				return prev(ctx, d, m)
			}
		}
		//nolint:staticcheck
		if r.Create != nil {
			log.Fatalf("%s: Create function is not supported", name)
		}
		if r.CreateContext != nil {
			prev := r.CreateContext
			r.CreateContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := createValidateFunc(name, d, m); err != nil {
					return diag.FromErr(err)
				}
				return prev(ctx, d, m)
			}
		}
		resources[name] = r
	}
	return resources
}
