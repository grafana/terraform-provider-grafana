package frontendo11y

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestConvertClientModelToTFModel_OriginsSorted(t *testing.T) {
	app := frontendo11yapi.App{
		ID:   1,
		Name: "test-app",
		CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{
			{URL: "https://z.example.com"},
			{URL: "https://a.example.com"},
			{URL: "https://m.example.com"},
		},
	}

	result, diags := convertClientModelToTFModel(123, app)
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}

	var origins []types.String
	diags = result.AllowedOrigins.ElementsAs(nil, &origins, false)
	if diags.HasError() {
		t.Fatalf("failed to extract origins: %v", diags)
	}

	expected := []string{
		"https://a.example.com",
		"https://m.example.com",
		"https://z.example.com",
	}

	if len(origins) != len(expected) {
		t.Fatalf("expected %d origins, got %d", len(expected), len(origins))
	}

	for i, exp := range expected {
		if origins[i].ValueString() != exp {
			t.Errorf("origin[%d]: expected %q, got %q", i, exp, origins[i].ValueString())
		}
	}
}
