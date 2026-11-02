package interviewrepository

import "time"

// ProgressModel maps interview_progress independently of the aggregate and the application record.
// Timestamps are never populated by GORM hooks so the trusted application clock stays authoritative.
type ProgressModel struct {
	ExecutionID         string     `gorm:"column:execution_id;type:uuid;primaryKey"`
	FollowUpLimit       int        `gorm:"column:follow_up_limit;not null"`
	CurrentStage        string     `gorm:"column:current_stage;not null"`
	StageActivatedAt    time.Time  `gorm:"column:stage_activated_at;not null"`
	CommittedVersion    int64      `gorm:"column:committed_version;not null"`
	ControllerID        *string    `gorm:"column:controller_id;type:uuid"`
	FollowUpEntryIDs    string     `gorm:"column:follow_up_entry_ids;type:jsonb;not null"`
	ReconnectEpisodes   int        `gorm:"column:reconnect_episodes;not null"`
	ReconnectAttempts   int        `gorm:"column:reconnect_attempts;not null"`
	ReconnectDetectedAt *time.Time `gorm:"column:reconnect_detected_at"`
	CreatedAt           time.Time  `gorm:"column:created_at;not null;autoCreateTime:false"`
	UpdatedAt           time.Time  `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the progress table used by GORM.
func (ProgressModel) TableName() string { return "interview_progress" }

// CoverageModel maps one interview_coverage_items row.
type CoverageModel struct {
	ID                   string    `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID          string    `gorm:"column:execution_id;type:uuid;not null"`
	QuestionID           string    `gorm:"column:question_id;not null"`
	Stage                string    `gorm:"column:stage;not null"`
	State                string    `gorm:"column:state;not null"`
	OmissionReason       *string   `gorm:"column:omission_reason"`
	ClarificationCount   int       `gorm:"column:clarification_count;not null"`
	AnswerEntryID        *string   `gorm:"column:answer_entry_id;type:uuid"`
	ClarificationEntryID *string   `gorm:"column:clarification_entry_id;type:uuid"`
	OmissionEntryID      *string   `gorm:"column:omission_entry_id;type:uuid"`
	UpdatedAt            time.Time `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the coverage table used by GORM.
func (CoverageModel) TableName() string { return "interview_coverage_items" }

// SessionModel maps one interview_voice_sessions row.
type SessionModel struct {
	ID                string     `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID       string     `gorm:"column:execution_id;type:uuid;not null"`
	Stage             string     `gorm:"column:stage;not null"`
	AttemptID         string     `gorm:"column:attempt_id;type:uuid;not null"`
	ProviderSessionID *string    `gorm:"column:provider_session_id"`
	ReadinessAt       *time.Time `gorm:"column:readiness_at"`
	ReplacedSessionID *string    `gorm:"column:replaced_session_id;type:uuid"`
	OpenedAt          time.Time  `gorm:"column:opened_at;not null"`
	ClosedAt          *time.Time `gorm:"column:closed_at"`
}

// TableName fixes the voice session table used by GORM.
func (SessionModel) TableName() string { return "interview_voice_sessions" }

// EntryModel maps one interview_transcript_entries row that the caller already classified.
type EntryModel struct {
	ID                string    `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID       string    `gorm:"column:execution_id;type:uuid;not null"`
	SessionID         string    `gorm:"column:session_id;type:uuid;not null"`
	AttemptID         string    `gorm:"column:attempt_id;type:uuid;not null"`
	Speaker           string    `gorm:"column:speaker;not null"`
	Ordinal           int       `gorm:"column:ordinal;not null"`
	Revision          int       `gorm:"column:revision;not null"`
	EvidenceState     string    `gorm:"column:evidence_state;not null"`
	Body              string    `gorm:"column:body;not null"`
	CitesEntryID      *string   `gorm:"column:cites_entry_id;type:uuid"`
	SupersedesEntryID *string   `gorm:"column:supersedes_entry_id;type:uuid"`
	CommittedAt       time.Time `gorm:"column:committed_at;not null"`
}

// TableName fixes the transcript table used by GORM.
func (EntryModel) TableName() string { return "interview_transcript_entries" }

// LossModel maps one interview_technical_losses row.
type LossModel struct {
	ID          string     `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID string     `gorm:"column:execution_id;type:uuid;not null"`
	QuestionID  string     `gorm:"column:question_id;not null"`
	SessionID   string     `gorm:"column:session_id;type:uuid;not null"`
	AttemptID   string     `gorm:"column:attempt_id;type:uuid;not null"`
	ResolvedAt  *time.Time `gorm:"column:resolved_at"`
	CreatedAt   time.Time  `gorm:"column:created_at;not null;autoCreateTime:false"`
}

// TableName fixes the technical loss table used by GORM.
func (LossModel) TableName() string { return "interview_technical_losses" }

// HandoffModel maps one interview_handoffs row.
type HandoffModel struct {
	ID          string    `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID string    `gorm:"column:execution_id;type:uuid;not null"`
	SourceStage string    `gorm:"column:source_stage;not null"`
	TargetStage *string   `gorm:"column:target_stage"`
	Revision    int       `gorm:"column:revision;not null"`
	Summary     *string   `gorm:"column:summary"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime:false"`
}

// TableName fixes the handoff table used by GORM.
func (HandoffModel) TableName() string { return "interview_handoffs" }

// CheckpointModel maps one interview_checkpoints row.
type CheckpointModel struct {
	ID             string    `gorm:"column:id;type:uuid;primaryKey"`
	ExecutionID    string    `gorm:"column:execution_id;type:uuid;not null"`
	SessionID      string    `gorm:"column:session_id;type:uuid;not null"`
	AttemptID      string    `gorm:"column:attempt_id;type:uuid;not null"`
	ThroughOrdinal int       `gorm:"column:through_ordinal;not null"`
	CreatedAt      time.Time `gorm:"column:created_at;not null;autoCreateTime:false"`
}

// TableName fixes the checkpoint table used by GORM.
func (CheckpointModel) TableName() string { return "interview_checkpoints" }
