package testutils

import (
	"context"
	"fmt"
	"os"
	"slices"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/cloud"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/grafana"
	"github.com/grafana/terraform-provider-grafana/v4/pkg/provider"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// CheckLister is a resource.TestCheckFunc that checks that the resource's lister
// function returns the given ID.
// This is meant to be used at least once in every resource's tests to ensure that
// the resource's lister function is working correctly.
func CheckLister(terraformResource string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		// Get the resource from the state
		rs, ok := s.RootModule().Resources[terraformResource]
		if !ok {
			return fmt.Errorf("resource not found: %s", terraformResource)
		}
		id := rs.Primary.ID

		// Find the resource info
		resource, ok := provider.ResourcesMap()[rs.Type]
		if !ok {
			return fmt.Errorf("resource type %s not found", rs.Type)
		}

		// Get the resource's lister function
		lister := resource.ListIDsFunc
		if lister == nil {
			return fmt.Errorf("resource %s does not have a lister function", terraformResource)
		}

		// Get the list of IDs from the lister function
		ctx := context.Background()
		var listerData any = grafana.NewListerData(false, false)
		if resource.Category == common.CategoryCloud {
			listerData = cloud.NewListerData(os.Getenv("GRAFANA_CLOUD_ORG"))
		}
		// Asserts resources are stack-scoped, so we need stack ID for listing
		if resource.Category == common.CategoryAsserts {
			listerData = os.Getenv("GRAFANA_CLOUD_PROVIDER_TEST_STACK_ID")
		}
		ids, err := lister(ctx, Provider.Meta().(*common.Client), listerData)
		if err != nil {
			return fmt.Errorf("error listing %s: %w", terraformResource, err)
		}

		// Check that the ID is in the list
		if slices.Contains(ids, id) {
			return nil
		}

		return fmt.Errorf("resource %s with ID %s not found in list: %v", terraformResource, id, ids)
	}
}
