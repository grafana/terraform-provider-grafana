# AppPlatform Field Type Mapping

Reference for mapping Go SDK types to Terraform Plugin Framework types. Each entry covers: schema attribute, model field, SpecParser (TF→Go), SpecSaver (Go→TF), and the `attr.Type` for `ObjectValue()` calls.

---

## Scalar Types

### `string` (required)

```go
// Schema
"name": schema.StringAttribute{Required: true, Description: "..."},

// Model
Name types.String `tfsdk:"name"`

// SpecParser (TF → Go)
dst.Spec.Name = data.Name.ValueString()

// SpecSaver (Go → TF)
data.Name = types.StringValue(src.Spec.Name)

// AttrTypes
"name": types.StringType,
```

### `string` (optional / pointer `*string`)

```go
// Schema
"description": schema.StringAttribute{Optional: true, Description: "..."},

// Model
Description types.String `tfsdk:"description"`

// SpecParser — pointer variant
if !data.Description.IsNull() && !data.Description.IsUnknown() {
    dst.Spec.Description = data.Description.ValueStringPointer()
}
// SpecParser — value variant
if !data.Description.IsNull() && !data.Description.IsUnknown() {
    dst.Spec.Description = data.Description.ValueString()
}

// SpecSaver — pointer variant
if src.Spec.Description != nil {
    data.Description = types.StringValue(*src.Spec.Description)
} else {
    data.Description = types.StringNull()
}
// SpecSaver — value variant (empty string = absent)
if src.Spec.Description != "" {
    data.Description = types.StringValue(src.Spec.Description)
} else {
    data.Description = types.StringNull()
}

// AttrTypes
"description": types.StringType,
```

### `int64` / `*int64`

```go
// Schema
"max_lines": schema.Int64Attribute{Optional: true, Description: "..."},

// Model
MaxLines types.Int64 `tfsdk:"max_lines"`

// SpecParser
if !data.MaxLines.IsNull() && !data.MaxLines.IsUnknown() {
    dst.Spec.MaxLines = data.MaxLines.ValueInt64()
}

// SpecSaver
if src.Spec.MaxLines != 0 {
    data.MaxLines = types.Int64Value(src.Spec.MaxLines)
} else {
    data.MaxLines = types.Int64Null()
}

// AttrTypes
"max_lines": types.Int64Type,
```

### `bool` / `*bool`

```go
// Schema
"enabled": schema.BoolAttribute{Optional: true, Description: "..."},

// Model
Enabled types.Bool `tfsdk:"enabled"`

// SpecParser
if !data.Enabled.IsNull() && !data.Enabled.IsUnknown() {
    dst.Spec.Enabled = data.Enabled.ValueBool()
}

// SpecSaver
data.Enabled = types.BoolValue(src.Spec.Enabled)
// or for pointer:
if src.Spec.Enabled != nil {
    data.Enabled = types.BoolValue(*src.Spec.Enabled)
} else {
    data.Enabled = types.BoolNull()
}

// AttrTypes
"enabled": types.BoolType,
```

### `float64` / `*float64`

```go
// Schema
"threshold": schema.Float64Attribute{Optional: true, Description: "..."},

// Model
Threshold types.Float64 `tfsdk:"threshold"`

// SpecParser
if !data.Threshold.IsNull() && !data.Threshold.IsUnknown() {
    dst.Spec.Threshold = data.Threshold.ValueFloat64()
}

// SpecSaver
if src.Spec.Threshold != 0 {
    data.Threshold = types.Float64Value(src.Spec.Threshold)
} else {
    data.Threshold = types.Float64Null()
}

// AttrTypes
"threshold": types.Float64Type,
```

---

## Collection Types

### `[]string` (list of strings)

```go
// Schema
"tags": schema.ListAttribute{
    Optional:    true,
    ElementType: types.StringType,
    Description: "...",
},

// Model
Tags types.List `tfsdk:"tags"`

// SpecParser
if !data.Tags.IsNull() && !data.Tags.IsUnknown() {
    tags := make([]string, 0, len(data.Tags.Elements()))
    for _, el := range data.Tags.Elements() {
        tags = append(tags, el.(types.String).ValueString())
    }
    dst.Spec.Tags = tags
}

// SpecSaver
if len(src.Spec.Tags) > 0 {
    tagVals := make([]attr.Value, len(src.Spec.Tags))
    for i, t := range src.Spec.Tags {
        tagVals[i] = types.StringValue(t)
    }
    tags, diags := types.ListValue(types.StringType, tagVals)
    if diags.HasError() {
        return diags
    }
    data.Tags = tags
} else {
    data.Tags = types.ListNull(types.StringType)
}

// AttrTypes
"tags": types.ListType{ElemType: types.StringType},
```

### `map[string]string`

```go
// Schema
"labels": schema.MapAttribute{
    Optional:    true,
    ElementType: types.StringType,
    Description: "...",
},

// Model
Labels types.Map `tfsdk:"labels"`

// SpecParser
if !data.Labels.IsNull() && !data.Labels.IsUnknown() {
    labels := make(map[string]string, len(data.Labels.Elements()))
    for k, v := range data.Labels.Elements() {
        labels[k] = v.(types.String).ValueString()
    }
    dst.Spec.Labels = labels
}

// SpecSaver
if len(src.Spec.Labels) > 0 {
    labelVals := make(map[string]attr.Value, len(src.Spec.Labels))
    for k, v := range src.Spec.Labels {
        labelVals[k] = types.StringValue(v)
    }
    labels, diags := types.MapValue(types.StringType, labelVals)
    if diags.HasError() {
        return diags
    }
    data.Labels = labels
} else {
    data.Labels = types.MapNull(types.StringType)
}

// AttrTypes
"labels": types.MapType{ElemType: types.StringType},
```

---

## Nested Object Types

### Struct → `schema.SingleNestedBlock` / `schema.SingleNestedAttribute`

Use `SingleNestedBlock` when the field is a block (not an attribute):

```go
// Schema — as Block
SpecBlocks: map[string]schema.Block{
    "config": schema.SingleNestedBlock{
        Description: "...",
        Attributes: map[string]schema.Attribute{
            "host": schema.StringAttribute{Required: true},
            "port": schema.Int64Attribute{Optional: true},
        },
    },
},

// Model — nested struct
type MySpecModel struct {
    Config MyConfigModel `tfsdk:"config"`
}

type MyConfigModel struct {
    Host types.String `tfsdk:"host"`
    Port types.Int64  `tfsdk:"port"`
}

// IMPORTANT: When SpecAttributes contains a nested block, the SpecParser
// receives src as types.Object whose "config" key is types.Object (not the struct).
// Use src.As() with the full model.

// SpecParser — nested via full model As()
var data MySpecModel
if diag := src.As(ctx, &data, basetypes.ObjectAsOptions{
    UnhandledNullAsEmpty:    true,
    UnhandledUnknownAsEmpty: true,
}); diag.HasError() {
    return diag
}
// data.Config is now a MyConfigModel with Host/Port populated
dst.Spec.Host = data.Config.Host.ValueString()
```

Use `SingleNestedAttribute` when the field is an attribute (no HCL `{}` block required):

```go
// Schema — as Attribute
"config": schema.SingleNestedAttribute{
    Optional: true,
    Attributes: map[string]schema.Attribute{
        "host": schema.StringAttribute{Required: true},
    },
},

// AttrTypes for ObjectValue
"config": types.ObjectType{AttrTypes: map[string]attr.Type{
    "host": types.StringType,
}},
```

---

## List of Objects

### `[]struct` → `schema.ListAttribute{ElementType: objectType}`

```go
// Define the element type as a package-level var
var itemType = types.ObjectType{
    AttrTypes: map[string]attr.Type{
        "type":  types.StringType,
        "value": types.StringType,
    },
}

// Schema
"items": schema.ListAttribute{
    Required:    true,
    ElementType: itemType,
    Description: "...",
},

// Model struct for items
type ItemModel struct {
    Type  types.String `tfsdk:"type"`
    Value types.String `tfsdk:"value"`
}

// SpecParser — list of objects
if !data.Items.IsNull() && !data.Items.IsUnknown() {
    items := make([]MySdkItem, 0, len(data.Items.Elements()))
    for _, el := range data.Items.Elements() {
        obj := el.(types.Object)
        var m ItemModel
        if diag := obj.As(ctx, &m, basetypes.ObjectAsOptions{
            UnhandledNullAsEmpty:    true,
            UnhandledUnknownAsEmpty: true,
        }); diag.HasError() {
            return diag
        }
        items = append(items, MySdkItem{
            Type:  m.Type.ValueString(),
            Value: m.Value.ValueString(),
        })
    }
    dst.Spec.Items = items
}

// SpecSaver — list of objects
if len(src.Spec.Items) > 0 {
    models := make([]ItemModel, 0, len(src.Spec.Items))
    for _, item := range src.Spec.Items {
        models = append(models, ItemModel{
            Type:  types.StringValue(item.Type),
            Value: types.StringValue(item.Value),
        })
    }
    list, diags := types.ListValueFrom(ctx, itemType, models)
    if diags.HasError() {
        return diags
    }
    data.Items = list
} else {
    data.Items = types.ListNull(itemType)
}

// AttrTypes
"items": types.ListType{ElemType: itemType},
```

---

## JSON Blob

### `map[string]any` / raw JSON → `jsontypes.NormalizedType{}`

Use when the field is an opaque JSON object (e.g., Grafana panel JSON, raw query).

```go
import "github.com/hashicorp/terraform-plugin-framework-jsontypes/jsontypes"

// Schema
"raw_json": schema.StringAttribute{
    Optional:    true,
    CustomType:  jsontypes.NormalizedType{},
    Description: "...",
},

// Model
RawJSON jsontypes.Normalized `tfsdk:"raw_json"`

// SpecParser
if !data.RawJSON.IsNull() && !data.RawJSON.IsUnknown() {
    dst.Spec.RawJSON = data.RawJSON.ValueString()
}

// SpecSaver
if src.Spec.RawJSON != "" {
    data.RawJSON = jsontypes.NewNormalizedValue(src.Spec.RawJSON)
} else {
    data.RawJSON = jsontypes.NewNormalizedNull()
}

// AttrTypes
"raw_json": jsontypes.NormalizedType{},
```

`jsontypes.NormalizedType` normalizes whitespace during plan comparison, avoiding spurious diffs from re-formatted JSON.

---

## Quick Reference Table

| Go SDK Type | TF Schema Attribute | Model Field Type | attr.Type for ObjectValue |
|-------------|--------------------|--------------------|--------------------------|
| `string` | `schema.StringAttribute{}` | `types.String` | `types.StringType` |
| `*string` | `schema.StringAttribute{Optional: true}` | `types.String` | `types.StringType` |
| `int64` | `schema.Int64Attribute{}` | `types.Int64` | `types.Int64Type` |
| `*int64` | `schema.Int64Attribute{Optional: true}` | `types.Int64` | `types.Int64Type` |
| `bool` | `schema.BoolAttribute{}` | `types.Bool` | `types.BoolType` |
| `*bool` | `schema.BoolAttribute{Optional: true}` | `types.Bool` | `types.BoolType` |
| `float64` | `schema.Float64Attribute{}` | `types.Float64` | `types.Float64Type` |
| `[]string` | `schema.ListAttribute{ElementType: types.StringType}` | `types.List` | `types.ListType{ElemType: types.StringType}` |
| `map[string]string` | `schema.MapAttribute{ElementType: types.StringType}` | `types.Map` | `types.MapType{ElemType: types.StringType}` |
| `[]struct` | `schema.ListAttribute{ElementType: objectType}` | `types.List` | `types.ListType{ElemType: objectType}` |
| struct (block) | `schema.SingleNestedBlock{Attributes: ...}` | nested struct | `types.ObjectType{AttrTypes: ...}` |
| struct (attr) | `schema.SingleNestedAttribute{Attributes: ...}` | nested struct | `types.ObjectType{AttrTypes: ...}` |
| raw JSON | `schema.StringAttribute{CustomType: jsontypes.NormalizedType{}}` | `jsontypes.Normalized` | `jsontypes.NormalizedType{}` |
