package grafana

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestUnitDataSourcesRead(t *testing.T) {
	for _, test := range []struct {
		name      string
		orgID     types.String
		wantOrgID string
		status    int
		body      string
		wantCount int
		wantError bool
	}{
		{
			name:      "all sources including provisioned",
			orgID:     types.StringNull(),
			wantOrgID: "1",
			status:    http.StatusOK,
			body:      `[{"id":2,"uid":"z-loki","name":"Logs","type":"loki","url":"http://loki:3100","isDefault":false},{"id":1,"uid":"a-prometheus","name":"Metrics","type":"prometheus","url":"http://prometheus:9090","isDefault":true,"readOnly":true}]`,
			wantCount: 2,
		},
		{
			name:      "empty organization",
			orgID:     types.StringNull(),
			wantOrgID: "1",
			status:    http.StatusOK,
			body:      `[]`,
		},
		{
			name:      "explicit organization",
			orgID:     types.StringValue("2"),
			wantOrgID: "2",
			status:    http.StatusOK,
			body:      `[]`,
		},
		{
			name:      "API error",
			orgID:     types.StringNull(),
			wantOrgID: "1",
			status:    http.StatusForbidden,
			body:      `{"message":"access denied"}`,
			wantError: true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				require.Equal(t, http.MethodGet, request.Method)
				require.Equal(t, "/api/datasources", request.URL.Path)
				require.Equal(t, test.wantOrgID, request.Header.Get("X-Grafana-Org-Id"))
				writer.Header().Set("Content-Type", "application/json")
				writer.WriteHeader(test.status)
				fmt.Fprint(writer, test.body)
			}))
			defer server.Close()

			ctx := context.Background()
			source := &dataSourcesDataSource{
				basePluginFrameworkDataSource: basePluginFrameworkDataSource{
					client: testGrafanaClient(t, server.URL).WithOrgID(1),
				},
			}
			var schemaResponse datasource.SchemaResponse
			source.Schema(ctx, datasource.SchemaRequest{}, &schemaResponse)
			config := tfsdk.State{Schema: schemaResponse.Schema}
			require.False(t, config.Set(ctx, &dataSourcesDataSourceModel{
				ID:    types.StringNull(),
				OrgID: test.orgID,
			}).HasError())
			request := datasource.ReadRequest{
				Config: tfsdk.Config{Schema: schemaResponse.Schema, Raw: config.Raw},
			}
			response := datasource.ReadResponse{State: tfsdk.State{Schema: schemaResponse.Schema}}
			source.Read(ctx, request, &response)
			require.Equal(t, test.wantError, response.Diagnostics.HasError(), "%v", response.Diagnostics)
			if test.wantError {
				return
			}

			var state dataSourcesDataSourceModel
			require.False(t, response.State.Get(ctx, &state).HasError())
			require.Equal(t, test.wantOrgID+":data_sources", state.ID.ValueString())
			require.Equal(t, test.wantOrgID, state.OrgID.ValueString())
			require.NotNil(t, state.DataSources)
			require.Len(t, state.DataSources, test.wantCount)
			if test.wantCount > 0 {
				require.Equal(t, dataSourceListElementModel{
					ID:        types.Int64Value(1),
					UID:       types.StringValue("a-prometheus"),
					Name:      types.StringValue("Metrics"),
					Type:      types.StringValue("prometheus"),
					URL:       types.StringValue("http://prometheus:9090"),
					IsDefault: types.BoolValue(true),
				}, state.DataSources[0])
				require.Equal(t, "z-loki", state.DataSources[1].UID.ValueString())
			}
		})
	}
}
