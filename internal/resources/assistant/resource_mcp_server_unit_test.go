package assistant

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/grafana/terraform-provider-grafana/v4/internal/common/assistantapi"
	"github.com/grafana/terraform-provider-grafana/v4/internal/util"
)

func testGitHubApp(appID, installationID int64, privateKey string) *mcpGitHubAppModel {
	return &mcpGitHubAppModel{
		AppID:          types.Int64Value(appID),
		InstallationID: types.Int64Value(installationID),
		PrivateKey:     types.StringValue(privateKey),
		PrivateKeyHash: types.StringValue(hashPrivateKey(privateKey)),
	}
}

func testMCPServerModel(githubApp *mcpGitHubAppModel) mcpServerModel {
	return mcpServerModel{
		Scope:   types.StringValue("tenant"),
		Name:    types.StringValue("github"),
		Enabled: types.BoolValue(true),
		Configuration: mcpConfigurationModel{
			URL: types.StringValue("https://api.githubcopilot.com/mcp/"),
		},
		GitHubApp: githubApp,
	}
}

func TestUnitMCPPlanToCreateGitHubApp(t *testing.T) {
	t.Parallel()

	body, diags := mcpPlanToCreate(context.Background(), testMCPServerModel(testGitHubApp(123, 456, "pem")))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body.GitHubAppID == nil || *body.GitHubAppID != 123 {
		t.Fatalf("unexpected app ID: %v", body.GitHubAppID)
	}
	if body.GitHubAppInstallationID == nil || *body.GitHubAppInstallationID != 456 {
		t.Fatalf("unexpected installation ID: %v", body.GitHubAppInstallationID)
	}
	if body.GitHubAppPrivateKey == nil || *body.GitHubAppPrivateKey != "pem" {
		t.Fatalf("unexpected private key: %v", body.GitHubAppPrivateKey)
	}

	body, diags = mcpPlanToCreate(context.Background(), testMCPServerModel(nil))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if body.GitHubAppID != nil || body.GitHubAppInstallationID != nil || body.GitHubAppPrivateKey != nil {
		t.Fatalf("expected no GitHub App credentials, got %+v", body)
	}
}

func TestUnitMCPPlanToUpdateGitHubApp(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		plan           *mcpGitHubAppModel
		state          *mcpGitHubAppModel
		planURL        string
		stateURL       string
		wantAppID      *int64
		wantInstallID  *int64
		wantPrivateKey *string
	}{
		{
			name: "no GitHub App",
		},
		{
			name:           "added",
			plan:           testGitHubApp(123, 456, "pem"),
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:          "unchanged key is not resent",
			plan:          testGitHubApp(123, 456, "pem"),
			state:         testGitHubApp(123, 456, "pem"),
			wantAppID:     util.Ptr(int64(123)),
			wantInstallID: util.Ptr(int64(456)),
		},
		{
			name:           "changed app ID resends key",
			plan:           testGitHubApp(123, 456, "pem"),
			state:          testGitHubApp(999, 456, "pem"),
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:           "changed installation ID resends key",
			plan:           testGitHubApp(123, 789, "pem"),
			state:          testGitHubApp(123, 456, "pem"),
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(789)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:           "changed key",
			plan:           testGitHubApp(123, 456, "new-pem"),
			state:          testGitHubApp(123, 456, "pem"),
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("new-pem"),
		},
		{
			name: "key missing from imported state",
			plan: testGitHubApp(123, 456, "pem"),
			state: &mcpGitHubAppModel{
				AppID:          types.Int64Value(123),
				InstallationID: types.Int64Value(456),
				PrivateKey:     types.StringNull(),
				PrivateKeyHash: types.StringNull(),
			},
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:           "changed URL host resends key",
			plan:           testGitHubApp(123, 456, "pem"),
			state:          testGitHubApp(123, 456, "pem"),
			planURL:        "https://copilot-api.acme.ghe.com/mcp",
			stateURL:       "https://api.githubcopilot.com/mcp/",
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:           "changed URL port resends key",
			plan:           testGitHubApp(123, 456, "pem"),
			state:          testGitHubApp(123, 456, "pem"),
			planURL:        "https://api.githubcopilot.com/mcp/",
			stateURL:       "https://api.githubcopilot.com:8443/mcp/",
			wantAppID:      util.Ptr(int64(123)),
			wantInstallID:  util.Ptr(int64(456)),
			wantPrivateKey: util.Ptr("pem"),
		},
		{
			name:          "changed URL path does not resend key",
			plan:          testGitHubApp(123, 456, "pem"),
			state:         testGitHubApp(123, 456, "pem"),
			planURL:       "https://api.githubcopilot.com/mcp/x/issues",
			stateURL:      "https://api.githubcopilot.com/mcp/x/repos",
			wantAppID:     util.Ptr(int64(123)),
			wantInstallID: util.Ptr(int64(456)),
		},
		{
			name:          "changed URL host case does not resend key",
			plan:          testGitHubApp(123, 456, "pem"),
			state:         testGitHubApp(123, 456, "pem"),
			planURL:       "https://API.GITHUBCOPILOT.COM/mcp/",
			stateURL:      "https://api.githubcopilot.com/mcp/",
			wantAppID:     util.Ptr(int64(123)),
			wantInstallID: util.Ptr(int64(456)),
		},
		{
			name:      "removed",
			state:     testGitHubApp(123, 456, "pem"),
			wantAppID: util.Ptr(int64(0)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			plan, state := testMCPServerModel(tt.plan), testMCPServerModel(tt.state)
			if tt.planURL != "" {
				plan.Configuration.URL = types.StringValue(tt.planURL)
			}
			if tt.stateURL != "" {
				state.Configuration.URL = types.StringValue(tt.stateURL)
			}
			body, diags := mcpPlanToUpdate(context.Background(), plan, state)
			if diags.HasError() {
				t.Fatalf("unexpected diagnostics: %v", diags)
			}
			assertInt64Ptr(t, "app ID", body.GitHubAppID, tt.wantAppID)
			assertInt64Ptr(t, "installation ID", body.GitHubAppInstallationID, tt.wantInstallID)
			if (body.GitHubAppPrivateKey == nil) != (tt.wantPrivateKey == nil) ||
				(body.GitHubAppPrivateKey != nil && *body.GitHubAppPrivateKey != *tt.wantPrivateKey) {
				t.Fatalf("unexpected private key: got %v, want %v", body.GitHubAppPrivateKey, tt.wantPrivateKey)
			}
		})
	}
}

func TestUnitMCPUpdateClearsGitHubAppWithZeroID(t *testing.T) {
	t.Parallel()

	body, diags := mcpPlanToUpdate(context.Background(), testMCPServerModel(nil), testMCPServerModel(testGitHubApp(123, 456, "pem")))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("failed to marshal update: %v", err)
	}
	if !strings.Contains(string(raw), `"githubAppId":0`) {
		t.Fatalf("expected githubAppId 0 in update body, got %s", raw)
	}
}

func TestUnitMCPToModelGitHubApp(t *testing.T) {
	t.Parallel()

	integration := assistantapi.Integration{
		ID:                      "id",
		Name:                    "github",
		Type:                    "mcp",
		Scope:                   "tenant",
		Configuration:           json.RawMessage(`{"url":"https://api.githubcopilot.com/mcp/"}`),
		GitHubAppID:             util.Ptr(int64(123)),
		GitHubAppInstallationID: util.Ptr(int64(456)),
	}

	model, diags := mcpToModel(context.Background(), integration, types.MapNull(types.StringType), types.StringValue(hashPrivateKey("pem")))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if model.GitHubApp == nil {
		t.Fatal("expected GitHub App credentials in model")
	}
	if model.GitHubApp.AppID.ValueInt64() != 123 || model.GitHubApp.InstallationID.ValueInt64() != 456 {
		t.Fatalf("unexpected GitHub App IDs: %+v", model.GitHubApp)
	}
	if !model.GitHubApp.PrivateKey.IsNull() {
		t.Fatal("expected write-only private key to be null in state")
	}
	if model.GitHubApp.PrivateKeyHash.ValueString() != hashPrivateKey("pem") {
		t.Fatalf("expected private key hash to be carried over, got %q", model.GitHubApp.PrivateKeyHash.ValueString())
	}

	integration.GitHubAppID = nil
	integration.GitHubAppInstallationID = nil
	model, diags = mcpToModel(context.Background(), integration, types.MapNull(types.StringType), types.StringValue(hashPrivateKey("pem")))
	if diags.HasError() {
		t.Fatalf("unexpected diagnostics: %v", diags)
	}
	if model.GitHubApp != nil {
		t.Fatalf("expected no GitHub App credentials, got %+v", model.GitHubApp)
	}
}

func assertInt64Ptr(t *testing.T, name string, got, want *int64) {
	t.Helper()
	if (got == nil) != (want == nil) || (got != nil && *got != *want) {
		t.Fatalf("unexpected %s: got %v, want %v", name, got, want)
	}
}

func TestUnitIsGitHubMCPURL(t *testing.T) {
	t.Parallel()

	for url, want := range map[string]bool{
		"https://api.githubcopilot.com/mcp":           true,
		"https://api.githubcopilot.com/mcp/":          true,
		"https://api.githubcopilot.com/mcp/x/repos":   true,
		"https://API.GITHUBCOPILOT.COM/mcp/":          true,
		"https://copilot-api.acme.ghe.com/mcp":        true,
		"https://copilot-api.acme.ghe.com/mcp/x/repo": true,
		"https://COPILOT-API.ACME.GHE.COM/mcp":        true,
		"":                                            false,
		"not a url":                                   false,
		"https://example.com/mcp":                     false,
		"https://api.github.com/mcp":                  false,
		"https://api.githubcopilot.com.evil.com/mcp":  false,
		"https://evil.com/api.githubcopilot.com/mcp":  false,
		"https://ghe.com/mcp":                         false,
		"https://acme.ghe.com.evil.com/mcp":           false,
		"https://api.githubcopilot.com/mcpx":          false,
		"https://acme.ghe.com/mcpx":                   false,
		"https://api.githubcopilot.com/":              false,
		"https://api.githubcopilot.com/MCP":           false,
		"http://api.githubcopilot.com/mcp":            false,
		"http://copilot-api.acme.ghe.com/mcp":         false,
		"https://token@api.githubcopilot.com/mcp":     false,
		"https://user@copilot-api.acme.ghe.com/mcp":   false,
		"https://api.githubcopilot.com:8443/mcp":      false,
		"https://copilot-api.acme.ghe.com:443/mcp":    false,
	} {
		if got := isGitHubMCPURL(url); got != want {
			t.Errorf("isGitHubMCPURL(%q) = %v, want %v", url, got, want)
		}
	}
}
