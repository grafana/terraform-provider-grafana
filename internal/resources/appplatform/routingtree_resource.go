package appplatform

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/apps/alerting/notifications/pkg/apis/alertingnotifications/v1beta1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	"github.com/grafana/terraform-provider-grafana/v4/internal/common"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// supportedRouteTreeDepth bounds how deeply nested routes can be expressed in
// the Terraform schema. Terraform does not allow infinitely recursive schemas,
// so we mirror the legacy grafana_notification_policy resource and cap the depth.
// This can be increased without breaking backwards compatibility.
const supportedRouteTreeDepth = 5

// routeMatcherType is the Terraform object type for a single route matcher.
var routeMatcherType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"type":  types.StringType,
		"label": types.StringType,
		"value": types.StringType,
	},
}

// routingTreeDefaultsType is the Terraform object type for the defaults block.
var routingTreeDefaultsType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"receiver":        types.StringType,
		"group_by":        types.ListType{ElemType: types.StringType},
		"group_wait":      types.StringType,
		"group_interval":  types.StringType,
		"repeat_interval": types.StringType,
	},
}

// routeObjectType returns the Terraform object type for a route at the given
// remaining depth. The nested "routes" attribute is only present while depth > 1,
// matching the schema produced by routeBlock.
func routeObjectType(depth uint) types.ObjectType {
	attrs := map[string]attr.Type{
		"receiver":              types.StringType,
		"continue":              types.BoolType,
		"group_by":              types.ListType{ElemType: types.StringType},
		"mute_time_intervals":   types.ListType{ElemType: types.StringType},
		"active_time_intervals": types.ListType{ElemType: types.StringType},
		"group_wait":            types.StringType,
		"group_interval":        types.StringType,
		"repeat_interval":       types.StringType,
		"matchers":              types.ListType{ElemType: routeMatcherType},
	}
	if depth > 1 {
		attrs["routes"] = types.ListType{ElemType: routeObjectType(depth - 1)}
	}
	return types.ObjectType{AttrTypes: attrs}
}

// routeBlock recursively builds the schema block for a route to a finite depth.
func routeBlock(depth uint) schema.ListNestedBlock {
	nested := schema.NestedBlockObject{
		Attributes: map[string]schema.Attribute{
			"receiver": schema.StringAttribute{
				Optional:    true,
				Description: "The contact point to route notifications that match this rule to. If not set, inherits from the nearest ancestor route that has it configured, ultimately falling back to `spec.defaults`.",
			},
			"continue": schema.BoolAttribute{
				Optional:    true,
				Description: "Whether to continue matching subsequent sibling routes if an alert matches this route. Defaults to false.",
			},
			"group_by": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group by all labels. If not set, inherits from the nearest ancestor route that has it configured, ultimately falling back to `spec.defaults`.",
			},
			"matchers": schema.ListAttribute{
				Optional:    true,
				ElementType: routeMatcherType,
				Validators:  []validator.List{routingTreeMatcherValidator{}},
				Description: "Matchers that an alert has to fulfill to match this route. When multiple matchers are supplied, an alert must match ALL of them.",
			},
			"mute_time_intervals": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of time interval names that mute this route during the specified times.",
			},
			"active_time_intervals": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "A list of time interval names that activate this route only during the specified times.",
			},
			"group_wait": schema.StringAttribute{
				Optional:    true,
				Description: "Time to wait to buffer alerts of the same group before sending a notification. If not set, inherits from the nearest ancestor route that has it configured, ultimately falling back to `spec.defaults`.",
			},
			"group_interval": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum time interval between two notifications for the same group. If not set, inherits from the nearest ancestor route that has it configured, ultimately falling back to `spec.defaults`.",
			},
			"repeat_interval": schema.StringAttribute{
				Optional:    true,
				Description: "Minimum time interval for re-sending a notification if an alert is still firing. If not set, inherits from the nearest ancestor route that has it configured, ultimately falling back to `spec.defaults`.",
			},
		},
	}

	if depth > 1 {
		nested.Blocks = map[string]schema.Block{
			"routes": routeBlock(depth - 1),
		}
	}

	return schema.ListNestedBlock{
		NestedObject: nested,
		Description:  "Child routes of this route.",
	}
}

type routingTreeSpecModel struct {
	Defaults          types.Object `tfsdk:"defaults"`
	Routes            types.List   `tfsdk:"routes"`
	DisableProvenance types.Bool   `tfsdk:"disable_provenance"`
}

// RoutingTree creates a new Grafana RoutingTree resource.
func RoutingTree() NamedResource {
	return NewNamedResource[*v1beta1.RoutingTree, *v1beta1.RoutingTreeList](
		common.CategoryAlerting,
		ResourceConfig[*v1beta1.RoutingTree]{
			Kind: v1beta1.RoutingTreeKind(),
			Schema: ResourceSpecSchema{
				Description: "Manages the Grafana notification routing tree (notification policy tree).",
				MarkdownDescription: `
Manages the Grafana notification routing tree using the Grafana Alerting Notifications API.

The routing tree determines which contact point an alert is routed to, based on its labels.

Requires Grafana 13.0+ with the ` + "`alertingMultiplePolicies`" + ` [feature toggle](https://grafana.com/docs/grafana/latest/setup-grafana/configure-grafana/feature-toggles/) enabled.

* [Official documentation](https://grafana.com/docs/grafana/latest/alerting/configure-notifications/create-notification-policy/)
`,
				SpecAttributes: map[string]schema.Attribute{
					"disable_provenance": schema.BoolAttribute{
						Optional:    true,
						Computed:    true,
						Default:     booldefault.StaticBool(false),
						Description: "Set to `true` to allow editing this resource from other sources (UI, API). Defaults to `false`, which locks the resource to Terraform management only.",
					},
				},
				SpecBlocks: map[string]schema.Block{
					"defaults": schema.SingleNestedBlock{
						Description: "The default values applied to alerts that do not match any specific route.",
						Attributes: map[string]schema.Attribute{
							"receiver": schema.StringAttribute{
								Required:    true,
								Description: "The default contact point to route all unmatched notifications to.",
							},
							"group_by": schema.ListAttribute{
								Optional:    true,
								ElementType: types.StringType,
								Description: "A list of alert labels to group alerts into notifications by. Use the special label `...` to group by all labels.",
							},
							"group_wait": schema.StringAttribute{
								Optional:    true,
								Description: "Time to wait to buffer alerts of the same group before sending a notification. Default is 30 seconds.",
							},
							"group_interval": schema.StringAttribute{
								Optional:    true,
								Description: "Minimum time interval between two notifications for the same group. Default is 5 minutes.",
							},
							"repeat_interval": schema.StringAttribute{
								Optional:    true,
								Description: "Minimum time interval for re-sending a notification if an alert is still firing. Default is 4 hours.",
							},
						},
					},
					"routes": routeBlock(supportedRouteTreeDepth),
				},
			},
			SpecParser: parseRoutingTreeSpec,
			SpecSaver:  saveRoutingTreeSpec,
		})
}

type routingTreeMatcherValidator struct{}

func (v routingTreeMatcherValidator) Description(_ context.Context) string {
	return "matcher must have a valid type (one of: =, !=, =~, !~) and a non-empty label"
}

func (v routingTreeMatcherValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v routingTreeMatcherValidator) ValidateList(ctx context.Context, req validator.ListRequest, resp *validator.ListResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	for i, el := range req.ConfigValue.Elements() {
		obj, ok := el.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()

		label, _ := attrs["label"].(types.String)
		if label.ValueString() == "" {
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid Matcher",
				fmt.Sprintf("Matcher at index %d: label must not be empty", i),
			)
		}

		matchType, _ := attrs["type"].(types.String)
		switch v1beta1.RoutingTreeMatcherType(matchType.ValueString()) {
		case v1beta1.RoutingTreeMatcherTypeEqual,
			v1beta1.RoutingTreeMatcherTypeNotEqual,
			v1beta1.RoutingTreeMatcherTypeEqualRegex,
			v1beta1.RoutingTreeMatcherTypeNotEqualRegex:
			// valid
		default:
			resp.Diagnostics.AddAttributeError(
				req.Path.AtListIndex(i),
				"Invalid Matcher",
				fmt.Sprintf("Matcher at index %d: invalid type %q; allowed types are: =, !=, =~, !~", i, matchType.ValueString()),
			)
		}
	}
}

func parseRoutingTreeSpec(ctx context.Context, src types.Object, dst *v1beta1.RoutingTree) diag.Diagnostics {
	var data routingTreeSpecModel
	if diags := src.As(ctx, &data, basetypes.ObjectAsOptions{
		UnhandledNullAsEmpty:    true,
		UnhandledUnknownAsEmpty: true,
	}); diags.HasError() {
		return diags
	}

	var diags diag.Diagnostics
	spec := v1beta1.RoutingTreeSpec{
		Routes: []v1beta1.RoutingTreeRoute{},
	}

	if !data.Defaults.IsNull() && !data.Defaults.IsUnknown() {
		defaults, d := parseRoutingTreeDefaults(data.Defaults)
		diags.Append(d...)
		spec.Defaults = defaults
	}

	if !data.Routes.IsNull() && !data.Routes.IsUnknown() {
		routes, d := parseRoutes(data.Routes, supportedRouteTreeDepth)
		diags.Append(d...)
		spec.Routes = routes
	}

	if diags.HasError() {
		return diags
	}

	if err := dst.SetSpec(spec); err != nil {
		return diag.Diagnostics{
			diag.NewErrorDiagnostic("failed to set spec", err.Error()),
		}
	}

	meta, err := utils.MetaAccessor(dst)
	if err == nil {
		if data.DisableProvenance.ValueBool() {
			if meta.GetAnnotations() == nil {
				meta.SetAnnotations(map[string]string{})
			}
			annotations := meta.GetAnnotations()
			annotations[provenanceAnnotationKey] = provenanceNone
			meta.SetAnnotations(annotations)
		} else {
			meta.SetAnnotation(provenanceAnnotationKey, provenanceAPI)
		}
	}

	return diags
}

func parseRoutingTreeDefaults(src types.Object) (v1beta1.RoutingTreeRouteDefaults, diag.Diagnostics) {
	var diags diag.Diagnostics
	defaults := v1beta1.RoutingTreeRouteDefaults{}
	attrs := src.Attributes()

	if v, ok := attrs["receiver"].(types.String); ok && !v.IsNull() && !v.IsUnknown() {
		defaults.Receiver = v.ValueString()
	}
	defaults.GroupBy = specStringList(attrs["group_by"])
	defaults.GroupWait = optStringPtr(attrs["group_wait"])
	defaults.GroupInterval = optStringPtr(attrs["group_interval"])
	defaults.RepeatInterval = optStringPtr(attrs["repeat_interval"])

	return defaults, diags
}

func parseRoutes(src types.List, depth uint) ([]v1beta1.RoutingTreeRoute, diag.Diagnostics) {
	var diags diag.Diagnostics
	elements := src.Elements()
	routes := make([]v1beta1.RoutingTreeRoute, 0, len(elements))
	for _, el := range elements {
		obj, ok := el.(types.Object)
		if !ok {
			continue
		}
		route, d := parseRoute(obj, depth)
		diags.Append(d...)
		routes = append(routes, route)
	}
	return routes, diags
}

func parseRoute(src types.Object, depth uint) (v1beta1.RoutingTreeRoute, diag.Diagnostics) {
	var diags diag.Diagnostics
	route := v1beta1.RoutingTreeRoute{}
	attrs := src.Attributes()

	if v, ok := attrs["continue"].(types.Bool); ok && !v.IsNull() && !v.IsUnknown() {
		route.Continue = v.ValueBool()
	}
	route.Receiver = optStringPtr(attrs["receiver"])
	route.GroupBy = specStringList(attrs["group_by"])
	route.MuteTimeIntervals = specStringList(attrs["mute_time_intervals"])
	route.ActiveTimeIntervals = specStringList(attrs["active_time_intervals"])
	route.GroupWait = optStringPtr(attrs["group_wait"])
	route.GroupInterval = optStringPtr(attrs["group_interval"])
	route.RepeatInterval = optStringPtr(attrs["repeat_interval"])

	if v, ok := attrs["matchers"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
		route.Matchers = parseRouteMatchers(v)
	}

	if depth > 1 {
		if v, ok := attrs["routes"].(types.List); ok && !v.IsNull() && !v.IsUnknown() {
			children, d := parseRoutes(v, depth-1)
			diags.Append(d...)
			route.Routes = children
		}
	}

	return route, diags
}

func parseRouteMatchers(src types.List) []v1beta1.RoutingTreeMatcher {
	elements := src.Elements()
	matchers := make([]v1beta1.RoutingTreeMatcher, 0, len(elements))
	for _, el := range elements {
		obj, ok := el.(types.Object)
		if !ok {
			continue
		}
		attrs := obj.Attributes()
		matcher := v1beta1.RoutingTreeMatcher{}
		if v, ok := attrs["type"].(types.String); ok {
			matcher.Type = v1beta1.RoutingTreeMatcherType(v.ValueString())
		}
		if v, ok := attrs["label"].(types.String); ok {
			matcher.Label = v.ValueString()
		}
		if v, ok := attrs["value"].(types.String); ok {
			matcher.Value = v.ValueString()
		}
		matchers = append(matchers, matcher)
	}
	return matchers
}

func saveRoutingTreeSpec(ctx context.Context, src *v1beta1.RoutingTree, dst *ResourceModel) diag.Diagnostics {
	var diags diag.Diagnostics

	defaults, d := saveRoutingTreeDefaults(src.Spec.Defaults)
	diags.Append(d...)

	routes, d := saveRoutes(src.Spec.Routes, supportedRouteTreeDepth)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	disableProvenance := types.BoolValue(false)
	if meta, err := utils.MetaAccessor(src); err == nil {
		disableProvenance = types.BoolValue(meta.GetAnnotation(provenanceAnnotationKey) == provenanceNone)
	}

	spec, d := types.ObjectValue(
		map[string]attr.Type{
			"defaults":           routingTreeDefaultsType,
			"routes":             types.ListType{ElemType: routeObjectType(supportedRouteTreeDepth)},
			"disable_provenance": types.BoolType,
		},
		map[string]attr.Value{
			"defaults":           defaults,
			"routes":             routes,
			"disable_provenance": disableProvenance,
		},
	)
	diags.Append(d...)
	if diags.HasError() {
		return diags
	}

	dst.Spec = spec
	return diags
}

func saveRoutingTreeDefaults(src v1beta1.RoutingTreeRouteDefaults) (types.Object, diag.Diagnostics) {
	return types.ObjectValue(
		routingTreeDefaultsType.AttrTypes,
		map[string]attr.Value{
			"receiver":        types.StringValue(src.Receiver),
			"group_by":        stringListToTf(src.GroupBy),
			"group_wait":      optStringToTf(src.GroupWait),
			"group_interval":  optStringToTf(src.GroupInterval),
			"repeat_interval": optStringToTf(src.RepeatInterval),
		},
	)
}

func saveRoutes(src []v1beta1.RoutingTreeRoute, depth uint) (types.List, diag.Diagnostics) {
	elemType := routeObjectType(depth)
	if len(src) == 0 {
		return types.ListNull(elemType), nil
	}

	var diags diag.Diagnostics
	elements := make([]attr.Value, 0, len(src))
	for _, route := range src {
		obj, d := saveRoute(route, depth)
		diags.Append(d...)
		elements = append(elements, obj)
	}
	if diags.HasError() {
		return types.ListNull(elemType), diags
	}

	list, d := types.ListValue(elemType, elements)
	diags.Append(d...)
	return list, diags
}

func saveRoute(src v1beta1.RoutingTreeRoute, depth uint) (types.Object, diag.Diagnostics) {
	objType := routeObjectType(depth)

	values := map[string]attr.Value{
		"receiver":              optStringToTf(src.Receiver),
		"continue":              types.BoolValue(src.Continue),
		"group_by":              stringListToTf(src.GroupBy),
		"mute_time_intervals":   stringListToTf(src.MuteTimeIntervals),
		"active_time_intervals": stringListToTf(src.ActiveTimeIntervals),
		"group_wait":            optStringToTf(src.GroupWait),
		"group_interval":        optStringToTf(src.GroupInterval),
		"repeat_interval":       optStringToTf(src.RepeatInterval),
		"matchers":              routeMatchersToTf(src.Matchers),
	}

	if depth > 1 {
		routes, diags := saveRoutes(src.Routes, depth-1)
		if diags.HasError() {
			return types.ObjectNull(objType.AttrTypes), diags
		}
		values["routes"] = routes
	}

	return types.ObjectValue(objType.AttrTypes, values)
}

func routeMatchersToTf(matchers []v1beta1.RoutingTreeMatcher) types.List {
	if len(matchers) == 0 {
		return types.ListNull(routeMatcherType)
	}
	elements := make([]attr.Value, 0, len(matchers))
	for _, m := range matchers {
		obj, _ := types.ObjectValue(routeMatcherType.AttrTypes, map[string]attr.Value{
			"type":  types.StringValue(string(m.Type)),
			"label": types.StringValue(m.Label),
			"value": types.StringValue(m.Value),
		})
		elements = append(elements, obj)
	}
	list, _ := types.ListValue(routeMatcherType, elements)
	return list
}

// optStringPtr converts an optional Terraform string attribute to a *string.
func optStringPtr(v attr.Value) *string {
	s, ok := v.(types.String)
	if !ok || s.IsNull() || s.IsUnknown() {
		return nil
	}
	return s.ValueStringPointer()
}

// optStringToTf converts a *string to a Terraform string value (null if nil).
func optStringToTf(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// specStringList converts a Terraform list attribute to a []string.
func specStringList(v attr.Value) []string {
	list, ok := v.(types.List)
	if !ok || list.IsNull() || list.IsUnknown() {
		return nil
	}
	elements := list.Elements()
	out := make([]string, 0, len(elements))
	for _, el := range elements {
		if s, ok := el.(types.String); ok {
			out = append(out, s.ValueString())
		}
	}
	return out
}

// stringListToTf converts a []string to a Terraform list value (null if empty).
func stringListToTf(values []string) types.List {
	if len(values) == 0 {
		return types.ListNull(types.StringType)
	}
	elements := make([]attr.Value, 0, len(values))
	for _, v := range values {
		elements = append(elements, types.StringValue(v))
	}
	list, _ := types.ListValue(types.StringType, elements)
	return list
}
