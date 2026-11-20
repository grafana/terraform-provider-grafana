resource "grafana_team" "example" {
  name = "Example team"
}

data "grafana_oncall_team" "example" {
  name       = grafana_team.example.name
  depends_on = [grafana_team.example]
}

// Up to three named intervals. The team is in hours when the current time is in any range of any interval.
// Times are read in the team's timezone (set in IRM > Settings > Teams, UTC by default).
resource "grafana_oncall_team_hours" "example" {
  team_id = data.grafana_oncall_team.example.id

  interval {
    name = "Business hours"
    monday {
      start = "09:00"
      end   = "12:00"
    }
    monday {
      start = "13:00"
      end   = "17:00"
    }
    tuesday {
      start = "09:00"
      end   = "17:00"
    }
    wednesday {
      start = "09:00"
      end   = "17:00"
    }
    thursday {
      start = "09:00"
      end   = "17:00"
    }
    friday {
      start = "09:00"
      end   = "17:00"
    }
  }

  interval {
    name = "Weekend coverage"
    saturday {
      start = "10:00"
      end   = "14:00"
    }
  }
}

// Hold escalation until the team is in its hours
resource "grafana_oncall_escalation_chain" "example" {
  name = "Example chain"
}

resource "grafana_oncall_escalation" "wait_for_team_hours" {
  escalation_chain_id    = grafana_oncall_escalation_chain.example.id
  type                   = "pause_until_team_hours"
  notify_to_team_members = data.grafana_oncall_team.example.id
  position               = 0
  depends_on             = [grafana_oncall_team_hours.example]
}
