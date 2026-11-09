package frontendo11y

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/frontendo11yapi"
)

// newTestResource returns a resource whose Grafana Cloud API and Frontend Observability API
// both point at one test server that knows the given apps.
func newTestResource(t *testing.T, apps []frontendo11yapi.App) *resourceFrontendO11yApp {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/instances/1564802"):
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id": 1564802, "slug": "mystack", "regionSlug": "prod-eu-west-2", "createdAt": "2026-01-01T00:00:00Z",
			})
		case strings.HasSuffix(r.URL.Path, "/api/v1/app"):
			_ = json.NewEncoder(w).Encode(apps)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	faroClient, err := frontendo11yapi.NewClient(srv.URL, "grafana.net", "token", srv.Client(), "test", nil)
	if err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(srv.URL)
	cfg := gcom.NewConfiguration()
	cfg.Host = u.Host
	cfg.Scheme = u.Scheme
	cfg.HTTPClient = srv.Client()

	return &resourceFrontendO11yApp{client: faroClient, gcomClient: gcom.NewAPIClient(cfg)}
}

func stateWithModel(ctx context.Context, t *testing.T, r *resourceFrontendO11yApp, m FrontendO11yAppTFModel) tfsdk.State {
	t.Helper()
	var schemaResp resource.SchemaResponse
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	state := tfsdk.State{Schema: schemaResp.Schema}
	if diags := state.Set(ctx, m); diags.HasError() {
		t.Fatalf("building state: %v", diags)
	}
	return state
}

func baseModel(id string) FrontendO11yAppTFModel {
	return FrontendO11yAppTFModel{
		ID:                 types.StringValue(id),
		StackID:            types.Int64Value(1564802),
		Name:               types.StringValue("my-app"),
		AllowedOrigins:     types.ListNull(types.StringType),
		ExtraLogAtrributes: types.MapNull(types.StringType),
		Settings:           types.MapNull(types.StringType),
		CollectorEndpoint:  types.StringNull(),
	}
}

// Crossplane reads a resource before it creates it, with an empty ID. A missing app must
// leave the state, so that Terraform and Crossplane create it.
func TestReadRemovesMissingApp(t *testing.T) {
	ctx := context.Background()
	r := newTestResource(t, []frontendo11yapi.App{{ID: 1, Name: "other-app"}})
	state := stateWithModel(ctx, t, r, baseModel(""))

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}
	if !resp.State.Raw.IsNull() {
		t.Errorf("expected the resource to be removed from the state, got %s", resp.State.Raw)
	}
}

func TestReadFindsAppAndSetsStringID(t *testing.T) {
	ctx := context.Background()
	r := newTestResource(t, []frontendo11yapi.App{{
		ID:                 8234,
		Name:               "my-app",
		Key:                "abc",
		CollectEndpointURL: "https://faro-collector.example.net/collect",
		CORSAllowedOrigins: []frontendo11yapi.AllowedOrigin{{URL: "https://example.com"}},
	}})
	state := stateWithModel(ctx, t, r, baseModel("8234"))

	resp := &resource.ReadResponse{State: state}
	r.Read(ctx, resource.ReadRequest{State: state}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read: %v", resp.Diagnostics)
	}

	var got FrontendO11yAppTFModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("reading state: %v", diags)
	}
	if got.ID.ValueString() != "8234" {
		t.Errorf("ID = %q, want %q", got.ID.ValueString(), "8234")
	}
	if got.CollectorEndpoint.ValueString() != "https://faro-collector.example.net/collect/abc" {
		t.Errorf("collector_endpoint = %q", got.CollectorEndpoint.ValueString())
	}
}
