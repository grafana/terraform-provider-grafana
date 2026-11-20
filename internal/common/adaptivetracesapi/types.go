package adaptivetracesapi

import "encoding/json"

// Policy is a trace sampling policy. Body is a type-specific JSON object kept as raw JSON.
type Policy struct {
	ID        string          `json:"id,omitempty"`
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Body      json.RawMessage `json:"body"`
	ExpiresAt *string         `json:"expires_at,omitempty"` // RFC 3339; nil means the policy never expires

	// Read-only fields set by the server.
	VersionCreatedAt string `json:"version_created_at,omitempty"`
	VersionCreatedBy string `json:"version_created_by,omitempty"`
}

// policyRequest is the writable part of a policy, sent on create and update.
type policyRequest struct {
	Type      string          `json:"type"`
	Name      string          `json:"name"`
	Body      json.RawMessage `json:"body"`
	ExpiresAt *string         `json:"expires_at,omitempty"`
}

// newPolicyRequest drops the ID and read-only fields, which the server sets.
func newPolicyRequest(policy Policy) policyRequest {
	return policyRequest{
		Type:      policy.Type,
		Name:      policy.Name,
		Body:      policy.Body,
		ExpiresAt: policy.ExpiresAt,
	}
}
