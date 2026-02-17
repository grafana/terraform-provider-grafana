# Secure Values for App Platform Resources

This guide explains how to add `secure` support to a resource built with `Resource[T, L]` in
`internal/resources/appplatform`.

## When to use this

Use `secure` when your resource needs secrets (tokens, keys, client secrets) that must **not**
be stored in Terraform state.

The framework supports this via:
- `secure { ... }` block with framework-defined write-only string attributes
- `secure_version` trigger field at resource root

## Quick checklist

1. Add `SecureValueAttributes` to your resource schema.
2. Set `SecureParser`:
   - Use `DefaultSecureParser[*MyType]` for normal string secrets.
   - Use a custom parser when you need validation that depends on `spec` (auth mode, mutually exclusive fields, required combinations).
3. Document for users: bump `secure_version` to force re-apply of secure values.

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
    token         = var.token
    client_secret = var.client_secret
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
	Token      types.String `tfsdk:"token"`
	PrivateKey types.String `tfsdk:"private_key"`
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
		if model.Token.IsNull() || model.Token.IsUnknown() || model.Token.ValueString() == "" {
			diags.AddError("invalid secure configuration", "`secure.token` is required when `spec.auth_type = \"pat\"`")
			return diags
		}
	case "github_app":
		if model.PrivateKey.IsNull() || model.PrivateKey.IsUnknown() || model.PrivateKey.ValueString() == "" {
			diags.AddError("invalid secure configuration", "`secure.private_key` is required when `spec.auth_type = \"github_app\"`")
			return diags
		}
	default:
		diags.AddError("invalid spec configuration", "unsupported `spec.auth_type`")
		return diags
	}

	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		diags.AddError("failed to parse secure values", err.Error())
		return diags
	}

	secureValues := apicommon.InlineSecureValues{}
	if !model.Token.IsNull() && !model.Token.IsUnknown() && model.Token.ValueString() != "" {
		secureValues["token"] = apicommon.InlineSecureValue{
			Create: apicommon.NewSecretValue(model.Token.ValueString()),
		}
	}
	if !model.PrivateKey.IsNull() && !model.PrivateKey.IsUnknown() && model.PrivateKey.ValueString() != "" {
		secureValues["privateKey"] = apicommon.InlineSecureValue{
			Create: apicommon.NewSecretValue(model.PrivateKey.ValueString()),
		}
	}

	err = meta.SetSecureValues(secureValues)
	if err != nil {
		diags.AddError("failed to parse secure values", err.Error())
	}

	return diags
}
```

## Runtime behavior

- Secure values are read from `req.Config`, not `req.Plan`.
- `secure` values are write-only and stored as `null` in state.
- `secure_version` is stored in state and used as the update trigger.
- If `secure` is omitted, parser receives null/unknown and should no-op.

## Common errors

- `SecureValueAttributes is configured, but SecureParser is nil`
  - Add `SecureParser` in `ResourceConfig`.
- `SecureParser is configured, but SecureValueAttributes is empty`
  - Remove parser or define `SecureValueAttributes`.
