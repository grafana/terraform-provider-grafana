package assistant

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
)

func TestUnitHeadersFromMapMarksModified(t *testing.T) {
	t.Parallel()

	headers, diags := headersFromMap(types.MapValueMust(types.StringType, map[string]attr.Value{
		"Authorization": types.StringValue("Bearer token"),
	}))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	for name, body := range map[string]any{
		"create": assistantapi.IntegrationCreate{CustomHeaders: headers},
		"update": assistantapi.IntegrationUpdate{CustomHeaders: &headers},
	} {
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("%s: marshal: %v", name, err)
		}
		if want := `"customHeaders":[{"key":"Authorization","value":"Bearer token","modified":true}]`; !strings.Contains(string(raw), want) {
			t.Fatalf("%s: body %s does not contain %s", name, raw, want)
		}
	}
}
