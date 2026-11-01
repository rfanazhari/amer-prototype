package executionrepository

import "time"

// ExecutionModel maps agent_executions independently of the aggregate and the application DTO.
// Timestamps are never populated by GORM hooks so the trusted application clock stays authoritative.
type ExecutionModel struct {
	ID             string     `gorm:"type:uuid;primaryKey"`
	InstanceID     string     `gorm:"column:instance_id;type:uuid;not null"`
	RevisionID     string     `gorm:"column:published_revision_id;type:uuid;not null"`
	State          string     `gorm:"not null"`
	StateVersion   int64      `gorm:"column:state_version;not null"`
	ProgressRef    *string    `gorm:"column:committed_progress_ref;type:uuid"`
	ActivatedAt    *time.Time `gorm:"column:activated_at"`
	TerminalAt     *time.Time `gorm:"column:terminal_at"`
	TerminalReason *string    `gorm:"column:terminal_reason"`
	CreatedAt      time.Time  `gorm:"column:created_at;not null;autoCreateTime:false"`
	UpdatedAt      time.Time  `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the shared execution table used by GORM.
func (ExecutionModel) TableName() string { return "agent_executions" }

// TransitionModel maps one append-only agent_execution_transitions row.
type TransitionModel struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	ExecutionID  string    `gorm:"column:execution_id;type:uuid;not null"`
	FromState    *string   `gorm:"column:from_state"`
	ToState      string    `gorm:"column:to_state;not null"`
	StateVersion int64     `gorm:"column:state_version;not null"`
	ActorRef     string    `gorm:"column:actor_ref;not null"`
	OperationKey *string   `gorm:"column:operation_key"`
	Reason       *string   `gorm:"column:reason"`
	OccurredAt   time.Time `gorm:"column:occurred_at;not null"`
}

// TableName fixes the transition history table used by GORM.
func (TransitionModel) TableName() string { return "agent_execution_transitions" }

// PinModel maps interview_executions. Its created_at column is left to the database default.
type PinModel struct {
	ID           string `gorm:"type:uuid;primaryKey"`
	AssignmentID string `gorm:"column:assignment_id;type:uuid;not null"`
	InstanceID   string `gorm:"column:instance_id;type:uuid;not null"`
	RevisionID   string `gorm:"column:published_revision_id;type:uuid;not null"`
}

// TableName fixes the interview admission pin table used by GORM.
func (PinModel) TableName() string { return "interview_executions" }

// CommandModel maps one agent_execution_commands row, the idempotent identity of a caller command.
type CommandModel struct {
	ID          string    `gorm:"type:uuid;primaryKey"`
	Scope       string    `gorm:"not null"`
	CommandKind string    `gorm:"column:command_kind;not null"`
	RequestKey  string    `gorm:"column:request_key;not null"`
	Fingerprint string    `gorm:"column:intent_fingerprint;not null"`
	ExecutionID string    `gorm:"column:execution_id;type:uuid;not null"`
	CreatedAt   time.Time `gorm:"column:created_at;not null;autoCreateTime:false"`
}

// TableName fixes the command identity table used by GORM.
func (CommandModel) TableName() string { return "agent_execution_commands" }
