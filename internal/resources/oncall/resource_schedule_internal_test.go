package oncall

import (
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// scheduleConfig builds the raw config value that Terraform passes to
// CustomizeDiff, with every attribute the validation looks at present and null
// unless the test sets it.
func scheduleConfig(attrs map[string]cty.Value) cty.Value {
	config := map[string]cty.Value{
		"type":             cty.NullVal(cty.String),
		"shifts":           cty.NullVal(cty.Set(cty.String)),
		"ical_url_primary": cty.NullVal(cty.String),
		"time_zone":        cty.NullVal(cty.String),
	}
	for k, v := range attrs {
		config[k] = v
	}

	return cty.ObjectVal(config)
}

func TestUnitValidateScheduleConfig(t *testing.T) {
	shifts := cty.SetVal([]cty.Value{cty.StringVal("SBM4FGN10MWYR")})

	for _, tc := range []struct {
		name    string
		config  cty.Value
		wantErr string
	}{
		{
			name:   "shifts on a calendar schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("calendar"), "shifts": shifts}),
		},
		{
			name:    "shifts on a web schedule",
			config:  scheduleConfig(map[string]cty.Value{"type": cty.StringVal("web"), "shifts": shifts}),
			wantErr: "shifts can not be set with type: web",
		},
		{
			name:    "shifts on an ical schedule",
			config:  scheduleConfig(map[string]cty.Value{"type": cty.StringVal("ical"), "shifts": shifts}),
			wantErr: "shifts can not be set with type: ical",
		},
		{
			name:   "empty shifts on a web schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("web"), "shifts": cty.SetValEmpty(cty.String)}),
		},
		{
			name:   "unknown shifts on a web schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("web"), "shifts": cty.UnknownVal(cty.Set(cty.String))}),
		},
		{
			name: "shifts with an unresolved element on a web schedule",
			config: scheduleConfig(map[string]cty.Value{
				"type":   cty.StringVal("web"),
				"shifts": cty.SetVal([]cty.Value{cty.UnknownVal(cty.String)}),
			}),
			wantErr: "shifts can not be set with type: web",
		},
		{
			name:   "ical_url_primary on an ical schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("ical"), "ical_url_primary": cty.StringVal("https://example.com/cal.ics")}),
		},
		{
			name:    "ical_url_primary on a web schedule",
			config:  scheduleConfig(map[string]cty.Value{"type": cty.StringVal("web"), "ical_url_primary": cty.StringVal("https://example.com/cal.ics")}),
			wantErr: "ical_url_primary can not be set with type: web",
		},
		{
			name:    "ical_url_primary on a calendar schedule",
			config:  scheduleConfig(map[string]cty.Value{"type": cty.StringVal("calendar"), "ical_url_primary": cty.StringVal("https://example.com/cal.ics")}),
			wantErr: "ical_url_primary can not be set with type: calendar",
		},
		{
			name:   "time_zone on a calendar schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("calendar"), "time_zone": cty.StringVal("UTC")}),
		},
		{
			name:   "time_zone on a web schedule",
			config: scheduleConfig(map[string]cty.Value{"type": cty.StringVal("web"), "time_zone": cty.StringVal("UTC")}),
		},
		{
			name:    "time_zone on an ical schedule",
			config:  scheduleConfig(map[string]cty.Value{"type": cty.StringVal("ical"), "time_zone": cty.StringVal("UTC")}),
			wantErr: "time_zone can not be set with type: ical",
		},
		{
			name: "shifts reported before the other attributes",
			config: scheduleConfig(map[string]cty.Value{
				"type":             cty.StringVal("web"),
				"shifts":           shifts,
				"ical_url_primary": cty.StringVal("https://example.com/cal.ics"),
			}),
			wantErr: "shifts can not be set with type: web",
		},
		{
			name:   "no type set",
			config: scheduleConfig(map[string]cty.Value{"shifts": shifts}),
		},
		{
			name:   "unknown type",
			config: scheduleConfig(map[string]cty.Value{"type": cty.UnknownVal(cty.String), "shifts": shifts}),
		},
		{
			name:   "null config",
			config: cty.NullVal(cty.EmptyObject),
		},
		{
			name:   "unknown config",
			config: cty.UnknownVal(cty.EmptyObject),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := validateScheduleConfig(tc.config)

			switch {
			case tc.wantErr == "" && err != nil:
				t.Fatalf("expected no error, got %q", err)
			case tc.wantErr != "" && err == nil:
				t.Fatalf("expected error %q, got none", tc.wantErr)
			case tc.wantErr != "" && err.Error() != tc.wantErr:
				t.Fatalf("expected error %q, got %q", tc.wantErr, err)
			}
		})
	}
}
