# SDKv2 Resource Pattern (Legacy)

Used by ~65 resources: all of `grafana` package (OSS/Enterprise), `cloud`, `oncall`, `machinelearning`, `slo`, `syntheticmonitoring`, `asserts`.

## Registration

```go
// In package-level Resources slice:
common.NewLegacySDKResource(
    common.CategoryGrafanaOSS,
    "grafana_folder",
    orgResourceIDString("uid"),   // ResourceID type
    resourceFolder(),             // *schema.Resource
).
    WithLister(listFolders).
    WithPreferredResourceNameField("title")
```

- Sets `.Schema` field on `*common.Resource`
- `legacySDKResources()` in `pkg/provider/resources.go:107` filters for non-nil `.Schema`

## Canonical File Structure

```
resource_<name>.go
├── resourceXxx() *common.Resource          ← factory registered in resources.go
│   └── &schema.Resource{
│         CreateContext: common.WithFolderMutex[schema.CreateContextFunc](CreateXxx),
│         ReadContext:   ReadXxx,
│         UpdateContext: common.WithFolderMutex[schema.UpdateContextFunc](UpdateXxx),
│         DeleteContext: common.WithFolderMutex[schema.DeleteContextFunc](DeleteXxx),
│         Importer: &schema.ResourceImporter{StateContext: schema.ImportStatePassthroughContext},
│         Schema: map[string]*schema.Schema{
│             "org_id": orgIDAttribute(),   // for org-scoped resources
│             "uid":    { Type: TypeString, Computed: true, ... },
│         },
│       }
├── listXxx(ctx, *Client, data) []string    ← lister for code generation
├── CreateXxx(ctx, *ResourceData, any) diag.Diagnostics
├── ReadXxx(ctx, *ResourceData, any) diag.Diagnostics
├── UpdateXxx(ctx, *ResourceData, any) diag.Diagnostics
└── DeleteXxx(ctx, *ResourceData, any) diag.Diagnostics
```

## Org-Scoped Resources

Most core Grafana resources are org-scoped. IDs are `orgID:resourceIdentifier` (e.g., `1:my-folder-uid`).

```
Create:  d.Get("org_id") ──► OAPIClientFromNewOrgResource(meta, d)
                              returns (client *goapi.GrafanaHTTPAPI, orgID int64)
                              then: d.SetId(MakeOrgResourceID(orgID, uid))

Read/Update/Delete:
         d.Id() = "1:uid" ──► OAPIClientFromExistingOrgResource(meta, d.Id())
                               returns (client, orgID=1, restOfID="uid")
```

Key helpers in `internal/resources/grafana/oss_org_id.go`:
- `orgIDAttribute()` — `TypeString, Optional, ForceNew, DiffSuppressFunc: suppress when empty`
- `orgResourceIDString(field)` / `orgResourceIDInt(field)` — factory for `[OptInt, String/Int]` ResourceID
- `MakeOrgResourceID(orgID, id)` / `SplitOrgResourceID(id)` — construct/parse
- `OAPIClientFromNewOrgResource(meta, d)` — for Create
- `OAPIClientFromExistingOrgResource(meta, id)` — for Read/Update/Delete

## CRUD Patterns

### Create
```go
func CreateFolder(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    client, orgID := OAPIClientFromNewOrgResource(meta, d)
    folder, err := client.Folders.CreateFolder(...)
    if err != nil { return diag.FromErr(err) }
    d.SetId(MakeOrgResourceID(orgID, folder.Payload.UID))
    return ReadFolder(ctx, d, meta)  // populate computed fields
}
```

### Read (404 handling)
```go
func ReadFolder(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
    folder, err := client.Folders.GetFolderByUID(uid, nil)
    if errDiags, shouldReturn := common.CheckReadError("folder", d, err); shouldReturn {
        return errDiags  // nil on 404 (d.SetId("") already called), error diag otherwise
    }
    d.Set("title", folder.Payload.Title)
    return nil
}
```

### Delete (ignore already-deleted)
```go
func DeleteFolder(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
    client, _, uid := OAPIClientFromExistingOrgResource(meta, d.Id())
    _, err := client.Folders.DeleteFolder(uid, nil)
    diags, _ := common.CheckReadError("folder", d, err)
    return diags  // nil on 404, error otherwise
}
```

## Mutex Protection

Three mutexes for resources prone to concurrent-write races:

```go
// In resource schema definition:
CreateContext: common.WithAlertingMutex[schema.CreateContextFunc](createFn),
UpdateContext: common.WithFolderMutex[schema.UpdateContextFunc](updateFn),
DeleteContext: common.WithDashboardMutex[schema.DeleteContextFunc](deleteFn),
```

| Mutex | Used by |
|-------|---------|
| `WithAlertingMutex` | contact_point, message_template, mute_timing, notification_policy |
| `WithFolderMutex` | folder |
| `WithDashboardMutex` | dashboard |

Reads are NOT wrapped (concurrent reads are safe).

## Lister Functions

```go
type ResourceListIDsFunc func(ctx context.Context, client *Client, data any) ([]string, error)

// Typical lister:
func listFolders(ctx context.Context, client *common.Client, data any) ([]string, error) {
    // data is *grafana.ListerData — provides lazy-cached list of org IDs
    return listerFunctionOrgResource(func(ctx context.Context, client *goapi.GrafanaHTTPAPI, orgID int64) ([]string, error) {
        resp, err := client.Folders.GetFolders(folders.NewGetFoldersParams(), nil)
        if err != nil { return nil, err }
        ids := make([]string, len(resp.Payload))
        for i, f := range resp.Payload {
            ids[i] = MakeOrgResourceID(orgID, f.UID)
        }
        return ids, nil
    })(ctx, client, data)
}
```

The `listerFunctionOrgResource` wrapper iterates all org IDs (from lazy-cached `ListerData`) and calls the inner function once per org.

## Shared Abstractions

### Permission Resources (`common_resource_permission_sdk2.go`)
Five resources (folder, dashboard, datasource, service account permission) share `resourcePermissionsHelper`:
```go
type resourcePermissionsHelper struct {
    resourceType  string                                    // "folders", "dashboards"
    roleAttribute string                                    // schema attribute name
    getResource   func(*schema.ResourceData, any) (string, error)
}
```
Provides CRUD methods that compute delta-based permission updates (only add/remove changed items).

## Schema Helpers (`internal/common/schema.go`)

```go
// Clone resource schema for a data source (all fields become Computed):
CloneResourceSchemaForDatasource(r *schema.Resource, overrides map[string]*schema.Schema)
// Note: nil value in overrides deletes that field from the clone

// Convenience constructors:
ComputedString() *schema.Schema
ComputedInt() *schema.Schema
ComputedStringWithDescription(desc string) *schema.Schema

// Validators:
ValidateDuration(i any, p cty.Path) diag.Diagnostics      // "5m", "1h30m"
ValidateDurationWithDays(i any, p cty.Path) diag.Diagnostics  // also accepts "1d"
AllowedValuesDescription(desc string, values []string) string // appends "Allowed values: `a`, `b`."
```

## Cross-Domain Patterns

| Package | Client access | Org scoping | Uses mutexes |
|---------|--------------|-------------|-------------|
| `grafana` | `OAPIClientFromNew/ExistingOrgResource` | Yes, composite IDs | Yes (alerting, folder, dashboard) |
| `cloud` | `withClient[*gcom.APIClient]` generic wrapper | No (region-scoped) | No |
| `oncall` | `withClient[*onCallAPI.Client]` generic wrapper | No | No |
| `machinelearning`, `slo`, `syntheticmonitoring` | Direct from `meta.(*common.Client)` | No | No |
