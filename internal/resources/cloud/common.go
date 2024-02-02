package cloud

import (
	"github.com/grafana/grafana-com-public-clients/go/gcom"
	"github.com/hashicorp/go-uuid"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
)

func ClientRequestID() string {
	uuid, err := uuid.GenerateUUID()
	if err != nil {
		return ""
	}
	return "tf-" + uuid
}

func apiError(err error) diag.Diagnostics {
	if err == nil {
		return nil
	}
	detail := err.Error()
	if err, ok := err.(*gcom.GenericOpenAPIError); ok {
		detail += "\n" + string(err.Body())
	}
	return diag.Diagnostics{
		diag.Diagnostic{
			Severity: diag.Error,
			Summary:  err.Error(),
			Detail:   detail,
		},
	}
}
