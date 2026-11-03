package interviewrepository

import (
	"context"
	"errors"
	"time"

	"github.com/rfanazhari/amer-prototype/application/repository/interviewport"
	"github.com/rfanazhari/amer-prototype/application/service"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"gorm.io/gorm"
)

// Commands persists session reservations and immutable replay results in joined transactions.
type Commands struct {
	DB    *gorm.DB
	Clock service.Clock
}

// commandModel maps the forward-migrated reservation and its nullable legacy fields.
type commandModel struct {
	ID               string
	ExecutionID      string
	Key              string `gorm:"column:idempotency_key"`
	Intent           string
	AttemptID        string
	State            string
	SessionID        *string
	AnswerSDP        *string
	ExecutionVersion *int64
	CreatedAt        time.Time `gorm:"autoCreateTime:false"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime:false"`
}

// TableName binds the candidate-session-scoped command table.
func (commandModel) TableName() string { return "interview_session_commands" }

// Find returns the original reservation or successful replay fields without creating anything.
func (s Commands) Find(ctx context.Context, id, key string) (interviewport.SessionCommand, error) {
	var row commandModel
	err := transaction.Session(ctx, s.DB).Where("execution_id = ? AND idempotency_key = ?", id, key).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return interviewport.SessionCommand{}, interviewport.ErrNotFound
	}
	result := interviewport.SessionCommand{ID: row.ID, ExecutionID: row.ExecutionID, Key: row.Key, Intent: row.Intent, AttemptID: row.AttemptID, State: row.State, CreatedAt: row.CreatedAt.UTC()}
	if row.SessionID != nil {
		result.SessionID = *row.SessionID
	}
	if row.AnswerSDP != nil {
		result.AnswerSDP = *row.AnswerSDP
	}
	if row.ExecutionVersion != nil {
		result.ExecutionVersion = *row.ExecutionVersion
	}
	return result, err
}

// Insert reserves a pending identity with its session shell before provider I/O.
func (s Commands) Insert(ctx context.Context, c interviewport.SessionCommand) error {
	if c.State != "pending" || c.AnswerSDP != "" || c.SessionID == "" || c.ExecutionVersion < 1 {
		return interviewport.ErrConflict
	}
	now := s.Clock.Now().UTC().Truncate(time.Millisecond)
	row := commandModel{ID: c.ID, ExecutionID: c.ExecutionID, Key: c.Key, Intent: c.Intent, AttemptID: c.AttemptID, State: c.State, SessionID: &c.SessionID, ExecutionVersion: &c.ExecutionVersion, CreatedAt: c.CreatedAt, UpdatedAt: now}
	return classify(transaction.Session(ctx, s.DB).Create(&row).Error)
}

// Finalize atomically stores the original answer with the caller's successful progress write.
func (s Commands) Finalize(ctx context.Context, c interviewport.SessionCommand) error {
	result := transaction.Session(ctx, s.DB).Model(&commandModel{}).Where("execution_id = ? AND idempotency_key = ? AND attempt_id = ? AND session_id = ? AND state = ?", c.ExecutionID, c.Key, c.AttemptID, c.SessionID, "pending").Updates(map[string]any{"state": "succeeded", "answer_sdp": c.AnswerSDP, "execution_version": c.ExecutionVersion, "updated_at": s.Clock.Now().UTC().Truncate(time.Millisecond)})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return interviewport.ErrConflict
	}
	return nil
}

// Fail records a known unsuccessful setup; the original key remains non-retryable.
func (s Commands) Fail(ctx context.Context, id, key string) error {
	return transaction.Within(ctx, s.DB, func(_ context.Context, tx *gorm.DB) error {
		result := tx.Model(&commandModel{}).Where("execution_id = ? AND idempotency_key = ? AND state = ?", id, key, "pending").Updates(map[string]any{"state": "failed", "updated_at": s.Clock.Now().UTC().Truncate(time.Millisecond)})
		if result.Error != nil {
			return result.Error
		}
		return tx.Exec(`UPDATE agent_execution_attempts SET outcome = 'FAILED' WHERE execution_id = ? AND id IN (SELECT attempt_id FROM interview_session_commands WHERE execution_id = ? AND idempotency_key = ? AND state = 'failed')`, id, id, key).Error
	})
}
