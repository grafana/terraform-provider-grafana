# AppPlatform Generic Resource Pattern

Located in `internal/resources/appplatform/`. These 11 resources use Kubernetes-style APIs (`/apis/...`) via the Grafana App SDK, backed by a Go generics-based `Resource[T, L]` implementation.

## Architecture

```
AppPlatformResources()          [pkg/provider/resources.go:83]
  └── []NamedResource
        ├── .Resource  →  Resource[T, L]  (generic CRUD implementation)
        ├── .Name      →  "grafana_apps_<group>_<kind>_<version>"
        └── .Category

Resource[T sdkresource.Object, L sdkresource.ListObject]
  ├── config    ResourceConfig[T]           (resource definition)
  ├── client    *NamespacedClient[T, L]     (K8s API client)
  └── namespace string                      ("stacks-N" or "org-N")
```

**These resources bypass `*common.Resource` entirely** — they are registered via `AppPlatformResources()` and wired directly into the Plugin Framework provider.

## ResourceConfig[T] — The Resource Definition

```go
type ResourceConfig[T sdkresource.Object] struct {
    Schema        ResourceSpecSchema       // Terraform schema attributes/blocks for spec
    Kind          sdkresource.Kind         // API group/version/kind; provides ZeroValue()
    SpecParser    SpecParser[T]            // func(ctx, spec types.Object, dst T) diag.Diagnostics
    SpecSaver     SpecSaver[T]             // func(ctx, src T, dst *ResourceModel) diag.Diagnostics
    PlanModifier  ResourcePlanModifier     // optional: hook for custom plan logic
    UpdateDecider ResourceUpdateDecider    // optional: decide whether to call API update
    UseConfigSpec bool                     // read spec from raw Config instead of Plan
}
```

## Terraform State Models

```go
ResourceModel {
    id       types.String  // K8s UUID (server-assigned) — used as Terraform resource ID
    metadata types.Object  // ResourceMetadataModel
    spec     types.Object  // resource-specific spec (schema varies per resource)
    options  types.Object  // ResourceOptionsModel
}

ResourceMetadataModel {
    uid         types.String  // K8s .metadata.name (user-chosen, used in API calls)
    uuid        types.String  // K8s .metadata.uid (server-assigned UUID = Terraform ID)
    version     types.String  // K8s resourceVersion (for optimistic locking)
    folder_uid  types.String  // K8s namespace within stack (not Grafana folder)
    labels      types.Map     // K8s labels
    annotations types.Map     // K8s annotations
}

ResourceOptionsModel {
    overwrite bool  // bypass optimistic locking (set automatically after import)
}
```

**Critical naming distinction:** `uid` = K8s `.metadata.name` (human-chosen name); `uuid` = K8s `.metadata.uid` (server UUID). The `id` field in Terraform state is always the `uuid`.

## CRUD Lifecycle

### Configure
```
GrafanaAppPlatformAPI.ClientFor(kind) → *NamespacedClient[T, L]
namespaceForClient() → "stacks-<stackID>" (cloud) or "org-<orgID>" (local)
```

Namespace priority (from `resource.go:251`): **stackID checked first even though orgID defaults to 1**.

### Create
```
1. req.Plan.Get(ctx, &model)
2. If UseConfigSpec: req.Config.Get(ctx, &configModel) — use configModel.Spec instead
3. Kind.Schema.ZeroValue().(T) — create empty typed K8s object
4. setManagerProperties(obj, clientID) — set manager annotations
5. ParseResourceFromModel(model, obj) → SetMetadataFromModel + SpecParser(model.Spec, obj)
6. r.client.Create(ctx, obj, CreateOptions{})
7. SaveResourceToModel(response, &model) — fills UUID, version, etc.
8. resp.State.Set(ctx, model)
```

### Read
```
1. r.client.Get(ctx, uid)  [uid = metadata.uid from state]
2. SetMetadataFromModel(response, &model)  — refresh metadata only
3. Spec is NOT refreshed — preserved verbatim from prior state
4. resp.State.Set(ctx, model)
```

**Why spec isn't refreshed:** Avoids perpetual diffs caused by API returning normalized/transformed spec values.

### Update
```
1. Same as Create but uses r.client.Update with ResourceVersion for optimistic locking
2. If opts.Overwrite = true: clear ResourceVersion (bypass K8s optimistic locking)
3. If UpdateDecider returns skip=true: write prior state back, return without API call
```

### Delete
```
r.client.Delete(ctx, uid)  — 404 is silently ignored (idempotent)
```

### ImportState
```
1. r.client.Get(ctx, req.ID)  — req.ID is the UID (K8s name), NOT the UUID
2. SaveResourceToModel → fills metadata from response
3. r.config.SpecSaver(ctx, response, &model)  — ONLY place SpecSaver is called
4. model.options.overwrite = true  — prevents 409 Conflict on first apply after import
```

## SpecParser / SpecSaver Patterns

Three patterns depending on the resource:

### JSON blob (Dashboard)
```go
SpecParser: func(ctx, spec types.Object, dst *DashboardType) diag.Diagnostics {
    var model DashboardSpecModel
    diags.Append(spec.As(ctx, &model, ...)...)
    jsonBytes, _ := json.Marshal(map[string]any{...})  // unmarshaled from model.JSON
    dst.Spec.Object = jsonMap
    return diags
},
SpecSaver: func(ctx, src *DashboardType, dst *ResourceModel) diag.Diagnostics {
    jsonBytes, _ := json.Marshal(src.Spec.Object)
    // set dst.spec.json = string(jsonBytes)
},
```

### Structured (AlertRule, RecordingRule, InhibitionRule)
```go
SpecParser: func(ctx, spec types.Object, dst *AlertRuleType) diag.Diagnostics {
    var model AlertRuleSpecModel
    diags.Append(spec.As(ctx, &model, ...)...)
    dst.Spec.Title = model.Title.ValueString()
    // ... map each field
    for k, v := range model.Data {
        dst.Spec.Data = append(dst.Spec.Data, parseExpression(k, v))
    }
    return diags
},
```

### Write-only secrets (SecureValue)
Uses all three advanced features — see below.

## Three Advanced Features

### 1. UpdateDecider
Determines at runtime whether to actually call the API update. Used by `SecureValue` to detect if the write-only `value` field actually changed:

```go
UpdateDecider: func(ctx, plan *ResourceModel, prior *ResourceModel) (skip bool, diags diag.Diagnostics) {
    if plan.Spec == prior.Spec { return true, nil }  // skip if no spec change
    return false, nil
},
```

### 2. UseConfigSpec
When `true`, reads `spec` from the raw Terraform Config (pre-plan) instead of Plan. Needed when fields are `WriteOnly: true` — Plan nullifies write-only fields, but Config still has the value.

### 3. PlanModifier
Called during plan to customize the plan before it's stored. `SecureValue` uses this to:
1. Compute SHA-256 hash of the write-only `value` field
2. Store hash in `value_hash` computed attribute
3. Mark other computed fields as Unknown when `value` changes

## Error Handling

`ErrorToDiagnostics` in `errors.go` converts K8s `*apierrors.StatusError`:
- `StatusReasonInvalid` → `FieldErrorsFromCauses` maps K8s field paths to `path.Path` attribute errors
- Other reasons → generic `resp.Diagnostics.AddError()`
- Hardcoded hack: for dashboard v1alpha1, `spec.*` fields are remapped to `spec.json.*`

## Resource Naming

```
grafana_apps_<first-segment-of-group>_<lowercase-kind>_<version>

Example:
  Group=dashboard.grafana.app, Kind=Dashboard, Version=v1beta1
  → grafana_apps_dashboard_dashboard_v1beta1
```

Exception: `KeeperActivation` hardcodes its name (breaks the pattern).

## Concrete Resource Catalog

| Factory | TF Resource Name | Kind | Version | Category |
|---------|-----------------|------|---------|----------|
| `Dashboard()` | `grafana_apps_dashboard_dashboard_v1beta1` | Dashboard | v1beta1 | GrafanaApps |
| `Playlist()` | `grafana_apps_playlist_playlist_v0alpha1` | Playlist | v0alpha1 | GrafanaApps |
| `AlertEnrichment()` | `grafana_apps_alertenrichment_alertenrichment_v1beta1` | AlertEnrichment | v1beta1 | Alerting |
| `AlertRule()` | `grafana_apps_alerting_alertrule_v0alpha1` | AlertRule | v0alpha1 | Alerting |
| `InhibitionRule()` | `grafana_apps_alertingnotifications_inhibitionrule_v0alpha1` | InhibitionRule | v0alpha1 | Alerting |
| `RecordingRule()` | `grafana_apps_alerting_recordingrule_v0alpha1` | RecordingRule | v0alpha1 | Alerting |
| `AppO11yConfigResource()` | `grafana_apps_productactivation_appo11yconfig_v1alpha1` | AppO11yConfig | v1alpha1 | Cloud |
| `K8sO11yConfigResource()` | `grafana_apps_productactivation_k8so11yconfig_v1alpha1` | K8sO11yConfig | v1alpha1 | Cloud |
| `Keeper()` | `grafana_apps_secret_keeper_v1beta1` | Keeper | v1beta1 | Enterprise |
| `SecureValue()` | `grafana_apps_secret_securevalue_v1beta1` | SecureValue | v1beta1 | Enterprise |
| `KeeperActivation()` | `grafana_apps_secret_keeper_activation_v1beta1` | (custom) | — | Enterprise |

## How to Add a New AppPlatform Resource

```
1. Create internal/resources/appplatform/<name>_resource.go

2. Define spec schema:
   var mySchema = ResourceSpecSchema{
       SpecAttributes: map[string]schema.Attribute{ ... },
       SpecBlocks:     map[string]schema.Block{ ... },
   }

3. Implement SpecParser[*v1.MyType]:
   func parseMySpec(ctx, spec types.Object, dst *v1.MyType) diag.Diagnostics { ... }

4. Implement SpecSaver[*v1.MyType] (for import):
   func saveMySpec(ctx, src *v1.MyType, dst *ResourceModel) diag.Diagnostics { ... }

5. Create factory function:
   func MyResource() NamedResource {
       return NewNamedResource[*v1.MyType, *v1.MyTypeList](
           common.CategoryGrafanaApps,
           ResourceConfig[*v1.MyType]{
               Schema:     mySchema,
               Kind:       v1.Kind(),
               SpecParser: parseMySpec,
               SpecSaver:  saveMySpec,
           },
       )
   }

6. Register in pkg/provider/resources.go:
   func AppPlatformResources() []appplatform.NamedResource {
       return []appplatform.NamedResource{
           ...,
           appplatform.MyResource(),
       }
   }

7. Add example: examples/resources/grafana_apps_<group>_<kind>_<version>/resource.tf

8. Run: go generate ./...
```

## Unit Testing AppPlatform Resources

AppPlatform resources have **pure unit tests** in `resource_test.go` (no live Grafana needed):

```go
func TestMyResourceSpec(t *testing.T) {
    cases := []struct{ name string; spec types.Object; expected v1.MyType }{ ... }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            var result v1.MyType
            diags := parseMySpec(ctx, tc.spec, &result)
            require.False(t, diags.HasError())
            require.Equal(t, tc.expected, result)
        })
    }
}
```
