resource "grafana_contact_point" "a_contact_point" {
  name = "A Contact Point"

  email {
    addresses = ["one@company.org", "two@company.org"]
    message   = "{{ len .Alerts.Firing }} firing."
  }
}

resource "grafana_mute_timing" "a_mute_timing" {
  name = "Some Mute Timing"

  intervals {
    weekdays = ["monday"]
  }
}

resource "grafana_mute_timing" "working_hours" {
  name = "Working Hours"
  intervals {
    times {
      start = "09:00"
      end   = "18:00"
    }
  }
}


resource "grafana_notification_policy" "my_notification_policy" {
  group_by      = ["..."]
  contact_point = grafana_contact_point.a_contact_point.name

  group_wait      = "45s"
  group_interval  = "6m"
  repeat_interval = "3h"

  policy {
    matcher {
      label = "mylabel"
      match = "="
      value = "myvalue"
    }
    matcher {
      label = "alertname"
      match = "="
      value = "CPU Usage"
    }
    matcher {
      label = "Name"
      match = "=~"
      value = "host.*|host-b.*"
    }
    contact_point  = grafana_contact_point.a_contact_point.name // This can be omitted to inherit from the parent
    continue       = true
    mute_timings   = [grafana_mute_timing.a_mute_timing.name]
    active_timings = [grafana_mute_timing.working_hours.name]

    group_wait      = "45s"
    group_interval  = "6m"
    repeat_interval = "3h"

    policy {
      matcher {
        label = "sublabel"
        match = "="
        value = "subvalue"
      }
      contact_point = grafana_contact_point.a_contact_point.name // This can also be omitted to inherit from the parent's parent
      group_by      = ["..."]
    }
  }

  policy {
    matcher {
      label = "anotherlabel"
      match = "=~"
      value = "another value.*"
    }
    contact_point = grafana_contact_point.a_contact_point.name
    group_by      = ["..."]
  }
}
