package oncall

import (
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestUnitTeamHoursRoundTrip(t *testing.T) {
	prior := []teamHoursIntervalModel{{
		Name:   types.StringValue("Business hours"),
		Monday: []teamHoursRangeModel{{Start: types.StringValue("09:00"), End: types.StringValue("17:00")}},
	}}

	api := intervalsToAPI(prior)
	require.Equal(t, "09:00", api[0].Hours.Monday[0].Start)

	// The API returns HH:MM:SS and empty days; the configured form is preserved and empty days are empty blocks.
	got := intervalsFromAPI([]onCallAPI.TeamHoursInterval{{
		Name: "Business hours",
		Hours: onCallAPI.WeekHours{
			Monday:  []onCallAPI.TimeRange{{Start: "09:00:00", End: "17:00:00"}},
			Tuesday: []onCallAPI.TimeRange{},
		},
	}}, prior)
	require.Equal(t, prior[0].Monday, got[0].Monday)
	require.Empty(t, got[0].Tuesday)
	require.NotNil(t, got[0].Tuesday)

	// A real change on the server is surfaced.
	got = intervalsFromAPI([]onCallAPI.TeamHoursInterval{{
		Name:  "Business hours",
		Hours: onCallAPI.WeekHours{Monday: []onCallAPI.TimeRange{{Start: "10:00:00", End: "17:00:00"}}},
	}}, prior)
	require.Equal(t, "10:00:00", got[0].Monday[0].Start.ValueString())
}
