terraform import grafana_folder_permission.my_folder {{folder_uid}} # To use the default provider org
terraform import grafana_folder_permission.my_folder {{org_id}}:{{folder_uid}} # When "org_id" is set on the resource
