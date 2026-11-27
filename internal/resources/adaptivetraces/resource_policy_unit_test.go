package adaptivetraces

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/adaptivetracesapi"
)

const (
	testPolicyID   = "123e4567-e89b-12d3-a456-426614174000"
	testPolicyPath = "/adaptive-traces/api/v1/policies/" + testPolicyID
)

func testPolicyJSON(t *testing.T, policy adaptivetracesapi.Policy) string {
	t.Helper()
	if policy.ID == "" {
		policy.ID = testPolicyID
	}
	if policy.VersionCreatedAt == "" {
		policy.VersionCreatedAt = "2026-01-02T03:04:05Z"
		policy.VersionCreatedBy = "user@example.com"
	}
	out, err := json.Marshal(policy)
	require.NoError(t, err)
	return string(out)
}

func newTestPolicyResource(t *testing.T, handler http.HandlerFunc) *policyResource {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return &policyResource{client: adaptivetracesapi.NewClient("123:token", server.URL, nil, "test", nil)}
}

func testPolicySchema(t *testing.T) schema.Schema {
	t.Helper()
	resp := &resource.SchemaResponse{}
	(&policyResource{}).Schema(context.Background(), resource.SchemaRequest{}, resp)
	require.False(t, resp.Diagnostics.HasError())
	return resp.Schema
}

func testPolicyState(t *testing.T, model *policyModel) tfsdk.State {
	t.Helper()
	s := testPolicySchema(t)
	state := tfsdk.State{Schema: s, Raw: tftypes.NewValue(s.Type().TerraformType(context.Background()), nil)}
	if model != nil {
		require.False(t, state.Set(context.Background(), model).HasError())
	}
	return state
}

func testPolicyModel() *policyModel {
	return &policyModel{
		ID:               types.StringValue(testPolicyID),
		Name:             types.StringValue("Keep errors"),
		Type:             types.StringValue("status_code"),
		Body:             jsontypes.NewNormalizedValue(`{"status_codes":["ERROR"]}`),
		VersionCreatedAt: types.StringValue("2026-01-02T03:04:05Z"),
		VersionCreatedBy: types.StringValue("user@example.com"),
	}
}

func writeAPIError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	_, _ = fmt.Fprintf(w, `{"error": %q}`, message)
}

func TestUnitPolicySetFromPolicy(t *testing.T) {
	ctx := context.Background()

	t.Run("fresh model gets canonical body", func(t *testing.T) {
		var model policyModel
		diags := model.setFromPolicy(ctx, &adaptivetracesapi.Policy{
			ID:               testPolicyID,
			Type:             "string_attribute",
			Name:             "Checkout",
			Body:             json.RawMessage(`{"values": ["checkout"], "key": "service.name"}`),
			VersionCreatedAt: "2026-01-02T03:04:05Z",
			VersionCreatedBy: "user@example.com",
		})
		require.False(t, diags.HasError())
		require.Equal(t, testPolicyID, model.ID.ValueString())
		require.Equal(t, "Checkout", model.Name.ValueString())
		require.Equal(t, "string_attribute", model.Type.ValueString())
		require.Equal(t, `{"key":"service.name","values":["checkout"]}`, model.Body.ValueString())
		require.Equal(t, "2026-01-02T03:04:05Z", model.VersionCreatedAt.ValueString())
		require.Equal(t, "user@example.com", model.VersionCreatedBy.ValueString())
	})

	t.Run("equivalent body keeps the configured string", func(t *testing.T) {
		model := policyModel{
			Body: jsontypes.NewNormalizedValue("{\n  \"values\": [\"checkout\"],\n  \"key\": \"service.name\"\n}"),
		}
		before := model
		diags := model.setFromPolicy(ctx, &adaptivetracesapi.Policy{
			ID:   testPolicyID,
			Type: "string_attribute",
			Name: "Checkout",
			Body: json.RawMessage(`{"key": "service.name", "values": ["checkout"]}`),
		})
		require.False(t, diags.HasError())
		require.Equal(t, before.Body, model.Body)
	})

	t.Run("drift replaces the configured body", func(t *testing.T) {
		model := policyModel{
			Body: jsontypes.NewNormalizedValue(`{"key":"service.name","values":["checkout"]}`),
		}
		diags := model.setFromPolicy(ctx, &adaptivetracesapi.Policy{
			ID:   testPolicyID,
			Type: "string_attribute",
			Name: "Checkout",
			Body: json.RawMessage(`{"key": "service.name", "values": ["cart"]}`),
		})
		require.False(t, diags.HasError())
		require.Equal(t, `{"key":"service.name","values":["cart"]}`, model.Body.ValueString())
	})

	t.Run("missing version fields are null", func(t *testing.T) {
		var model policyModel
		diags := model.setFromPolicy(ctx, &adaptivetracesapi.Policy{ID: testPolicyID, Type: "diversity", Name: "d", Body: json.RawMessage(`{}`)})
		require.False(t, diags.HasError())
		require.True(t, model.VersionCreatedAt.IsNull())
		require.True(t, model.VersionCreatedBy.IsNull())
	})

	t.Run("invalid body", func(t *testing.T) {
		var model policyModel
		diags := model.setFromPolicy(ctx, &adaptivetracesapi.Policy{ID: testPolicyID, Body: json.RawMessage(`{`)})
		require.True(t, diags.HasError())
	})
}

func TestUnitPolicyRead(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		handler     http.HandlerFunc
		wantRemoved bool
		wantErr     string
		wantName    string
	}{
		"not found removes the resource": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantRemoved: true,
		},
		"active policy refreshes the state": {
			handler: func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodGet, req.Method)
				require.Equal(t, testPolicyPath, req.URL.Path)
				_, _ = w.Write([]byte(testPolicyJSON(t, adaptivetracesapi.Policy{
					Type: "status_code", Name: "Renamed", Body: json.RawMessage(`{"status_codes": ["ERROR"]}`),
				})))
			},
			wantName: "Renamed",
		},
		"API errors are reported": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeAPIError(w, http.StatusServiceUnavailable, "database unavailable")
			},
			wantErr: "database unavailable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestPolicyResource(t, tc.handler)
			state := testPolicyState(t, testPolicyModel())
			resp := &resource.ReadResponse{State: state}
			r.Read(ctx, resource.ReadRequest{State: state}, resp)

			if tc.wantErr != "" {
				require.True(t, resp.Diagnostics.HasError())
				require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tc.wantErr)
				return
			}
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			require.Equal(t, tc.wantRemoved, resp.State.Raw.IsNull())
			if tc.wantName != "" {
				var got policyModel
				require.False(t, resp.State.Get(ctx, &got).HasError())
				require.Equal(t, tc.wantName, got.Name.ValueString())
				require.Equal(t, `{"status_codes":["ERROR"]}`, got.Body.ValueString())
			}
		})
	}
}

func TestUnitPolicyDelete(t *testing.T) {
	ctx := context.Background()

	for name, tc := range map[string]struct {
		handler http.HandlerFunc
		wantErr string
	}{
		"deleted policy": {
			handler: func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, http.MethodDelete, req.Method)
				require.Equal(t, testPolicyPath, req.URL.Path)
				w.WriteHeader(http.StatusNoContent)
			},
		},
		"not found is reported": {
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantErr: "not found",
		},
		"API errors are reported": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeAPIError(w, http.StatusServiceUnavailable, "database unavailable")
			},
			wantErr: "database unavailable",
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestPolicyResource(t, tc.handler)
			state := testPolicyState(t, testPolicyModel())
			resp := &resource.DeleteResponse{State: state}
			r.Delete(ctx, resource.DeleteRequest{State: state}, resp)

			if tc.wantErr != "" {
				require.True(t, resp.Diagnostics.HasError())
				require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tc.wantErr)
				return
			}
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
		})
	}
}

func TestUnitPolicyImportStateInvalidID(t *testing.T) {
	r := newTestPolicyResource(t, func(http.ResponseWriter, *http.Request) {
		t.Error("the API must not be called for a malformed ID")
	})

	for name, id := range map[string]string{
		"empty":            "",
		"not a uuid":       "my-policy",
		"no hyphens":       strings.ReplaceAll(testPolicyID, "-", ""),
		"braces":           "{" + testPolicyID + "}",
		"urn":              "urn:uuid:" + testPolicyID,
		"trailing garbage": testPolicyID + "/",
	} {
		t.Run(name, func(t *testing.T) {
			resp := &resource.ImportStateResponse{State: testPolicyState(t, nil)}
			r.ImportState(context.Background(), resource.ImportStateRequest{ID: id}, resp)
			require.True(t, resp.Diagnostics.HasError())
			require.Equal(t, "Invalid import ID", resp.Diagnostics.Errors()[0].Summary())
			require.True(t, resp.State.Raw.IsNull())
		})
	}
}

func TestUnitPolicyImportState(t *testing.T) {
	ctx := context.Background()
	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	for name, tc := range map[string]struct {
		handler http.HandlerFunc
		wantErr string
	}{
		"policy": {
			handler: func(w http.ResponseWriter, req *http.Request) {
				require.Equal(t, testPolicyPath, req.URL.Path)
				_, _ = w.Write([]byte(testPolicyJSON(t, adaptivetracesapi.Policy{
					Type: "status_code", Name: "Keep errors", Body: json.RawMessage(`{"status_codes": ["ERROR"]}`),
				})))
			},
		},
		"not found": {
			handler: func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNotFound) },
			wantErr: "does not exist",
		},
		"expiring": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(testPolicyJSON(t, adaptivetracesapi.Policy{
					Type: "status_code", Name: "Keep errors", Body: json.RawMessage(`{"status_codes": ["ERROR"]}`), ExpiresAt: &future,
				})))
			},
			wantErr: "Expiring policies cannot be managed",
		},
		"policy created by the API": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				_, _ = w.Write([]byte(testPolicyJSON(t, adaptivetracesapi.Policy{
					Type: "anomaly", Name: "Anomalies", Body: json.RawMessage(`{"span_name": "a", "service_name": "b", "sample_errors": true}`),
				})))
			},
			wantErr: "created by the API",
		},
		"API error": {
			handler: func(w http.ResponseWriter, _ *http.Request) {
				writeAPIError(w, http.StatusForbidden, "missing scope adaptive-traces-policies:read")
			},
			wantErr: "missing scope",
		},
	} {
		t.Run(name, func(t *testing.T) {
			r := newTestPolicyResource(t, tc.handler)
			resp := &resource.ImportStateResponse{State: testPolicyState(t, nil)}
			r.ImportState(ctx, resource.ImportStateRequest{ID: testPolicyID}, resp)

			if tc.wantErr != "" {
				require.True(t, resp.Diagnostics.HasError())
				require.Contains(t, resp.Diagnostics.Errors()[0].Detail(), tc.wantErr)
				require.True(t, resp.State.Raw.IsNull())
				return
			}
			require.False(t, resp.Diagnostics.HasError(), resp.Diagnostics)
			var state policyModel
			require.False(t, resp.State.Get(ctx, &state).HasError())
			require.Equal(t, testPolicyID, state.ID.ValueString())
			require.Equal(t, "status_code", state.Type.ValueString())
			require.Equal(t, `{"status_codes":["ERROR"]}`, state.Body.ValueString())
		})
	}
}
