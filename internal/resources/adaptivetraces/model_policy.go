package adaptivetraces

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/adaptivetracesapi"
)

type policyModel struct {
	ID               types.String         `tfsdk:"id"`
	Name             types.String         `tfsdk:"name"`
	Type             types.String         `tfsdk:"type"`
	Body             jsontypes.Normalized `tfsdk:"body"`
	VersionCreatedAt types.String         `tfsdk:"version_created_at"`
	VersionCreatedBy types.String         `tfsdk:"version_created_by"`
}

// toPolicy builds the full-replace payload. It has no expiry, so an update clears one set outside Terraform.
func (m *policyModel) toPolicy() adaptivetracesapi.Policy {
	return adaptivetracesapi.Policy{
		Type: m.Type.ValueString(),
		Name: m.Name.ValueString(),
		Body: json.RawMessage(m.Body.ValueString()),
	}
}

// setFromPolicy copies the API policy into the model. The body keeps its current string when it means the same
// JSON as the API's, so its stored formatting is not rewritten.
func (m *policyModel) setFromPolicy(ctx context.Context, policy *adaptivetracesapi.Policy) diag.Diagnostics {
	var diags diag.Diagnostics

	body, err := canonicalJSON(policy.Body)
	if err != nil {
		diags.AddError("Invalid policy body", "The API returned a body that is not valid JSON: "+err.Error())
		return diags
	}
	newBody := jsontypes.NewNormalizedValue(body)
	if !m.Body.IsNull() && !m.Body.IsUnknown() {
		if equal, _ := m.Body.StringSemanticEquals(ctx, newBody); equal {
			newBody = m.Body
		}
	}

	m.ID = types.StringValue(policy.ID)
	m.Name = types.StringValue(policy.Name)
	m.Type = types.StringValue(policy.Type)
	m.Body = newBody
	m.VersionCreatedAt = stringValueOrNull(policy.VersionCreatedAt)
	m.VersionCreatedBy = stringValueOrNull(policy.VersionCreatedBy)
	return diags
}

// canonicalJSON compacts raw and sorts object keys, matching what jsonencode() produces.
func canonicalJSON(raw json.RawMessage) (string, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()

	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", err
	}
	out, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func stringValueOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}
