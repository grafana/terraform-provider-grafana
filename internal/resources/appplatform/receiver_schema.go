package appplatform

import (
	"fmt"
	"strconv"

	"github.com/grafana/terraform-provider-grafana/v4/internal/resources/appplatform/integrationschemas"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// The schema of the receiver resource, built from integrationschemas.IntegrationType: the spec, and
// one repeated block per integration type.

// Attributes of the receiver spec besides the integration blocks.
const (
	receiverTitleAttr             = "title"
	receiverDisableProvenanceAttr = "disable_provenance"
)

// receiverDefinition holds the integration types the receiver resource exposes, derived from the
// integration schema snapshot, and the Terraform type of the spec.
type receiverDefinition struct {
	types    []integrationschemas.IntegrationType
	byAPI    map[string]integrationschemas.IntegrationType
	specType types.ObjectType
}

func newReceiverDefinition() (*receiverDefinition, schema.SingleNestedBlock, error) {
	integrationTypes, err := integrationschemas.Load()
	if err != nil {
		return nil, schema.SingleNestedBlock{}, err
	}
	if err := validateReceiverIntegrationNames(integrationTypes); err != nil {
		return nil, schema.SingleNestedBlock{}, err
	}

	d := &receiverDefinition{types: integrationTypes, byAPI: make(map[string]integrationschemas.IntegrationType, len(integrationTypes))}
	for _, t := range integrationTypes {
		d.byAPI[t.APIType] = t
	}

	spec := schema.SingleNestedBlock{
		Attributes: map[string]schema.Attribute{
			receiverTitleAttr: schema.StringAttribute{
				Required:    true,
				Description: "The name of the receiver. It is the name routes and alert rules use to reference the receiver, and can be changed without recreating the receiver.",
			},
			receiverDisableProvenanceAttr: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Set to `true` to allow editing this resource from other sources (UI, API). Defaults to `false`, which locks the resource to Terraform management only.",
			},
		},
		Blocks: receiverIntegrationBlocks(integrationTypes),
	}
	specType, ok := spec.Type().(types.ObjectType)
	if !ok {
		return nil, spec, fmt.Errorf("unexpected receiver spec type %T", spec.Type())
	}
	d.specType = specType
	return d, spec, nil
}

// Attributes every integration block has besides the generated settings fields.
const (
	receiverIntegrationUID                   = "uid"
	receiverIntegrationVersion               = "version"
	receiverIntegrationDisableResolveMessage = "disable_resolve_message"
)

// receiverReservedFieldNames are integration-level names the settings fields must not collide with.
var receiverReservedFieldNames = map[string]struct{}{
	receiverIntegrationUID:                   {},
	receiverIntegrationVersion:               {},
	receiverIntegrationDisableResolveMessage: {},
}

// receiverSpecReservedNames are spec-level names the integration blocks must not collide with.
var receiverSpecReservedNames = map[string]struct{}{
	receiverTitleAttr:             {},
	receiverDisableProvenanceAttr: {},
}

// validateReceiverIntegrationNames checks that the generated names do not collide with the
// attributes the resource defines itself.
func validateReceiverIntegrationNames(integrationTypes []integrationschemas.IntegrationType) error {
	for _, it := range integrationTypes {
		if _, ok := receiverSpecReservedNames[it.TFName]; ok {
			return fmt.Errorf("integration type %q maps to reserved receiver spec name %q", it.APIType, it.TFName)
		}
		for _, f := range it.Fields {
			if _, ok := receiverReservedFieldNames[f.TFName]; ok {
				return fmt.Errorf("field %s/%s maps to reserved name %q", it.APIType, f.APIName, f.TFName)
			}
		}
	}
	return nil
}

// receiverIntegrationBlocks builds the spec blocks, one repeated block per exposed integration type.
func receiverIntegrationBlocks(integrationTypes []integrationschemas.IntegrationType) map[string]schema.Block {
	blocks := make(map[string]schema.Block, len(integrationTypes))
	for _, t := range integrationTypes {
		attrs := map[string]schema.Attribute{
			receiverIntegrationUID: schema.StringAttribute{
				Required:    true,
				Description: "The UID of the integration. It identifies the integration within the receiver and is unique across all receivers in the organization.",
			},
			receiverIntegrationVersion: schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(t.Version),
				Description: fmt.Sprintf("The version of the integration settings. Defaults to `%s`, which is currently the only supported version.", t.Version),
				Validators:  []validator.String{stringvalidator.OneOf(t.Version)},
			},
			receiverIntegrationDisableResolveMessage: schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Whether to disable sending resolve messages. Defaults to `false`.",
			},
		}
		nestedBlocks := map[string]schema.Block{}
		addReceiverFields(attrs, nestedBlocks, t.Fields, false)

		blocks[t.TFName] = schema.ListNestedBlock{
			Description: fmt.Sprintf("A %s integration.", t.Name),
			NestedObject: schema.NestedBlockObject{
				Attributes: attrs,
				Blocks:     nestedBlocks,
			},
		}
	}
	return blocks
}

// addReceiverFields adds the attributes and blocks of fields. insideObject is whether the fields
// are in a single nested block (an optional block, like webhook payload).
func addReceiverFields(attrs map[string]schema.Attribute, blocks map[string]schema.Block, fields []*integrationschemas.Field, insideObject bool) {
	for _, f := range fields {
		switch f.Kind {
		case integrationschemas.KindObject:
			nestedAttrs := map[string]schema.Attribute{}
			nestedBlocks := map[string]schema.Block{}
			addReceiverFields(nestedAttrs, nestedBlocks, f.Children, true)
			blocks[f.TFName] = schema.SingleNestedBlock{
				Description: f.Description,
				Attributes:  nestedAttrs,
				Blocks:      nestedBlocks,
			}
		case integrationschemas.KindObjectList:
			nestedAttrs := map[string]schema.Attribute{}
			nestedBlocks := map[string]schema.Block{}
			addReceiverFields(nestedAttrs, nestedBlocks, f.Children, insideObject)
			blocks[f.TFName] = schema.ListNestedBlock{
				Description: f.Description,
				NestedObject: schema.NestedBlockObject{
					Attributes: nestedAttrs,
					Blocks:     nestedBlocks,
				},
			}
		default:
			attrs[f.TFName] = receiverAttribute(f, insideObject)
		}
	}
}

// receiverAttribute returns the Terraform attribute of a field that is not a nested block.
//
// The framework validates the required attributes of a single nested block even when the block is
// not configured (https://github.com/hashicorp/terraform-plugin-framework/issues/740), so inside
// one (insideObject) required fields are optional in the schema, and Grafana reports a missing
// value when the block is used. A SingleNestedAttribute, which needs protocol version 6, would let
// these fields be required again.
func receiverAttribute(f *integrationschemas.Field, insideObject bool) schema.Attribute {
	desc := f.Description
	required := f.Required && !insideObject
	if f.Required && insideObject {
		desc += " Required when the block is set."
	}
	optional, sensitive := !required, f.Sensitive
	switch f.Kind {
	case integrationschemas.KindInt:
		a := schema.Int64Attribute{Optional: optional, Required: required, Sensitive: sensitive, Description: desc}
		if len(f.AllowedValues) > 0 {
			allowed := make([]int64, 0, len(f.AllowedValues))
			for _, v := range f.AllowedValues {
				if n, err := strconv.ParseInt(v, 10, 64); err == nil {
					allowed = append(allowed, n)
				}
			}
			a.Validators = []validator.Int64{int64validator.OneOf(allowed...)}
		}
		return a
	case integrationschemas.KindBool:
		return schema.BoolAttribute{Optional: optional, Required: required, Sensitive: sensitive, Description: desc}
	case integrationschemas.KindStringList, integrationschemas.KindJoinedStringList:
		return schema.ListAttribute{ElementType: types.StringType, Optional: optional, Required: required, Sensitive: sensitive, Description: desc}
	case integrationschemas.KindStringMap:
		return schema.MapAttribute{ElementType: types.StringType, Optional: optional, Required: required, Sensitive: sensitive, Description: desc}
	default:
		a := schema.StringAttribute{Optional: optional, Required: required, Sensitive: sensitive, Description: desc}
		if len(f.AllowedValues) > 0 {
			a.Validators = []validator.String{stringvalidator.OneOf(f.AllowedValues...)}
		}
		return a
	}
}
