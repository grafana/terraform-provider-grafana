terraform import grafana_data_source.by_integer_id {{datasource_id}} # To use the default provider org
terraform import grafana_data_source.by_uid {{datasource_uid}} # To use the default provider org

terraform import grafana_data_source.by_integer_id {{org_id}}:{{datasource_id}} # When "org_id" is set on the resource
terraform import grafana_data_source.by_uid {{org_id}}:{{datasource_uid}} # When "org_id" is set on the resource