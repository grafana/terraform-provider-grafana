resource "grafana_incident_custom_field" "affected_team" {
  name        = "Affected team"
  slug        = "affected_team"
  type        = "single-select"
  description = "The team that owns the affected service."

  select_option {
    value = "payments"
    label = "Payments"
  }

  select_option {
    value       = "checkout"
    label       = "Checkout"
    description = "Cart, checkout and order confirmation."
  }

  # Archived options can't be selected on new incidents, but incidents that
  # already use them keep them.
  select_option {
    value    = "billing"
    label    = "Billing"
    archived = true
  }
}

resource "grafana_incident_custom_field" "customer_ticket" {
  name     = "Customer ticket"
  slug     = "customer_ticket"
  type     = "string"
  required = true
}
