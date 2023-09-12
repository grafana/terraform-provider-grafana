# To use the default provider org
terraform import grafana_data_source.by_integer_id {{datasource id}}
terraform import grafana_data_source.by_uid {{datasource uid}}

# When "org_id" is set on the resource
terraform import grafana_data_source.by_integer_id {{org_id}}:{{datasource id}}
terraform import grafana_data_source.by_uid {{org_id}}:{{datasource uid}}
