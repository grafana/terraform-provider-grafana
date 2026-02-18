# Secure Values for App Platform Resources

This guide explains how to add `secure` support to a resource built with `Resource[T, L]` in
`internal/resources/appplatform`.

## When to use this

Use `secure` when your resource needs secrets (tokens, keys, client secrets) that must **not**
be stored in Terraform state.

The framework supports this via:
- `secure { ... }` block with framework-defined write-only secure attributes
- `secure_version` trigger field at resource root

## Quick checklist

1. Add `SecureValueAttributes` to your resource schema.
   - If neither `Required` nor `Optional` is set on a field, the framework defaults it to `Optional: true`.
   - Set `APIName` when Terraform key differs from API secure key (for example `client_secret` -> `clientSecret`).
2. Set `SecureParser`:
   - Use `DefaultSecureParser[*MyType]` for normal secure fields.
   - Use a custom parser when you need validation that depends on `spec` (auth mode, mutually exclusive fields, required combinations).
3. Expose secure subresource accessors:
   - If the resource only exposes `secure`, embed `secureSubresourceSupport[MySecure]`.
   - If the resource exposes other subresources too, use `addSecureSubresource`, `getSecureSubresource`, and `setSecureSubresource`.
4. Model `Secure` as `apicommon.InlineSecureValues` by default.
5. Document for users: bump `secure_version` to force re-apply of secure values.

## Secure field shape: map or struct 

```go
// v0alpha1/my_resource_types.go
type MyResourceSpec struct {
	Name     string `json:"name,omitempty"`
	AuthType string `json:"authType,omitempty"`
}

type MyResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MyResourceSpec              `json:"spec,omitempty"`
	Secure            apicommon.InlineSecureValues `json:"secure,omitempty"` // map form
}

type MyResourceSecure struct {
	Token        apicommon.InlineSecureValue `json:"token,omitzero,omitempty"`
	ClientSecret apicommon.InlineSecureValue `json:"clientSecret,omitzero,omitempty"`
}

type MyResourceWithStructSecure struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MyResourceSpec  `json:"spec,omitempty"`
	Secure            MyResourceSecure `json:"secure,omitempty"` // struct form
}
```

`Secure` does **not** have to be `apicommon.InlineSecureValues`.

With `DefaultSecureParser`, the destination can be either:
- `map[string]apicommon.InlineSecureValue` (most common)
- a custom struct whose fields are `apicommon.InlineSecureValue`

Recommended default:
- Use `apicommon.InlineSecureValues` unless you have a concrete reason not to.

Use a custom `Secure` struct only when:
- The API type already exposes `secure` as a struct and you want to keep that model.
- You want explicit per-field typing and JSON-tag mapping in one place.

Requirements for `DefaultSecureParser`:
- Resource object has an exported `Secure` field.
- `secure.<key>` must be an object with exactly one of:
  - `create` (set/rotate inline secret value)
  - `name` (reference existing secret by name)
- `DefaultSecureParser` uses exact key matching.
- If Terraform key differs from API key, configure `SecureValueAttribute.APIName`.
- Schema stores each secure key as a write-only `map(string)` (compatibility with current tf6->tf5 mux path), but HCL usage remains object-style (`{ create = ... }` / `{ name = ... }`).

## Example: default parser (recommended)

```go
func MyResource() NamedResource {
	return NewNamedResource[*v0alpha1.MyResource, *v0alpha1.MyResourceList](
		common.CategoryGrafanaApps,
		ResourceConfig[*v0alpha1.MyResource]{
			Kind: v0alpha1.MyResourceKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages my resource.",
				SpecAttributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{Required: true},
				},
				SecureValueAttributes: map[string]SecureValueAttribute{
					"token": {
						Optional: true,
					},
					"client_secret": {
						Optional: true,
						APIName:  "clientSecret",
					},
				},
			},
			SpecParser:   parseMySpec,
			SpecSaver:    saveMySpec,
			SecureParser: DefaultSecureParser[*v0alpha1.MyResource],
		},
	)
}
```

## Expose secure subresource methods

### Secure-only subresource (recommended when applicable)

```go
type MyResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              v0alpha1.MyResourceSpec `json:"spec,omitempty"`

	secureSubresourceSupport[v0alpha1.MyResourceSecure]
}
```

### Resource with `secure` plus other subresources

```go
func (o *MyResource) GetSubresources() map[string]any {
	return addSecureSubresource(map[string]any{
		"status": o.Status,
	}, o.Secure)
}

func (o *MyResource) GetSubresource(name string) (any, bool) {
	if value, ok := getSecureSubresource(name, o.Secure); ok {
		return value, true
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
	if name == "status" {
		cast, ok := value.(v0alpha1.MyResourceStatus)
		if !ok {
			return fmt.Errorf("cannot set status type %#v, not of type MyResourceStatus", value)
		}
		o.Status = cast
		return nil
	}
	return fmt.Errorf("subresource '%s' does not exist", name)
}
```

The helper preserves secure payload behavior, including forwarding raw `create` values in API
subresource payloads.

## Example HCL

```hcl
resource "grafana_apps_example_myresource_v0alpha1" "this" {
  metadata {
    uid = "my-resource"
  }

  spec {
    name = "example"
  }

  secure {
    token = {
      create = var.token
    }
    client_secret = {
      name = "existing-client-secret"
    }
  }

  secure_version = 1
}
```

## Example: custom parser

Use a custom parser only if you need it, for example - if `spec.auth_type` is:
- `pat`: `secure.token` is required
- `github_app`: `secure.private_key` is required

`DefaultSecureParser` cannot enforce these cross-field rules; a custom parser can.

```go
type mySecureModel struct {
	Token      types.Object `tfsdk:"token"`
	PrivateKey types.Object `tfsdk:"private_key"`
}

func isSecureValueProvided(v types.Object) bool {
	if v.IsNull() || v.IsUnknown() {
		return false
	}

	nameValue := v.Attributes()["name"]
	if !nameValue.IsNull() && !nameValue.IsUnknown() {
		return true
	}

	createValue := v.Attributes()["create"]
	if !createValue.IsNull() && !createValue.IsUnknown() {
		return true
	}

	return false
}

func parseMySecure(ctx context.Context, secure types.Object, dst *v0alpha1.MyResource) diag.Diagnostics {
	var diags diag.Diagnostics

	if secure.IsNull() || secure.IsUnknown() {
		return diags
	}

	var model mySecureModel
	diags.Append(secure.As(ctx, &model, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	})...)
	if diags.HasError() {
		return diags
	}

	// Example cross-field validation using already-parsed spec.
	switch dst.Spec.AuthType {
	case "pat":
		if !isSecureValueProvided(model.Token) {
			diags.AddError("invalid secure configuration", "`secure.token` is required when `spec.auth_type = \"pat\"`")
			return diags
		}
	case "github_app":
		if !isSecureValueProvided(model.PrivateKey) {
			diags.AddError("invalid secure configuration", "`secure.private_key` is required when `spec.auth_type = \"github_app\"`")
			return diags
		}
	default:
		diags.AddError("invalid spec configuration", "unsupported `spec.auth_type`")
		return diags
	}

	// Reuse framework mapping after custom validation.
	diags.Append(DefaultSecureParser(ctx, secure, dst)...)
	return diags
}
```

## Runtime behavior

- Secure values are read from `req.Config`, not `req.Plan`.
- `secure` values are write-only and never persisted in state.
- `secure` state shape is either:
  - `null` when `secure` is omitted, or
  - an object with null child values when `secure` is configured.
- `secure_version` is stored in state; changing it creates a Terraform diff that triggers Update.
- On Update, any secure key declared in `SecureValueAttributes` but omitted from current config is sent as `InlineSecureValue{Remove: true}`.
- No secure-key baseline is persisted in Terraform private state.
- `remove` is intentionally not exposed in Terraform schema; removal is framework-managed.
- If `secure` is omitted, parser receives null/unknown and no create/name values are set; on Update, schema-driven reconciliation treats all declared secure keys as omitted and sends removals.

## Common errors

- `SecureValueAttributes is configured, but SecureParser is nil`
  - Add `SecureParser` in `ResourceConfig`.
- `SecureParser is configured, but SecureValueAttributes is empty`
  - Remove parser or define `SecureValueAttributes`.
