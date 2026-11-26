package grafana

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestUnitTeam_PreferencesTimezoneValidation(t *testing.T) {
	ctx := context.Background()
	resp := &resource.SchemaResponse{}
	(&teamResource{}).Schema(ctx, resource.SchemaRequest{}, resp)

	block, ok := resp.Schema.Blocks["preferences"].(schema.ListNestedBlock)
	if !ok {
		t.Fatal("preferences block not found in the team schema")
	}
	attr, ok := block.NestedObject.Attributes["timezone"].(schema.StringAttribute)
	if !ok {
		t.Fatal("timezone attribute not found in the preferences block")
	}

	cases := []struct {
		value   string
		wantErr bool
	}{
		{value: "", wantErr: false},
		{value: "utc", wantErr: false},
		{value: "browser", wantErr: false},
		{value: "Europe/Madrid", wantErr: false},
		{value: "America/New_York", wantErr: false},
		{value: "UTC", wantErr: false},
		{value: "Europe/Nowhere", wantErr: true},
		{value: "not a timezone", wantErr: true},
	}

	for _, tc := range cases {
		t.Run(tc.value, func(t *testing.T) {
			req := validator.StringRequest{
				Path:        path.Root("preferences").AtListIndex(0).AtName("timezone"),
				ConfigValue: types.StringValue(tc.value),
			}
			validateResp := &validator.StringResponse{}
			for _, v := range attr.Validators {
				v.ValidateString(ctx, req, validateResp)
			}

			if got := validateResp.Diagnostics.HasError(); got != tc.wantErr {
				t.Fatalf("timezone %q: expected error %t, got %t: %v", tc.value, tc.wantErr, got, validateResp.Diagnostics)
			}
		})
	}
}
