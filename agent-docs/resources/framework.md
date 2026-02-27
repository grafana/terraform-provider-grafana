# Plugin Framework Resource Pattern

Used by ~26 resources across: `k6` (all), `fleetmanagement` (all), `connections` (all), `cloudprovider` (all), `frontendo11y` (all), plus permission-item resources and `cloud_org_member` in the `grafana`/`cloud` packages.

## Registration

```go
// In package-level Resources slice:
common.NewResource(
    common.CategoryK6,
    "grafana_k6_project",
    common.NewResourceID(common.StringIDField("id")),
    &k6ProjectResource{},   // resource.ResourceWithConfigure
)
```

- Sets `.PluginFrameworkSchema` field on `*common.Resource`
- `pluginFrameworkResources()` in `pkg/provider/resources.go:119` filters for non-nil `.PluginFrameworkSchema`

## Canonical Resource Structure

```go
// 1. Resource type
type myResource struct {
    client *SomeAPIClient   // set in Configure
}

// 2. Required interface methods
func (r *myResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
    resp.TypeName = req.ProviderTypeName + "_my_resource"
}

func (r *myResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
    resp.Schema = schema.Schema{
        Attributes: map[string]schema.Attribute{
            "id": schema.StringAttribute{
                Computed: true,
                PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
            },
            "name": schema.StringAttribute{Required: true},
        },
    }
}

// 3. Configure — idempotent guard is required
func (r *myResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
    if req.ProviderData == nil || r.client != nil { return }  // idempotent guard
    client, ok := req.ProviderData.(*common.Client)
    if !ok { resp.Diagnostics.AddError("Unexpected type", ...); return }
    r.client = client.SomeAPIClient
}

// 4. CRUD methods
func (r *myResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    var plan myResourceModel
    resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
    if resp.Diagnostics.HasError() { return }
    // ... API call ...
    resp.Diagnostics.Append(resp.State.Set(ctx, plan)...)
}

func (r *myResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
    var state myResourceModel
    resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
    // ... API call ...
    if isNotFound { resp.State.RemoveResource(ctx); return }  // 404 handling
    resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *myResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
    resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

## Model Structs

Typed structs with `tfsdk` tags replace `*schema.ResourceData`:

```go
type myResourceModel struct {
    ID          types.String `tfsdk:"id"`
    Name        types.String `tfsdk:"name"`
    Description types.String `tfsdk:"description"`
    Tags        types.List   `tfsdk:"tags"`
    Config      types.Object `tfsdk:"config"`
}
```

**Important:** Plugin Framework does NOT support struct embedding for `tfsdk` fields. Permission-item resources work around this with explicit `ToBase()`/`SetFromBase()` converter methods.

## Configure Pattern Variants

Three variants exist, all using the idempotent guard (`if req.ProviderData == nil || r.client != nil { return }`):

**1. Shared base struct** (grafana permission-items, k6):
```go
type myResource struct {
    basePluginFrameworkResource  // embedded, provides Configure() for free
}
// basePluginFrameworkResource is in grafana/common_plugin_framework.go
```

**2. Per-resource with helper** (connections, fleet management):
```go
func withClientForResource[T any](fn func(*T, ...)) resource.ConfigureFunc { ... }
```

**3. Direct implementation** (cloudprovider, frontendo11y, k6 variants):
Each resource implements `Configure` directly.

## Framework-Specific Features

### Plan Modifiers
```go
// Immutable field (changing forces recreate):
schema.StringAttribute{
    PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
}

// Computed ID (preserve unknown value across plan):
schema.StringAttribute{
    Computed: true,
    PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
}

// Org-scoped ID (suppress diff when empty — suppress when org set at provider level):
schema.StringAttribute{
    Optional: true, Computed: true,
    PlanModifiers: []planmodifier.String{orgIDAttributePlanModifier{}},
}
```

### Validators
```go
schema.StringAttribute{
    Validators: []validator.String{
        stringvalidator.ExactlyOneOf(
            path.MatchRoot("role"),
            path.MatchRoot("team_id"),
            path.MatchRoot("user_id"),
        ),
    },
}
```

### State Upgrade (k6 only)
`k6ProjectResource` implements `resource.ResourceWithUpgradeState` to migrate IDs from int32 to string between schema versions.

## Permission Resource Abstraction (Framework)

Four permission-item resources (`folder_permission_item`, `dashboard_permission_item`, `datasource_permission_item`, `service_account_permission_item`) share `resourcePermissionBase` (`common_resource_permission.go`):

```
resourcePermissionBase
  ├── addInSchemaAttributes()  →  injects org_id, built-in_role, team_id, user_id, permission
  ├── readItem(ctx, orgID, uid, model) → GET access control API, filter by target type
  └── writeItem(ctx, orgID, uid, model) → PUT access control API

Per-resource workaround (no struct embedding in tfsdk):
  myModel.ToBase() → *resourcePermissionBaseModel
  myModel.SetFromBase(*resourcePermissionBaseModel)
```

## Org-Scoped IDs in Framework

For Framework resources in the `grafana` package that still need org scoping, custom plan modifiers handle suppression:

```go
// orgIDAttributePlanModifier: suppress diff when new value is empty
// orgScopedAttributePlanModifier: ignore org ID prefix in compound plan values
```

These are in `internal/resources/grafana/common_plugin_framework.go`.

## Package Distribution

| Package | All Framework? | Notes |
|---------|---------------|-------|
| `k6` | Yes | All 5 resources + data sources |
| `fleetmanagement` | Yes | Uses ConnectRPC client |
| `connections` | Yes | |
| `cloudprovider` | Yes | |
| `frontendo11y` | Yes | |
| `grafana` | Mixed | Permission-item resources are Framework; alerting/teams/etc. are SDKv2 |
| `cloud` | Mixed | `cloud_org_member` is Framework; rest are SDKv2 |
| `asserts` | No | Newer package but uses SDKv2 (anomaly) |
