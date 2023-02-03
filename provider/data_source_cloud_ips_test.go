package provider

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccDataSourceCloudIPsRead(t *testing.T) {
	CheckCloudAPITestsEnabled(t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccExample(t, "data-sources/grafana_cloud_ips/data-source.tf"),
				Check: func(s *terraform.State) error {
					rs := s.RootModule().Resources["data.grafana_cloud_ips.test"]

					for k, v := range rs.Primary.Attributes {
						// Attributes have two parts, the count of a list and the list items
						if strings.HasSuffix(k, ".#") {
							// This is the count
							intValue, err := strconv.Atoi(v)
							if err != nil {
								return fmt.Errorf("could not convert attribute %s (value: %s) to int: %s", k, v, err)
							}
							if intValue == 0 {
								return fmt.Errorf("attribute %s is empty", k)
							}
						} else if k != "id" && k != "%" {
							// Other items are IPs
							if parsed := net.ParseIP(v); parsed == nil {
								return fmt.Errorf("invalid IP in attribute %s: %s", k, v)
							}
						}
					}

					return nil
				},
			},
		},
	})
}
