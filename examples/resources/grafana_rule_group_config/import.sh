terraform import grafana_rule_group_config.name "{{ folderUID }}:{{ ruleGroupName }}"
terraform import grafana_rule_group_config.name "{{ orgID }}:{{ folderUID }}:{{ ruleGroupName }}"
