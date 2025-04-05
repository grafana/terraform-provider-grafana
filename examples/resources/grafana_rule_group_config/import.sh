terraform import grafana_rule_group_config.name "{{ folderUID }}:{{ title }}"
terraform import grafana_rule_group_config.name "{{ orgID }}:{{ folderUID }}:{{ title }}"
