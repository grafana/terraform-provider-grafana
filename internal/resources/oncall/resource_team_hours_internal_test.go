package oncall

import (
	"testing"

	onCallAPI "github.com/grafana/amixr-api-go-client"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/stretchr/testify/require"
)

func TestUnitTeamHoursRoundTrip(t *testing.T) {
	cfg := []teamHoursIntervalModel{{
		Name:   types.StringValue("Business hours"),
		Monday: []teamHoursRangeModel{{Start: types.StringValue("09:00"), End: types.StringValue("17:00")}},
		Friday: []teamHoursRangeModel{{Start: types.StringValue("09:00"), End: types.StringValue("23:59:59")}},
	}}

	api := intervalsToAPI(cfg)
	require.Equal(t, "09:00", api[0].Hours.Monday[0].Start)

	// The API returns HH:MM:SS and empty days. Zero seconds are dropped so the result matches
	// the configured form with no prior state (as after an import); non-zero seconds are kept.
	got := intervalsFromAPI([]onCallAPI.TeamHoursInterval{{
		Name: "Business hours",
		Hours: onCallAPI.WeekHours{
			Monday:  []onCallAPI.TimeRange{{Start: "09:00:00", End: "17:00:00"}},
			Friday:  []onCallAPI.TimeRange{{Start: "09:00:00", End: "23:59:59"}},
			Tuesday: []onCallAPI.TimeRange{},
		},
	}})
	require.Equal(t, cfg[0].Monday, got[0].Monday)
	require.Equal(t, cfg[0].Friday, got[0].Friday)
	require.Empty(t, got[0].Tuesday)
	require.NotNil(t, got[0].Tuesday)

	// A real change on the server is surfaced.
	got = intervalsFromAPI([]onCallAPI.TeamHoursInterval{{
		Name:  "Business hours",
		Hours: onCallAPI.WeekHours{Monday: []onCallAPI.TimeRange{{Start: "10:00:00", End: "17:00:00"}}},
	}})
	require.Equal(t, "10:00", got[0].Monday[0].Start.ValueString())
}

func TestUnitTeamHoursTimeOfDayRegexp(t *testing.T) {
	for _, ok := range []string{"00:00", "09:00", "23:59", "23:59:59", "09:00:01"} {
		require.True(t, timeOfDayRegexp.MatchString(ok), ok)
	}
	for _, bad := range []string{"09:00:00", "9:00", "24:00", "09:60", "09:00:60", "09:00:"} {
		require.False(t, timeOfDayRegexp.MatchString(bad), bad)
	}
}
