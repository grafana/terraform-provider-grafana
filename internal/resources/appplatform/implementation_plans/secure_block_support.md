# App Platform Generic Secure Block Support — Implementation Plan

## Summary

Extend the generic `Resource[T, L]` framework in `resource.go` to support an optional `secure`
block with **write-only attributes** (Terraform 1.11+). This is a reusable framework-level change
that any app platform resource with secret fields can opt into.

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
values when incremented. This keeps the `secure` block a clean 1:1 mirror of the API's `Secure`
struct — no Terraform-only version fields mixed in.

**Why write-only over `Sensitive: true`:**
- `Sensitive: true` only masks values in CLI output — secrets are still stored **plaintext in state**
- `WriteOnly: true` means the value is passed to the provider at apply time then **discarded entirely**

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

Resources without secrets simply omit `SecureAttributes` and get the current behavior unchanged.

### Secure Rotation Semantics

Because write-only values are not stored in plan/state, changing only `secure.*` values without
changing a persisted argument does not guarantee an update operation. The contract for this design
is:

- users increment `secure_version` when they want secure values re-applied
- providers document this explicitly on secure-enabled resources

### No-Op Guarantee for Existing Resources

The implementation must not alter behavior for existing app platform resources that do not define
`SecureAttributes`.

Concretely:

- Keep the base `ResourceModel` unchanged for non-secure resources.
- Use a secure-enabled model path only when `SecureAttributes` is non-empty.
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
    SecureAttributes    map[string]schema.Attribute  // NEW — when non-empty, enables secure block
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
`SecureAttributes`.

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
and populates the `Secure` field on the API object using `InlineSecureValue{Create: value}`.

To avoid resource-by-resource boilerplate for the common case, provide a framework helper that
resources can pass directly:

```go
SecureParser: DefaultSecureParser[*MyType],
```

`DefaultSecureParser` iterates the `secure` object attributes and writes all non-null string values
to the destination object's `Secure` field as `InlineSecureValue{Create: ...}`.

For struct-typed `Secure` fields, the helper maps Terraform snake_case attribute names to JSON
tag field names (including camelCase tags such as `clientSecret` and `webhookSecret`) by matching
against the struct field JSON tags. Tag option suffixes (for example `,omitzero,omitempty`) are
treated as metadata and not part of the field name.

For map-typed `Secure` fields, the helper normalizes Terraform snake_case keys to lower camelCase
before writing to the secure map (for example `client_secret` -> `clientSecret`).

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

    // Conditionally add secure support
    if len(sch.SecureAttributes) > 0 {
        // Validation guardrails
        if r.config.SecureParser == nil {
            res.Diagnostics.AddError("Invalid resource secure configuration",
                "SecureAttributes is configured, but SecureParser is nil.")
        }
        for name, attr := range sch.SecureAttributes {
            if !attr.IsWriteOnly() {
                res.Diagnostics.AddError("Invalid secure attribute configuration",
                    fmt.Sprintf("Secure attribute %q must set WriteOnly: true.", name))
            }
        }

        blocks["secure"] = schema.SingleNestedBlock{
            Description: "Sensitive credentials. Values are write-only and never stored in Terraform state.",
            Attributes:  sch.SecureAttributes,
        }
        attrs["secure_version"] = schema.Int64Attribute{
            Optional:    true,
            Description: "Increment this value to trigger re-application of all secure values.",
        }
    } else if r.config.SecureParser != nil {
        res.Diagnostics.AddError("Invalid resource secure configuration",
            "SecureParser is configured, but SecureAttributes is empty.")
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
    }

    res, err := r.client.Create(ctx, obj, sdkresource.CreateOptions{})
    // ... existing logic ...
}
```

Same pattern for `Update`.

**Read** — secure-enabled resources write `secure` back to state as `null` object and preserve
`secure_version`; non-secure resources remain unchanged.

**Delete** — no changes needed.

**ImportState** — secure values cannot be imported (they're write-only). For secure-enabled
resources, `secure` is set to `null` and `secure_version` is unset after import. The user must
provide secure values in config after importing.

### 6. Handle Absent `secure` Block

When a resource has `SecureAttributes` but the user doesn't provide a `secure` block (e.g.,
a repository using Connection-based auth instead of a direct token), the `SecureParser` receives
a null `types.Object`. The parser should handle this gracefully by simply not setting any
secure values on the API object.

---

## Files Modified

| File | Change |
|---|---|
| `internal/resources/appplatform/resource.go` | Add `SecureAttributes`, `SecureParser`, attribute-path handling for secure fields, schema validation guardrails, config reading in Create/Update |
| `internal/resources/appplatform/resource_test.go` | Add framework unit tests for secure schema inclusion/exclusion and guardrail validation |

## Files NOT Modified

Existing resources (Dashboard, Playlist, AlertRule, AlertEnrichment, RecordingRule, AppO11yConfig,
K8sO11yConfig) are unchanged — they don't set `SecureAttributes` so they get the current behavior.

---

## Example: How a Resource Opts In

A resource with secrets provides `SecureAttributes` and, in the common case, uses the default
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
                SecureAttributes: map[string]schema.Attribute{
                    "token": schema.StringAttribute{
                        Optional:    true,
                        WriteOnly:   true,
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

Use a custom parser only when secure fields are not simple string secrets, the mapping is not
field-to-field, or resource-specific transformation logic is required.

---

## Testing Plan

The secure block framework must be tested and working independently before any resource (e.g.,
git sync Repository/Connection) builds on top of it.

### 1. Unit Tests (`resource_test.go`)

Extend the existing unit tests to verify framework-level behavior without needing a live Grafana
instance. These run with `go test` — no `TF_ACC` required.

**Test: Schema generation with SecureAttributes**

Verify that when `SecureAttributes` is non-empty, the generated schema includes the `secure`
block and `secure_version` attribute. When `SecureAttributes` is empty/nil, verify these are absent.

```go
func TestSchemaIncludesSecureBlock(t *testing.T) {
    // Create a Resource with SecureAttributes populated
    r := NewResource[*MockType, *MockTypeList](ResourceConfig[*MockType]{
        Kind: mockKind(),
        Schema: ResourceSpecSchema{
            Description: "test",
            SpecAttributes: map[string]schema.Attribute{
                "name": schema.StringAttribute{Required: true},
            },
            SecureAttributes: map[string]schema.Attribute{
                "token": schema.StringAttribute{Optional: true, WriteOnly: true},
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

func TestSchemaExcludesSecureBlockWhenNoSecureAttributes(t *testing.T) {
    // Create a Resource WITHOUT SecureAttributes (existing behavior)
    r := NewResource[*MockType, *MockTypeList](ResourceConfig[*MockType]{
        Kind: mockKind(),
        Schema: ResourceSpecSchema{
            Description: "test",
            SpecAttributes: map[string]schema.Attribute{
                "name": schema.StringAttribute{Required: true},
            },
            // SecureAttributes: nil — not set
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
- `SecureAttributes` without `SecureParser` returns schema diagnostics.
- `SecureParser` without `SecureAttributes` returns schema diagnostics.

**Test: DefaultSecureParser writes secure values**

Verify that `DefaultSecureParser` copies non-null string fields from the `secure` object into
the destination object's `Secure` fields as `InlineSecureValue{Create: ...}`, and rejects
unsupported non-string secure fields with diagnostics.

Include both map-typed and struct-typed `Secure` destinations. For struct-typed destinations,
cover snake_case Terraform attributes mapped to camelCase JSON tags and tags that include options
such as `,omitzero,omitempty`.

For map-typed destinations, verify snake_case Terraform keys are normalized to lower camelCase
before writing secure values.

**Test: DefaultSecureParser null/unknown inputs**

Verify that null and unknown secure objects return no diagnostics and do not mutate destination
secure fields.

Also verify a non-null object with all secure attributes set to null (`secure {}` with no values)
is treated as a no-op.

**Test: Helper edge coverage**

Verify snake_case conversion handles acronym-style field names (`URLToken` -> `url_token`) and
cover the MetaAccessor fallback path with an object that has no direct `Secure` field.

Cover defensive error paths for:
- struct secure destination with unknown secure keys
- map secure destination with incompatible key/value destination types

**Test: Secure-path field sync with ResourceModel**

Verify secure-path helpers read/write exactly the same base attribute set as `ResourceModel`
(`id`, `metadata`, `spec`, `options`) to prevent drift between non-secure and secure code paths.

**Test: Existing resources unchanged**

Verify that `Playlist`, `Dashboard`, etc. — resources that don't set `SecureAttributes` —
produce identical schemas before and after the framework change (regression test).
Prefer a table-driven check over all currently registered app platform resources.

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
| Schema unit test (with SecureAttributes) | `secure` block and `secure_version` appear in schema |
| Schema unit test (without SecureAttributes) | Existing resources are unaffected (regression) |
| Schema validation (secure attrs must be write-only) | Secret fields cannot accidentally fall back to state-stored values |
| Schema validation (parser/schema pairing, both directions) | Opt-in contract is explicit and safe |
| Default secure parser unit test (map + struct secure fields) | Resources can use `SecureParser: DefaultSecureParser[...]` with snake_case->camelCase field mapping |
| Default secure parser null/unknown unit test | Omitted secure block remains a safe no-op |
| Helper edge coverage unit test | Acronym snake_case conversion and MetaAccessor fallback behavior are protected |
| Parser defensive-path unit test | Unknown struct keys and incompatible map key/value destination types return clear diagnostics |
| Secure-path field sync unit test | Secure helper read/write paths stay aligned with `ResourceModel` fields |
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

The resource author would provide only the secret fields in `SecureAttributes`. The framework
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

In `Schema()`, for each entry in `SecureAttributes`, the framework would add a second attribute
with the `_version` suffix:

```go
if len(sch.SecureAttributes) > 0 {
    secureAttrs := make(map[string]schema.Attribute)
    for name, attr := range sch.SecureAttributes {
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
  in the resource author's `SecureAttributes` map. This makes the schema harder to reason about —
  `SecureAttributes` no longer represents the full set of attributes in the `secure` block.

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

1. Resource authors define `SecureAttributes` by looking at the Go struct — no mental translation
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
