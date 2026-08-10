package agento11yapi

import (
	"encoding/json"
	"time"
)

// listResponse is the standard cursor-paginated envelope returned by the
// agent observability evaluation control plane: {"items": [...], "next_cursor": "..."}.
type listResponse[T any] struct {
	Items      []T    `json:"items"`
	NextCursor string `json:"next_cursor"`
}

// itemsResponse is the non-paginated list envelope used by rule actions:
// {"items": [...]}.
type itemsResponse[T any] struct {
	Items []T `json:"items"`
}

// OutputKey describes a single score output produced by an evaluator.
type OutputKey struct {
	Key           string   `json:"key"`
	Type          string   `json:"type,omitempty"`
	Description   string   `json:"description,omitempty"`
	Unit          string   `json:"unit,omitempty"`
	PassThreshold *float64 `json:"pass_threshold,omitempty"`
	Enum          []string `json:"enum,omitempty"`
	Min           *float64 `json:"min,omitempty"`
	Max           *float64 `json:"max,omitempty"`
	PassMatch     []string `json:"pass_match,omitempty"`
	PassValue     *bool    `json:"pass_value,omitempty"`
}

// Evaluator is an evaluator definition returned by the API.
type Evaluator struct {
	TenantID              string          `json:"tenant_id"`
	EvaluatorID           string          `json:"evaluator_id"`
	Version               string          `json:"version"`
	Kind                  string          `json:"kind"`
	Description           string          `json:"description,omitempty"`
	Config                json.RawMessage `json:"config"`
	OutputKeys            []OutputKey     `json:"output_keys"`
	IsPredefined          bool            `json:"is_predefined"`
	SourceTemplateID      string          `json:"source_template_id,omitempty"`
	SourceTemplateVersion string          `json:"source_template_version,omitempty"`
	CreatedBy             string          `json:"created_by,omitempty"`
	UpdatedBy             string          `json:"updated_by,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
	UpdatedAt             time.Time       `json:"updated_at"`
}

// EvaluatorWrite is the request body to create or update an evaluator.
type EvaluatorWrite struct {
	EvaluatorID string          `json:"evaluator_id"`
	Version     string          `json:"version"`
	Kind        string          `json:"kind"`
	Description string          `json:"description,omitempty"`
	Config      json.RawMessage `json:"config"`
	OutputKeys  []OutputKey     `json:"output_keys"`
}

// Rule is an asynchronous (online) evaluation rule returned by the API.
type Rule struct {
	TenantID          string          `json:"tenant_id"`
	RuleID            string          `json:"rule_id"`
	Enabled           bool            `json:"enabled"`
	Selector          string          `json:"selector"`
	Match             json.RawMessage `json:"match,omitempty"`
	SampleRate        float64         `json:"sample_rate"`
	EvaluatorIDs      []string        `json:"evaluator_ids"`
	ExecutionMode     string          `json:"execution_mode,omitempty"`
	AlertRuleUIDs     []string        `json:"alert_rule_uids,omitempty"`
	FilterableTagKeys []string        `json:"filterable_tag_keys,omitempty"`
	MinIdleSeconds    *int            `json:"min_idle_seconds,omitempty"`
	CreatedBy         string          `json:"created_by,omitempty"`
	UpdatedBy         string          `json:"updated_by,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// RuleWrite is the request body to create (or upsert) an evaluation rule.
type RuleWrite struct {
	RuleID            string          `json:"rule_id"`
	Enabled           *bool           `json:"enabled,omitempty"`
	Selector          string          `json:"selector,omitempty"`
	Match             json.RawMessage `json:"match,omitempty"`
	SampleRate        *float64        `json:"sample_rate,omitempty"`
	EvaluatorIDs      []string        `json:"evaluator_ids"`
	ExecutionMode     string          `json:"execution_mode,omitempty"`
	AlertRuleUIDs     []string        `json:"alert_rule_uids,omitempty"`
	FilterableTagKeys []string        `json:"filterable_tag_keys,omitempty"`
	MinIdleSeconds    *int            `json:"min_idle_seconds,omitempty"`
}

// RulePatch is the request body to partially update an evaluation rule.
type RulePatch struct {
	Enabled           *bool           `json:"enabled,omitempty"`
	Selector          *string         `json:"selector,omitempty"`
	Match             json.RawMessage `json:"match,omitempty"`
	SampleRate        *float64        `json:"sample_rate,omitempty"`
	EvaluatorIDs      []string        `json:"evaluator_ids,omitempty"`
	ExecutionMode     *string         `json:"execution_mode,omitempty"`
	AlertRuleUIDs     []string        `json:"alert_rule_uids,omitempty"`
	FilterableTagKeys []string        `json:"filterable_tag_keys"`
	MinIdleSeconds    *int            `json:"min_idle_seconds,omitempty"`
}

// RuleActionCondition is the trigger expression attached to a rule action.
type RuleActionCondition struct {
	Kind string `json:"kind"`
}

// RuleActionConfig carries the action-type-specific configuration.
type RuleActionConfig struct {
	Kind          string   `json:"kind"`
	CollectionIDs []string `json:"collection_ids,omitempty"`
}

// RuleAction is a side effect attached to an evaluation rule.
type RuleAction struct {
	TenantID     string              `json:"tenant_id"`
	ActionID     string              `json:"action_id"`
	RuleID       string              `json:"rule_id"`
	Condition    RuleActionCondition `json:"condition"`
	ActionConfig RuleActionConfig    `json:"action_config"`
	Enabled      bool                `json:"enabled"`
	CreatedBy    string              `json:"created_by,omitempty"`
	UpdatedBy    string              `json:"updated_by,omitempty"`
	CreatedAt    time.Time           `json:"created_at"`
	UpdatedAt    time.Time           `json:"updated_at"`
}

// RuleActionCreate is the request body to create a rule action.
type RuleActionCreate struct {
	Condition    RuleActionCondition `json:"condition"`
	ActionConfig RuleActionConfig    `json:"action_config"`
	Enabled      *bool               `json:"enabled,omitempty"`
}

// RuleActionUpdate is the request body to patch a rule action. Nil fields are
// left untouched.
type RuleActionUpdate struct {
	Condition    *RuleActionCondition `json:"condition,omitempty"`
	ActionConfig *RuleActionConfig    `json:"action_config,omitempty"`
	Enabled      *bool                `json:"enabled,omitempty"`
}

// Collection is a named group of saved conversations returned by the API.
type Collection struct {
	TenantID     string    `json:"tenant_id"`
	CollectionID string    `json:"collection_id"`
	Name         string    `json:"name"`
	Description  string    `json:"description,omitempty"`
	MemberCount  int       `json:"member_count"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// CollectionCreate is the request body to create a collection. The API assigns
// the collection ID and derives created_by from the request actor, so neither
// is sent.
type CollectionCreate struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CollectionPatch is the request body to update a collection. It replaces both
// fields, so an empty description clears the stored value.
type CollectionPatch struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// HookToolFilter blocks tool calls whose names match any of the glob patterns.
type HookToolFilter struct {
	BlockedNames []string `json:"blocked_names"`
}

// HookRedactPattern is a single regex redaction step.
type HookRedactPattern struct {
	ID    string `json:"id,omitempty"`
	Regex string `json:"regex"`
}

// HookRedactConfig is the ordered set of regex redaction patterns applied to a hook rule.
type HookRedactConfig struct {
	Patterns []HookRedactPattern `json:"patterns"`
}

// HookRule is a synchronous (request-path) guard rule returned by the API.
type HookRule struct {
	TenantID     string            `json:"tenant_id"`
	RuleID       string            `json:"rule_id"`
	Enabled      bool              `json:"enabled"`
	Phase        string            `json:"phase"`
	Priority     int               `json:"priority"`
	Selector     string            `json:"selector"`
	Match        json.RawMessage   `json:"match,omitempty"`
	EvaluatorIDs []string          `json:"evaluator_ids"`
	ActionOnFail string            `json:"action_on_fail"`
	ShortCircuit bool              `json:"short_circuit"`
	ToolFilter   *HookToolFilter   `json:"tool_filter,omitempty"`
	Redact       *HookRedactConfig `json:"redact,omitempty"`
	CreatedBy    string            `json:"created_by,omitempty"`
	UpdatedBy    string            `json:"updated_by,omitempty"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}

// HookRuleWrite is the request body to create or upsert a hook rule.
type HookRuleWrite struct {
	RuleID       string            `json:"rule_id,omitempty"`
	Enabled      *bool             `json:"enabled,omitempty"`
	Phase        string            `json:"phase,omitempty"`
	Priority     *int              `json:"priority,omitempty"`
	Selector     string            `json:"selector,omitempty"`
	Match        json.RawMessage   `json:"match,omitempty"`
	EvaluatorIDs []string          `json:"evaluator_ids,omitempty"`
	ActionOnFail string            `json:"action_on_fail,omitempty"`
	ShortCircuit *bool             `json:"short_circuit,omitempty"`
	ToolFilter   *HookToolFilter   `json:"tool_filter,omitempty"`
	Redact       *HookRedactConfig `json:"redact,omitempty"`
}
