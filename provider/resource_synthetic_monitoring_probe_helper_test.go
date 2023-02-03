package provider

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func TestImportProbeStateWithToken(t *testing.T) {
	CheckCloudInstanceTestsEnabled(t)

	testcases := map[string]struct {
		input             string
		expectError       bool
		expectedID        string
		expectedAuthToken string
	}{
		"valid id, no auth_token": {
			input:             "1",
			expectError:       false,
			expectedID:        "1",
			expectedAuthToken: "",
		},
		"valid id, valid auth_token": {
			input:             "1:aGVsbG8=",
			expectError:       false,
			expectedID:        "1",
			expectedAuthToken: "aGVsbG8=",
		},
		"valid id, invalid auth_token": {
			input:       "1:xxx",
			expectError: true,
		},
		"invalid id, valid auth_token": {
			input:       ":aGVsbG8=",
			expectError: true,
		},
	}

	for name, tc := range testcases {
		t.Run(name, func(t *testing.T) {
			d := schema.TestResourceDataRaw(t, ResourceSyntheticMonitoringProbe().Schema, nil)
			d.SetId(tc.input)

			res, err := importProbeStateWithToken(context.Background(), d, nil)
			switch {
			case tc.expectError && err == nil:
				t.Fatalf("calling importProbeStateWithToken with id %q, expecting error, got nil", tc.input)

			case !tc.expectError && err != nil:
				t.Fatalf("calling importProbeStateWithToken with id %q, expecting no error, got %s", tc.input, err)

			case !tc.expectError:
				if len(res) != 1 {
					t.Fatalf("expecting 1 ResourceData, got %d", len(res))
				}

				if tc.expectedID != res[0].Id() {
					t.Fatalf("expecting id %q, got %q", tc.expectedID, res[0].Id())
				}

				if tc.expectedAuthToken != "" {
					output, ok := res[0].GetOk("auth_token")
					if !ok {
						t.Fatalf("expecting auth_token to be set")
					} else if str, ok := output.(string); !ok || str != tc.expectedAuthToken {
						t.Fatalf("expecting auth_token to match string %q, got %#v", tc.expectedAuthToken, output)
					}
				}
			}
		})
	}
}
