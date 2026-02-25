# AppPlatform Resource Gotchas

Known pitfalls when implementing AppPlatform resources. Read this before generating any code.

---

## 1. Read does NOT refresh spec from API

**Problem**: The generic `Read` implementation in `resource.go` only refreshes **metadata** from the API response (UUID, version, URL, annotations). The `Spec` field in state is **never updated** from the API.

**Why**: AppPlatform resources are designed so that Terraform is the source of truth for spec. The spec is only written into state during Create/Update (from the plan) and ImportState (via SpecSaver).

**Implication**: Do not write a SpecSaver that makes assumptions about round-tripping all fields. Fields that the API computes or transforms will appear as a diff on the next plan if you naively save them.

**Reference**: `resource.go` — `Read` method only calls `saveMetadataToState`.

---

## 2. SpecSaver is ONLY called during ImportState

**Problem**: Many engineers expect SpecSaver to be called during `Read`, like in standard Provider Framework resources. It is not.

**Why**: See gotcha #1 — spec is not read from API.

**Implication**: If SpecSaver has a bug, it only manifests during `terraform import`, not during normal `apply`/`plan`. Always test import explicitly.

**Test pattern** (from `alertenrichment_resource_acc_test.go`):
```go
{
    ResourceName:      resourceName,
    ImportState:       true,
    ImportStateVerify: true,
    ImportStateVerifyIgnore: []string{
        "options.%",
        "options.overwrite",
    },
    ImportStateIdFunc: importStateIDFunc(resourceName),
},
```

---

## 3. ObjectAsOptions required in SpecParser

**Problem**: Without `UnhandledNullAsEmpty` and `UnhandledUnknownAsEmpty`, parsing a `types.Object` into a struct will fail with confusing errors if the object has null/unknown attributes.

**Always use this pattern** in SpecParser:
```go
if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
    UnhandledNullAsEmpty:    true,
    UnhandledUnknownAsEmpty: true,
}); diag.HasError() {
    return diag
}
```

Also use in SpecSaver when calling `dst.Spec.As(...)`.

---

## 4. AttrTypes must match SpecAttributes keys exactly

**Problem**: When building a `types.ObjectValue()` or `types.ObjectValueFrom()` in SpecSaver, the `map[string]attr.Type{}` must have **exactly the same keys** as the `SpecAttributes` map in your schema.

**Symptom**: Build succeeds but terraform apply panics with "mismatch between schema and state" or a nil pointer dereference.

**Wrong** (missing a key, or extra key):
```go
spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
    "title": types.StringType,
    // "interval" missing — will panic at runtime
}, &data)
```

**Correct**:
```go
spec, diags := types.ObjectValueFrom(ctx, map[string]attr.Type{
    "title":    types.StringType,
    "interval": types.StringType,
    "items":    types.ListType{ElemType: PlaylistItemType},
}, &data)
```

**Pro tip**: Define a package-level `var <Name>SpecAttrTypes = map[string]attr.Type{...}` and use it in both the schema and SpecSaver to keep them in sync.

---

## 5. Import test needs importStateIDFunc

**Problem**: By default, Terraform import uses the resource's `id` attribute. AppPlatform resources set `id` to the Kubernetes UUID (a generated value). To import by name/uid, you need the K8s metadata `uid` (a human-assigned identifier).

**Why**: The K8s API is queried by UID (the K8s resource name), not UUID (the K8s metadata UUID).

**Pattern** (from `alertenrichment_resource_acc_test.go`):
```go
func importStateIDFunc(resourceName string) terraformresource.ImportStateIdFunc {
    return func(s *terraform.State) (string, error) {
        rs, ok := s.RootModule().Resources[resourceName]
        if !ok {
            return "", fmt.Errorf("resource not found: %s", resourceName)
        }
        uid := rs.Primary.Attributes["metadata.uid"]
        if uid == "" {
            return "", fmt.Errorf("UID is empty in resource %s", resourceName)
        }
        return uid, nil
    }
}
```

This function is defined in `alertenrichment_resource_acc_test.go` and is available to all tests in the `appplatform_test` package.

---

## 6. ImportStateVerifyIgnore for options

**Problem**: The `options` block (with `overwrite`) is not returned by the API, so it will always differ between the imported state and the actual config.

**Always include**:
```go
ImportStateVerifyIgnore: []string{
    "options.%",
    "options.overwrite",
},
```

---

## 7. Resource naming uses first group segment only

**Problem**: The TF resource name formula uses only the first dot-segment of the API group.

**Formula** (`resource.go:formatResourceType`):
```go
g := strings.Split(kind.Group(), ".")[0]
return fmt.Sprintf("grafana_apps_%s_%s_%s", g, strings.ToLower(kind.Kind()), kind.Version())
```

**Examples**:
- `rules.alerting.grafana.app` → first segment is `rules` → `grafana_apps_rules_*`
- `notifications.alerting.grafana.app` → first segment is `notifications` → `grafana_apps_notifications_*`
- `alerting.grafana.app` → first segment is `alerting` → `grafana_apps_alerting_*`

---

## 8. Nullable optional fields in SpecSaver

**Problem**: When saving optional fields that may be absent, using Go zero values (`""`, `0`, `false`) instead of the null/unknown variants causes Terraform to always see a diff.

**Correct patterns**:
```go
// Optional string field
if src.Spec.SomeField != "" {
    data.SomeField = types.StringValue(src.Spec.SomeField)
} else {
    data.SomeField = types.StringNull()
}

// Optional pointer field
if src.Spec.SomePtr != nil {
    data.SomeField = types.StringValue(*src.Spec.SomePtr)
} else {
    data.SomeField = types.StringNull()
}

// Optional list
if len(src.Spec.Items) > 0 {
    items, diags := types.ListValueFrom(ctx, itemType, items)
    // ...
} else {
    data.Items = types.ListNull(itemType)
}
```

---

## 9. Test package is `appplatform_test`

**Problem**: Acceptance test files must use the `appplatform_test` external test package, not `appplatform`.

**Why**: Allows testing the public API only; prevents accidentally relying on internal symbols.

**Correct**:
```go
package appplatform_test
```

**Wrong**:
```go
package appplatform
```

---

## 10. examples_test.go gating

**Problem**: Every resource example is automatically tested by `TestAccExamples` in `internal/resources/examples_test.go`. If your resource needs gating different from the category default, you must add an explicit `case`.

**When to add a case**:
- Resource requires Enterprise tests but the category default is OSS
- Resource requires a specific minimum Grafana version
- Resource is behind a feature flag and should be skipped for now

**Pattern**:
```go
{
    category: "Alerting",
    testCheck: func(t *testing.T, filename string) {
        switch {
        case strings.Contains(filename, "grafana_apps_alertenrichment"):
            testutils.CheckEnterpriseTestsEnabled(t, ">=12.2.0")
        case strings.Contains(filename, "grafana_apps_rules"):
            t.Skip() // TODO: Enable once feature flag removed
        default:
            testutils.CheckOSSTestsEnabled(t, ">=11.0.0")
        }
    },
},
```

---

## 11. Hand-rolled K8s types pattern

If the SDK package doesn't provide `<Kind>`, `<Kind>List`, `<Kind>Kind()`, you must hand-roll them. Follow `appo11y_config_resource.go`:

```go
// local type definitions
type MyResource struct {
    k8sutils.TypeMeta   `json:",inline"`
    k8sutils.ObjectMeta `json:"metadata,omitempty"`
    Spec                MyResourceSpec `json:"spec,omitempty"`
}

type MyResourceList struct {
    k8sutils.TypeMeta `json:",inline"`
    Items             []MyResource `json:"items"`
}

type MyResourceSpec struct {
    // fields
}

// implement sdkresource.Object interface
func (r *MyResource) GetSpec() any { return r.Spec }
func (r *MyResource) SetSpec(spec any) error { ... }
// etc.

// Kind definition
func MyResourceKind() sdkresource.Kind { ... }
```

Read `appo11y_config_resource.go` in full before attempting this pattern.

---

## 12. KeeperActivation breaks the generic pattern

**Note**: `secret_keeper_activation_resource.go` is an anomaly — it has a hardcoded resource name, no spec block, and special singleton handling. Do not use it as a reference for normal resources.
