terraform import grafana_rule_group.name "{{ folderUID }}:{{ name }}"
terraform import grafana_rule_group.name "{{ orgID }}:{{ folderUID }}:{{ name }}"
