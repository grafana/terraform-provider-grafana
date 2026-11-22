package frontendo11y

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

func TestParseAppID(t *testing.T) {
	cases := []struct {
		name    string
		in      types.String
		want    int64
		wantErr bool
	}{
		{name: "number", in: types.StringValue("8234"), want: 8234},
		{name: "empty", in: types.StringValue(""), want: 0},
		{name: "null", in: types.StringNull(), want: 0},
		{name: "unknown", in: types.StringUnknown(), want: 0},
		{name: "not a number", in: types.StringValue("abc"), wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, diags := parseAppID(tc.in)
			if diags.HasError() != tc.wantErr {
				t.Fatalf("parseAppID(%s) errors = %v, want error %v", tc.in, diags, tc.wantErr)
			}
			if !tc.wantErr && got != tc.want {
				t.Errorf("parseAppID(%s) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestModelConversionRoundTrip(t *testing.T) {
	ctx := context.Background()
	app := frontendo11yapi.App{
		ID:                 8234,
		Name:               "my-app",
		CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{{URL: "https://example.com"}},
		ExtraLogLabels:     []frontendo11yapi.LogLabel{{Label: "team", Value: "web"}},
		Settings:           map[string]string{"geolocation.enabled": "1"},
		CollectEndpointURL: "https://faro-collector.example.net/collect",
		Key:                "abc",
	}

	tfModel, diags := convertClientModelToTFModel(1564802, app)
	if diags.HasError() {
		t.Fatalf("convertClientModelToTFModel: %v", diags)
	}
	if tfModel.ID.ValueString() != "8234" {
		t.Errorf("ID = %q, want %q", tfModel.ID.ValueString(), "8234")
	}

	back, diags := tfModel.toClientModel(ctx)
	if diags.HasError() {
		t.Fatalf("toClientModel: %v", diags)
	}
	if back.ID != app.ID {
		t.Errorf("round trip ID = %d, want %d", back.ID, app.ID)
	}
}

func TestToClientModelRejectsInvalidID(t *testing.T) {
	model := FrontendO11yAppTFModel{
		ID:                 types.StringValue("not-a-number"),
		AllowedOrigins:     types.ListNull(types.StringType),
		ExtraLogAtrributes: types.MapNull(types.StringType),
		Settings:           types.MapNull(types.StringType),
	}
	if _, diags := model.toClientModel(context.Background()); !diags.HasError() {
		t.Fatal("expected an error for a non-numeric ID")
	}
}

func TestUpgradeStateV0ToV1(t *testing.T) {
	ctx := context.Background()
	r := &resourceFrontendO11yApp{}
	upgrader, ok := r.UpgradeState(ctx)[0]
	if !ok {
		t.Fatal("missing state upgrader for version 0")
	}

	listOfStrings := tftypes.List{ElementType: tftypes.String}
	mapOfStrings := tftypes.Map{ElementType: tftypes.String}
	priorType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"id":                   tftypes.Number,
		"stack_id":             tftypes.Number,
		"name":                 tftypes.String,
		"collector_endpoint":   tftypes.String,
		"allowed_origins":      listOfStrings,
		"extra_log_attributes": mapOfStrings,
		"settings":             mapOfStrings,
	}}
	priorValue := tftypes.NewValue(priorType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.Number, 8234),
		"stack_id":           tftypes.NewValue(tftypes.Number, 1564802),
		"name":               tftypes.NewValue(tftypes.String, "my-app"),
		"collector_endpoint": tftypes.NewValue(tftypes.String, "https://faro-collector.example.net/collect/abc"),
		"allowed_origins": tftypes.NewValue(listOfStrings, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "https://example.com"),
		}),
		"extra_log_attributes": tftypes.NewValue(mapOfStrings, map[string]tftypes.Value{
			"team": tftypes.NewValue(tftypes.String, "web"),
		}),
		"settings": tftypes.NewValue(mapOfStrings, map[string]tftypes.Value{
			"geolocation.enabled": tftypes.NewValue(tftypes.String, "1"),
		}),
	})

	var currentSchema resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &currentSchema)

	req := resource.UpgradeStateRequest{
		State: &tfsdk.State{Raw: priorValue, Schema: *upgrader.PriorSchema},
	}
	resp := &resource.UpgradeStateResponse{
		State: tfsdk.State{Schema: currentSchema.Schema},
	}
	upgrader.StateUpgrader(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("state upgrade failed: %v", resp.Diagnostics)
	}

	var upgraded FrontendO11yAppTFModel
	if diags := resp.State.Get(ctx, &upgraded); diags.HasError() {
		t.Fatalf("reading upgraded state: %v", diags)
	}
	if upgraded.ID.ValueString() != "8234" {
		t.Errorf("upgraded ID = %q, want %q", upgraded.ID.ValueString(), "8234")
	}
	if upgraded.StackID.ValueInt64() != 1564802 || upgraded.Name.ValueString() != "my-app" {
		t.Errorf("upgraded state lost attributes: %+v", upgraded)
	}
	if upgraded.CollectorEndpoint.ValueString() != "https://faro-collector.example.net/collect/abc" {
		t.Errorf("upgraded collector_endpoint = %q", upgraded.CollectorEndpoint.ValueString())
	}
}

func TestSchemaVersionAndIDType(t *testing.T) {
	ctx := context.Background()
	var resp resource.SchemaResponse
	(&resourceFrontendO11yApp{}).Schema(ctx, resource.SchemaRequest{}, &resp)
	if resp.Schema.Version != 1 {
		t.Errorf("schema version = %d, want 1", resp.Schema.Version)
	}
	if got := resp.Schema.Attributes["id"].GetType(); got != types.StringType {
		t.Errorf("id type = %s, want %s", got, types.StringType)
	}
}
