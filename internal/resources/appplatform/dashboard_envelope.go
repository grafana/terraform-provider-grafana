package appplatform

import (
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
)

// extractSpecIfEnvelope detects if the given JSON is a full Kubernetes-style envelope
// (i.e. has an "apiVersion" field, as produced by the Grafana UI dashboard export).
// If so, it extracts and returns only the "spec" field, so the caller can unmarshal
// it directly into a DashboardSpec without having to manually unwrap it.
//
// This allows users to pass the full exported dashboard JSON to spec.json instead of
// having to extract .spec themselves:
//
//	spec { json = jsonencode(jsondecode(file("dashboard.json"))) }        # now works
//	spec { json = jsonencode(jsondecode(file("dashboard.json")).spec) }   # still works
//
// If the JSON is not an envelope, it is returned unchanged.
func extractSpecIfEnvelope(normalized jsontypes.Normalized) jsontypes.Normalized {
	var envelope struct {
		APIVersion string          `json:"apiVersion"`
		Spec       json.RawMessage `json:"spec"`
	}

	if err := json.Unmarshal([]byte(normalized.ValueString()), &envelope); err != nil {
		return normalized
	}

	if envelope.APIVersion == "" || len(envelope.Spec) == 0 {
		return normalized
	}

	return jsontypes.NewNormalizedValue(string(envelope.Spec))
}
