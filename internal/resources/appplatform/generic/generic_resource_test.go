package generic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/grafana/grafana-app-sdk/k8s"
	goapi "github.com/grafana/grafana-openapi-client-go/client"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	tfrsc "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
)

func TestResolveGenericInputMergesManifestAndOverrides(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"annotations": map[string]any{
				"from_manifest": "1",
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

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"annotations": map[string]any{
			"from_override":     "1",
			utils.AnnoKeyFolder: "folder-1",
		},
	})
	require.False(t, diags.HasError())

	spec, diags := goToDynamicValue(ctx, map[string]any{
		"nested": map[string]any{
			"change": "override",
		},
		"description": "added",
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
		Metadata: metadata,
		Spec:     spec,
	})
	require.False(t, diags.HasError())

	require.Equal(t, "iam.grafana.app", resolved.APIGroup)
	require.Equal(t, "v0alpha1", resolved.Version)
	require.Equal(t, "Team", resolved.Kind)
	require.Equal(t, "team-a", resolved.Name)
	require.Equal(t, map[string]any{
		"title":       "Team A",
		"description": "added",
		"nested": map[string]any{
			"keep":   "yes",
			"change": "override",
		},
	}, resolved.Object.Spec)

	meta, err := utils.MetaAccessor(resolved.Object)
	require.NoError(t, err)
	require.Equal(t, "folder-1", meta.GetFolder())
	require.Equal(t, "1", resolved.Object.GetAnnotations()["from_manifest"])
	require.Equal(t, "1", resolved.Object.GetAnnotations()["from_override"])
	require.Equal(t, "folder-1", resolved.Object.GetAnnotations()[utils.AnnoKeyFolder])
}

func TestResolveGenericInputAllowsEmptySpecOverrideMapToClearManifestField(t *testing.T) {
	ctx := context.Background()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
		},
		"spec": map[string]any{
			"nested": map[string]any{
				"keep": "manifest",
			},
		},
	})
	require.False(t, diags.HasError())

	spec, diags := goToDynamicValue(ctx, map[string]any{
		"nested": map[string]any{},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
		Spec:     spec,
	})
	require.False(t, diags.HasError())
	require.Equal(t, map[string]any{}, resolved.Object.Spec["nested"])
}

func TestResolveGenericInputAllowsEmptyMetadataOverrideMapToClearManifestField(t *testing.T) {
	ctx := context.Background()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "team-a",
			"annotations": map[string]any{
				"managed": "manifest",
			},
		},
	})
	require.False(t, diags.HasError())

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"annotations": map[string]any{},
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
		Metadata: metadata,
	})
	require.False(t, diags.HasError())
	require.Empty(t, resolved.Object.GetAnnotations())
}

func TestResolveGenericInputRejectsConflictingTopLevelMetadataUID(t *testing.T) {
	ctx := context.Background()

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name": "manifest-team",
		},
	})
	require.False(t, diags.HasError())

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"uid": "terraform-team",
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		Manifest: manifest,
		Metadata: metadata,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Conflicting metadata identifier")
}

func TestResolveGenericInputRejectsTopLevelMetadataNameAlias(t *testing.T) {
	ctx := context.Background()

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"name": "team-a",
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		APIGroup: types.StringValue("iam.grafana.app"),
		Version:  types.StringValue("v0alpha1"),
		Kind:     types.StringValue("Team"),
		Metadata: metadata,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputRejectsConflictingMetadataNameAndUID(t *testing.T) {
	ctx := context.Background()

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"name": "team-a",
		"uid":  "team-b",
	})
	require.False(t, diags.HasError())

	_, diags = resolveGenericInput(ctx, GenericResourceModel{
		APIGroup: types.StringValue("iam.grafana.app"),
		Version:  types.StringValue("v0alpha1"),
		Kind:     types.StringValue("Team"),
		Metadata: metadata,
	})
	require.True(t, diags.HasError())
}

func TestResolveGenericInputSupportsManifestMetadataUIDAlias(t *testing.T) {
	ctx := context.Background()

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

func TestResolveGenericInputRejectsSecureInManifest(t *testing.T) {
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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

func TestResolveGenericInputAcceptsArbitraryTopLevelMetadataField(t *testing.T) {
	ctx := context.Background()

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"uid":       "team-a",
		"namespace": "custom-ns",
	})
	require.False(t, diags.HasError())

	resolved, diags := resolveGenericInput(ctx, GenericResourceModel{
		APIGroup: types.StringValue("iam.grafana.app"),
		Version:  types.StringValue("v0alpha1"),
		Kind:     types.StringValue("Team"),
		Metadata: metadata,
	})
	require.False(t, diags.HasError())
	require.Equal(t, "custom-ns", resolved.Object.GetNamespace())
}

func TestResolveGenericInputRejectsNonStringMetadataLabels(t *testing.T) {
	ctx := context.Background()

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
	ctx := context.Background()

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

func TestResolvePluralUsesDiscovery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/apis/iam.grafana.app/v0alpha1", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{},
		},
	}

	plural, err := r.resolvePlural(context.Background(), "iam.grafana.app", "v0alpha1", "Team")
	require.NoError(t, err)
	require.Equal(t, "teams", plural)
}

func TestResolvePluralSendsConfiguredOrgIDHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/apis/iam.grafana.app/v0alpha1", req.URL.Path)
		require.Equal(t, "17", req.Header.Get(grafanaOrgIDHeader))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{OrgID: 17},
		},
	}

	plural, err := r.resolvePlural(context.Background(), "iam.grafana.app", "v0alpha1", "Team")
	require.NoError(t, err)
	require.Equal(t, "teams", plural)
}

func TestResolvePluralRejectsClusterScopedKind(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/apis/iam.grafana.app/v0alpha1", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":false}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{},
		},
	}

	_, err = r.resolvePlural(context.Background(), "iam.grafana.app", "v0alpha1", "Team")
	require.Error(t, err)
	require.Contains(t, err.Error(), "cluster-scoped")
}

func TestResolveNamespaceUsesConfiguredStackIDWithoutBootdata(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatalf("bootdata should not be called when stack_id is explicitly configured")
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaStackID:      123,
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{},
		},
	}

	namespace, diags := r.resolveNamespace(context.Background())
	require.False(t, diags.HasError())
	require.Equal(t, "stacks-123", namespace)
}

func TestResolveNamespaceBootdataSendsConfiguredOrgIDHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/bootdata", r.URL.Path)
		require.Equal(t, "17", r.Header.Get(grafanaOrgIDHeader))
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"settings":{"namespace":"stacks-321"}}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{OrgID: 17},
		},
	}

	namespace, diags := r.resolveNamespace(context.Background())
	require.False(t, diags.HasError())
	require.Equal(t, "stacks-321", namespace)
}

func TestResolveNamespaceErrorsWhenAutodiscoveryFailsWithoutExplicitOrgID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/bootdata", r.URL.Path)
		http.Error(w, "blocked", http.StatusUnauthorized)
	}))
	defer server.Close()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	r := &genericResource{
		client: &common.Client{
			GrafanaOrgID:        1,
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{},
		},
	}

	_, diags := r.resolveNamespace(context.Background())
	require.True(t, diags.HasError())
}

func TestResolveResourceRejectsManifestNamespaceOutsideProviderContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/apis/iam.grafana.app/v0alpha1", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx := context.Background()
	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
		"metadata": map[string]any{
			"name":      "team-a",
			"namespace": "custom-ns",
		},
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, server, genericResourceTestProviderConfig{OrgID: 2, OrgConfigured: true})
	_, diags = resource.resolveResource(ctx, GenericResourceModel{
		Manifest: manifest,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Namespace does not match provider context")
}

func TestResolveResourceRejectsTopLevelNamespaceOutsideProviderContextAtTopLevelPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		require.Equal(t, "/apis/iam.grafana.app/v0alpha1", req.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`))
		require.NoError(t, err)
	}))
	defer server.Close()

	ctx := context.Background()
	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"uid":       "team-a",
		"namespace": "custom-ns",
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, server, genericResourceTestProviderConfig{OrgID: 2, OrgConfigured: true})
	_, diags = resource.resolveResource(ctx, GenericResourceModel{
		APIGroup: types.StringValue("iam.grafana.app"),
		Version:  types.StringValue("v0alpha1"),
		Kind:     types.StringValue("Team"),
		Metadata: metadata,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Namespace does not match provider context")
	diagWithPath, ok := diags[0].(diag.DiagnosticWithPath)
	require.True(t, ok)
	require.Equal(t, path.Root("metadata").AtMapKey("namespace"), diagWithPath.Path())
}

func TestResolveResourceFailsNamespaceAutodiscoveryBeforeRouteDiscovery(t *testing.T) {
	discoveryCalls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/bootdata":
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"settings":{"namespace":"org-17"}}`))
			require.NoError(t, err)
		case "/apis/iam.grafana.app/v0alpha1":
			discoveryCalls++
			http.Error(w, "discovery should not run when namespace autodiscovery fails", http.StatusInternalServerError)
		default:
			t.Fatalf("unexpected request path %q", req.URL.Path)
		}
	}))
	defer server.Close()

	ctx := context.Background()
	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"uid": "team-a",
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, server, genericResourceTestProviderConfig{})
	_, diags = resource.resolveResource(ctx, GenericResourceModel{
		APIGroup: types.StringValue("iam.grafana.app"),
		Version:  types.StringValue("v0alpha1"),
		Kind:     types.StringValue("Team"),
		Metadata: metadata,
	})
	require.True(t, diags.HasError())
	requireDiagnosticsContain(t, diags, "Set either provider-level `org_id` or `stack_id` explicitly")
	require.Equal(t, 0, discoveryCalls)
}

func TestImportStateRejectsFivePartImportID(t *testing.T) {
	resource := &genericResource{}
	resp := newGenericImportStateResponse(t, resource)

	resource.ImportState(context.Background(), tfrsc.ImportStateRequest{
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
			resp := newGenericImportStateResponse(t, resource)

			resource.ImportState(context.Background(), tfrsc.ImportStateRequest{
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
		case "/apis/iam.grafana.app/v0alpha1":
			w.Header().Set("Content-Type", "application/json")
			_, err := w.Write([]byte(`{"resources":[{"name":"teams","kind":"Team","namespaced":true}]}`))
			require.NoError(t, err)
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

	ctx := context.Background()
	tfSchema := newGenericResourceSchema(t)

	manifest, diags := goToDynamicValue(ctx, map[string]any{
		"apiVersion": "iam.grafana.app/v0alpha1",
		"kind":       "Team",
	})
	require.False(t, diags.HasError())

	metadata, diags := goToDynamicValue(ctx, map[string]any{
		"uid": "team-a",
	})
	require.False(t, diags.HasError())

	resource := newGenericResourceForTests(t, server, genericResourceTestProviderConfig{OrgID: 2, OrgConfigured: true})
	req := tfrsc.DeleteRequest{
		State: newGenericStateFromModel(t, tfSchema, GenericResourceModel{
			ID:            types.StringValue("uuid-1"),
			Manifest:      manifest,
			Metadata:      metadata,
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
	OrgID         int64
	OrgConfigured bool
	StackID       int64
}

func newGenericResourceForTests(
	t *testing.T,
	server *httptest.Server,
	cfg genericResourceTestProviderConfig,
) *genericResource {
	t.Helper()

	parsedURL, err := url.Parse(server.URL)
	require.NoError(t, err)

	return &genericResource{
		client: &common.Client{
			GrafanaAPIURLParsed: parsedURL,
			GrafanaAPIConfig:    &goapi.TransportConfig{},
			GrafanaAppPlatformAPI: k8s.NewClientRegistry(rest.Config{
				Host:    server.URL,
				APIPath: "/apis",
			}, k8s.ClientConfig{}),
			GrafanaAppPlatformAPIClientID: "terraform-provider-grafana-test",
			GrafanaOrgID:                  cfg.OrgID,
			GrafanaOrgIDConfigured:        cfg.OrgConfigured,
			GrafanaStackID:                cfg.StackID,
		},
	}
}

func newGenericResourceSchema(t *testing.T) schema.Schema {
	t.Helper()

	var schemaResp tfrsc.SchemaResponse
	(&genericResource{}).Schema(context.Background(), tfrsc.SchemaRequest{}, &schemaResp)
	require.False(t, schemaResp.Diagnostics.HasError(), schemaResp.Diagnostics.Errors())
	return schemaResp.Schema
}

func newGenericPlanFromModel(t *testing.T, tfSchema schema.Schema, model GenericResourceModel) tfsdk.Plan {
	t.Helper()

	plan := tfsdk.Plan{
		Schema: tfSchema,
		Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.Background()), nil),
	}
	diags := plan.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags.Errors())
	return plan
}

func newGenericConfigFromModel(t *testing.T, tfSchema schema.Schema, model GenericResourceModel) tfsdk.Config {
	t.Helper()

	plan := newGenericPlanFromModel(t, tfSchema, model)
	return tfsdk.Config{
		Schema: tfSchema,
		Raw:    plan.Raw,
	}
}

func newGenericStateFromModel(t *testing.T, tfSchema schema.Schema, model GenericResourceModel) tfsdk.State {
	t.Helper()

	state := tfsdk.State{
		Schema: tfSchema,
		Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.Background()), nil),
	}
	diags := state.Set(context.Background(), &model)
	require.False(t, diags.HasError(), diags.Errors())
	return state
}

func newGenericNullState(tfSchema schema.Schema) tfsdk.State {
	return tfsdk.State{
		Schema: tfSchema,
		Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.Background()), nil),
	}
}

func newGenericImportStateResponse(t *testing.T, resource *genericResource) tfrsc.ImportStateResponse {
	t.Helper()

	tfSchema := newGenericResourceSchema(t)

	resp := tfrsc.ImportStateResponse{
		State: tfsdk.State{
			Schema: tfSchema,
			Raw:    tftypes.NewValue(tfSchema.Type().TerraformType(context.Background()), nil),
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

func decodeJSONBody(t *testing.T, req *http.Request) map[string]any {
	t.Helper()

	body, err := io.ReadAll(req.Body)
	require.NoError(t, err)
	req.Body.Close()

	var decoded map[string]any
	err = json.Unmarshal(body, &decoded)
	require.NoError(t, err)
	return decoded
}
