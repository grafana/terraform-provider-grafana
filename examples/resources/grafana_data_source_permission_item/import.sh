terraform import grafana_data_source_permission_item.name "{{ datasourceUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
terraform import grafana_data_source_permission_item.name "{{ orgID }}:{{ datasourceUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
