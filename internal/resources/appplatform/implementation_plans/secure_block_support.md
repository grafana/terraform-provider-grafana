# App Platform Generic Secure Block Support — Implementation Plan

## Summary

Extend the generic `Resource[T, L]` framework in `resource.go` to support an optional `secure`
block with **write-only attributes** (Terraform 1.11+). This is a reusable framework-level change
that any app platform resource with secret fields can opt into.

Also add framework-managed secure-key reconciliation: on Update, schema-declared secure keys
omitted from current config are translated to API-side `InlineSecureValue{Remove: true}`.

## Motivation

Several Grafana app platform resources (provisioning Repository, provisioning Connection, and
potentially future resources) have a `Secure` field at the root level alongside `Spec`:

```
SomeResource {
  Spec   SomeSpec     // declarative config
  Secure SomeSecure   // tokens, keys, secrets
  Status SomeStatus   // read-only
}
```

The current `Resource[T, L]` framework only handles `metadata`, `spec`, and `options`. There is
no mechanism to pass secrets to the API without storing them in Terraform state.

## Design

### Write-Only Attributes (Terraform 1.11+)

Secrets use `WriteOnly: true` attributes so they are **never stored in Terraform state** — not
even in encrypted or sensitive form. This is the state-of-the-art approach adopted by AWS, Azure,
and Vault providers:

- **Terraform 1.11** introduced write-only arguments ([announcement blog post](https://www.hashicorp.com/en/blog/terraform-1-11-ephemeral-values-managed-resources-write-only-arguments)), with official guidance in [Use temporary write-only arguments](https://developer.hashicorp.com/terraform/language/manage-sensitive-data/write-only)
- **AWS provider** added `password_wo`/`master_password_wo` to RDS resources (`aws_db_instance`, `aws_rds_cluster`) and KMS/Secrets Manager resources starting in v5.87+ ([CHANGELOG](https://github.com/hashicorp/terraform-provider-aws/blob/main/CHANGELOG.md))
- **AzureRM provider** added write-only attributes to `azurerm_key_vault_secret`, `azurerm_mssql_server`, `azurerm_mysql_flexible_server`, and `azurerm_postgresql_flexible_server` in v4.34.0, with a formal [contributor guide for adding write-only attributes](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/guide-new-write-only-attribute.md)
- **Vault provider** added `data_json_wo` to `vault_kv_secret_v2`, `credentials_wo` to `vault_gcp_secret_backend`, and `password_wo` to `vault_database_secret_backend_connection` ([official guide](https://registry.terraform.io/providers/hashicorp/vault/latest/docs/guides/using_write_only_attributes.html))

Provider framework documentation: [Plugin Framework: Write-only Arguments](https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-arguments)

A single `secure_version` attribute at the resource root triggers re-application of all secure
changes when incremented. This keeps the `secure` block a clean 1:1 mirror of the API's `Secure`
struct — no Terraform-only version fields and no exposed `remove` action in user schema.

**Why write-only over `Sensitive: true`:**
- `Sensitive: true` only masks values in CLI output — secrets are still stored **plaintext in state**
- `WriteOnly: true` means the value is passed to the provider at apply time then **discarded entirely**

### Provider Protocol Constraint (Current Mux Architecture)

This provider currently serves through a tf5 mux path with framework resources downgraded from
protocol v6 to v5 (`pkg/provider/provider.go`). Under this architecture:

- `WriteOnly` is supported by the downgrade layer.
- `SchemaAttribute.NestedType` (framework nested attributes) is **not** supported.

Plan implication:

- Secure fields must be modeled without framework nested attributes.
- The `secure` block remains `schema.SingleNestedBlock`, and each secure key is represented as a
  write-only map/object-shaped value (`map(string)`), preserving user UX:
  `secure.<key> = { create = ... }` or `secure.<key> = { name = ... }`.

This keeps secure support compatible with current provider-version/mux constraints while avoiding
future migration blockers.

### How Existing Grafana Provider Resources Handle Secrets

Every non-app-platform resource in the provider currently uses `Sensitive: true` — there are no
existing uses of `WriteOnly: true`. Common patterns:

| Resource | Secret Field | Pattern |
|---|---|---|
| `grafana_service_account_token` | `key` | `Computed: true, Sensitive: true` — token returned once on creation, stored in state |
| `grafana_cloud_access_policy_token` | `token` | Same — computed sensitive, stored in state |
| `grafana_sso_settings` | `client_secret` | `Optional: true, Sensitive: true` — OAuth2 client secret stored in state |
| `grafana_data_source` | `secure_json_data`, `http_headers` | `Sensitive: true` with custom `DiffSuppressFunc` for JSON normalization |
| `grafana_user` | `password` | `Required: true, Sensitive: true` — never returned by API on read, but stored in state |
| `grafana_contact_point` | `basic_auth_password`, `api_token`, etc. | `Sensitive: true` per notifier type — dozens of secret fields |
| `grafana_oncall_outgoing_webhook` | `password`, `authorization_header` | `Sensitive: true` with `DiffSuppressFunc` for preset-controlled fields |
| `grafana_cloud_provider_azure_credential` | `client_secret` | `Required: true, Sensitive: true` (plugin framework) — set to `""` on import |
| `grafana_connections_metrics_endpoint_scrape_job` | `authentication_bearer_token`, `authentication_basic_password` | `Sensitive: true` — intentionally not set during Read (state preserves original) |

**Common workarounds for API not returning secrets on read:**
- Most resources simply don't overwrite the sensitive field during Read, relying on Terraform's
  state to preserve the value from the last Create/Update
- `grafana_data_source` uses dummy values (`"true"`) for HTTP header values the API won't return
- `grafana_user` uses `HasChange("password")` to avoid unnecessary Update calls

**Why this is problematic:**
- All these secrets are stored **plaintext in `terraform.tfstate`** (local or remote backend)
- Anyone with state access (S3 bucket, Terraform Cloud, local filesystem) can read every secret
- `Sensitive: true` only redacts values from `terraform plan` CLI output — it is a UI concern, not a security boundary

The `WriteOnly: true` approach adopted in this plan eliminates this entire class of risk for
new app platform resources.

### Resulting Terraform Schema (when a resource opts in)

```hcl
resource "grafana_apps_<group>_<kind>_<version>" "example" {
  metadata { uid = "..." }

  spec {
    # resource-specific config
  }

  secure {
    # resource-specific secrets — all WriteOnly, never in state
    some_token = var.my_token
  }
  secure_version = 1  # increment to re-apply all secure values

  options {
    overwrite = true
  }
}
```

Resources without secrets simply omit `SecureValueAttributes` and get the current behavior unchanged.

### UX Note: Secure Input Shape

Each secure key is configured as an object with exactly one of:
- `create` for inline secret creation/rotation
- `name` for referencing an existing secret

Internally (schema), this is modeled as a write-only `map(string)` per secure key to stay
compatible with the current tf6->tf5 downgrade path.

```hcl
secure {
  token = {
    create = var.token
  }
  client_secret = {
    name = "my-existing-secret-name"
  }
}
```

### Secure Key Deletion Semantics (No Exposed `remove` Attribute)

The Terraform `secure` block remains write-only and does **not** expose `secure.<key>.remove`.
Secure values are provided as object form (`create` or `name`).

Framework behavior is schema-driven:

1. Parse configured secure values from `req.Config` and set create/name values on the API object.
2. On Update, for every key declared in `SecureValueAttributes` that is **not** configured in
   current `secure` config, inject `InlineSecureValue{Remove: true}`.

This yields expected UX:

```hcl
secure {
  token = {
    create = var.token
  }
}
secure_version = 1
```

Later:

```hcl
secure {
  # token removed from config
}
secure_version = 2
```

`token` is removed in the API, without storing secret values in state and without adding a
Terraform `remove` field.

Important scope note:
- This only applies to keys declared in `SecureValueAttributes`.
- The provider cannot remove arbitrary unknown remote secure keys that are not part of schema.

### Secure Rotation Semantics

Because write-only values are not stored in plan/state, changing only `secure.*` values without
changing a persisted argument does not guarantee an update operation. The contract for this design
is:

- users increment `secure_version` when they want secure changes (create/update/remove) re-applied
- providers document this explicitly on secure-enabled resources

If an update operation runs for any reason, secure-key reconciliation is applied from current
config and schema-known key set. In practice, for secure-only changes, `secure_version` is the trigger.

### No-Op Guarantee for Existing Resources

The implementation must not alter behavior for existing app platform resources that do not define
`SecureValueAttributes`.

Concretely:

- Keep the base `ResourceModel` unchanged for non-secure resources.
- Use a secure-enabled model path only when `SecureValueAttributes` is non-empty.
- Do not rely on conditional schema fields with an unconditional model struct, because plugin
  framework object/struct mapping requires a field-for-field match.

---

## Changes to `resource.go`

### 1. Extend `ResourceSpecSchema`

```go
type ResourceSpecSchema struct {
    Description         string
    MarkdownDescription string
    DeprecationMessage  string
    SpecAttributes      map[string]schema.Attribute
    SpecBlocks          map[string]schema.Block
    SecureValueAttributes map[string]SecureValueAttribute  // NEW — write-only secure object fields
}

type SecureValueAttribute struct {
    Description         string
    MarkdownDescription string
    DeprecationMessage  string
    Required            bool
    Optional            bool
    APIName             string // optional explicit API key; defaults to Terraform key
}
```

### 2. Keep `ResourceModel` Unchanged; Use Attribute-Path Access for Secure Fields

```go
type ResourceModel struct {
    ID       types.String `tfsdk:"id"`
    Metadata types.Object `tfsdk:"metadata"`
    Spec     types.Object `tfsdk:"spec"`
    Options  types.Object `tfsdk:"options"`
}
```

For secure-enabled resources, read/write `secure` and `secure_version` via top-level attribute
paths (`GetAttribute` / `SetAttribute`) while reusing `ResourceModel` for shared fields
(`id`, `metadata`, `spec`, `options`). This preserves existing behavior for resources without
`SecureValueAttributes`.

#### Why We Cannot Add `Secure` to `ResourceModel`

If we add:

```go
Secure        types.Object `tfsdk:"secure"`
SecureVersion types.Int64  `tfsdk:"secure_version"`
```

directly to `ResourceModel`, non-secure resources fail model decoding because their schema does
not contain those attributes. Terraform Plugin Framework struct/object decoding requires an exact
field match between `tfsdk` tags and schema fields in both directions.

Practical consequence:
- `req.Plan.Get(ctx, &data)` / `req.State.Get(ctx, &data)` returns diagnostics for non-secure
  resources (for example: struct fields not found in object, such as `secure`).
- This is not a panic; it is a hard diagnostic error that blocks CRUD.
- There is no conditional `tfsdk` tag mechanism to make struct fields optional by schema shape.

Because of this, secure attributes must be handled conditionally via attribute-path access, while
the base `ResourceModel` remains unchanged.

### 3. Add `SecureParser` to `ResourceConfig`

```go
type SecureParser[T sdkresource.Object] func(ctx context.Context, secure types.Object, dst T) diag.Diagnostics

type ResourceConfig[T sdkresource.Object] struct {
    Schema      ResourceSpecSchema
    Kind        sdkresource.Kind
    SpecParser  SpecParser[T]
    SpecSaver   SpecSaver[T]
    SecureParser SecureParser[T]  // NEW — nil means no secure support
}
```

The `SecureParser` reads write-only values from a `types.Object` (extracted from `req.Config`)
and populates the `Secure` field on the API object using `InlineSecureValue` values.

To avoid resource-by-resource boilerplate for the common case, provide a framework helper that
resources can pass directly:

```go
SecureParser: DefaultSecureParser[*MyType],
```

`DefaultSecureParser` iterates the `secure` object attributes and writes each configured nested
object to the destination object's `Secure` field as either:
- `InlineSecureValue{Create: ...}` from `create`
- `InlineSecureValue{Name: ...}` from `name`

Key mapping is explicit:
- Terraform uses `SecureValueAttributes` map keys (typically snake_case).
- API destination key defaults to the same key.
- If API key differs, resource author sets `SecureValueAttribute.APIName`.
  Example: `client_secret` (Terraform) -> `clientSecret` (API).

This avoids heuristic case conversion and keeps parser behavior deterministic.

### 4. Schema Generation

In the `Schema()` method, conditionally add the `secure` block and `secure_version` attribute:

```go
func (r *Resource[T, L]) Schema(ctx context.Context, req resource.SchemaRequest, res *resource.SchemaResponse) {
    sch := r.config.Schema

    attrs := map[string]schema.Attribute{
        "id": schema.StringAttribute{ /* existing */ },
    }
    blocks := map[string]schema.Block{
        "metadata": /* existing */,
        "spec":     /* existing */,
        "options":  /* existing */,
    }

    if len(sch.SecureValueAttributes) > 0 {
        if r.config.SecureParser == nil {
            res.Diagnostics.AddError("Invalid resource secure configuration",
                "SecureValueAttributes is configured, but SecureParser is nil.")
        }
        secureAttrs, secureDiags := buildSecureValueSchemaAttributes(sch.SecureValueAttributes)
        res.Diagnostics.Append(secureDiags...)

        blocks["secure"] = schema.SingleNestedBlock{
            Description: "Sensitive credentials. Values are write-only and never stored in Terraform state.",
            Attributes:  secureAttrs,
        }
        attrs["secure_version"] = schema.Int64Attribute{
            Optional:    true,
            Description: "Increment this value to trigger re-application of all secure values.",
        }
    } else if r.config.SecureParser != nil {
        res.Diagnostics.AddError("Invalid resource secure configuration",
            "SecureParser is configured, but SecureValueAttributes is empty.")
    }

    res.Schema = schema.Schema{
        Description:         sch.Description,
        MarkdownDescription: sch.MarkdownDescription,
        DeprecationMessage:  sch.DeprecationMessage,
        Attributes:          attrs,
        Blocks:              blocks,
    }
}
```

### 5. CRUD Changes

**Create and Update** — read secure values from `req.Config` (not `req.Plan`), and use the
secure-enabled model path only when secure schema is configured:

```go
func (r *Resource[T, L]) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
    // non-secure resources continue using ResourceModel (no-op behavior)
    // secure-enabled resources still reuse ResourceModel for shared fields
    // and use GetAttribute/SetAttribute for secure + secure_version

    // For secure-enabled resources only: parse write-only values from req.Config
    if r.hasSecureSchema() {
        var secureObj types.Object
        diags := req.Config.GetAttribute(ctx, path.Root("secure"), &secureObj)
        resp.Diagnostics.Append(diags...)
        if resp.Diagnostics.HasError() {
            return
        }
        if diags := r.config.SecureParser(ctx, secureObj, obj); diags.HasError() {
            resp.Diagnostics.Append(diags...)
            return
        }

        // On Create, omitted secure keys are simply absent.
    }

    res, err := r.client.Create(ctx, obj, sdkresource.CreateOptions{})
    // ... existing logic ...
}
```

`Update` adds schema-driven removal reconciliation:

```go
if r.hasSecureSchema() {
    var secureObj types.Object
    diags := req.Config.GetAttribute(ctx, path.Root("secure"), &secureObj)
    // ... error handling ...

    // Parse current create/update secure values from config.
    diags = r.config.SecureParser(ctx, secureObj, obj)
    // ... error handling ...

    // For every schema-declared key omitted in config, inject Remove:true.
    err = applySchemaBasedSecureRemovals(obj, secureObj, r.config.Schema.SecureValueAttributes)
    // ... error handling ...
}
```

Secure state-shape handling (planned):

- If `secure` is configured in plan/config, write back a structurally present `secure` object with
  null child values (write-only values are never persisted).
- If `secure` is omitted, write back `secure = null`.

This avoids write-only presence/absence inconsistencies during apply.

Secure payload handling (planned):

- Secure values must be sent in the API payload as top-level `secure` content.
- Because secure `create` values are redacted by default JSON marshalling wrappers, secure payload
  serialization must explicitly preserve raw `create` strings when constructing request content.

**Read** — secure-enabled resources preserve `secure` block structural presence semantics
(present-empty vs null, as described above) and preserve `secure_version`; non-secure resources
remain unchanged.

**Delete** — no changes needed.

**ImportState** — secure values cannot be imported (they're write-only). For secure-enabled
resources, `secure` is set to `null` and `secure_version` is unset after import.

### 6. Handle Absent `secure` Block

When a resource has `SecureValueAttributes` but the user doesn't provide a `secure` block (e.g.,
a repository using Connection-based auth instead of a direct token), the `SecureParser` receives
a null `types.Object`. The parser should handle this gracefully by simply not setting any
secure values on the API object.

### 7. Generic Secure Subresource Support for Resource Authors

To minimize per-resource boilerplate when exposing API `secure` subresources, add a reusable helper layer:

- `secureSubresourceSupport[T]` (embeddable) for resources that only expose `secure`.
- `addSecureSubresource`, `getSecureSubresource`, and `setSecureSubresource` for resources that expose
  `secure` plus additional subresources (for example, `status`).

Secure payload construction is generic for both struct and map secure models containing
`InlineSecureValue` fields. API payload uses `inlineSecureValueSubresource(...)` so `create` values
are emitted as raw strings (not redacted wrapper JSON) when sending requests.

Secure-only resource shape:

```go
type MyResource struct {
    // ...
    secureSubresourceSupport[myv0alpha1.MySecure]
}
```

Resource with mixed subresources:

```go
func (o *MyResource) GetSubresources() map[string]any {
    return addSecureSubresource(map[string]any{"status": o.Status}, o.Secure)
}

func (o *MyResource) GetSubresource(name string) (any, bool) {
    if v, ok := getSecureSubresource(name, o.Secure); ok {
        return v, true
    }
    if name == "status" {
        return o.Status, true
    }
    return nil, false
}

func (o *MyResource) SetSubresource(name string, value any) error {
    if handled, err := setSecureSubresource(name, value, &o.Secure); handled {
        return err
    }
    // ... status branch ...
    return fmt.Errorf("subresource '%s' does not exist", name)
}
```

---

## Files Modified

| File | Change |
|---|---|
| `internal/resources/appplatform/resource.go` | Add `SecureValueAttributes`, `SecureParser`, attribute-path handling for secure fields, schema validation guardrails, config reading in Create/Update |
| `internal/resources/appplatform/secure_subresource_support.go` | Add generic secure subresource helpers (embeddable + composable) for API payload/accessor boilerplate |
| `internal/resources/appplatform/connection_resource.go` | Use generic secure subresource support for connection secure payload/accessors |
| `internal/resources/appplatform/repository_resource.go` | Use generic secure subresource support for repository secure payload/accessors |
| `internal/resources/appplatform/resource_test.go` | Add framework unit tests for secure schema inclusion/exclusion, guardrail validation, and schema-driven secure-key reconciliation |
| `internal/resources/appplatform/secure_subresource_support_test.go` | Add helper-focused unit tests for secure subresource payload/accessor behavior |

## Files NOT Modified

Existing resources (Dashboard, Playlist, AlertRule, AlertEnrichment, RecordingRule, AppO11yConfig,
K8sO11yConfig) are unchanged — they don't set `SecureValueAttributes` so they get the current behavior.

---

## Example: How a Resource Opts In

A resource with secrets provides `SecureValueAttributes` and, in the common case, uses the default
helper directly for `SecureParser`:

```go
func MyResource() NamedResource {
    return NewNamedResource[*MyType, *MyTypeList](
        common.CategoryGrafanaApps,
        ResourceConfig[*MyType]{
            Kind: MyTypeKind(),
            Schema: ResourceSpecSchema{
                Description: "...",
                SpecAttributes: map[string]schema.Attribute{
                    // ... spec fields ...
                },
                SecureValueAttributes: map[string]SecureValueAttribute{
                    "token": {
                        Optional:    true,
                        Description: "Auth token. Never stored in state.",
                    },
                },
            },
            SpecParser:   parseMySpec,
            SpecSaver:    saveMySpec,
            SecureParser: DefaultSecureParser[*MyType],
        },
    )
}
```

Use a custom parser only when secure fields need extra validation or transformation, the mapping is not
field-to-field, or resource-specific transformation logic is required.

---

## Testing Plan

The secure block framework must be tested and working independently before any resource (e.g.,
git sync Repository/Connection) builds on top of it.

### 1. Unit Tests (`resource_test.go`, `secure_subresource_support_test.go`)

Extend the existing unit tests to verify framework-level behavior without needing a live Grafana
instance. These run with `go test` — no `TF_ACC` required.

**Test: Schema generation with SecureValueAttributes**

Verify that when `SecureValueAttributes` is non-empty, the generated schema includes the `secure`
block and `secure_version` attribute. When `SecureValueAttributes` is empty/nil, verify these are absent.

```go
func TestSchemaIncludesSecureBlock(t *testing.T) {
    // Create a Resource with SecureValueAttributes populated
    r := NewResource[*MockType, *MockTypeList](ResourceConfig[*MockType]{
        Kind: mockKind(),
        Schema: ResourceSpecSchema{
            Description: "test",
            SpecAttributes: map[string]schema.Attribute{
                "name": schema.StringAttribute{Required: true},
            },
            SecureValueAttributes: map[string]SecureValueAttribute{
                "token": SecureValueAttribute{Optional: true},
            },
        },
        SpecParser:   mockSpecParser,
        SpecSaver:    mockSpecSaver,
        SecureParser: mockSecureParser,
    })

    var req resource.SchemaRequest
    var res resource.SchemaResponse
    r.Schema(context.Background(), req, &res)

    // Assert "secure" block exists
    _, hasSecure := res.Schema.Blocks["secure"]
    require.True(t, hasSecure, "schema should include secure block")

    // Assert "secure_version" attribute exists
    _, hasVersion := res.Schema.Attributes["secure_version"]
    require.True(t, hasVersion, "schema should include secure_version attribute")
}

func TestSchemaExcludesSecureBlockWhenNoSecureValueAttributes(t *testing.T) {
    // Create a Resource WITHOUT SecureValueAttributes (existing behavior)
    r := NewResource[*MockType, *MockTypeList](ResourceConfig[*MockType]{
        Kind: mockKind(),
        Schema: ResourceSpecSchema{
            Description: "test",
            SpecAttributes: map[string]schema.Attribute{
                "name": schema.StringAttribute{Required: true},
            },
            // SecureValueAttributes: nil — not set
        },
        SpecParser: mockSpecParser,
        SpecSaver:  mockSpecSaver,
    })

    var req resource.SchemaRequest
    var res resource.SchemaResponse
    r.Schema(context.Background(), req, &res)

    _, hasSecure := res.Schema.Blocks["secure"]
    require.False(t, hasSecure, "schema should NOT include secure block")

    _, hasVersion := res.Schema.Attributes["secure_version"]
    require.False(t, hasVersion, "schema should NOT include secure_version attribute")
}
```

**Test: SecureParser receives null when secure block is absent**

Verify that when a user omits the `secure` block, the `SecureParser` receives a null object
and the resource creates successfully without errors.

**Test: Schema validation for parser/attributes mismatch (both directions)**

Verify both guardrails:
- `SecureValueAttributes` without `SecureParser` returns schema diagnostics.
- `SecureParser` without `SecureValueAttributes` returns schema diagnostics.

**Test: DefaultSecureParser writes secure values**

Verify that `DefaultSecureParser` parses each `secure.<key>` object into destination `Secure`
fields as `InlineSecureValue{Create: ...}` or `InlineSecureValue{Name: ...}`, and rejects
invalid secure object shapes with diagnostics.

Include both map-typed and struct-typed `Secure` destinations. For struct-typed destinations,
cover explicit `APIName` mapping (for example `client_secret` -> `clientSecret`) and tags that
include options such as `,omitzero,omitempty`.

For map-typed destinations, verify keys are written exactly as resolved by the explicit mapping.

**Test: DefaultSecureParser null/unknown inputs**

Verify that null and unknown secure objects return no diagnostics and do not mutate destination
secure fields.

Also verify a non-null object with all secure attributes set to null (`secure {}` with no values)
is treated as a no-op.

**Test: Helper edge coverage**

Verify explicit APIName mapping validation (duplicate/blank API names) and cover the MetaAccessor
fallback path with an object that has no direct `Secure` field.

Cover defensive error paths for:
- struct secure destination with unknown secure keys
- map secure destination with incompatible key/value destination types

**Test: Secure-path field sync with ResourceModel**

Verify secure-path helpers read/write exactly the same base attribute set as `ResourceModel`
(`id`, `metadata`, `spec`, `options`) to prevent drift between non-secure and secure code paths.

**Test: Schema-driven secure key reconciliation**

Verify helper logic derives configured secure API keys from current config and applies
`InlineSecureValue{Remove: true}` for schema-declared keys omitted in config.

**Test: Omitted secure block removes all schema-declared keys**

Given no configured secure keys in current config, verify reconciliation applies
`InlineSecureValue{Remove: true}` for every key declared in `SecureValueAttributes`.

**Test: Existing resources unchanged**

Verify that `Playlist`, `Dashboard`, etc. — resources that don't set `SecureValueAttributes` —
produce identical schemas before and after the framework change (regression test).
Prefer a table-driven check over all currently registered app platform resources.

**Test: Secure subresource helper coverage**

Verify `secureSubresourceSupport[T]` and helper functions:
- secure-only accessor behavior (`GetSubresources`, `GetSubresource`, `SetSubresource`)
- mixed-subresource merge behavior via `addSecureSubresource(...)`
- payload filtering for zero/unsupported fields across struct and map secure models
- typed assignment errors from `setSecureSubresource(...)` remain clear for resource authors

### 2. Acceptance Test Strategy

For framework-only work, rely on unit tests first. Acceptance lifecycle validation should come
from the first real secure app platform resource (e.g., Repository/Connection) that is registered
in provider resources.

Avoid adding a test-only app platform resource in `_test.go` as the primary acceptance strategy:
provider resource registration is static, so package-local test-only resources are not
automatically exposed through acceptance test provider factories.

### 3. What the Tests Validate

| Test | What it proves |
|---|---|
| Schema unit test (with SecureValueAttributes) | `secure` block and `secure_version` appear in schema |
| Schema unit test (without SecureValueAttributes) | Existing resources are unaffected (regression) |
| Schema validation (secure value attribute configuration) | Invalid required/optional combinations are rejected |
| Schema validation (parser/schema pairing, both directions) | Opt-in contract is explicit and safe |
| Default secure parser unit test (map + struct secure fields) | Resources can use `SecureParser: DefaultSecureParser[...]` with explicit Terraform-key -> API-key mapping |
| Default secure parser null/unknown unit test | Omitted secure block remains a safe no-op |
| Helper edge coverage unit test | APIName validation and MetaAccessor fallback behavior are protected |
| Parser defensive-path unit test | Unknown struct keys and incompatible map key/value destination types return clear diagnostics |
| Secure-path field sync unit test | Secure helper read/write paths stay aligned with `ResourceModel` fields |
| Schema-driven reconciliation unit test | Missing schema-declared keys become `InlineSecureValue{Remove:true}` on Update |
| Omitted secure block reconciliation unit test | Empty/null secure config removes all schema-declared keys |
| Secure subresource helper unit test | Generic secure helper APIs keep resource-level subresource code minimal and predictable |
| Acceptance with first real secure resource | Full Create/Read/Update/Import lifecycle for `secure` + `secure_version` |
| Existing resource tests pass | Playlist, Dashboard, AlertRule etc. tests continue to pass without changes |

### 4. Docker Compose / CI Requirements

The unit tests require no infrastructure. The acceptance tests require:

- A running Grafana instance (the existing `docker-compose.yml` setup)
- `TF_ACC=1` and `GRAFANA_URL`/`GRAFANA_AUTH` environment variables set
- Terraform >= 1.11 in the test runner (write-only attributes are a 1.11 feature)

Add to `docker-compose.yml` `GF_FEATURE_TOGGLES_ENABLE` if any provisioning-specific flags are
needed (none for the framework test itself, but will be needed later for git sync resource tests).

---

## Discarded Alternative: Per-Field `_version` Inside the Secure Block

An alternative considered was having the framework automatically generate a companion `_version`
attribute for every field inside the `secure` block, following the `_wo` / `_wo_version` naming
convention used by the AWS and AzureRM providers.

### What It Would Look Like

The resource author would provide only the secret fields in `SecureValueAttributes`. The framework
would automatically inject a sibling `<field>_version` attribute for each one:

```hcl
resource "grafana_apps_provisioning_repository_v0alpha1" "example" {
  metadata { uid = "my-repo" }

  spec { ... }

  secure {
    token           = var.github_token         # WriteOnly — never in state
    token_version   = 1                        # auto-generated by framework, stored in state

    webhook_secret         = var.webhook_secret # WriteOnly — never in state
    webhook_secret_version = 1                  # auto-generated by framework, stored in state
  }
}
```

### Framework Implementation

In `Schema()`, for each entry in `SecureValueAttributes`, the framework would add a second attribute
with the `_version` suffix:

```go
if len(sch.SecureValueAttributes) > 0 {
    secureAttrs := make(map[string]schema.Attribute)
    for name, attr := range sch.SecureValueAttributes {
        secureAttrs[name] = attr
        secureAttrs[name+"_version"] = schema.Int64Attribute{
            Optional:    true,
            Description: fmt.Sprintf("Increment to trigger re-application of %s.", name),
        }
    }
    blocks["secure"] = schema.SingleNestedBlock{
        Description: "Sensitive credentials. Values are write-only and never stored in Terraform state.",
        Attributes:  secureAttrs,
    }
}
```

### Why This Was Discarded

**Cons:**

- **Breaks 1:1 mapping with API struct.** The API's `Secure` struct has fields like `Token`,
  `WebhookSecret`, `ClientSecret`. The Terraform `secure` block would have twice as many fields
  (`token`, `token_version`, `webhook_secret`, `webhook_secret_version`). This makes the secure
  block no longer a clean mirror of the API — resource authors and users need to mentally filter
  out the version fields.

- **Clutters the user-facing HCL.** For a resource with 3 secrets (e.g., Connection has
  `private_key`, `client_secret`, `token`), the secure block balloons to 6 fields. The version
  fields are boilerplate that adds noise to every configuration.

- **Harder to "rotate all secrets at once".** If a user wants to re-apply all secrets (e.g.,
  after a credential rotation), they must increment N separate version fields. With the chosen
  design, they increment a single `secure_version`.

- **Framework magic is surprising.** The framework silently injects attributes that don't appear
  in the resource author's `SecureValueAttributes` map. This makes the schema harder to reason about —
  `SecureValueAttributes` no longer represents the full set of attributes in the `secure` block.

- **`SecureParser` complexity.** The parser receives a `types.Object` that now contains both
  secret values (write-only, null in state) and version values (normal, stored in state). The
  parser must distinguish between them, adding logic that doesn't exist in the simpler design.

**Pros:**

- **Per-secret change detection.** A user can rotate just one secret without re-sending all of
  them. With the chosen single `secure_version` approach, incrementing it re-applies every secret
  even if only one changed.

- **Matches AWS/Azure convention.** The `_wo` / `_wo_version` pattern is documented as the
  standard approach in the [Terraform plugin framework docs](https://developer.hashicorp.com/terraform/plugin/framework/resources/write-only-arguments)
  and the [AzureRM contributor guide](https://github.com/hashicorp/terraform-provider-azurerm/blob/main/contributing/topics/guide-new-write-only-attribute.md).
  Users familiar with `aws_db_instance.password_wo_version` would find this pattern recognizable.

- **No unnecessary API writes.** If only `token` changed but `webhook_secret` didn't, a
  per-field version lets the `SecureParser` skip unchanged secrets. With single `secure_version`,
  the parser always sends all secrets (though the API is generally idempotent for unchanged values).

### Why Single `secure_version` Was Chosen Instead

The Grafana app platform's `Secure` struct is a well-defined API contract. Keeping the Terraform
`secure` block as a 1:1 mirror of that struct is more valuable than per-secret granularity because:

1. Resource authors define `SecureValueAttributes` by looking at the Go struct — no mental translation
2. Users write HCL that mirrors the API docs — no extra `_version` fields to learn
3. Secret rotation is typically an "all at once" operation anyway (credentials from one vault path)
4. The overhead of re-sending unchanged secrets is negligible — the API is idempotent
5. The single version field lives outside the `secure` block, keeping concerns separated

---

## Source File References

| File | Purpose |
|---|---|
| `internal/resources/appplatform/resource.go` | Generic Resource[T, L] framework — primary file to modify |
| `internal/resources/appplatform/appo11y_config_resource.go` | Example of self-contained resource (no secure fields) |
| `internal/resources/appplatform/playlist_resource.go` | Example of resource using external Kind (no secure fields) |
