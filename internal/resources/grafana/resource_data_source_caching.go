package grafana

import "github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

func ResrouceDatasourceCache() *schema.Resource {
	return &schema.Resource{
		Description: `
* [HTTP API](https://grafana.com/docs/grafana/latest/developers/http_api/query_and_resource_caching/)
		`,
	}
}
