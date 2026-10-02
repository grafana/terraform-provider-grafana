package assistant

import (
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func scopeAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "Whether the resource is visible to the whole tenant (`tenant`) or only the creating user (`user`).",
		Required:    true,
		Validators: []validator.String{
			stringvalidator.OneOf("user", "tenant"),
		},
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.RequiresReplace(),
		},
	}
}

func enabledAttribute() schema.BoolAttribute {
	return schema.BoolAttribute{
		Description: "Whether the resource is enabled.",
		Optional:    true,
		Computed:    true,
		Default:     booldefault.StaticBool(true),
	}
}

func applicationsAttribute() schema.ListAttribute {
	return schema.ListAttribute{
		Description: "Applications where this resource applies. Valid values: `assistant`, `loop`, `infrastructure_memory` (rules only), `all`. Defaults to all applications when unset.",
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
	}
}

func applicationsAttributeMCP() schema.ListAttribute {
	return schema.ListAttribute{
		Description: "Applications where this resource applies. Valid values: `assistant`, `loop`, `all`. Defaults to all applications when unset.",
		Optional:    true,
		Computed:    true,
		ElementType: types.StringType,
	}
}

func idAttribute() schema.StringAttribute {
	return schema.StringAttribute{
		Description: "The UUID of the resource.",
		Computed:    true,
		PlanModifiers: []planmodifier.String{
			stringplanmodifier.UseStateForUnknown(),
		},
	}
}
