package generic

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/client"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	tfrsc "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func TestResolveGenericInputFromManifest(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"annotations": map[string]any{
				"from_manifest":     "1",
				utils.AnnoKeyFolder: "folder-1",
			},
		},
		"spec": map[string]any{
			"title": "Team A",
			"nested": map[string]any{
				"keep":   "yes",
				"change": "manifest",
			},
		},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.False(t, diags.HasError())

	require.Equal(t, "iam.grafana.app", resolved.APIGroup)
	require.Equal(t, "v0alpha1", resolved.Version)
	require.Equal(t, "Team", resolved.Kind)
	require.Equal(t, "team-a", resolved.Name)
	require.Equal(t, map[string]any{
		"title": "Team A",
		"nested": map[string]any{
			"keep":   "yes",
			"change": "manifest",
		},
	}, resolved.Object.Spec)

	meta, err := utils.MetaAccessor(resolved.Object)
	require.NoError(t, err)
	require.Equal(t, "folder-1", meta.GetFolder())
	require.Equal(t, "1", resolved.Object.GetAnnotations()["from_manifest"])
	require.Equal(t, "folder-1", resolved.Object.GetAnnotations()[utils.AnnoKeyFolder])
}

func TestResolveGenericInputSupportsManifestMetadataUIDAlias(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"uid": "team-a",
		},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "team-a", resolved.Name)
}

func TestResolveGenericInputRejectsConflictingManifestNameAndUID(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"uid":  "team-b",
		},
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputRejectsSecureInManifest(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
		},
		"secure": map[string]any{
			"api_token": map[string]any{
				"create": "secret",
			},
		},
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputAcceptsIgnoredManifestStatus(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"status": map[string]any{
			"phase": "ready",
		},
		"metadata": map[string]any{
			"name": "team-a",
		},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "team-a", resolved.Name)
}

func TestResolveGenericInputRejectsUnsupportedManifestField(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"data": map[string]any{
			"unexpected": true,
		},
		"metadata": map[string]any{
			"name": "team-a",
		},
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputAcceptsManifestServerMetadataField(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name":            "team-a",
			"resourceVersion": "12",
		},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "team-a", resolved.Name)
	require.Equal(t, "12", resolved.Object.GetResourceVersion())
}

func TestResolveGenericInputRejectsNonStringMetadataLabels(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"labels": map[string]any{
				"tier": true,
			},
		},
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputRejectsNonStringMetadataAnnotations(t *testing.T) {
	ctx := context.TODO()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"annotations": map[string]any{
				"tier": 7,
			},
		},
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
}

func TestRefreshConfigScopedSpecDetectsNestedServerAddedFields(t *testing.T) {
	configSpec := map[string]any{
		"title": "My Dashboard",
		"timeSettings": map[string]any{
			"from": "now-6h",
			"to":   "now",
		},
	}

	liveSpec := map[string]any{
		"title": "My Dashboard",
		"timeSettings": map[string]any{
			"from":                 "now-6h",
			"to":                   "now",
			"timezone":             "UTC",       // server-added nested field
			"autoRefreshIntervals": []any{"5s"}, // server-added nested field
		},
		"editable": true, // server-added top-level field (handled by refreshManifestState)
	}

	refreshed := refreshConfigScopedSpec(configSpec, liveSpec)

	// Nested server-added fields must appear in refreshed spec for drift detection.
	ts, ok := refreshed["timeSettings"].(map[string]any)
	require.True(t, ok, "expected timeSettings to be a map")
	require.Equal(t, "now-6h", ts["from"])
	require.Equal(t, "now", ts["to"])
	require.Equal(t, "UTC", ts["timezone"], "nested server-added field 'timezone' should be included for drift detection")
	require.Equal(t, []any{"5s"}, ts["autoRefreshIntervals"], "nested server-added field 'autoRefreshIntervals' should be included for drift detection")

	// Top-level server-added field is NOT included by refreshConfigScopedSpec
	// (that's handled separately by refreshManifestState's top-level loop).
	_, hasEditable := refreshed["editable"]
	require.False(t, hasEditable, "refreshConfigScopedSpec should not add top-level server keys — that's refreshManifestState's job")
}

func TestRefreshConfigScopedSpecNestedServerAddedFieldsMissedWithoutFix(t *testing.T) {
	// This test documents the exact scenario the reviewer flagged:
	// config has timeSettings.from and timeSettings.to, server adds
	// timeSettings.fiscalYearStartMonth. Without the recursive fix,
	// this nested addition would be silently dropped.
	configSpec := map[string]any{
		"timeSettings": map[string]any{
			"from": "now-6h",
		},
	}

	liveSpec := map[string]any{
		"timeSettings": map[string]any{
			"from":                 "now-6h",
			"fiscalYearStartMonth": float64(0),
		},
	}

	refreshed := refreshConfigScopedSpec(configSpec, liveSpec)
	ts := refreshed["timeSettings"].(map[string]any)

	require.Contains(t, ts, "fiscalYearStartMonth",
		"server-added nested field 'fiscalYearStartMonth' under configured 'timeSettings' must be detected as drift")
}

func TestValidateGenericSecureConfigValueRejectsInvalidKey(t *testing.T) {
	secure := types.DynamicValue(types.ObjectValueMust(map[string]attr.Type{
		"api_token": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"invalid": types.StringType,
			},
		},
	}, map[string]attr.Value{
		"api_token": types.ObjectValueMust(map[string]attr.Type{
			"invalid": types.StringType,
		}, map[string]attr.Value{
			"invalid": types.StringValue("secret"),
		}),
	}))

	diags := validateGenericSecureConfigValue(secure)
	require.True(t, diags.HasError())
}

func TestValidateGenericSecureConfigValueRejectsEmptyObject(t *testing.T) {
	secure := types.DynamicValue(types.ObjectValueMust(map[string]attr.Type{
		"api_token": types.ObjectType{AttrTypes: map[string]attr.Type{}},
	}, map[string]attr.Value{
		"api_token": types.ObjectValueMust(map[string]attr.Type{}, map[string]attr.Value{}),
	}))

	diags := validateGenericSecureConfigValue(secure)
	require.True(t, diags.HasError())
}

func TestValidateGenericSecureConfigValueRejectsNullName(t *testing.T) {
	secure := types.DynamicValue(types.ObjectValueMust(map[string]attr.Type{
		"api_token": types.ObjectType{
			AttrTypes: map[string]attr.Type{
				"name": types.StringType,
			},
		},
	}, map[string]attr.Value{
		"api_token": types.ObjectValueMust(map[string]attr.Type{
			"name": types.StringType,
		}, map[string]attr.Value{
			"name": types.StringNull(),
		}),
	}))

	diags := validateGenericSecureConfigValue(secure)
	require.True(t, diags.HasError())
}

func TestResolveNamespaceFallsBackToConfiguredStackID(t *testing.T) {
	r := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		StackID: 123,
		APICalls: map[string][]byte{
			"/bootdata": []byte(`{"settings":{"namespace":"default"}}`),
		},
	})

	namespace, diags := r.resolveNamespace(context.TODO())
	require.False(t, diags.HasError())
	require.Equal(t, "stacks-123", namespace)
}

func TestResolveNamespaceErrorsOnStackIDMismatch(t *testing.T) {
	r := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		StackID: 99,
		APICalls: map[string][]byte{
			"/bootdata": []byte(`{"settings":{"namespace":"stacks-42"}}`),
		},
	})

	_, diags := r.resolveNamespace(context.TODO())
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Stack ID mismatch")
}

func TestResolveNamespaceFallsBackToOrgIDWhenBootdataFails(t *testing.T) {
	r := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		OrgID: 1,
		APICalls: map[string][]byte{
			"/bootdata": []byte(`{"settings":{"namespace":"not a namespace"}}`),
		},
	})

	namespace, diags := r.resolveNamespace(context.TODO())
	require.False(t, diags.HasError())
	require.Equal(t, "default", namespace) // OrgNamespaceFormatter(1) returns "default"
}

func TestResolveNamespaceErrorsWhenAllFallbacksFail(t *testing.T) {
	r := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		APICalls: map[string][]byte{
			"/bootdata": []byte(`{"settings":{"namespace":"not a namespace"}}`),
		},
	})

	_, diags := r.resolveNamespace(context.TODO())
	require.True(t, diags.HasError())
}

func TestResolveResourceRejectsManifestNamespaceOutsideProviderContext(t *testing.T) {
	ctx := context.TODO()
	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name":      "team-a",
			"namespace": "custom-ns",
		},
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		OrgID: 2,
		APICalls: map[string][]byte{
			"/bootdata":                      []byte(`{"settings":{"namespace":"default"}}`),
			"/apis/iam.grafana.app/v0alpha1": []byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`),
		},
	})

	_, diags = resource.resolveResource(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Namespace does not match provider context")
}

func TestResolveResourceFailsNamespaceAutodiscoveryBeforeRouteDiscovery(t *testing.T) {
	ctx := context.TODO()
	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
		},
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		APICalls: map[string][]byte{
			"/bootdata": []byte(`{"settings":{"namespace":"org-17"}}`),
		},
	})
	_, diags = resource.resolveResource(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Set either provider-level `org_id` or `stack_id` explicitly")
}

func TestImportStateRejectsFivePartImportID(t *testing.T) {
	resource := &genericResource{}
	resp := newGenericImportStateResponse(t)

	resource.ImportState(context.TODO(), tfrsc.ImportStateRequest{
		ID: "iam.grafana.app/v0alpha1/Team/teams/team-a",
	}, &resp)
	require.True(t, resp.Diagnostics.HasError())
	requireDiagnosticsContain(t, resp.Diagnostics, "Invalid import ID")
}

func TestImportStateRejectsEmptyImportSegments(t *testing.T) {
	testCases := []string{
		"iam.grafana.app/v0alpha1/Team/",
		"/v0alpha1/Team/team-a",
		"iam.grafana.app//Team/team-a",
		"iam.grafana.app/v0alpha1//team-a",
	}

	for _, importID := range testCases {
		t.Run(importID, func(t *testing.T) {
			resource := &genericResource{}
			resp := newGenericImportStateResponse(t)

			resource.ImportState(context.TODO(), tfrsc.ImportStateRequest{
				ID: importID,
			}, &resp)
			require.True(t, resp.Diagnostics.HasError())
			requireDiagnosticsContain(t, resp.Diagnostics, "Invalid import ID")
		})
	}
}

func TestDeleteErrorsWhenUIDPreconditionDetectsReplacement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/apis/iam.grafana.app/v0alpha1/namespaces/org-2/teams/team-a":
			require.Equal(t, http.MethodDelete, req.Method)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			_, err := w.Write([]byte(`{"kind":"Status","apiVersion":"v1","metadata":{},"status":"Failure","message":"uid precondition failed","reason":"Conflict","code":409}`))
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request path %q", req.URL.Path)
		}
	}))
	defer server.Close()

	ctx := context.TODO()
	tfSchema := newGenericResourceSchema(t)

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
		},
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, genericResourceTestProviderConfig{
		Host:  server.URL,
		OrgID: 2,
		APICalls: map[string][]byte{
			"/bootdata":                      []byte(`{"settings":{"namespace":"default"}}`),
			"/apis/iam.grafana.app/v0alpha1": []byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`),
		},
	})
	req := tfrsc.DeleteRequest{
		State: newGenericStateFromModel(t, tfSchema, GenericResourceModel{
			ID:            types.StringValue("uuid-1"),
			Manifest:      manifest,
			Secure:        types.DynamicNull(),
			SecureVersion: types.Int64Null(),
		}),
	}
	resp := tfrsc.DeleteResponse{}

	resource.Delete(ctx, req, &resp)
	require.True(t, resp.Diagnostics.HasError())
	requireDiagnosticsContain(t, resp.Diagnostics, "Resource replaced outside Terraform")
}

type genericResourceTestProviderConfig struct {
	Host     string
	OrgID    int64
	StackID  int64
	APICalls map[string][]byte
}

type commonClient struct {
	grafanaGetFunc func(subpath string) ([]byte, error)
}

func (c *commonClient) GrafanaGet(_ context.Context, subpath string) ([]byte, error) {
	return c.grafanaGetFunc(subpath)
}

func newGenericResourceForTests(t *testing.T, cfg genericResourceTestProviderConfig) *genericResource {
	t.Helper()

	appPlatformClient := client.New(rest.Config{
		Host:    cfg.Host,
		APIPath: "/apis",
	}, &commonClient{
		grafanaGetFunc: func(subpath string) ([]byte, error) {
			if response, ok := cfg.APICalls[subpath]; ok {
				return response, nil
			}

			return nil, fmt.Errorf("unexpected call to '%s'", subpath)
		},
	})

	return &genericResource{
		client: &common.Client{
			GrafanaAPIConfig:              &goapi.TransportConfig{},
			GrafanaAppPlatformAPI:         appPlatformClient,
			GrafanaAppPlatformAPIClientID: "terraform-provider-grafana-test",
			GrafanaOrgID:                  cfg.OrgID,
			GrafanaStackID:                cfg.StackID,
		},
	}
}

func newGenericResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	var schemaResp tfrsc.SchemaResponse
	(&genericResource{}).Schema(context.TODO(), tfrsc.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), schemaResp.Diagnostics.Errors())
	return schemaResp.Schema
}

func newGenericStateFromModel(t *testing.T, tfSchema schema.Schema, model GenericResourceModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{
		Schema: tfSchema,
		Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.TODO()), nil),
	}
	diags := state.Set(context.TODO(), &model)
	require.False(t, diags.HasError(), diags.Errors())
	return state
}

func newGenericImportStateResponse(t *testing.T) tfrsc.ImportStateResponse {
	t.Helper()

	tfSchema := newGenericResourceSchema(t)
	resp := tfrsc.ImportStateResponse{
		State: tfsdk.State{
			Schema: tfSchema,
			Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.TODO()), nil),
		},
	}
	return resp
}

func requireDiagnosticsContain(t *testing.T, diags diag.Diagnostics, needle string) {
	t.Helper()

	for _, diagnostic := range diags {
		if strings.Contains(diagnostic.Summary(), needle) || strings.Contains(diagnostic.Detail(), needle) {
			return
		}
	}

	t.Fatalf("expected diagnostics to contain %q, got %#v", needle, diags)
}
