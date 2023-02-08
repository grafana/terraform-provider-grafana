package grafana_test

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

var (
	defaultOrgIDRegexp = regexp.MustCompile(`^(0|1):[a-zA-Z0-9-_]+$`)
	// https://regex101.com/r/icTmfm/1
	nonDefaultOrgIDRegexp = regexp.MustCompile(`^([^0-1]\d*|1\d+):[a-zA-Z0-9-_]+$`)
)

func checkResourceIsInOrg(resourceName, orgResourceName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resourceOrgID, err := strconv.Atoi(s.RootModule().Resources[resourceName].Primary.Attributes["org_id"])
		if err != nil {
			return err
		}

		if resourceOrgID <= 1 {
			return fmt.Errorf("resource org ID %d is not greater than 1", resourceOrgID)
		}

		orgID, err := strconv.Atoi(s.RootModule().Resources[orgResourceName].Primary.ID)
		if err != nil {
			return err
		}

		if orgID != resourceOrgID {
			return fmt.Errorf("expected org ID %d, got %d", orgID, resourceOrgID)
		}

		return nil
	}
}
