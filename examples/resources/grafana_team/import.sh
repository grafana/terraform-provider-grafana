terraform import grafana_team.team_name {{team_id}} # To use the default provider org
terraform import grafana_team.team_name {{org_id}}:{{team_id}} # When "org_id" is set on the resource
