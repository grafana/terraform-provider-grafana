package appplatform

import (
	"encoding/json"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
)

// rejectIfEnvelope detects if the given JSON is a full Kubernetes-style envelope
// (i.e. has an "apiVersion" field at the top level). If so, it returns an error
// diagnostic explaining that only the spec is expected.
//
// The json field expects the dashboard spec object only:
//
//	spec { json = jsonencode(jsondecode(file("dashboard.json")).spec) }  # correct
//	spec { json = jsonencode(jsondecode(file("dashboard.json")))       }  # wrong — full envelope
func rejectIfEnvelope(normalized jsontypes.Normalized) diag.Diagnostics {
	var probe struct {
		APIVersion string `json:"apiVersion"`
	}

	if err := json.Unmarshal([]byte(normalized.ValueString()), &probe); err != nil {
		return nil
	}

	if probe.APIVersion == "" {
		return nil
	}

	return diag.Diagnostics{
		diag.NewErrorDiagnostic(
			"Full Kubernetes envelope passed to spec.json",
			fmt.Sprintf(
				"The `json` field expects only the dashboard spec object, not the full Kubernetes envelope.\n\n"+
					"The value you provided has `apiVersion: %q` at the top level, which means it is the full "+
					"envelope (apiVersion + kind + metadata + spec) as exported by the Grafana UI.\n\n"+
					"Fix: extract the spec before encoding:\n\n"+
					"  json = jsonencode(jsondecode(file(\"dashboard.json\")).spec)\n\n"+
					"Accepting the full envelope is intentionally unsupported — metadata fields "+
					"(name, namespace, labels) are managed by Terraform, not the spec.",
				probe.APIVersion,
			),
		),
	}
}
