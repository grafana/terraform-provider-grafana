package incident_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/grafana/terraform-provider-grafana/v4/internal/testutils"
)

// noIncidentAPI points the provider at a server that fails the test on any
// request, for tests that only plan.
func noIncidentAPI(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected API request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	t.Cleanup(server.Close)
	t.Setenv("GRAFANA_URL", server.URL)
	t.Setenv("GRAFANA_AUTH", "admin:admin")
}

func customFieldTypeConfig(fieldType, selectOption string) string {
	return fmt.Sprintf(`
resource "grafana_incident_custom_field" "test" {
  name = "tf-unit-test-field"
  slug = "tf_unit_test_field"
  type = %q
%s
}
`, fieldType, selectOption)
}

// TestUnitIncidentCustomField_BoolAndDateTypes pins that bool and date fields
// plan like the other non-select types: accepted without select options, and
// refused with them before the API is called.
func TestUnitIncidentCustomField_BoolAndDateTypes(t *testing.T) {
	noIncidentAPI(t)

	for _, fieldType := range []string{"bool", "date"} {
		t.Run(fieldType, func(t *testing.T) {
			resource.UnitTest(t, resource.TestCase{
				ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
				Steps: []resource.TestStep{
					{
						Config:             customFieldTypeConfig(fieldType, ""),
						PlanOnly:           true,
						ExpectNonEmptyPlan: true,
					},
					{
						Config: customFieldTypeConfig(fieldType, `
  select_option {
    value = "yes"
    label = "Yes"
  }`),
						PlanOnly:    true,
						ExpectError: regexp.MustCompile(`Select options require a select field`),
					},
				},
			})
		})
	}
}

// TestUnitIncidentCustomField_UnknownType pins that the type is one of the
// API's values, so a name the UI shows, such as boolean, is refused at plan
// time.
func TestUnitIncidentCustomField_UnknownType(t *testing.T) {
	noIncidentAPI(t)

	resource.UnitTest(t, resource.TestCase{
		ProtoV5ProviderFactories: testutils.ProtoV5ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config:      customFieldTypeConfig("boolean", ""),
				PlanOnly:    true,
				ExpectError: regexp.MustCompile(`value must be one of`),
			},
		},
	})
}
