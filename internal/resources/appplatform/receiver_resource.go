package appplatform

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// receiverListSeparator joins and splits the values of lists the API stores as one string.
const receiverListSeparator = ";"

// Receiver returns the grafana_apps_notifications_receiver_v1beta1 resource.
func Receiver() NamedResource {
	def, spec, err := newReceiverDefinition()
	if err != nil {
		// The snapshot is embedded and covered by tests, so this is a build-time bug.
		panic(fmt.Sprintf("building receiver schema: %v", err))
	}

	return NewNamedResource[*v1beta1.Receiver, *v1beta1.ReceiverList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.Receiver]{
			Kind: v1beta1.ReceiverKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages a Grafana Alerting receiver (contact point) and its integrations. Requires Grafana 13.4.0 or later.",
				MarkdownDescription: `
Manages a Grafana Alerting receiver (contact point) and its integrations.

Requires Grafana 13.4.0 or later.

This resource replaces ` + "`grafana_contact_point`" + `, which uses the deprecated legacy provisioning API scheduled for removal in Grafana 14.

Each integration is identified by its ` + "`uid`" + `, which must be unique across all receivers in the organization. Credentials are
sent from the configuration on every update, so a credential that is removed from the configuration is cleared. The API does not
return credentials, so they are not recovered by import and must be present in the configuration.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/configure-notifications/manage-contact-points/)
`,
				SpecAttributes: spec.Attributes,
				SpecBlocks:     spec.Blocks,
			},
			PlanModifier: def.modifyPlan,
			// Shares the org's Alertmanager config with other resources, so concurrent creates can 409.
			RetryCreateOnConflict: true,
			SpecParser:            def.parseSpec,
			SpecSaver:             def.saveSpec,
		})
}

// ---- Terraform -> API ----

func (d *receiverDefinition) parseSpec(ctx context.Context, src types.Object, dst *v1beta1.Receiver) diag.Diagnostics {
	var diags diag.Diagnostics
	attrs := src.Attributes()

	spec := v1beta1.ReceiverSpec{Integrations: []v1beta1.ReceiverIntegration{}}
	if v, ok := attrs[receiverTitleAttr].(types.String); ok {
		spec.Title = v.ValueString()
	}
	disableProvenance := false
	if v, ok := attrs[receiverDisableProvenanceAttr].(types.Bool); ok {
		disableProvenance = v.ValueBool()
	}

	for _, it := range d.types {
		list, ok := attrs[it.TFName].(types.List)
		if !ok || list.IsNull() || list.IsUnknown() {
			continue
		}
		for i, elem := range list.Elements() {
			obj, ok := elem.(types.Object)
			if !ok {
				diags.AddError("Invalid integration", fmt.Sprintf("%s[%d] is not an object", it.TFName, i))
				continue
			}
			integration, d2 := parseReceiverIntegration(it, obj)
			diags.Append(d2...)
			spec.Integrations = append(spec.Integrations, integration)
		}
	}
	if diags.HasError() {
		return diags
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to set spec", err.Error())}
	}
	meta, err := utils.MetaAccessor(dst)
	if err != nil {
		return diag.Diagnostics{diag.NewErrorDiagnostic("failed to get metadata accessor", err.Error())}
	}
	if disableProvenance {
		meta.SetAnnotation(v1beta1.ProvenanceStatusAnnotationKey, provenanceNone)
	} else {
		meta.SetAnnotation(v1beta1.ProvenanceStatusAnnotationKey, provenanceAPI)
	}
	return diags
}

func parseReceiverIntegration(it integrationschemas.IntegrationType, obj types.Object) (v1beta1.ReceiverIntegration, diag.Diagnostics) {
	attrs := obj.Attributes()
	uid, _ := attrs[receiverIntegrationUID].(types.String)
	disableResolve, _ := attrs[receiverIntegrationDisableResolveMessage].(types.Bool)

	settings, diags := receiverSettingsFromAttrs(it.Fields, attrs)

	uidValue := uid.ValueString()
	disable := disableResolve.ValueBool()
	return v1beta1.ReceiverIntegration{
		Uid:                   &uidValue,
		Type:                  it.APIType,
		Version:               it.Version,
		DisableResolveMessage: &disable,
		Settings:              settings,
	}, diags
}

// receiverSettingsFromAttrs converts the configured fields of an integration (or subform) into API
// settings. Null and unknown values are omitted, so a secret that is not configured is not sent.
func receiverSettingsFromAttrs(fields []*integrationschemas.Field, attrs map[string]attr.Value) (map[string]any, diag.Diagnostics) {
	var diags diag.Diagnostics
	settings := map[string]any{}
	for _, f := range fields {
		v, ok := attrs[f.TFName]
		if !ok || v == nil || v.IsNull() || v.IsUnknown() {
			continue
		}
		value, include, d := receiverValueToAPI(f, v)
		diags.Append(d...)
		if include {
			settings[f.APIName] = value
		}
	}
	return settings, diags
}

func receiverValueToAPI(f *integrationschemas.Field, v attr.Value) (value any, include bool, diags diag.Diagnostics) {
	bad := func() (any, bool, diag.Diagnostics) {
		diags.AddError("Invalid integration setting", fmt.Sprintf("%s has unexpected type %T", f.TFName, v))
		return nil, false, diags
	}

	switch f.Kind {
	case integrationschemas.KindInt:
		n, ok := v.(types.Int64)
		if !ok {
			return bad()
		}
		return n.ValueInt64(), true, nil
	case integrationschemas.KindJoinedStringList:
		l, ok := v.(types.List)
		if !ok {
			return bad()
		}
		strs, d := stringsFromList(l)
		return strings.Join(strs, receiverListSeparator), true, d
	case integrationschemas.KindBool:
		b, ok := v.(types.Bool)
		if !ok {
			return bad()
		}
		return b.ValueBool(), true, nil
	case integrationschemas.KindStringList:
		l, ok := v.(types.List)
		if !ok {
			return bad()
		}
		strs, d := stringsFromList(l)
		return strs, true, d
	case integrationschemas.KindStringMap:
		m, ok := v.(types.Map)
		if !ok {
			return bad()
		}
		out := make(map[string]string, len(m.Elements()))
		for k, e := range m.Elements() {
			s, ok := e.(types.String)
			if !ok {
				return bad()
			}
			out[k] = s.ValueString()
		}
		return out, true, nil
	case integrationschemas.KindObject:
		o, ok := v.(types.Object)
		if !ok {
			return bad()
		}
		nested, d := receiverSettingsFromAttrs(f.Children, o.Attributes())
		// An empty block is the same as an absent one.
		return nested, len(nested) > 0, d
	case integrationschemas.KindObjectList:
		l, ok := v.(types.List)
		if !ok {
			return bad()
		}
		items := make([]any, 0, len(l.Elements()))
		for _, e := range l.Elements() {
			o, ok := e.(types.Object)
			if !ok {
				return bad()
			}
			nested, d := receiverSettingsFromAttrs(f.Children, o.Attributes())
			diags.Append(d...)
			items = append(items, nested)
		}
		return items, true, diags
	default: // integrationschemas.KindString
		s, ok := v.(types.String)
		if !ok {
			return bad()
		}
		return s.ValueString(), true, nil
	}
}

func stringsFromList(l types.List) ([]string, diag.Diagnostics) {
	var diags diag.Diagnostics
	out := make([]string, 0, len(l.Elements()))
	for _, e := range l.Elements() {
		s, ok := e.(types.String)
		if !ok {
			diags.AddError("Invalid list element", fmt.Sprintf("expected a string, got %T", e))
			continue
		}
		out = append(out, s.ValueString())
	}
	return out, diags
}

// ---- API -> Terraform (import) ----

// saveSpec converts the API receiver into the Terraform spec. The shared engine only calls it on
// import. Credentials are not returned by the API, so they are left null.
func (d *receiverDefinition) saveSpec(ctx context.Context, src *v1beta1.Receiver, dst *ResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	values := map[string]attr.Value{
		receiverTitleAttr:             types.StringValue(src.Spec.Title),
		receiverDisableProvenanceAttr: types.BoolValue(receiverProvenanceDisabled(src)),
	}

	byType := map[string][]attr.Value{}
	for _, in := range src.Spec.Integrations {
		it, ok := d.byAPI[in.Type]
		if !ok {
			diags.AddError("Unsupported integration",
				fmt.Sprintf("Integration %q has type %q, which this resource does not support (it may be a deprecated type or one that cannot be created).", derefString(in.Uid), in.Type))
			continue
		}
		if in.Version != it.Version {
			diags.AddError("Unsupported integration version",
				fmt.Sprintf("Integration %q of type %q uses settings version %q; this resource only supports %q (block %s).", derefString(in.Uid), in.Type, in.Version, it.Version, it.TFName))
			continue
		}
		elemType := d.elemType(it)
		obj, d2 := receiverIntegrationToObject(it, elemType, in)
		diags.Append(d2...)
		if d2.HasError() {
			continue
		}
		byType[it.TFName] = append(byType[it.TFName], obj)
	}
	if diags.HasError() {
		return diags
	}

	for _, it := range d.types {
		listType, ok := d.specType.AttrTypes[it.TFName].(types.ListType)
		if !ok {
			diags.AddError("Invalid receiver schema", fmt.Sprintf("%s is not a list", it.TFName))
			return diags
		}
		list, d2 := types.ListValue(listType.ElemType, byType[it.TFName])
		diags.Append(d2...)
		values[it.TFName] = list
	}

	spec, d2 := types.ObjectValue(d.specType.AttrTypes, values)
	diags.Append(d2...)
	if diags.HasError() {
		return diags
	}
	dst.Spec = spec
	return diags
}

func receiverIntegrationToObject(it integrationschemas.IntegrationType, typ types.ObjectType, in v1beta1.ReceiverIntegration) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	disable := false
	if in.DisableResolveMessage != nil {
		disable = *in.DisableResolveMessage
	}
	values := map[string]attr.Value{
		receiverIntegrationUID:                   types.StringValue(derefString(in.Uid)),
		receiverIntegrationDisableResolveMessage: types.BoolValue(disable),
	}

	remaining := make(map[string]any, len(in.Settings))
	for k, v := range in.Settings {
		remaining[k] = v
	}
	d2 := receiverFieldsFromAPI(it.Fields, typ, remaining, values, in.Type)
	diags.Append(d2...)
	if len(remaining) > 0 && !diags.HasError() {
		diags.AddError("Unsupported integration settings",
			fmt.Sprintf("Integration %q of type %q has settings this resource does not support: %s.", derefString(in.Uid), in.Type, sortedKeys(remaining)))
	}
	if diags.HasError() {
		return nil, diags
	}
	obj, d3 := types.ObjectValue(typ.AttrTypes, values)
	diags.Append(d3...)
	return obj, diags
}

// receiverFieldsFromAPI fills values for every attribute of typ that corresponds to a field,
// consuming the matching keys from remaining so unsupported settings can be reported.
func receiverFieldsFromAPI(fields []*integrationschemas.Field, typ types.ObjectType, remaining map[string]any, values map[string]attr.Value, integrationType string) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, f := range fields {
		key, raw, found := takeSetting(remaining, f.APIName)
		fieldType := typ.AttrTypes[f.TFName]
		if fieldType == nil {
			diags.AddError("Invalid receiver schema", fmt.Sprintf("%s has no attribute %s", integrationType, f.TFName))
			continue
		}
		if !found || raw == nil {
			values[f.TFName] = nullOf(fieldType)
			continue
		}
		delete(remaining, key)

		v, d := receiverValueFromAPI(f, fieldType, raw)
		diags.Append(d...)
		values[f.TFName] = v
	}
	return diags
}

// takeSetting finds a setting by exact name, then case-insensitively, since the API treats secret
// names case-insensitively.
func takeSetting(settings map[string]any, name string) (string, any, bool) {
	if v, ok := settings[name]; ok {
		return name, v, true
	}
	for k, v := range settings {
		if strings.EqualFold(k, name) {
			return k, v, true
		}
	}
	return "", nil, false
}

func receiverValueFromAPI(f *integrationschemas.Field, typ attr.Type, raw any) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	fail := func() (attr.Value, diag.Diagnostics) {
		diags.AddError("Unsupported integration setting", fmt.Sprintf("Setting %q has an unexpected value of type %T.", f.APIName, raw))
		return nullOf(typ), diags
	}

	switch f.Kind {
	case integrationschemas.KindInt:
		switch n := raw.(type) {
		case float64:
			return types.Int64Value(int64(n)), nil
		case int64:
			return types.Int64Value(n), nil
		case int:
			return types.Int64Value(int64(n)), nil
		case string:
			// Grafana accepts numbers as strings, so a value saved that way (for example in the UI) is a string.
			if parsed, err := strconv.ParseInt(strings.TrimSpace(n), 10, 64); err == nil {
				return types.Int64Value(parsed), nil
			}
		}
		return fail()
	case integrationschemas.KindJoinedStringList:
		s, ok := raw.(string)
		if !ok {
			return fail()
		}
		var elems []attr.Value
		for _, part := range strings.FieldsFunc(s, func(r rune) bool { return r == ';' || r == ',' || r == '\n' }) {
			elems = append(elems, types.StringValue(strings.TrimSpace(part)))
		}
		return types.ListValue(types.StringType, elems)
	case integrationschemas.KindBool:
		b, ok := raw.(bool)
		if !ok {
			return fail()
		}
		return types.BoolValue(b), nil
	case integrationschemas.KindStringList:
		items, ok := raw.([]any)
		if !ok {
			return fail()
		}
		elems := make([]attr.Value, 0, len(items))
		for _, item := range items {
			s, ok := item.(string)
			if !ok {
				return fail()
			}
			elems = append(elems, types.StringValue(s))
		}
		return types.ListValue(types.StringType, elems)
	case integrationschemas.KindStringMap:
		m, ok := raw.(map[string]any)
		if !ok {
			return fail()
		}
		elems := make(map[string]attr.Value, len(m))
		for k, item := range m {
			s, ok := item.(string)
			if !ok {
				return fail()
			}
			elems[k] = types.StringValue(s)
		}
		return types.MapValue(types.StringType, elems)
	case integrationschemas.KindObject:
		m, ok := raw.(map[string]any)
		if !ok {
			return fail()
		}
		return receiverObjectFromAPI(f, typ.(types.ObjectType), m)
	case integrationschemas.KindObjectList:
		items, ok := raw.([]any)
		if !ok {
			return fail()
		}
		elemType := typ.(types.ListType).ElemType.(types.ObjectType)
		elems := make([]attr.Value, 0, len(items))
		for _, item := range items {
			m, ok := item.(map[string]any)
			if !ok {
				return fail()
			}
			obj, d := receiverObjectFromAPI(f, elemType, m)
			diags.Append(d...)
			elems = append(elems, obj)
		}
		list, d := types.ListValue(elemType, elems)
		diags.Append(d...)
		return list, diags
	default: // integrationschemas.KindString
		switch s := raw.(type) {
		case string:
			return types.StringValue(s), nil
		case float64: // text inputs whose API value is a number
			return types.StringValue(strconv.FormatFloat(s, 'f', -1, 64)), nil
		}
		return fail()
	}
}

func receiverObjectFromAPI(f *integrationschemas.Field, typ types.ObjectType, m map[string]any) (attr.Value, diag.Diagnostics) {
	var diags diag.Diagnostics
	remaining := make(map[string]any, len(m))
	for k, v := range m {
		remaining[k] = v
	}
	values := map[string]attr.Value{}
	diags.Append(receiverFieldsFromAPI(f.Children, typ, remaining, values, f.APIName)...)
	if len(remaining) > 0 && !diags.HasError() {
		diags.AddError("Unsupported integration settings", fmt.Sprintf("Setting %q has fields this resource does not support: %s.", f.APIName, sortedKeys(remaining)))
	}
	if diags.HasError() {
		return types.ObjectNull(typ.AttrTypes), diags
	}
	obj, d := types.ObjectValue(typ.AttrTypes, values)
	diags.Append(d...)
	return obj, diags
}

// sortedKeys returns the keys of m, sorted and comma separated, for error messages.
func sortedKeys(m map[string]any) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func nullOf(t attr.Type) attr.Value {
	switch tt := t.(type) {
	case types.ObjectType:
		return types.ObjectNull(tt.AttrTypes)
	case types.ListType:
		return types.ListNull(tt.ElemType)
	case types.MapType:
		return types.MapNull(tt.ElemType)
	case basetypes.StringType:
		return types.StringNull()
	case basetypes.BoolType:
		return types.BoolNull()
	case basetypes.Int64Type:
		return types.Int64Null()
	}
	return types.StringNull()
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// receiverProvenanceDisabled reports whether the receiver has no provenance, which Grafana reports
// as an empty value (the annotation is omitted).
func receiverProvenanceDisabled(r *v1beta1.Receiver) bool {
	switch r.GetProvenanceStatus() {
	case "", v1beta1.ProvenanceStatusNone:
		return true
	}
	return false
}

// ---- plan-time validation ----

// modifyPlan rejects duplicate integration UIDs within the configuration. Grafana rejects them too,
// so this only exists to report the problem at plan time, with a pointer to the offending block,
// instead of as an error during apply.
func (d *receiverDefinition) modifyPlan(ctx context.Context, req resource.ModifyPlanRequest, resp *resource.ModifyPlanResponse) {
	if req.Plan.Raw.IsNull() {
		return // destroy
	}
	var plan ResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	seen := map[string]string{}
	for _, it := range d.types {
		for i, uid := range d.uidsOf(plan.Spec, it) {
			if uid == "" {
				continue // unknown at plan time
			}
			if other, ok := seen[uid]; ok {
				resp.Diagnostics.AddAttributeError(
					path.Root("spec").AtName(it.TFName).AtListIndex(i).AtName(receiverIntegrationUID),
					"Duplicate integration uid",
					fmt.Sprintf("The uid %q is already used by %s in this receiver. Integration uids must be unique.", uid, other),
				)
			}
			seen[uid] = it.TFName
		}
	}
}

func (d *receiverDefinition) uidsOf(spec types.Object, it integrationschemas.IntegrationType) []string {
	if spec.IsNull() || spec.IsUnknown() {
		return nil
	}
	list, ok := spec.Attributes()[it.TFName].(types.List)
	if !ok || list.IsNull() || list.IsUnknown() {
		return nil
	}
	uids := make([]string, 0, len(list.Elements()))
	for _, e := range list.Elements() {
		uid := ""
		if o, ok := e.(types.Object); ok {
			if s, ok := o.Attributes()[receiverIntegrationUID].(types.String); ok && !s.IsUnknown() && !s.IsNull() {
				uid = s.ValueString()
			}
		}
		uids = append(uids, uid)
	}
	return uids
}
