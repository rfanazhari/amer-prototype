package interviewreportrepository

import "time"

// ReportModel maps interview_reports independently of the aggregate and the application DTO.
// Timestamps are never populated by GORM hooks so the trusted application clock stays authoritative.
type ReportModel struct {
	ID           string    `gorm:"type:uuid;primaryKey"`
	ExecutionID  string    `gorm:"column:execution_id;type:uuid;not null"`
	AssignmentID string    `gorm:"column:assignment_id;type:uuid;not null"`
	CreatedAt    time.Time `gorm:"column:created_at;not null;autoCreateTime:false"`
}

// TableName fixes the report table used by GORM.
func (ReportModel) TableName() string { return "interview_reports" }

// GenerationModel maps interview_report_generations. A nil FailureReason is stored as null.
type GenerationModel struct {
	ID               string    `gorm:"type:uuid;primaryKey"`
	ReportID         string    `gorm:"column:report_id;type:uuid;not null"`
	GenerationNumber int       `gorm:"column:generation_number;not null"`
	State            string    `gorm:"not null"`
	FailureReason    *string   `gorm:"column:failure_reason"`
	CreatedAt        time.Time `gorm:"column:created_at;not null;autoCreateTime:false"`
	UpdatedAt        time.Time `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the generation table used by GORM.
func (GenerationModel) TableName() string { return "interview_report_generations" }

// StatementModel maps interview_report_statements. Nil evidence pointers are stored as null.
type StatementModel struct {
	ID               string  `gorm:"type:uuid;primaryKey"`
	GenerationID     string  `gorm:"column:generation_id;type:uuid;not null"`
	Ordinal          int     `gorm:"not null"`
	Kind             string  `gorm:"not null"`
	Body             string  `gorm:"not null"`
	EvidenceEntryID  *string `gorm:"column:evidence_entry_id;type:uuid"`
	EvidenceRevision *int    `gorm:"column:evidence_revision"`
	PresentsAsAnswer bool    `gorm:"column:presents_as_answer;not null"`
}

// TableName fixes the statement table used by GORM.
func (StatementModel) TableName() string { return "interview_report_statements" }

// JobModel maps interview_report_jobs. Nil window and attempt pointers are stored as null.
type JobModel struct {
	ID                string     `gorm:"type:uuid;primaryKey"`
	GenerationID      string     `gorm:"column:generation_id;type:uuid;not null"`
	State             string     `gorm:"not null"`
	AttemptCount      int        `gorm:"column:attempt_count;not null"`
	FencingToken      int64      `gorm:"column:fencing_token;not null"`
	CreatedAt         time.Time  `gorm:"column:created_at;not null;autoCreateTime:false"`
	WaitDeadlineAt    time.Time  `gorm:"column:wait_deadline_at;not null"`
	WindowStartedAt   *time.Time `gorm:"column:window_started_at"`
	DeadlineAt        *time.Time `gorm:"column:deadline_at"`
	EligibleAt        time.Time  `gorm:"column:eligible_at;not null"`
	AttemptDeadlineAt *time.Time `gorm:"column:attempt_deadline_at"`
	UpdatedAt         time.Time  `gorm:"column:updated_at;not null;autoUpdateTime:false"`
}

// TableName fixes the job table used by GORM.
func (JobModel) TableName() string { return "interview_report_jobs" }

// AttemptModel maps interview_report_attempts. Nil finish and outcome are stored as null.
type AttemptModel struct {
	ID            string     `gorm:"type:uuid;primaryKey"`
	JobID         string     `gorm:"column:job_id;type:uuid;not null"`
	AttemptNumber int        `gorm:"column:attempt_number;not null"`
	FencingToken  int64      `gorm:"column:fencing_token;not null"`
	StartedAt     time.Time  `gorm:"column:started_at;not null"`
	DeadlineAt    time.Time  `gorm:"column:deadline_at;not null"`
	FinishedAt    *time.Time `gorm:"column:finished_at"`
	Outcome       *string    `gorm:"column:outcome"`
}

// TableName fixes the attempt table used by GORM.
func (AttemptModel) TableName() string { return "interview_report_attempts" }

// OutboxModel maps outbox_messages, the durable publish intent.
type OutboxModel struct {
	ID               string     `gorm:"type:uuid;primaryKey"`
	JobKind          string     `gorm:"column:job_kind;not null"`
	JobID            string     `gorm:"column:job_id;type:uuid;not null"`
	CreatedAt        time.Time  `gorm:"column:created_at;not null;autoCreateTime:false"`
	PublishClaimedAt *time.Time `gorm:"column:publish_claimed_at"`
	ConfirmedAt      *time.Time `gorm:"column:confirmed_at"`
}

// TableName fixes the outbox table used by GORM.
func (OutboxModel) TableName() string { return "outbox_messages" }
