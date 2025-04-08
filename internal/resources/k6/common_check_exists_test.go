package k6_test

import (
	"context"
	"fmt"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/grafana/k6-cloud-openapi-client-go/k6"

	"github.com/grafana/terraform-provider-grafana/v3/internal/common"
	"github.com/grafana/terraform-provider-grafana/v3/internal/common/k6providerapi"
	"github.com/grafana/terraform-provider-grafana/v3/internal/testutils"
)

// Helpers that check if a resource exists or doesn't. To define a new one, use the newCheckExistsHelper function.
// A function that gets a resource by their Terraform id is required.
var (
	projectCheckExists = newCheckExistsHelper(
		func(p *k6.ProjectApiModel) int32 { return p.GetId() },
		func(client *k6.APIClient, config *k6providerapi.K6APIConfig, id int32) (*k6.ProjectApiModel, error) {
			ctx := context.WithValue(context.Background(), k6.ContextAccessToken, config.Token)
			m, _, err := client.ProjectsAPI.ProjectsRetrieve(ctx, id).
				XStackId(config.StackID).
				Execute()
			return payloadOrError(m, err)
		},
	)
)

type checkExistsGetResourceFunc[T interface{}] func(client *k6.APIClient, config *k6providerapi.K6APIConfig, id int32) (*T, error)
type checkExistsGetIDFunc[T interface{}] func(*T) int32

type checkExistsHelper[T interface{}] struct {
	getIDFunc       func(*T) int32
	getResourceFunc checkExistsGetResourceFunc[T]
}

// newCheckExistsHelper creates a test helper that checks if a resource exists or not.
// The getIDFunc function should return the id of the resource.
// The getResourceFunc function should return the resource from the given id.
func newCheckExistsHelper[T interface{}](getIDFunc checkExistsGetIDFunc[T], getResourceFunc checkExistsGetResourceFunc[T]) checkExistsHelper[T] {
	return checkExistsHelper[T]{getIDFunc: getIDFunc, getResourceFunc: getResourceFunc}
}

// exists checks that the resource exists.
func (h *checkExistsHelper[T]) exists(rn string, v *T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[rn]
		if !ok {
			return fmt.Errorf("resource not found: %s\n %#v", rn, s.RootModule().Resources)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("resource id not set")
		}

		var resourceID int32
		if resID, err := strconv.Atoi(rs.Primary.ID); err != nil {
			return fmt.Errorf("resource id is not a valid int32")
		} else {
			resourceID = int32(resID)
		}

		obj, err := h.getResourceFunc(
			testutils.Provider.Meta().(*common.Client).K6APIClient,
			testutils.Provider.Meta().(*common.Client).K6APIConfig,
			resourceID,
		)
		if err != nil {
			return fmt.Errorf("error getting resource %s with id %q: %s", rn, rs.Primary.ID, err)
		}

		// Sanity check: The "destroyed" function should fail here because the resource exists
		if err := h.destroyed(obj)(s); err == nil {
			return fmt.Errorf("the destroyed check passed but shouldn't for resource %s with id %q. This is a bug in the test", rn, rs.Primary.ID)
		}

		*v = *obj
		return nil
	}
}

// destroyed checks that the resource doesn't exist.
func (h *checkExistsHelper[T]) destroyed(v *T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceID := h.getIDFunc(v)
		_, err := h.getResourceFunc(
			testutils.Provider.Meta().(*common.Client).K6APIClient,
			testutils.Provider.Meta().(*common.Client).K6APIConfig,
			resourceID,
		)
		if err == nil {
			return fmt.Errorf("%T %d still exists", v, resourceID)
		} else if !common.IsNotFoundError(err) {
			return fmt.Errorf("error checking if resource with id %d was destroyed: %s", resourceID, err)
		}
		return nil
	}
}

// payloadOrError returns the error if not nil, or the payload otherwise.
func payloadOrError[T interface{}](t *T, err error) (*T, error) {
	if err != nil {
		return nil, err
	}
	return t, nil
}
