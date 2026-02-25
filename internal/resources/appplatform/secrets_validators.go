package appplatform

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	k8svalidation "k8s.io/apimachinery/pkg/util/validation"
)

type DNS1123SubdomainValidator struct{}

func (v DNS1123SubdomainValidator) Description(_ context.Context) string {
	return "value must be DNS-1123 compatible"
}

func (v DNS1123SubdomainValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v DNS1123SubdomainValidator) ValidateString(
	ctx context.Context, req validator.StringRequest, resp *validator.StringResponse,
) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}

	value := req.ConfigValue.ValueString()
	for _, msg := range k8svalidation.IsDNS1123Subdomain(value) {
		resp.Diagnostics.AddAttributeError(
			req.Path,
			v.Description(ctx),
			fmt.Sprintf("%s: %s", req.Path, msg),
		)
	}
}
