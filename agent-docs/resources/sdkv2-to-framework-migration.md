# SDKv2 → Plugin Framework Migration Playbook

Support for **Terraform Plugin SDKv2** in this repo is being phased out: **new** REST resources and datasources should use the Plugin Framework (`common.NewResource`, Framework datasource constructors). Broader program context: [deployment_tools #475444](https://github.com/grafana/deployment_tools/issues/475444), [terraform-provider-grafana #2580](https://github.com/grafana/terraform-provider-grafana/issues/2580), [#2216](https://github.com/grafana/terraform-provider-grafana/issues/2216).

Reference migrations that shaped this doc: **`grafana_annotation`** ([PR #2546](https://github.com/grafana/terraform-provider-grafana/pull/2546) — optional attributes, RFC3339 validation, org-scoped lister), **`grafana_alerting_message_template`** ([PR #2567](https://github.com/grafana/terraform-provider-grafana/pull/2567) — alerting mutex, org plan modifiers, optional/computed), **`grafana_folder_permission`** ([PR #2608](https://github.com/grafana/terraform-provider-grafana/pull/2608)).

**See also:** [`framework.md`](./framework.md) — canonical Plugin Framework patterns (struct layout, `Configure` variants, plan modifiers, validators, models). **This file** is the SDKv2→Framework *delta* (audit, rewrite, mapping table, edge cases); use `framework.md` when you need the “how Framework works” reference.

### New `NewLegacySDK*` registrations and CI

New `NewLegacySDKResource` / `NewLegacySDKDataSource` registrations under `internal/resources/` trip the [SDKv2 migration check](../../.github/workflows/sdkv2-migration-check.yml) workflow and **fail CI**; the job log lists the offending lines. Prefer Framework registration for new work. (See also `AGENTS.md` “SDKv2 migration CI check”.)

---

## Quick Checklist

- [ ] Rewrite `resource_<name>.go` using Plugin Framework patterns
- [ ] **For every `DiffSuppressFunc` in the SDKv2 source: explicitly translate it or document why it is safe to drop** (see [DiffSuppressFunc handling](#diffsuppressfunc-handling) below)
- [ ] **For every non-zero `Default:` field: add a null guard if the value is read from state and used to control behavior in `Read`** (see [DiffSuppressFunc + Default as a signal for null-in-Read bugs](#diffsuppressfunc--default-as-a-signal-for-null-in-read-bugs))
- [ ] Update `resources.go`: rename factory call, change `addValidationToResources` entry if org-scoped
- [ ] Run `make docs` (`go generate ./...`) and commit updated `docs/resources/<name>.md`
- [ ] Check `pkg/generate/testdata/**/*.tf.tmpl` for this resource — update if `Computed` defaults changed
- [ ] Verify acceptance tests still pass: `TF_ACC=1 TF_ACC_OSS=true go test ./internal/resources/grafana/... -run TestAcc<Name> -v`
- [ ] Run linter: `make golangci-lint`

---

## Step-by-Step Migration

The walkthrough below is written for **resources**; **datasources** share the same registration and null-handling goals but use the Framework **data source** interfaces instead of `resource.Resource`.

### Step 1 — Audit the SDKv2 resource

Before writing any code, identify which SDKv2-specific patterns are used. Each requires a specific Framework equivalent (see mapping table below):

| SDKv2 feature | Present? |
|---|---|
| `ForceNew: true` on any field | |
| `DiffSuppressFunc` — list every field that has one, categorize by type (see below) | |
| `ValidateFunc` / `ValidateDiagFunc` | |
| `StateFunc` | |
| `Default:` (non-zero) — list every field; flag any used in `Read` logic | |
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

**Preserve validators:** re-home every SDKv2 `ValidateFunc` / `ValidateDiagFunc` on the Framework attribute as `Validators: [...]` (e.g. `stringvalidator.OneOf` for enums; RFC3339 and other bespoke checks as a custom `validator.String` — see the mapping table below and PR #2546). Do not drop validation unless you intend to change the contract.

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

**Org-scoped IDs — avoid copy-paste parsing:** If `read`, `Update`, and `Delete` all call `r.clientFromExistingOrgResource(resourceFooID, ...)` and then validate `split` length, type-assert the resource-local id (string, int, etc.), and surface the same diagnostics, extract a **single private helper** (for example `(client, orgID, uid, diags)` for a uid-based resource). That keeps behavior aligned and matches what reviewers expect after several migrations.

**Create — string UID vs numeric fallback:** Some APIs return a primary string identifier and sometimes a legacy numeric id. If you mirror the old SDK pattern `uid := payload.UID; if uid == "" { uid = strconv.FormatInt(payload.ID, 10) }`, only use the numeric branch when **`payload.ID != 0`**. If both are empty/zero, **return a diagnostic** instead of building a composite Terraform id containing `"0"` or another bogus value.

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

### Null vs empty or zero for optional fields in Read

For **optional** attributes, the API often returns an empty string or numeric **zero** when Terraform expects **`null`** for an unset attribute. If **Read** (or a shared private `read()` used from Create, Update, and ImportState) writes those raw API values into state while the user left the attribute unset, the next plan can show **Provider produced inconsistent result after apply** — the plan still has `null`, but state holds `""`, `0`, or another non-null shape.

**Normalize in Read:** when the attribute is optional and “unset” from Terraform’s perspective but the API returns empty string, zero, or an empty collection, set the corresponding model field to **`null`** using `types.StringNull()`, `types.Int64Null()` (and other numeric nulls as appropriate), `types.SetNull(elementType)`, `types.ListNull(elementType)`, etc., so state matches an unset configuration.

**Do not** set `types.StringValue("")` for an optional field that should be unset — you get the same failure mode because the plan had `null` but state gets `""`.

**Optional + Computed** attributes without a Terraform default are a common case: the API may omit the field or return an empty value. Apply the same rule — prefer the appropriate `*Null()` helper unless you are intentionally persisting a real default or computed value.

**Examples:** [PR #2546](https://github.com/grafana/terraform-provider-grafana/pull/2546) (`grafana_annotation`) and [PR #2567](https://github.com/grafana/terraform-provider-grafana/pull/2567) (`grafana_alerting_message_template`).

### DiffSuppressFunc handling

Every `DiffSuppressFunc` in the SDKv2 source **must be explicitly translated or explicitly justified as safe to drop**. Silently dropping one is a common source of breaking changes after migration (real examples: `grafana_team` incident i-2026-04-13-ocado-p1, `grafana_report` workdays_only regression).

Categorize each one during the Step 1 audit and apply the appropriate Framework pattern:

#### Category 1 — Null/absent state + non-zero default
Pattern: `old == "" && new == "somedefault"` or `old == new`.
These suppress spurious diffs caused by old state not storing the field.

**Framework treatment:** `Default: <staticDefault>` + `Computed: true` covers the plan-diff side. But also add a null guard in `Read` if the value drives behavior — see [DiffSuppressFunc + Default as a signal for null-in-Read bugs](#diffsuppressfunc--default-as-a-signal-for-null-in-read-bugs).

#### Category 2 — Value normalization (case, whitespace, format)
Pattern: `strings.EqualFold(old, new)`, `strings.TrimSpace(old) == strings.TrimSpace(new)`, date/time format equivalence.

**Framework treatment:** implement a custom `planmodifier.String` (or appropriate type) that normalizes the planned value to match the stored state when they are semantically equal. Do NOT drop these — dropping causes perpetual plan diffs whenever the API or user normalizes the value differently.

```go
// Example: case-insensitive string plan modifier
type caseInsensitiveModifier struct{}
func (m caseInsensitiveModifier) PlanModifyString(ctx context.Context, req planmodifier.StringRequest, resp *planmodifier.StringResponse) {
    if req.StateValue.IsNull() || req.PlanValue.IsNull() { return }
    if strings.EqualFold(req.StateValue.ValueString(), req.PlanValue.ValueString()) {
        resp.PlanValue = req.StateValue  // keep state value to suppress diff
    }
}
```

#### Category 3 — Conditional suppression based on another field
Pattern: `func(k, old, new string, d *schema.ResourceData) bool { return !someCondition(d.Get("other_field")) }`.
These suppress diffs on a field when a sibling field makes it irrelevant.

**Framework treatment:** implement a custom `PlanModifier` that reads the other attribute from the plan and returns the state value (suppressing the diff) when the condition is met. Do NOT just drop these — dropping causes perpetual plan diffs for configs that set the field to a value the API ignores.

```go
// Example: suppress workdays_only diff when frequency doesn't support it
func (m workdaysOnlyModifier) PlanModifyBool(ctx context.Context, req planmodifier.BoolRequest, resp *planmodifier.BoolResponse) {
    var frequency types.String
    req.Plan.GetAttribute(ctx, path.Root("schedule").AtListIndex(0).AtName("frequency"), &frequency)
    if !reportWorkdaysOnlyConfigAllowed(frequency.ValueString()) {
        resp.PlanValue = req.StateValue
    }
}
```

#### Category 4 — List/set count transitions
Pattern: `oldValue == "1" && newValue == "0"` (SDKv2 passes the collection count as a string for the `field.#` key).
These suppress diffs when a block is removed.

**Framework treatment:** Usually safe to drop if the block is fully optional and the API handles absence gracefully. Verify by checking what the API returns when the block is absent vs present. If the API echoes back an empty version of the block, ensure `Read` writes `null` to the field (not an empty list) when the block is absent, so state stays consistent with an unset config.

#### When it may be safe to drop a DiffSuppressFunc
Only drop without a replacement if **all** of the following hold:
- The suppressed case can no longer occur (e.g., the field was removed or the API behavior changed)
- OR `Computed: true` + the Framework's default/state machinery already prevents the diff
- AND you have verified this with an acceptance test that exercises the suppressed scenario

Document the rationale in a code comment on the schema attribute.

### DiffSuppressFunc + Default as a signal for null-in-Read bugs

When a field has **both** a `DiffSuppressFunc` and a non-zero `Default`, treat this as a red flag during audit. The `DiffSuppressFunc` is often suppressing a spurious plan diff caused by SDKv2 storing an absent field as `""` in state (e.g. `old == "" && new == "true"`). These are different symptoms of the same root cause: **existing state may be missing a value for this field**.

In Framework, absent state unmarshals to a null type (`types.BoolNull()`, `types.StringNull()`, etc.). If that null is then used directly in `Read` logic — e.g. `data.IgnoreExternallySyncedMembers.ValueBool()` — it returns the Go zero value (`false`, `""`, `0`) instead of the intended default. This can silently flip behavior on the next refresh after a provider upgrade, without any plan diff warning the user.

**Fix pattern:** add a null guard that treats null/unknown as the default value:

```go
// Matches pre-Framework behavior: SDKv2 Default: true meant absent state → true,
// but Framework's ValueBool() on a null returns false.
func effectiveIgnoreExternallySyncedMembers(b types.Bool) bool {
    if b.IsNull() || b.IsUnknown() {
        return true
    }
    return b.ValueBool()
}
```

Use the guard wherever the field value drives `Read` logic, not just plan diffing. The `Default: booldefault.StaticBool(true)` on the schema only applies during planning — it does not protect `Read`.

The `DiffSuppressFunc` mapping in the table below (→ custom plan modifier) covers the plan-diff side. This section covers the separate, higher-impact Read side.

**Real example:** `grafana_team`'s `ignore_externally_synced_members` (PR #2530 / incident i-2026-04-13-ocado-p1). After migration, users with old state lacking this field had it read as `false`, causing Terraform to try removing externally-synced team members on the next apply. Fixed in commit `df7cb05b`.

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
1. **ImportStateVerify failures**: if a field is now `null` in Framework where it was `""` in SDKv2 (or vice versa). Fix by normalizing in `read()` — see [Null vs empty or zero for optional fields in Read](#null-vs-empty-or-zero-for-optional-fields-in-read).
2. **Lister test failures** (`testutils.CheckLister`): if `.WithLister(...)` was dropped or the lister function signature changed.
3. **State drift on plan**: optional fields the API echoes back must use `null` vs non-null consistently with the plan; Framework is stricter than SDKv2. Same normalization rules as § [Null vs empty or zero for optional fields in Read](#null-vs-empty-or-zero-for-optional-fields-in-read).

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

When a regeneration would only change phrasing, prefer updating `Description` / `MarkdownDescription` on the schema in Go so `tfplugindocs` emits the right text, instead of hand-editing generated files under `docs/`.

---

## What to Provide When Starting a Migration

To get maximum value from an AI assistant on a migration, provide:

1. **The resource file** (`resource_<name>.go`) — to audit SDKv2 patterns (already the default)
2. **The test file** (`resource_<name>_test.go`) — to understand what scenarios must keep working, especially: what fields are checked with `ImportStateVerify`, whether there are org-scoped tests, what computed fields are asserted
3. **The examples directory** (`examples/resources/grafana_<name>/`) — for docs generation; shows what a minimal HCL config looks like, which affects what the `id` field description needs to cover
4. **Any generate testdata** that mentions this resource name — run `grep -r "grafana_<name>" pkg/generate/testdata/` to find them
5. **Confirm the target category**: OSS, Enterprise, or Alerting (affects mutex usage and test gating)
6. **Note any known behavioral quirks**: e.g. "the API always returns X even when unset" or "delete is actually a reset to defaults"

### Example agent prompt

> Migrate the `<name>` resource [or datasource] to use the Plugin Framework instead of SDKv2. Follow the migration steps in this playbook, and use PR #2546 (`grafana_annotation`) and PR #2567 (`grafana_alerting_message_template`) as examples.

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

---

## Shipping and collaboration

- Prefer the **`gh` CLI** for GitHub (`gh pr view`, `gh pr checkout`, `gh pr checks`). Run `gh auth status` if a command fails.
- **Remote CI**: If you publish work to a remote branch (so CI runs), keep iterating until required checks pass — same bar as local verification. Skipping push/CI is fine until you need branch-based review or merge.
- **Document gaps you had to fill:** if tests, CI, or API behavior forced extra steps this guide didn’t mention, add those learnings here (use `AGENTS.md` only for notes that apply beyond SDKv2→Framework migration).
