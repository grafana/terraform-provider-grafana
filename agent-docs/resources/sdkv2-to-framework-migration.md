# SDKv2 → Plugin Framework Migration Playbook

Derived from completed migrations: `grafana_annotation` (PR #2546), `grafana_alerting_message_template` (PR #2567), `grafana_folder_permission` (PR #2608).

**See also:** [`agent-docs/resources/framework.md`](./framework.md) — authoritative reference for Plugin Framework resource patterns (canonical struct layout, Configure variants, plan modifiers, validators, model structs). This playbook focuses on the *delta*: what changes from SDKv2 and what to watch out for during migration. When in doubt about how a Framework pattern works, consult `framework.md` first.

---

## Quick Checklist

- [ ] Rewrite `resource_<name>.go` using Plugin Framework patterns
- [ ] Update `resources.go`: rename factory call, change `addValidationToResources` entry if org-scoped
- [ ] Run `make docs` (`go generate ./...`) and commit updated `docs/resources/<name>.md`
- [ ] Check `pkg/generate/testdata/**/*.tf.tmpl` for this resource — update if `Computed` defaults changed
- [ ] Verify acceptance tests still pass: `TF_ACC=1 TF_ACC_OSS=true go test ./internal/resources/grafana/... -run TestAcc<Name> -v`
- [ ] Run linter: `make golangci-lint`

---

## Step-by-Step Migration

### Step 1 — Audit the SDKv2 resource

Before writing any code, identify which SDKv2-specific patterns are used. Each requires a specific Framework equivalent (see mapping table below):

| SDKv2 feature | Present? |
|---|---|
| `ForceNew: true` on any field | |
| `DiffSuppressFunc` | |
| `ValidateFunc` / `ValidateDiagFunc` | |
| `StateFunc` | |
| `Default:` (non-zero) | |
| `Computed: true` on optional field | |
| `d.HasChange("field")` in Update | |
| `common.WithAlertingMutex` / `WithDashboardMutex` / `WithFolderMutex` | |
| `OAPIGlobalClient` (instance-scoped, not org-scoped) | |
| Nested `schema.Resource` blocks (TypeSet/TypeList of resources) | |
| `schema.ImportStatePassthroughContext` only | |
| Lister: `listerFunctionOrgResource` vs `listerFunction` | |

### Step 2 — Rewrite `resource_<name>.go`

#### 2a. Registration

```go
// BEFORE (SDKv2)
func resourceFoo() *common.Resource {
    schema := &schema.Resource{ ... }
    return common.NewLegacySDKResource(
        common.CategoryGrafanaOSS,
        "grafana_foo",
        orgResourceIDString("uid"),
        schema,
    ).WithLister(listerFunctionOrgResource(listFoos))
}

// AFTER (Plugin Framework)
var (
    _ resource.Resource                = &fooResource{}
    _ resource.ResourceWithConfigure   = &fooResource{}
    _ resource.ResourceWithImportState = &fooResource{}

    resourceFooName = "grafana_foo"
    resourceFooID   = orgResourceIDString("uid")  // same ID helper
)

func makeResourceFoo() *common.Resource {
    return common.NewResource(
        common.CategoryGrafanaOSS,
        resourceFooName,
        resourceFooID,
        &fooResource{},
    ).WithLister(listerFunctionOrgResource(listFoos))
}
```

Note: factory function is renamed from `resourceFoo()` to `makeResourceFoo()`. The call site in `resources.go` must be updated to match.

#### 2b. Model struct

```go
type resourceFooModel struct {
    ID    types.String `tfsdk:"id"`
    OrgID types.String `tfsdk:"org_id"`
    Name  types.String `tfsdk:"name"`
    // etc — one field per schema attribute, tfsdk tag must match schema key exactly
}
```

Plugin Framework **does not support embedded structs** for `tfsdk` fields. Keep the model flat.

#### 2c. Resource struct and Configure

See `framework.md` § "Configure Pattern Variants" for the three variants. For org-scoped resources in the `grafana` package, embed `basePluginFrameworkResource` — this is the "Shared base struct" variant. It provides `r.client`, `r.config`, and `r.commonClient` (needed for alerting/dashboard/folder mutexes).

```go
type fooResource struct {
    basePluginFrameworkResource
}
```

For **instance-scoped** (global) resources (e.g. `grafana_user`) that use `OAPIGlobalClient`, implement `Configure` manually and reject API keys:
```go
func (r *fooResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil || r.client != nil { return }
    client, ok := req.ProviderData.(*common.Client)
    if !ok { resp.Diagnostics.AddError(...); return }
    if client.GrafanaAPIConfig.APIKey != "" {
        resp.Diagnostics.AddError("API key not supported", "Use basic auth for global-scope resources")
        return
    }
    r.client = client.GrafanaAPI.Clone().WithOrgID(0)
}
```

#### 2d. Metadata

```go
func (r *fooResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = resourceFooName
}
```

#### 2e. Schema

Replace `map[string]*schema.Schema` with `schema.Schema{ Attributes: map[string]schema.Attribute{...} }`. See `framework.md` § "Framework-Specific Features" for plan modifier and validator syntax, and the SDKv2 → Framework mapping table in Step 3 for field-by-field equivalents.

Always declare `Attributes` for flat fields. Use `Blocks` only when required for mux protocol v5 compatibility (nested permission sets — see `resourcePermissionBulkBase`).

#### 2f. CRUD methods

```go
func (r *fooResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var data resourceFooModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }

    client, orgID, err := r.clientFromNewOrgResource(data.OrgID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to get client", err.Error())
        return
    }

    // ... API call ...

    data.ID = types.StringValue(MakeOrgResourceID(orgID, apiResult.UID))
    // Read back for any computed fields:
    readData, diags := r.read(ctx, data.ID.ValueString())
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }
    resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *fooResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var data resourceFooModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }

    readData, diags := r.read(ctx, data.ID.ValueString())
    resp.Diagnostics.Append(diags...)
    if resp.Diagnostics.HasError() { return }
    if readData == nil {
        resp.State.RemoveResource(ctx)  // replaces d.SetId("")
        return
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}

func (r *fooResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
    var data resourceFooModel
    resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
    if resp.Diagnostics.HasError() { return }

    client, _, split, err := r.clientFromExistingOrgResource(resourceFooID, data.ID.ValueString())
    if err != nil {
        resp.Diagnostics.AddError("Failed to parse resource ID", err.Error())
        return
    }
    uid := split[0].(string)
    _, err = client.Foos.DeleteFoo(uid)
    if err != nil && !common.IsNotFoundError(err) {
        resp.Diagnostics.AddError("Failed to delete foo", err.Error())
    }
}
```

Factoring out a private `r.read(ctx, id string) (*resourceFooModel, diag.Diagnostics)` method is strongly recommended — it is reused by `Read`, `Create` (read-back), `Update` (read-back), and `ImportState`.

#### 2g. ImportState

```go
func (r *fooResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    readData, diags := r.read(ctx, req.ID)
    resp.Diagnostics = diags
    if resp.Diagnostics.HasError() { return }
    if readData == nil {
        resp.Diagnostics.AddError("Resource not found", "...")
        return
    }
    resp.Diagnostics.Append(resp.State.Set(ctx, readData)...)
}
```

Simple resources without computed fields can use `resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)` instead, but the read-then-set pattern is safer.

### Step 3 — Update `resources.go`

Change the registration line for the resource:
```go
// BEFORE
resourceFoo(),

// AFTER
makeResourceFoo(),
```

The `addValidationToResources(...)` wrapper is only meaningful for SDKv2 resources (it wraps `schema.Resource` CRUD funcs). Framework resources skip it automatically since `r.Schema` is nil for them — but you must still remove the old entry from the `addValidationToResources(...)` call and add the new `makeResourceFoo()` call within the same `Resources` slice.

### Step 4 — Regenerate docs

```sh
make docs   # or: go generate ./...
```

Commit the updated `docs/resources/<name>.md`. The doc changes are purely mechanical (schema is regenerated). The only manual change needed is if the description text itself changes.

### Step 5 — Check generate testdata

If the resource appears in `pkg/generate/testdata/generate/**/*.tf.tmpl`, update the golden file. The most common cause of changes:

- A field that was `Optional` with no `Computed` in SDKv2 but is now `Optional + Computed` in Framework will appear in the generated output with its default value. Example from PR #2567: `disable_provenance = false` began appearing explicitly.
- Field ordering may change (Framework emits alphabetically in generated configs).

Run the generate tests to catch this:
```sh
go test ./pkg/generate/... -run TestGenerate -v
```

---

## SDKv2 → Framework Mapping Table

| SDKv2 | Plugin Framework |
|---|---|
| `common.NewLegacySDKResource(...)` | `common.NewResource(...)` |
| `resourceFoo()` factory name | `makeResourceFoo()` |
| `func CreateFoo(ctx, d *schema.ResourceData, meta any)` | `func (r *fooResource) Create(ctx, req, resp)` |
| `d.Get("field").(string)` | `data.Field.ValueString()` |
| `d.Set("field", val)` | set field on model struct, then `resp.State.Set(ctx, model)` |
| `d.SetId("val")` | set `data.ID = types.StringValue("val")` |
| `d.Id()` | `data.ID.ValueString()` (from state read) |
| `common.CheckReadError("x", d, err)` | `if common.IsNotFoundError(err) { resp.State.RemoveResource(ctx); return }` |
| `schema.ImportStatePassthroughContext` | `resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)` or custom read-based ImportState |
| `ForceNew: true` | `PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}` |
| `DiffSuppressFunc: func(k, old, new string, d *schema.ResourceData) bool` | custom `planmodifier.String` implementing `PlanModifyString` |
| `ValidateFunc: validation.StringInSlice(...)` | `Validators: []validator.String{stringvalidator.OneOf(...)}` |
| `ValidateFunc: validation.IsRFC3339Time` | custom `validator.String` implementing `ValidateString` |
| `StateFunc: func(v any) string { return strings.TrimSpace(v.(string)) }` | `resource.ResourceWithModifyPlan` — implement `ModifyPlan` method |
| `Default: "somevalue"` | `Default: stringdefault.StaticString("somevalue")` + `Computed: true` |
| `d.HasChange("field")` in Update | read both `req.Plan` and `req.State` into models and compare |
| `OAPIClientFromNewOrgResource(meta, d)` | `r.clientFromNewOrgResource(data.OrgID.ValueString())` |
| `OAPIClientFromExistingOrgResource(meta, d.Id())` | `r.clientFromExistingOrgResource(resourceFooID, data.ID.ValueString())` |
| `OAPIGlobalClient(meta)` | use `r.client.Clone().WithOrgID(0)` directly; validate no API key in `Configure` |
| `orgIDAttribute()` | `pluginFrameworkOrgIDAttribute()` |
| `common.WithAlertingMutex[schema.CreateContextFunc](fn)` | `r.commonClient.WithAlertingLock(func() { ... })` wrapping the API call inline |
| `common.WithDashboardMutex[...]` | `r.commonClient.WithDashboardLock(func() { ... })` |
| `TypeSet` of `*schema.Schema{Type: TypeString}` | `schema.SetAttribute{ElementType: types.StringType}` |
| `TypeSet` of `*schema.Resource{...}` | `schema.SetNestedAttribute{...}` (or `schema.SetNestedBlock` for mux v5 compat) |
| `TypeList` of `*schema.Schema{Type: TypeString}` | `schema.ListAttribute{ElementType: types.StringType}` |
| `Sensitive: true` | `Sensitive: true` (same field, no change) |

---

## Special Cases

### Alerting resources

Alerting resources must serialize API calls via the alerting mutex. Use the inline lock pattern:

```go
var apiErr error
r.commonClient.WithAlertingLock(func() {
    _, apiErr = client.Provisioning.PutTemplate(params)
})
if apiErr != nil { ... }
```

The old `common.WithAlertingMutex[schema.CreateContextFunc](fn)` wrapper is SDKv2-only.

### Optional+Computed fields

When an optional field has no default and the API may or may not return it, use `types.StringNull()` / `types.Int64Null()` / `types.SetNull(elementType)` for the zero/empty case. Never set `types.StringValue("")` for an optional field that was unset — this causes "Provider produced inconsistent result after apply" errors because the plan had `null` but state gets `""`.

### Singleton resources (one per org, e.g. org preferences)

The ID is just the org ID (`types.StringValue(strconv.FormatInt(orgID, 10))`). Import receives the raw org ID string. The `clientFromExistingOrgResource` helper expects `<orgID>:<resourceID>` format — append a colon or use a special ResourceID definition:

```go
// Option A: define ID as optional orgID only (no resource-local part)
resourceFooID = common.NewResourceID(common.IntIDField("orgID"))

// Then in read/update/delete, parse via SplitOrgResourceID or just ParseInt directly
```

### Resources with no lister

If the SDKv2 resource had no `.WithLister(...)`, omit it in the Framework version too. The `grafana generate` command will skip this resource.

### Enterprise-only resources

Enterprise resources use `common.CategoryGrafanaEnterprise`. Tests must call `testutils.CheckEnterpriseTestsEnabled(t)` as the first line.

---

## Acceptance Test Requirements

Tests themselves generally **do not need code changes** — the test helpers (`ProtoV5ProviderFactories`, `testutils.CheckLister`, etc.) already work with both SDKv2 and Framework resources. The test `ProtoV5ProviderFactories` routes calls to the mux server which handles both plugin layers transparently.

What can break:
1. **ImportStateVerify failures**: if a computed field is now `null` in Framework where it was `""` in SDKv2 (or vice versa). Fix by normalizing the field value in `read()`.
2. **Lister test failures** (`testutils.CheckLister`): if `.WithLister(...)` was dropped or the lister function signature changed.
3. **State drift on plan**: if optional fields that the API echoes back are not correctly set to `null` vs non-null. Framework is stricter than SDKv2 about null/unknown consistency.

---

## Linter Requirements

The project uses `golangci-lint` (runs in Docker via `make golangci-lint`). Common issues after migration:

1. **Unused imports**: remove `"github.com/hashicorp/terraform-plugin-sdk/v2/..."` imports.
2. **Unused variables**: if the old factory function had an `_ = schema.Resource{...}` pattern, clean up.
3. **Missing interface assertions**: add `var _ resource.Resource = &fooResource{}` etc. at file top.
4. **`errcheck`**: all errors from `resp.Diagnostics.Append(...)` are already checked via `HasError()`. API errors must be checked (not silently ignored).

---

## Docs Requirements

`make docs` runs `go generate ./...` which calls `tfplugindocs`. The generated markdown changes when:

- Schema structure changes (field added/removed/renamed)
- `Description` / `MarkdownDescription` changes
- An `Optional` field becomes `Optional + Computed` (it moves from "Optional" section to "Optional" with a "(known after apply)" note)
- A `Default` value is added (appears in description automatically via `tfplugindocs`)

Always run `make docs` and commit the result. CI checks that docs are up-to-date.

---

## What to Provide When Starting a Migration

To get maximum value from an AI assistant on a migration, provide:

1. **The resource file** (`resource_<name>.go`) — to audit SDKv2 patterns (already the default)
2. **The test file** (`resource_<name>_test.go`) — to understand what scenarios must keep working, especially: what fields are checked with `ImportStateVerify`, whether there are org-scoped tests, what computed fields are asserted
3. **The examples directory** (`examples/resources/grafana_<name>/`) — for docs generation; shows what a minimal HCL config looks like, which affects what the `id` field description needs to cover
4. **Any generate testdata** that mentions this resource name — run `grep -r "grafana_<name>" pkg/generate/testdata/` to find them
5. **Confirm the target category**: OSS, Enterprise, or Alerting (affects mutex usage and test gating)
6. **Note any known behavioral quirks**: e.g. "the API always returns X even when unset" or "delete is actually a reset to defaults"

---

## Recommended Verification Steps

```sh
# 1. Build to catch compile errors
go build .

# 2. Run unit tests
go test ./... -run TestUnit

# 3. Run acceptance tests for this specific resource
GRAFANA_URL=http://localhost:3000 GRAFANA_AUTH=admin:admin \
  TF_ACC=1 TF_ACC_OSS=true GRAFANA_VERSION=11.0.0 \
  go test ./internal/resources/grafana/... -run TestAcc<ResourceName> -v -timeout 30m

# 4. Regenerate docs
make docs

# 5. Check generate testdata tests
go test ./pkg/generate/... -v

# 6. Lint (requires Docker)
make golangci-lint
```
