terraform import grafana_dashboard_permission_item.name "{{ dashboardUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
terraform import grafana_dashboard_permission_item.name "{{ orgID }}:{{ dashboardUID }}:{{ type (role, team, or user) }}:{{ identifier }}"
