package assistantapi

import (
	"encoding/json"
	"time"
)

// apiResponseWrapper is the standard Huma response envelope.
type apiResponseWrapper[T any] struct {
	Schema string `json:"$schema"`
	Status string `json:"status"`
	Data   T      `json:"data"`
}

// Terms represents the Assistant terms acceptance state returned by the API.
type Terms struct {
	AcceptedTermsAndConditions bool `json:"acceptedTermsAndConditions"`
}

// TermsAcceptanceUpdate is the request body for accepting or withdrawing Assistant terms.
type TermsAcceptanceUpdate struct {
	AcceptedTermsAndConditions bool `json:"acceptedTermsAndConditions"`
}

// pagination contains pagination information returned by list endpoints.
type pagination struct {
	Total  int64 `json:"total"`
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
}

// ruleListData is the data payload of a rules list response.
type ruleListData struct {
	Rules      []Rule     `json:"rules"`
	Pagination pagination `json:"pagination"`
}

// skillListData is the data payload of a skills list response.
type skillListData struct {
	Skills     []Skill    `json:"skills"`
	Pagination pagination `json:"pagination"`
}

// quickstartListData is the data payload of a quickstarts list response.
type quickstartListData struct {
	Quickstarts []Quickstart `json:"quickstarts"`
	Pagination  pagination   `json:"pagination"`
}

// integrationListData is the data payload of an integrations list response.
type integrationListData struct {
	Integrations []Integration `json:"integrations"`
	Pagination   pagination    `json:"pagination"`
}

// Rule represents an assistant rule returned by the API.
type Rule struct {
	ID           string   `json:"id"`
	Created      string   `json:"created,omitempty"`
	Modified     string   `json:"modified,omitempty"`
	CreatedBy    string   `json:"createdBy,omitempty"`
	UpdatedBy    string   `json:"updatedBy,omitempty"`
	UserID       *string  `json:"userId,omitempty"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	RuleContent  string   `json:"ruleContent"`
	Enabled      *bool    `json:"enabled"`
	Priority     int      `json:"priority"`
	Scope        string   `json:"scope"`
	Applications []string `json:"applications,omitempty"`
}

// RuleCreate is the request body for creating a rule.
type RuleCreate struct {
	Scope        string   `json:"scope"`
	Name         string   `json:"name"`
	Description  string   `json:"description,omitempty"`
	RuleContent  string   `json:"ruleContent"`
	Enabled      *bool    `json:"enabled,omitempty"`
	Priority     int      `json:"priority,omitempty"`
	Applications []string `json:"applications,omitempty"`
}

// RuleUpdate is the request body for updating a rule.
type RuleUpdate struct {
	Scope        string    `json:"scope"`
	Name         *string   `json:"name,omitempty"`
	Description  *string   `json:"description,omitempty"`
	RuleContent  *string   `json:"ruleContent,omitempty"`
	Enabled      *bool     `json:"enabled,omitempty"`
	Priority     *int      `json:"priority,omitempty"`
	Applications *[]string `json:"applications,omitempty"`
}

// Skill represents an assistant skill returned by the API.
type Skill struct {
	ID                     string          `json:"id"`
	Name                   string          `json:"name"`
	Body                   string          `json:"body"`
	CommandName            *string         `json:"commandName,omitempty"`
	CommandEnabledAt       *time.Time      `json:"commandEnabledAt,omitempty"`
	CommandEnabledBy       *string         `json:"commandEnabledBy,omitempty"`
	CreatedBy              string          `json:"createdBy,omitempty"`
	CreatedAt              time.Time       `json:"created"`
	UpdatedBy              string          `json:"updatedBy,omitempty"`
	UpdatedAt              time.Time       `json:"modified"`
	Version                int             `json:"version"`
	IncludeInKnowledgebase bool            `json:"includeInKnowledgebase"`
	ContextItems           json.RawMessage `json:"contextItems,omitempty"`
	AllowedTools           []AllowedTool   `json:"allowedTools,omitempty"`
	Scope                  string          `json:"scope"`
}

// AllowedTool identifies an MCP tool allowed for a skill.
type AllowedTool struct {
	IntegrationID string `json:"integrationId"`
	ToolName      string `json:"toolName"`
}

// SkillCreate is the request body for creating a skill.
type SkillCreate struct {
	Name                   string          `json:"name"`
	Body                   string          `json:"body"`
	IncludeInKnowledgebase *bool           `json:"includeInKnowledgebase,omitempty"`
	ContextItems           json.RawMessage `json:"contextItems,omitempty"`
	Scope                  *string         `json:"scope,omitempty"`
	AllowedTools           []AllowedTool   `json:"allowedTools,omitempty"`
}

// SkillUpdate is the request body for updating a skill.
type SkillUpdate struct {
	Name                   *string          `json:"name,omitempty"`
	Body                   *string          `json:"body,omitempty"`
	IncludeInKnowledgebase *bool            `json:"includeInKnowledgebase,omitempty"`
	ContextItems           *json.RawMessage `json:"contextItems,omitempty"`
	Scope                  *string          `json:"scope,omitempty"`
	AllowedTools           *[]AllowedTool   `json:"allowedTools,omitempty"`
}

// SkillCommandUpdate is the request body for setting or disabling a skill command.
type SkillCommandUpdate struct {
	CommandName *string `json:"commandName"`
}

// Quickstart represents an assistant quickstart prompt returned by the API.
type Quickstart struct {
	ID           string          `json:"id"`
	Created      string          `json:"created,omitempty"`
	Modified     string          `json:"modified,omitempty"`
	CreatedBy    string          `json:"createdBy,omitempty"`
	UpdatedBy    string          `json:"updatedBy,omitempty"`
	UserID       *string         `json:"userId,omitempty"`
	Title        *string         `json:"title,omitempty"`
	Prompt       string          `json:"prompt"`
	ContextItems json.RawMessage `json:"contextItems,omitempty"`
	Enabled      *bool           `json:"enabled"`
	Scope        string          `json:"scope"`
}

// QuickstartCreate is the request body for creating a quickstart.
type QuickstartCreate struct {
	Scope        string          `json:"scope"`
	Title        *string         `json:"title,omitempty"`
	Prompt       string          `json:"prompt"`
	ContextItems json.RawMessage `json:"contextItems,omitempty"`
	Enabled      *bool           `json:"enabled,omitempty"`
}

// QuickstartUpdate is the request body for updating a quickstart.
type QuickstartUpdate struct {
	Scope        string           `json:"scope"`
	Title        *string          `json:"title,omitempty"`
	Prompt       *string          `json:"prompt,omitempty"`
	ContextItems *json.RawMessage `json:"contextItems,omitempty"`
	Enabled      *bool            `json:"enabled,omitempty"`
}

// MCPConfig is the configuration for an MCP server integration.
type MCPConfig struct {
	URL                  string            `json:"url,omitempty"`
	BuiltinID            string            `json:"builtinId,omitempty"`
	ToolPreferences      map[string]string `json:"toolPreferences,omitempty"`
	ToolApprovalPolicies map[string]string `json:"toolApprovalPolicies,omitempty"`
}

// Header is a custom HTTP header for MCP integrations.
type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	// Modified marks Value as a new value rather than a carried-over stored secret.
	Modified bool `json:"modified,omitempty"`
}

// Integration represents an MCP server integration returned by the API.
type Integration struct {
	ID                   string          `json:"id"`
	Created              string          `json:"created,omitempty"`
	Modified             string          `json:"modified,omitempty"`
	CreatedBy            string          `json:"createdBy,omitempty"`
	UpdatedBy            string          `json:"updatedBy,omitempty"`
	UserID               *string         `json:"userId,omitempty"`
	Name                 string          `json:"name"`
	Description          string          `json:"description,omitempty"`
	Type                 string          `json:"type"`
	Enabled              *bool           `json:"enabled"`
	Scope                string          `json:"scope"`
	Applications         []string        `json:"applications,omitempty"`
	Configuration        json.RawMessage `json:"configuration,omitempty"`
	CustomHeaders        []Header        `json:"customHeaders,omitempty"`
	AuthenticationFailed *bool           `json:"authenticationFailed,omitempty"`
	// The GitHub App private key is write-only and never returned.
	GitHubAppID             *int64 `json:"githubAppId,omitempty"`
	GitHubAppInstallationID *int64 `json:"githubAppInstallationId,omitempty"`
}

// IntegrationCreate is the request body for creating an integration.
type IntegrationCreate struct {
	Scope                   string          `json:"scope"`
	Name                    string          `json:"name"`
	Description             string          `json:"description,omitempty"`
	Type                    string          `json:"type"`
	Enabled                 *bool           `json:"enabled,omitempty"`
	Applications            []string        `json:"applications,omitempty"`
	Configuration           json.RawMessage `json:"configuration,omitempty"`
	CustomHeaders           []Header        `json:"customHeaders,omitempty"`
	GitHubAppID             *int64          `json:"githubAppId,omitempty"`
	GitHubAppInstallationID *int64          `json:"githubAppInstallationId,omitempty"`
	GitHubAppPrivateKey     *string         `json:"githubAppPrivateKey,omitempty"`
}

// IntegrationUpdate is the request body for updating an integration.
//
// A GitHubAppID of 0 clears all GitHub App credentials. An omitted
// GitHubAppPrivateKey keeps the stored key.
type IntegrationUpdate struct {
	Scope                   string           `json:"scope"`
	Name                    *string          `json:"name,omitempty"`
	Description             *string          `json:"description,omitempty"`
	Enabled                 *bool            `json:"enabled,omitempty"`
	Applications            *[]string        `json:"applications,omitempty"`
	Configuration           *json.RawMessage `json:"configuration,omitempty"`
	CustomHeaders           *[]Header        `json:"customHeaders,omitempty"`
	GitHubAppID             *int64           `json:"githubAppId,omitempty"`
	GitHubAppInstallationID *int64           `json:"githubAppInstallationId,omitempty"`
	GitHubAppPrivateKey     *string          `json:"githubAppPrivateKey,omitempty"`
}

// automationListData is the data payload of an automations list response.
type automationListData struct {
	Automations []Automation `json:"automations"`
	NextCursor  string       `json:"nextCursor"`
}

// AutomationSlackTarget identifies where an automation posts its run result.
type AutomationSlackTarget struct {
	Type      string `json:"type"`
	TeamID    string `json:"teamId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	ChannelID string `json:"channelId,omitempty"`
}

// AutomationSlackNotification configures Slack delivery for automation runs.
type AutomationSlackNotification struct {
	Enabled  bool                   `json:"enabled"`
	NotifyOn []string               `json:"notifyOn,omitempty"`
	Target   *AutomationSlackTarget `json:"target,omitempty"`
}

// AutomationEmailNotification configures email delivery for automation runs.
type AutomationEmailNotification struct {
	Enabled   bool     `json:"enabled"`
	NotifyOn  []string `json:"notifyOn,omitempty"`
	Addresses []string `json:"addresses,omitempty"`
}

// AutomationNotifications is the notification provider set for an automation.
// Webhook notifications are not modelled: their URL and credentials use
// write-only secure settings that the API never returns.
//
// The fields deliberately omit `omitempty` for create serialization. Updates
// use AutomationUpdate.Notifications to emit only changed destinations: null
// deletes one provider, while an absent key preserves the stored configuration.
type AutomationNotifications struct {
	Slack *AutomationSlackNotification `json:"slack"`
	Email *AutomationEmailNotification `json:"email"`
}

// AutomationTrigger uses a caller-supplied UUID to retain identity across updates.
type AutomationTrigger struct {
	ID     string          `json:"id"`
	Type   string          `json:"type"`
	Config json.RawMessage `json:"config"`
}

// AutomationSchedule is the schedule reported on an automation response.
type AutomationSchedule struct {
	Cron      string `json:"cron"`
	Timezone  string `json:"timezone,omitempty"`
	NextRunAt string `json:"nextRunAt,omitempty"`
	EndsAt    string `json:"endsAt,omitempty"`
}

// Automation represents a saved Assistant prompt that runs manually or on a
// cron schedule.
type Automation struct {
	ID            string                   `json:"id"`
	Name          string                   `json:"name"`
	Description   string                   `json:"description,omitempty"`
	Prompt        string                   `json:"prompt"`
	Enabled       bool                     `json:"enabled"`
	Scope         string                   `json:"scope"`
	ExecutionMode string                   `json:"executionMode,omitempty"`
	Triggers      []AutomationTrigger      `json:"triggers,omitempty"`
	Labels        map[string]string        `json:"labels,omitempty"`
	Schedule      *AutomationSchedule      `json:"schedule,omitempty"`
	ContextItems  json.RawMessage          `json:"contextItems,omitempty"`
	Notifications *AutomationNotifications `json:"notifications,omitempty"`
	CreatedAt     string                   `json:"createdAt,omitempty"`
	CreatedBy     string                   `json:"createdBy,omitempty"`
	UpdatedAt     string                   `json:"updatedAt,omitempty"`
	UpdatedBy     string                   `json:"updatedBy,omitempty"`
}

// AutomationCreate is the request body for creating an automation.
type AutomationCreate struct {
	Name             string                   `json:"name"`
	Description      string                   `json:"description,omitempty"`
	Prompt           string                   `json:"prompt"`
	Enabled          bool                     `json:"enabled"`
	Scope            string                   `json:"scope,omitempty"`
	ExecutionMode    string                   `json:"executionMode,omitempty"`
	Triggers         []AutomationTrigger      `json:"triggers,omitempty"`
	Labels           map[string]string        `json:"labels,omitempty"`
	ScheduleEndsAt   string                   `json:"scheduleEndsAt,omitempty"`
	ScheduleCron     string                   `json:"scheduleCron,omitempty"`
	ScheduleTimezone string                   `json:"scheduleTimezone,omitempty"`
	ContextItems     json.RawMessage          `json:"contextItems,omitempty"`
	Notifications    *AutomationNotifications `json:"notifications,omitempty"`
}

// AutomationUpdate is the request body for updating an automation.
type AutomationUpdate struct {
	Name             *string                    `json:"name,omitempty"`
	Description      *string                    `json:"description,omitempty"`
	Prompt           *string                    `json:"prompt,omitempty"`
	Enabled          *bool                      `json:"enabled,omitempty"`
	Scope            *string                    `json:"scope,omitempty"`
	ExecutionMode    *string                    `json:"executionMode,omitempty"`
	Triggers         *[]AutomationTrigger       `json:"triggers,omitempty"`
	Labels           *map[string]string         `json:"labels,omitempty"`
	ScheduleEndsAt   *string                    `json:"scheduleEndsAt,omitempty"`
	ScheduleCron     *string                    `json:"scheduleCron,omitempty"`
	ScheduleTimezone *string                    `json:"scheduleTimezone,omitempty"`
	ContextItems     *json.RawMessage           `json:"contextItems,omitempty"`
	Notifications    map[string]json.RawMessage `json:"notifications,omitempty"`
}
