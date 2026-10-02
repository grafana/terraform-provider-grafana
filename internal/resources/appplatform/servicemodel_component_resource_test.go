package appplatform

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	tfresource "github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func serviceModelTestSpecObject(t *testing.T, overrides map[string]attr.Value) types.Object {
	t.Helper()

	vals := map[string]attr.Value{
		"title":           types.StringNull(),
		"description":     types.StringNull(),
		"type":            types.StringValue("service"),
		"identifiers":     types.ListNull(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}),
		"owner_ref":       types.ObjectNull(serviceModelRefAttrTypes),
		"depends_on_refs": types.ListNull(types.ObjectType{AttrTypes: serviceModelRefAttrTypes}),
		"links":           types.ListNull(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}),
	}
	for k, v := range overrides {
		vals[k] = v
	}

	obj, diags := types.ObjectValue(serviceModelComponentSpecAttrTypes, vals)
	require.False(t, diags.HasError(), "building test spec object: %v", diags)
	return obj
}

func serviceModelTestRefObject(t *testing.T, apiVersion, kind, name string) attr.Value {
	t.Helper()
	obj, diags := types.ObjectValue(serviceModelRefAttrTypes, map[string]attr.Value{
		"api_version": types.StringValue(apiVersion),
		"kind":        types.StringValue(kind),
		"name":        types.StringValue(name),
	})
	require.False(t, diags.HasError())
	return obj
}

func serviceModelTestFullSpecObject(t *testing.T) types.Object {
	t.Helper()

	identifier, diags := types.ObjectValue(serviceModelIdentifierAttrTypes, map[string]attr.Value{
		"key":   types.StringValue("namespace"),
		"value": types.StringValue("checkout-prod"),
	})
	require.False(t, diags.HasError())

	link, diags := types.ObjectValue(serviceModelLinkAttrTypes, map[string]attr.Value{
		"url":   types.StringValue("https://example.com/checkout"),
		"title": types.StringValue("Repository"),
		"icon":  types.StringNull(),
		"type":  types.StringValue("repository"),
	})
	require.False(t, diags.HasError())

	return serviceModelTestSpecObject(t, map[string]attr.Value{
		"title":       types.StringValue("Checkout"),
		"description": types.StringValue("Handles checkout."),
		"type":        types.StringValue("service"),
		"identifiers": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}, []attr.Value{identifier}),
		"owner_ref":   serviceModelTestRefObject(t, "iam.grafana.app/v0alpha1", "Team", "team-checkout"),
		"depends_on_refs": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelRefAttrTypes}, []attr.Value{
			serviceModelTestRefObject(t, "servicemodel.ext.grafana.com/v1alpha1", "Component", "payments"),
		}),
		"links": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}, []attr.Value{link}),
	})
}

func strPtrOf(s string) *string {
	return &s
}

func TestParseServiceModelComponentSpecFull(t *testing.T) {
	var dst ServiceModelComponent
	diags := parseServiceModelComponentSpec(context.Background(), serviceModelTestFullSpecObject(t), &dst)
	require.False(t, diags.HasError(), "%v", diags)

	require.Equal(t, ServiceModelComponentSpec{
		Title:       strPtrOf("Checkout"),
		Description: strPtrOf("Handles checkout."),
		Type:        "service",
		Identifiers: []ServiceModelIdentifier{{Key: "namespace", Value: "checkout-prod"}},
		Metadata: &ServiceModelBackstageMetadata{
			Name: "",
			Links: []ServiceModelBackstageLink{{
				URL:   "https://example.com/checkout",
				Title: strPtrOf("Repository"),
				Type:  strPtrOf("repository"),
			}},
		},
		OwnerRef:      &ServiceModelRef{APIVersion: "iam.grafana.app/v0alpha1", Kind: "Team", Name: "team-checkout"},
		DependsOnRefs: []ServiceModelRef{{APIVersion: "servicemodel.ext.grafana.com/v1alpha1", Kind: "Component", Name: "payments"}},
	}, dst.Spec)
}

func TestParseServiceModelComponentSpecMinimal(t *testing.T) {
	var dst ServiceModelComponent
	spec := serviceModelTestSpecObject(t, map[string]attr.Value{
		"title": types.StringValue("Checkout"),
	})

	diags := parseServiceModelComponentSpec(context.Background(), spec, &dst)
	require.False(t, diags.HasError(), "%v", diags)

	require.Equal(t, ServiceModelComponentSpec{
		Title:    strPtrOf("Checkout"),
		Type:     "service",
		Metadata: &ServiceModelBackstageMetadata{Name: ""},
	}, dst.Spec)
}

func TestParseServiceModelComponentSpecEmptyLists(t *testing.T) {
	var dst ServiceModelComponent
	spec := serviceModelTestSpecObject(t, map[string]attr.Value{
		"title":           types.StringValue("Checkout"),
		"identifiers":     types.ListValueMust(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}, []attr.Value{}),
		"depends_on_refs": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelRefAttrTypes}, []attr.Value{}),
		"links":           types.ListValueMust(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}, []attr.Value{}),
	})

	diags := parseServiceModelComponentSpec(context.Background(), spec, &dst)
	require.False(t, diags.HasError(), "%v", diags)

	// Empty lists behave exactly like null lists: omitted on the wire.
	require.Nil(t, dst.Spec.Identifiers)
	require.Nil(t, dst.Spec.DependsOnRefs)
	require.Nil(t, dst.Spec.Metadata.Links)
}

func TestParseServiceModelComponentSpecDoesNotSetName(t *testing.T) {
	var dst ServiceModelComponent
	diags := parseServiceModelComponentSpec(context.Background(), serviceModelTestFullSpecObject(t), &dst)
	require.False(t, diags.HasError(), "%v", diags)

	// Object name comes from metadata.uid and is set by the shared machinery,
	// never by the spec parser.
	require.Empty(t, dst.ObjectMeta.Name)
}

func TestParseServiceModelComponentSpecWireJSON(t *testing.T) {
	for _, tc := range []struct {
		name     string
		spec     types.Object
		expected string
	}{
		{
			name: "minimal",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title": types.StringValue("Checkout"),
			}),
			expected: `{"title":"Checkout","metadata":{"name":""},"type":"service"}`,
		},
		{
			// An explicitly-empty description must survive on the wire.
			name: "empty description",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title":       types.StringValue("Checkout"),
				"description": types.StringValue(""),
			}),
			expected: `{"title":"Checkout","description":"","metadata":{"name":""},"type":"service"}`,
		},
		{
			name: "full",
			spec: serviceModelTestFullSpecObject(t),
			expected: `{"title":"Checkout","description":"Handles checkout.",` +
				`"identifiers":[{"key":"namespace","value":"checkout-prod"}],` +
				`"metadata":{"name":"","links":[{"url":"https://example.com/checkout","title":"Repository","type":"repository"}]},` +
				`"type":"service",` +
				`"ownerRef":{"apiVersion":"iam.grafana.app/v0alpha1","kind":"Team","name":"team-checkout"},` +
				`"dependsOnRefs":[{"apiVersion":"servicemodel.ext.grafana.com/v1alpha1","kind":"Component","name":"payments"}]}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var dst ServiceModelComponent
			diags := parseServiceModelComponentSpec(context.Background(), tc.spec, &dst)
			require.False(t, diags.HasError(), "%v", diags)

			raw, err := json.Marshal(dst.Spec)
			require.NoError(t, err)
			require.Equal(t, tc.expected, string(raw))
		})
	}
}

func TestSaveServiceModelComponentSpecFull(t *testing.T) {
	src := &ServiceModelComponent{
		Spec: ServiceModelComponentSpec{
			Title:       strPtrOf("Checkout"),
			Description: strPtrOf("Handles checkout."),
			Type:        "service",
			Identifiers: []ServiceModelIdentifier{{Key: "namespace", Value: "checkout-prod"}},
			Metadata: &ServiceModelBackstageMetadata{
				Name: "",
				Links: []ServiceModelBackstageLink{{
					URL:   "https://example.com/checkout",
					Title: strPtrOf("Repository"),
					Type:  strPtrOf("repository"),
				}},
			},
			OwnerRef:      &ServiceModelRef{APIVersion: "iam.grafana.app/v0alpha1", Kind: "Team", Name: "team-checkout"},
			DependsOnRefs: []ServiceModelRef{{APIVersion: "servicemodel.ext.grafana.com/v1alpha1", Kind: "Component", Name: "payments"}},
		},
	}

	var dst ResourceModel
	diags := saveServiceModelComponentSpec(context.Background(), src, &dst)
	require.False(t, diags.HasError(), "%v", diags)
	require.Equal(t, serviceModelTestFullSpecObject(t), dst.Spec)
}

func TestSaveServiceModelComponentSpecEmptyToNull(t *testing.T) {
	src := &ServiceModelComponent{
		Spec: ServiceModelComponentSpec{
			Title: strPtrOf("Checkout"),
			Type:  "service",
			// Description nil, lists empty, OwnerRef nil, Metadata nil.
			Identifiers:   []ServiceModelIdentifier{},
			DependsOnRefs: []ServiceModelRef{},
		},
	}

	var dst ResourceModel
	diags := saveServiceModelComponentSpec(context.Background(), src, &dst)
	require.False(t, diags.HasError(), "%v", diags)

	expected := serviceModelTestSpecObject(t, map[string]attr.Value{
		"title": types.StringValue("Checkout"),
	})
	require.Equal(t, expected, dst.Spec)
}

func TestSaveServiceModelComponentSpecMetadataPresentNoLinks(t *testing.T) {
	src := &ServiceModelComponent{
		Spec: ServiceModelComponentSpec{
			Title:    strPtrOf("Checkout"),
			Type:     "service",
			Metadata: &ServiceModelBackstageMetadata{Name: "some-name"},
		},
	}

	var dst ResourceModel
	diags := saveServiceModelComponentSpec(context.Background(), src, &dst)
	require.False(t, diags.HasError(), "%v", diags)

	// metadata.name is intentionally not surfaced; links stay null.
	expected := serviceModelTestSpecObject(t, map[string]attr.Value{
		"title": types.StringValue("Checkout"),
	})
	require.Equal(t, expected, dst.Spec)
}

func TestSaveServiceModelComponentSpecDropsUnexposedFields(t *testing.T) {
	// Simulates importing an object whose spec carries fields this resource
	// does not manage; decoding must tolerate them and saving must drop them.
	payload := `{
		"apiVersion": "servicemodel.ext.grafana.com/v1alpha1",
		"kind": "Component",
		"metadata": {"name": "checkout", "namespace": "stacks-1"},
		"spec": {
			"identity": "checkout",
			"title": "Checkout",
			"type": "service",
			"lifecycle": "production",
			"owner": "group:default/team-a",
			"backstageMetadata": {"name": "checkout", "tags": ["payments"]},
			"providesApis": ["checkout-api"]
		},
		"status": {}
	}`

	var src ServiceModelComponent
	require.NoError(t, json.Unmarshal([]byte(payload), &src))

	var dst ResourceModel
	diags := saveServiceModelComponentSpec(context.Background(), &src, &dst)
	require.False(t, diags.HasError(), "%v", diags)

	expected := serviceModelTestSpecObject(t, map[string]attr.Value{
		"title": types.StringValue("Checkout"),
	})
	require.Equal(t, expected, dst.Spec)
}

func TestServiceModelComponentSpecRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name string
		spec types.Object
	}{
		{name: "full", spec: serviceModelTestFullSpecObject(t)},
		{
			name: "minimal",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title": types.StringValue("Checkout"),
			}),
		},
		{
			// The UI stores explicitly-empty descriptions; "" and null are
			// distinct states and both must survive import round-trips.
			name: "description empty string",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title":       types.StringValue("Checkout"),
				"description": types.StringValue(""),
			}),
		},
		{
			// Server objects may hold non-default ref shapes; import must
			// reproduce them exactly rather than reverting to defaults.
			name: "non-default refs",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title":     types.StringValue("Checkout"),
				"owner_ref": serviceModelTestRefObject(t, "iam.grafana.app/v1beta1", "Group", "group-x"),
				"depends_on_refs": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelRefAttrTypes}, []attr.Value{
					serviceModelTestRefObject(t, "other.grafana.app/v1", "Thing", "thing-1"),
				}),
			}),
		},
		{
			name: "multiple identifiers keep order",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title": types.StringValue("Checkout"),
				"identifiers": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}, []attr.Value{
					types.ObjectValueMust(serviceModelIdentifierAttrTypes, map[string]attr.Value{"key": types.StringValue("zeta"), "value": types.StringValue("1")}),
					types.ObjectValueMust(serviceModelIdentifierAttrTypes, map[string]attr.Value{"key": types.StringValue("alpha"), "value": types.StringValue("2")}),
					types.ObjectValueMust(serviceModelIdentifierAttrTypes, map[string]attr.Value{"key": types.StringValue("mid"), "value": types.StringValue("3")}),
				}),
			}),
		},
		{
			name: "owner ref only",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title":     types.StringValue("Checkout"),
				"owner_ref": serviceModelTestRefObject(t, "iam.grafana.app/v0alpha1", "Team", "team-checkout"),
			}),
		},
		{
			name: "links only",
			spec: serviceModelTestSpecObject(t, map[string]attr.Value{
				"title": types.StringValue("Checkout"),
				"links": types.ListValueMust(types.ObjectType{AttrTypes: serviceModelLinkAttrTypes}, []attr.Value{
					types.ObjectValueMust(serviceModelLinkAttrTypes, map[string]attr.Value{
						"url":   types.StringValue("https://example.com"),
						"title": types.StringNull(),
						"icon":  types.StringNull(),
						"type":  types.StringNull(),
					}),
				}),
			}),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var obj ServiceModelComponent
			diags := parseServiceModelComponentSpec(context.Background(), tc.spec, &obj)
			require.False(t, diags.HasError(), "%v", diags)

			var dst ResourceModel
			diags = saveServiceModelComponentSpec(context.Background(), &obj, &dst)
			require.False(t, diags.HasError(), "%v", diags)

			// parse -> save must reproduce the original spec object exactly;
			// this is what keeps `terraform plan` clean after import.
			require.Equal(t, tc.spec, dst.Spec)
		})
	}
}

func TestServiceModelRequireNameWhenConfigured(t *testing.T) {
	refObject := func(name attr.Value) types.Object {
		return types.ObjectValueMust(serviceModelRefAttrTypes, map[string]attr.Value{
			"api_version": types.StringValue("iam.grafana.app/v0alpha1"),
			"kind":        types.StringValue("Team"),
			"name":        name,
		})
	}

	for _, tc := range []struct {
		name      string
		value     types.Object
		wantError bool
	}{
		{name: "block not configured", value: types.ObjectNull(serviceModelRefAttrTypes), wantError: false},
		{name: "name known and non-empty", value: refObject(types.StringValue("team-checkout")), wantError: false},
		// Unknown = reference to a not-yet-created resource (e.g.
		// grafana_team.x.team_uid at plan time); must not error.
		{name: "name unknown", value: refObject(types.StringUnknown()), wantError: false},
		{name: "name null", value: refObject(types.StringNull()), wantError: true},
		{name: "name empty", value: refObject(types.StringValue("")), wantError: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			req := validator.ObjectRequest{ConfigValue: tc.value}
			var resp validator.ObjectResponse
			serviceModelRequireNameWhenConfigured{}.ValidateObject(context.Background(), req, &resp)
			require.Equal(t, tc.wantError, resp.Diagnostics.HasError())
		})
	}
}

func TestServiceModelComponentResourceName(t *testing.T) {
	named := ServiceModelComponentResource()
	require.Equal(t, "grafana_apps_servicemodel_component_v1alpha1", named.Name)
	require.True(t, strings.HasPrefix(named.Name, "grafana_apps_servicemodel_"))
}

var serviceModelTestMetadataAttrTypes = map[string]attr.Type{
	"uuid":        types.StringType,
	"uid":         types.StringType,
	"folder_uid":  types.StringType,
	"version":     types.StringType,
	"url":         types.StringType,
	"annotations": types.MapType{ElemType: types.StringType},
}

func serviceModelTestMetadataObject(t *testing.T, uid attr.Value) types.Object {
	t.Helper()
	obj, diags := types.ObjectValue(serviceModelTestMetadataAttrTypes, map[string]attr.Value{
		"uuid":        types.StringNull(),
		"uid":         uid,
		"folder_uid":  types.StringNull(),
		"version":     types.StringNull(),
		"url":         types.StringNull(),
		"annotations": types.MapNull(types.StringType),
	})
	require.False(t, diags.HasError())
	return obj
}

func TestServiceModelComponentPlanChecks(t *testing.T) {
	for _, tc := range []struct {
		name         string
		data         ResourceModel
		wantError    bool
		wantWarnings int
	}{
		{
			name: "valid",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: serviceModelTestMetadataObject(t, types.StringValue("checkout-service")),
			},
		},
		{
			// A null spec never reaches apply: the framework rejects the
			// config first because spec.title is required. Our checks add
			// no diagnostics of their own for it.
			name: "null spec produces no diagnostics",
			data: ResourceModel{
				Spec:     types.ObjectNull(serviceModelComponentSpecAttrTypes),
				Metadata: serviceModelTestMetadataObject(t, types.StringValue("checkout-service")),
			},
		},
		{
			name: "invalid uid errors",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: serviceModelTestMetadataObject(t, types.StringValue("Checkout_Service")),
			},
			wantError: true,
		},
		{
			name: "null metadata is skipped",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: types.ObjectNull(serviceModelTestMetadataAttrTypes),
			},
		},
		{
			// A uid referencing a not-yet-created resource must not error.
			name: "unknown uid is skipped",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: serviceModelTestMetadataObject(t, types.StringUnknown()),
			},
		},
		{
			name: "null uid is skipped",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: serviceModelTestMetadataObject(t, types.StringNull()),
			},
		},
		{
			name: "unknown metadata is skipped",
			data: ResourceModel{
				Spec: serviceModelTestSpecObject(t, map[string]attr.Value{
					"title": types.StringValue("Checkout"),
				}),
				Metadata: types.ObjectUnknown(serviceModelTestMetadataAttrTypes),
			},
		},
		{
			name: "null spec does not mask an invalid uid",
			data: ResourceModel{
				Spec:     types.ObjectNull(serviceModelComponentSpecAttrTypes),
				Metadata: serviceModelTestMetadataObject(t, types.StringValue("Checkout_Service")),
			},
			wantError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := serviceModelComponentPlanChecks(context.Background(), tc.data)
			require.Equal(t, tc.wantError, diags.HasError())
			require.Equal(t, tc.wantWarnings, diags.WarningsCount())
		})
	}
}

func TestValidateServiceModelComponentUID(t *testing.T) {
	long := strings.Repeat("a", 63)
	for _, tc := range []struct {
		name  string
		uid   string
		valid bool
	}{
		{name: "typical", uid: "checkout-service", valid: true},
		{name: "two chars", uid: "ab", valid: true},
		{name: "digits", uid: "0service9", valid: true},
		{name: "63 chars", uid: long, valid: true},
		{name: "64 chars", uid: long + "a", valid: false},
		{name: "single char", uid: "a", valid: false},
		{name: "empty", uid: "", valid: false},
		{name: "uppercase", uid: "Checkout", valid: false},
		{name: "leading dash", uid: "-checkout", valid: false},
		{name: "trailing dash", uid: "checkout-", valid: false},
		{name: "dot", uid: "checkout.service", valid: false},
		{name: "underscore", uid: "checkout_service", valid: false},
		{name: "mid dash", uid: "a-b", valid: true},
		{name: "consecutive dashes", uid: "a--b", valid: true},
		{name: "two digits", uid: "12", valid: true},
		{name: "multibyte", uid: "café", valid: false},
		{name: "embedded newline", uid: "ab\ncd", valid: false},
		{name: "space", uid: "a b", valid: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			msg := validateServiceModelComponentUID(tc.uid)
			if tc.valid {
				require.Empty(t, msg)
			} else {
				require.NotEmpty(t, msg)
			}
		})
	}
}

// TestServiceModelComponentIdentifiersSizeValidator exercises the validators
// attached to the identifiers block in the actual schema, so removing or
// loosening the size cap fails this test.
func TestServiceModelComponentIdentifiersSizeValidator(t *testing.T) {
	var resp tfresource.SchemaResponse
	ServiceModelComponentResource().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())

	spec, ok := resp.Schema.Blocks["spec"].(schema.SingleNestedBlock)
	require.True(t, ok, "spec must be a single nested block")
	identifiersBlock, ok := spec.Blocks["identifiers"].(schema.ListNestedBlock)
	require.True(t, ok, "identifiers must be a list nested block")
	require.NotEmpty(t, identifiersBlock.Validators, "identifiers block must carry validators")

	identifier := func(i int) attr.Value {
		return types.ObjectValueMust(serviceModelIdentifierAttrTypes, map[string]attr.Value{
			"key":   types.StringValue(fmt.Sprintf("key-%d", i)),
			"value": types.StringValue("v"),
		})
	}
	runValidators := func(n int) bool {
		vals := make([]attr.Value, n)
		for i := range vals {
			vals[i] = identifier(i)
		}
		req := validator.ListRequest{
			ConfigValue: types.ListValueMust(types.ObjectType{AttrTypes: serviceModelIdentifierAttrTypes}, vals),
		}
		var vresp validator.ListResponse
		for _, v := range identifiersBlock.Validators {
			v.ValidateList(context.Background(), req, &vresp)
		}
		return vresp.Diagnostics.HasError()
	}

	require.False(t, runValidators(5), "five identifiers must pass")
	require.True(t, runValidators(6), "six identifiers must fail")
}

// TestServiceModelComponentAttrTypesMatchSchema guards the hand-maintained
// attribute-type map against drifting from the schema: a mismatch would only
// surface as a type-conversion error during import at a user's site.
func TestServiceModelComponentAttrTypesMatchSchema(t *testing.T) {
	var resp tfresource.SchemaResponse
	ServiceModelComponentResource().Resource.Schema(context.Background(), tfresource.SchemaRequest{}, &resp)
	require.False(t, resp.Diagnostics.HasError())

	spec, ok := resp.Schema.Blocks["spec"]
	require.True(t, ok, "schema must have a spec block")
	require.Equal(t,
		types.ObjectType{AttrTypes: serviceModelComponentSpecAttrTypes},
		spec.Type(),
		"serviceModelComponentSpecAttrTypes must match the schema's spec object type",
	)
}
