resource "grafana_mute_timing" "my_mute_timing" {
    name = "My Mute Timing"

    intervals {
        times {
            start = "04:56"
            end = "14:17"
        }
        weekdays = ["monday", "tuesday:thursday"]
        days_of_month = ["1:7", "-1"]
        # TODO: This breaks on "december" but works on "12."
        months = ["1:3", "12"]
        years = ["2030", "2025:2026"]
    }
}