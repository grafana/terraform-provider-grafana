# A minimal service.
resource "grafana_apps_servicemodel_component_v1alpha1" "payments" {
  metadata {
    uid = "payments-api"
  }

  spec {
    title = "Payments API"
  }
}

resource "grafana_team" "checkout" {
  name = "Checkout Team"
}

# A service with ownership, dependencies and links.
resource "grafana_apps_servicemodel_component_v1alpha1" "checkout" {
  metadata {
    uid = "checkout-service"
  }

  spec {
    title       = "Checkout Service"
    description = "Handles checkout and payment orchestration."
    # type defaults to "service"

    # The uid is the implicit service_name; add an explicit one when the
    # telemetry value differs (e.g. characters the uid does not allow).
    identifiers {
      key   = "service_name"
      value = "Checkout_Service"
    }

    identifiers {
      key   = "namespace"
      value = "checkout-prod"
    }

    owner_ref {
      # api_version and kind default to a Grafana IAM team reference
      name = grafana_team.checkout.team_uid
    }

    depends_on_refs {
      name = grafana_apps_servicemodel_component_v1alpha1.payments.metadata.uid
    }

    links {
      url   = "https://github.com/example/checkout"
      title = "Source code"
      type  = "repository"
    }
  }
}