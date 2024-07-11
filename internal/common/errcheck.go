package common

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-openapi/runtime"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

const NotFoundError = "404"

// CheckReadError checks for common cases on resource read/delete paths:
// - If the resource no longer exists and 404s, it should be removed from state and return nil, to stop processing the read.
// - If there is an error, return the error.
// - Otherwise, do not return to continue processing the read.
func CheckReadError(resourceType string, d *schema.ResourceData, err error) (returnValue diag.Diagnostics, shouldReturn bool) {
	if err == nil {
		return nil, false
	}

	if !IsNotFoundError(err) {
		return diag.Errorf("error reading %s with ID`%s`: %v", resourceType, d.Id(), err), true
	}

	return WarnMissing(resourceType, d), true
}

func WarnMissing(resourceType string, d *schema.ResourceData) diag.Diagnostics {
	log.Printf("[WARN] removing %s with ID %q from state because it no longer exists in grafana", resourceType, d.Id())
	var diags diag.Diagnostics
	diags = append(diags, diag.Diagnostic{
		Severity: diag.Warning,
		Summary:  fmt.Sprintf("%s with ID %q is in Terraform state, but no longer exists in Grafana", resourceType, d.Id()),
		Detail:   fmt.Sprintf("%q will be recreated when you apply", d.Id()),
	})
	d.SetId("")
	return diags
}

func IsNotFoundError(err error) bool {
	if err, ok := err.(runtime.ClientResponseStatus); ok {
		return err.IsCode(404)
	}
	return strings.Contains(err.Error(), NotFoundError) // TODO: Remove when the old client is removed
}
