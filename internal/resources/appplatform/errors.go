package appplatform

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceAction is the action that is being performed on the resource.
type ResourceAction string

const (
	ResourceActionCreate ResourceAction = "create"
	ResourceActionUpdate ResourceAction = "update"
	ResourceActionDelete ResourceAction = "delete"
	ResourceActionRead   ResourceAction = "read"
)

// ErrorToDiagnostics formats an error from the Kubernetes API into Terraform diagnostics.
func ErrorToDiagnostics(action ResourceAction, resourceName, resourceType string, err error) diag.Diagnostics {
	res := make(diag.Diagnostics, 0)

	var serr apierrors.APIStatus
	if status, ok := err.(apierrors.APIStatus); ok || errors.As(err, &status) {
		serr = status
	} else {
		res.AddError(
			fmt.Sprintf("failed to %s resource %q: unknown error", action, resourceName),
			err.Error(),
		)

		return res
	}

	status := serr.Status()
	switch status.Reason {
	case metav1.StatusReasonInvalid:
		if len(status.Details.Causes) > 0 {
			errs := FieldErrorsFromCauses(resourceType, status.Details.Causes)

			for _, err := range errs {
				var msg strings.Builder
				for i, m := range err.Messages {
					msg.WriteString("* ")
					msg.WriteString(m)

					// Don't add a newline after the last message.
					if i < len(err.Messages)-1 {
						msg.WriteString("\n")
					}
				}

				res.AddAttributeError(err.Path, fmt.Sprintf("invalid value for field \"%s\":", err.Field), msg.String())
			}
		} else {
			res.AddError(
				fmt.Sprintf(
					"failed to %s resource %q: HTTP %d - %s", action, resourceName, status.Code, metav1.StatusReasonInvalid,
				),
				status.Message,
			)
		}
	case metav1.StatusReasonUnknown:
		res.AddError(
			fmt.Sprintf("failed to %s resource %q: HTTP %d - Unknown", action, resourceName, status.Code),
			status.Message,
		)
	default:
		res.AddError(
			fmt.Sprintf("failed to %s resource %q: HTTP %d - %s", action, resourceName, status.Code, status.Reason),
			status.Message,
		)
	}

	return res
}

// FieldError is a field-level error returned by the Kubernetes API,
// with the path and messages adapted to Terraform diagnostics.
type FieldError struct {
	Field    string
	Path     path.Path
	Messages []string
}

// FieldErrors is a map of field names to FieldError instances.
type FieldErrors map[string]FieldError

// FieldErrorsFromCauses converts a list of field errors returned by the Kubernetes API
// into a map of Terraform-acceptable field-level errors.
func FieldErrorsFromCauses(resourceType string, causes []metav1.StatusCause) FieldErrors {
	res := make(FieldErrors, len(causes))

	for _, cause := range causes {
		v, ok := res[cause.Field]

		if ok {
			v.Messages = append(v.Messages, cause.Message)
		} else {
			v = FieldError{
				Field:    cause.Field,
				Path:     ParseFieldPath(resourceType, cause.Field),
				Messages: []string{cause.Message},
			}
		}

		res[cause.Field] = v
	}

	return res
}

// ParseFieldPath converts a Kubernetes field path into a Terraform path.
func ParseFieldPath(resourceType, fieldPath string) path.Path {
	parts := strings.Split(fieldPath, ".")

	if len(parts) == 0 {
		return path.Root("")
	}

	var res path.Path
	switch parts[0] {
	case "metadata":
		res = path.Root("metadata")
	case "spec":
		res = path.Root("spec")
	default:
		// unknown, not supported
		return path.Root("")
	}

	if
	// TODO (@radiohead): this is a hack because the dashboard spec relies on the json field.
	// We need to find a better way to handle this.
	resourceType == "grafana_apps_dashboard_dashboard_v1alpha1" &&
		res.Equal(path.Root("spec")) &&
		parts[1] != "title" &&
		parts[1] != "tags" {
		res = res.AtName("json")

		for _, part := range parts[1:] {
			if idx, err := strconv.Atoi(part); err == nil {
				res = res.AtListIndex(idx)
			} else {
				res = res.AtMapKey(part)
			}
		}
	} else {
		for _, part := range parts[1:] {
			if idx, err := strconv.Atoi(part); err == nil {
				res = res.AtListIndex(idx)
			} else {
				res = res.AtName(part)
			}
		}
	}

	return res
}
