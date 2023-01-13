terraform import grafana_dashboard.dashboard_name {{dashboard_uid}} # To use the default provider org
terraform import grafana_dashboard.dashboard_name {{org_id}}:{{dashboard_uid}} # When "org_id" is set on the resource
