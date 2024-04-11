terraform import grafana_service_account_permission_item.name "{{ serviceAccountID }}:{{ type (role, team, or user) }}:{{ identifier }}"
terraform import grafana_service_account_permission_item.name "{{ orgID }}:{{ serviceAccountID }}:{{ type (role, team, or user) }}:{{ identifier }}"
