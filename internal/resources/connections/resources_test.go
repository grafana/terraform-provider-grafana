package connections_test

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/assert"

	"github.com/grafana/terraform-provider-grafana/v3/internal/resources/connections"
)

func Test_httpsURLValidator(t *testing.T) {
	t.Parallel()
	testCases := map[string]struct {
		providedURL   types.String
		expectedDiags diag.Diagnostics
	}{
		"valid url with https": {
			providedURL:   types.StringValue("https://dev.my-metrics-endpoint-url.com:9000/metrics"),
			expectedDiags: nil,
		},
		"null is considered valid": {
			providedURL:   types.StringNull(),
			expectedDiags: diag.Diagnostics(nil),
		},
		"unknown is considered valid": {
			providedURL:   types.StringUnknown(),
			expectedDiags: diag.Diagnostics(nil),
		},
		"invalid empty string": {
			providedURL: types.StringValue(""),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A valid URL is required.\n\nGiven Value: \"\"\n",
			)},
		},
		"invalid not a url": {
			providedURL: types.StringValue("this is not a url"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A URL was provided, protocol must be HTTPS.\n\nGiven Value: \"this is not a url\"\n",
			)},
		},
		"invalid leading space url": {
			providedURL: types.StringValue(" https://leading.space"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A string value was provided that is not a valid URL.\n\nGiven Value:  https://leading.space\nError: parse \" https://leading.space\": first path segment in URL cannot contain colon",
			)},
		},
		"invalid url without https": {
			providedURL: types.StringValue("www.google.com"),
			expectedDiags: diag.Diagnostics{diag.NewAttributeErrorDiagnostic(
				path.Root("test"),
				"value must be valid URL with HTTPS",
				"A URL was provided, protocol must be HTTPS.\n\nGiven Value: \"www.google.com\"\n",
			)},
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			res := validator.StringResponse{}
			connections.HTTPSURLValidator{}.ValidateString(
				context.Background(),
				validator.StringRequest{
					ConfigValue: tc.providedURL,
					Path:        path.Root("test"),
				},
				&res)

			assert.Equal(t, tc.expectedDiags, res.Diagnostics)
		})
	}
}
