resource "grafana_machine_learning_holiday" "ical" {
  name        = "My iCal holiday"
  description = "My Holiday"

  ical_url      = "https://calendar.google.com/calendar/ical/en.uk%23holiday%40group.v.calendar.google.com/public/basic.ics"
  ical_timezone = "Europe/London"
}
