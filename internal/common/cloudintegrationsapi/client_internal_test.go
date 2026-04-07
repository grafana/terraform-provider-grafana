package cloudintegrationsapi

import (
	"testing"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/cloudintegrationsapi/models"
	"github.com/stretchr/testify/assert"
)

// TestUnit_generateFolderUID tests that folder UID generation handles input in a similar manner to Grafana's
// integration plugin
func TestUnit_generateFolderUID(t *testing.T) {
	t.Parallel()

	c := &Client{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "spaces replaced with dashes", input: "Linux Node", expected: "linux-node"},
		{name: "single word lowercased", input: "Docker", expected: "docker"},
		{name: "empty string", input: "", expected: ""},
		{name: "already lowercase with dashes", input: "already-lower", expected: "already-lower"},
		{name: "multiple spaces", input: "A B C D", expected: "a-b-c-d"},
		{name: "mixed case no spaces", input: "GrafanaCloud", expected: "grafanacloud"},
		{name: "leading and trailing spaces", input: " Padded ", expected: "-padded-"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, c.generateFolderUID(tt.input))
		})
	}
}

// TestUnit_generateFolderUID_SpecialCharacters tests that non-alphanumeric characters
// are handled correctly by generateFolderUID.
func TestUnit_generateFolderUID_SpecialCharacters(t *testing.T) {
	t.Parallel()

	c := &Client{}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "dots and slashes unchanged", input: "project.100/team02/linux-node", expected: "project.100/team02/linux-node"},
		{name: "special characters unchanged", input: "Hello @ World!", expected: "hello-@-world!"},
		{name: "consecutive spaces to dashes", input: "a   b", expected: "a---b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, c.generateFolderUID(tt.input))
		})
	}
}

// TestUnit_resolveGrafanaRulesNamespace tests the logic for determining the namespace
// used for Grafana Alerting rules when installing an integration.
// Priority: dashboard_folder > rule_namespace > "Integration - <IntegrationName>"
func TestUnit_resolveGrafanaRulesNamespace(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name            string
		dashboardFolder string
		ruleNamespace   string
		integrationName string
		expected        string
	}{
		{
			name:            "dashboard_folder takes priority",
			dashboardFolder: "My Folder",
			ruleNamespace:   "ns",
			integrationName: "Docker",
			expected:        "My Folder",
		},
		{
			name:            "falls back to rule_namespace",
			dashboardFolder: "",
			ruleNamespace:   "ns",
			integrationName: "Docker",
			expected:        "ns",
		},
		{
			name:            "falls back to integration name",
			dashboardFolder: "",
			ruleNamespace:   "",
			integrationName: "Docker",
			expected:        "Integration - Docker",
		},
		{
			name:            "returns empty when all empty",
			dashboardFolder: "",
			ruleNamespace:   "",
			integrationName: "",
			expected:        "",
		},
		{
			name:            "dashboard_folder with empty others",
			dashboardFolder: "Solo",
			ruleNamespace:   "",
			integrationName: "",
			expected:        "Solo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := resolveGrafanaRulesNamespace(tt.dashboardFolder, tt.ruleNamespace, tt.integrationName)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestUnit_shouldInstallRulesOnInstall tests logic for determining whether rules should
// be installed to Grafana Alerting, during migration phase from Mimir to Grafana Alerting.
func TestUnit_shouldInstallRulesOnInstall(t *testing.T) {
	t.Parallel()

	rolloutLevelPtr := func(v models.RolloutLevel) *models.RolloutLevel { return &v }

	tests := []struct {
		name         string
		rolloutLevel *models.RolloutLevel
		expected     bool
	}{
		{name: "nil rollout level", rolloutLevel: nil, expected: false},
		{name: "level 0 (Mimir)", rolloutLevel: rolloutLevelPtr(RolloutLevelMimir), expected: false},
		{name: "level 1 (InstallOnly)", rolloutLevel: rolloutLevelPtr(RolloutLevelInstallOnly), expected: true},
		{name: "level 2 (Grafana)", rolloutLevel: rolloutLevelPtr(RolloutLevelGrafana), expected: true},
		{name: "level above max", rolloutLevel: rolloutLevelPtr(99), expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, shouldInstallRulesOnInstall(tt.rolloutLevel))
		})
	}
}
