---
layout: "grafana"
page_title: "Grafana: grafana_team_preferences"
sidebar_current: "docs-grafana-resource-team-preferences"
description: |-
  The grafana_team_preferences resource allows Team Preferences to be maintained. 
---

# grafana\_team\_preferences

The team preferences resource allows for team preferences to be set once a team 
has been created. Available preferences are a light or dark theme, the default
timezone to be used, and the dashboard to be displayed upon login. 

## Example Usage

```hcl
resource "grafana_dashboard" "metrics" {
  config_json = file("grafana-dashboard.json")
}

resource "grafana_team" "team" {
  name = "Team Name"
}

resource "grafana_team_preferences" "team_preferences" {
  team_id           = grafana_team.team.id
  theme             = "dark"
  timezone          = "browser"
  home_dashboard_id = grafana_dashboard.metrics.dashboard_id
}
```

## Argument Reference

The following arguments are supported:

* `team_id` - (Required) The numeric team ID.
* `theme` - (Optional) The theme for the specified team. Available themes are `light`, `dark`, or an empty string for the default theme. 
* `timezone` - (Optional) The timezone for the specified team. Available values are `utc`, `browser`, or an empty string for the default. 
* `home_dashboard_id` - (Optional) The numeric ID of the dashboard to display when a team member logs in.

## Import

Team preferences cannot be imported.
