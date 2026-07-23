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
}

// IntegrationCreate is the request body for creating an integration.
type IntegrationCreate struct {
	Scope         string          `json:"scope"`
	Name          string          `json:"name"`
	Description   string          `json:"description,omitempty"`
	Type          string          `json:"type"`
	Enabled       *bool           `json:"enabled,omitempty"`
	Applications  []string        `json:"applications,omitempty"`
	Configuration json.RawMessage `json:"configuration,omitempty"`
	CustomHeaders []Header        `json:"customHeaders,omitempty"`
}

// IntegrationUpdate is the request body for updating an integration.
type IntegrationUpdate struct {
	Scope         string           `json:"scope"`
	Name          *string          `json:"name,omitempty"`
	Description   *string          `json:"description,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
	Applications  *[]string        `json:"applications,omitempty"`
	Configuration *json.RawMessage `json:"configuration,omitempty"`
	CustomHeaders *[]Header        `json:"customHeaders,omitempty"`
}
