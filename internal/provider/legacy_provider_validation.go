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

type metadataValidation func(resourceName string, m interface{}) error

func grafanaClientPresent(resourceName string, m interface{}) error {
	if m.(*common.Client).GrafanaAPI == nil {
		return fmt.Errorf("the Grafana client is required for `%s`. Set the auth and url provider attributes", resourceName)
	}
	return nil
}

func smClientPresent(resourceName string, m interface{}) error {
	if m.(*common.Client).SMAPI == nil {
		return fmt.Errorf("the Synthetic Monitoring client is required for `%s`. Set the sm_access_token provider attribute", resourceName)
	}
	return nil
}

func cloudClientPresent(resourceName string, m interface{}) error {
	if m.(*common.Client).GrafanaCloudAPI == nil {
		return fmt.Errorf("the Cloud API client is required for `%s`. Set the cloud_api_key provider attribute", resourceName)
	}
	return nil
}

func onCallClientPresent(resourceName string, m interface{}) error {
	if m.(*common.Client).OnCallClient == nil {
		return fmt.Errorf("the Oncall client is required for `%s`. Set the oncall_access_token provider attribute", resourceName)
	}
	return nil
}

func addResourcesMetadataValidation(validateFunc metadataValidation, resources map[string]*schema.Resource) map[string]*schema.Resource {
	for name, r := range resources {
		name := name
		//nolint:staticcheck
		if r.Read != nil {
			log.Fatalf("%s: Read function is not supported", name)
		}
		if r.ReadContext != nil {
			prev := r.ReadContext
			r.ReadContext = func(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
				if err := validateFunc(name, m); err != nil {
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
				if err := validateFunc(name, m); err != nil {
					return diag.FromErr(err)
				}
				return prev(ctx, d, m)
			}
		}
		resources[name] = r
	}
	return resources
}
