terraform import grafana_folder_permission_item.name "{{ folderUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
terraform import grafana_folder_permission_item.name "{{ orgID }}:{{ folderUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
