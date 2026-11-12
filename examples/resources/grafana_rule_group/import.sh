terraform import grafana_rule_group.name "{{ folderUID }}:{{ ruleGroupName }}"
terraform import grafana_rule_group.name "{{ orgID }}:{{ folderUID }}:{{ ruleGroupName }}"
