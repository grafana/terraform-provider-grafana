package assistantapi

import "encoding/json"

// Watcher lifecycle states reported by the API. A watcher is only startable
// once it has been calibrated at least once.
const (
	WatcherStatusDraft       = "draft"
	WatcherStatusCalibrating = "calibrating"
	WatcherStatusReady       = "ready"
	WatcherStatusRunning     = "running"
	WatcherStatusPaused      = "paused"
	WatcherStatusError       = "error"
)

// WatcherQueryThresholds carries the numeric warning and critical boundaries
// for a calibrated check.
type WatcherQueryThresholds struct {
	Comparator string   `json:"comparator"`
	Warning    *float64 `json:"warning,omitempty"`
	Critical   *float64 `json:"critical,omitempty"`
	Source     string   `json:"source,omitempty"`
}

// WatcherQuery is a single calibrated check belonging to a watcher.
type WatcherQuery struct {
	ID            string                  `json:"id,omitempty"`
	Type          string                  `json:"type"`
	Expr          string                  `json:"expr"`
	DatasourceUID string                  `json:"datasourceUid,omitempty"`
	Comment       string                  `json:"comment,omitempty"`
	Enabled       *bool                   `json:"enabled,omitempty"`
	Role          string                  `json:"role,omitempty"`
	GoodWhen      string                  `json:"goodWhen,omitempty"`
	Thresholds    *WatcherQueryThresholds `json:"thresholds,omitempty"`
}

// WatcherSlackTarget identifies where a watcher posts its findings.
type WatcherSlackTarget struct {
	Type      string `json:"type"`
	ThreadTS  string `json:"threadTs,omitempty"`
	TeamID    string `json:"teamId,omitempty"`
	UserID    string `json:"userId,omitempty"`
	ChannelID string `json:"channelId,omitempty"`
}

// WatcherSlackAction configures Slack delivery for a watcher.
type WatcherSlackAction struct {
	MinSeverity string              `json:"minSeverity,omitempty"`
	Enabled     bool                `json:"enabled"`
	Target      *WatcherSlackTarget `json:"target,omitempty"`
}

// WatcherInvestigationAction configures whether a watcher may launch an
// Assistant Investigation on a critical assessment.
type WatcherInvestigationAction struct {
	Enabled   bool     `json:"enabled"`
	TeamNames []string `json:"teamNames,omitempty"`
}

// WatcherActions is the action policy evaluated after each watcher run.
type WatcherActions struct {
	MSTeams       json.RawMessage             `json:"msteams,omitempty"`
	Slack         *WatcherSlackAction         `json:"slack,omitempty"`
	Investigation *WatcherInvestigationAction `json:"investigation,omitempty"`
}

// Watcher represents a watcher agent returned by the API.
type Watcher struct {
	LearningEpoch          int64             `json:"learningEpoch"`
	RequestID              string            `json:"requestId,omitempty"`
	DefinitionVersion      int64             `json:"definitionVersion"`
	Labels                 map[string]string `json:"labels,omitempty"`
	Team                   *WatcherTeam      `json:"team,omitempty"`
	AutoStop               *WatcherAutoStop  `json:"autoStop,omitempty"`
	ArchivedAt             *string           `json:"archivedAt,omitempty"`
	ID                     string            `json:"id"`
	Name                   string            `json:"name"`
	Description            string            `json:"description,omitempty"`
	Prompt                 string            `json:"prompt"`
	Status                 string            `json:"status"`
	Sensitivity            string            `json:"sensitivity,omitempty"`
	TriggerIntervalSeconds int64             `json:"triggerIntervalSeconds,omitempty"`
	DisableDecisionSkip    *bool             `json:"disableDecisionSkip,omitempty"`
	DatasourceUIDs         []string          `json:"datasourceUids,omitempty"`
	CalibrationContext     string            `json:"calibrationContext,omitempty"`
	CalibratedAt           *string           `json:"calibratedAt,omitempty"`
	Queries                []WatcherQuery    `json:"queries,omitempty"`
	Actions                *WatcherActions   `json:"actions,omitempty"`
	CreatedAt              string            `json:"createdAt,omitempty"`
	CreatedBy              string            `json:"createdBy,omitempty"`
	UpdatedAt              string            `json:"updatedAt,omitempty"`
	LastRunAt              *string           `json:"lastRunAt,omitempty"`
	LastRunAssessment      string            `json:"lastRunAssessment,omitempty"`
	NextRunAt              *string           `json:"nextRunAt,omitempty"`
}

// WatcherCreate is the request body for creating a watcher. A freshly created
// watcher is in the draft state and must be calibrated before it can start.
type WatcherCreate struct {
	RequestID              string            `json:"requestId,omitempty"`
	Labels                 map[string]string `json:"labels,omitempty"`
	TeamUID                string            `json:"teamUid,omitempty"`
	AutoStop               *WatcherAutoStop  `json:"autoStop,omitempty"`
	Name                   string            `json:"name"`
	Description            string            `json:"description,omitempty"`
	Prompt                 string            `json:"prompt"`
	Sensitivity            string            `json:"sensitivity,omitempty"`
	TriggerIntervalSeconds int64             `json:"triggerIntervalSeconds,omitempty"`
	DisableDecisionSkip    *bool             `json:"disableDecisionSkip,omitempty"`
	DatasourceUIDs         []string          `json:"datasourceUids,omitempty"`
	Actions                *WatcherActions   `json:"actions,omitempty"`
}

// WatcherUpdate patches operator inputs without owning generated checks.
type WatcherUpdate struct {
	ExpectedDefinitionVersion *int64             `json:"expectedDefinitionVersion,omitempty"`
	Labels                    *map[string]string `json:"labels,omitempty"`
	TeamUID                   *string            `json:"teamUid,omitempty"`
	AutoStop                  *WatcherAutoStop   `json:"autoStop,omitempty"`
	Name                      *string            `json:"name,omitempty"`
	Description               *string            `json:"description,omitempty"`
	Prompt                    *string            `json:"prompt,omitempty"`
	Sensitivity               *string            `json:"sensitivity,omitempty"`
	TriggerIntervalSeconds    *int64             `json:"triggerIntervalSeconds,omitempty"`
	DisableDecisionSkip       *bool              `json:"disableDecisionSkip,omitempty"`
	DatasourceUIDs            *[]string          `json:"datasourceUids,omitempty"`
	Actions                   *WatcherActions    `json:"actions,omitempty"`
}

type WatcherTeam struct {
	UID string `json:"uid"`
}
type WatcherAutoStop struct {
	PauseAt string `json:"pauseAt,omitempty"`
	Archive bool   `json:"archive,omitempty"`
}
type watcherListData struct {
	Agents     []Watcher `json:"agents"`
	NextCursor string    `json:"nextCursor"`
}

// WatcherCalibrationRequest uses the public unattended calibration contract.
type WatcherCalibrationRequest struct {
	Unattended     bool `json:"unattended"`
	StartWhenReady bool `json:"startWhenReady"`
}
type WatcherCalibration struct {
	Status         string `json:"status"`
	RequestedBy    string `json:"requestedBy,omitempty"`
	StartedAt      string `json:"startedAt,omitempty"`
	Unattended     bool   `json:"unattended"`
	StartWhenReady bool   `json:"startWhenReady"`
	FailureReason  string `json:"failureReason,omitempty"`
	Summary        string `json:"summary,omitempty"`
	Message        string `json:"message,omitempty"`
}
