terraform import grafana_role_assignment_item.name "{{ roleUID }}:{{ type (user, team or service_account) }}:{{ identifier }}"
terraform import grafana_role_assignment_item.name "{{ orgID }}:{{ roleUID }}:{{ type (user, team or service_account) }}:{{ identifier }}"
