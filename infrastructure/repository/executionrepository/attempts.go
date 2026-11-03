package executionrepository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/application/repository/executionport"
	"github.com/rfanazhari/amer-prototype/infrastructure/database/transaction"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// executionProjection restores authority from the durable operation rows without a second pool query.
const executionProjection = `"agent_executions".*, COALESCE((SELECT MAX(current_authority_generation) FROM agent_execution_operations WHERE execution_id = agent_executions.id),0) AS current_generation`

// Ledger persists stage-independent attempts in the shared lifecycle transaction.
type Ledger struct{ DB *gorm.DB }

// operationModel maps the current authority of one lifecycle operation.
type operationModel struct {
	ExecutionID                string
	OperationKey               string
	State                      string
	CurrentAuthorityGeneration int64
	CreatedAt                  time.Time `gorm:"autoCreateTime:false"`
	UpdatedAt                  time.Time `gorm:"autoUpdateTime:false"`
}

// TableName binds the operation authority table.
func (operationModel) TableName() string { return "agent_execution_operations" }

// attemptModel stores one durable attempt without interview fields.
type attemptModel struct {
	ID                  string
	ExecutionID         string
	OperationKey        string
	AttemptNumber       int
	AuthorityGeneration int64
	Outcome             string
	DeadlineAt          time.Time
	CreatedAt           time.Time `gorm:"autoCreateTime:false"`
}

// TableName binds the generic attempt table.
func (attemptModel) TableName() string { return "agent_execution_attempts" }

// attemptFrom maps stored attempt facts to a detached application value.
func attemptFrom(m attemptModel) executionport.Attempt {
	return executionport.Attempt{ID: m.ID, ExecutionID: m.ExecutionID, OperationKey: m.OperationKey, Generation: m.AuthorityGeneration, Number: m.AttemptNumber, Outcome: m.Outcome, Deadline: m.DeadlineAt.UTC(), CreatedAt: m.CreatedAt.UTC()}
}

// List returns ordered attempts for the locked operation.
func (l Ledger) List(ctx context.Context, id, key string) ([]executionport.Attempt, error) {
	var rows []attemptModel
	if err := transaction.Session(ctx, l.DB).Where("execution_id = ? AND operation_key = ?", id, key).Order("authority_generation").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make([]executionport.Attempt, 0, len(rows))
	for _, row := range rows {
		result = append(result, attemptFrom(row))
	}
	return result, nil
}

// Find resolves an execution-owned authority UUID.
func (l Ledger) Find(ctx context.Context, id, attemptID string) (executionport.Attempt, error) {
	if _, err := uuid.Parse(attemptID); err != nil {
		return executionport.Attempt{}, executionport.ErrNotFound
	}
	var row attemptModel
	err := transaction.Session(ctx, l.DB).Where("execution_id = ? AND id = ?", id, attemptID).Take(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return executionport.Attempt{}, executionport.ErrNotFound
	}
	return attemptFrom(row), err
}

// Insert advances the operation and inserts the attempt atomically with the caller's execution mutation.
func (l Ledger) Insert(ctx context.Context, a executionport.Attempt) error {
	return transaction.Within(ctx, l.DB, func(_ context.Context, tx *gorm.DB) error {
		op := operationModel{ExecutionID: a.ExecutionID, OperationKey: a.OperationKey, State: "IN_PROGRESS", CurrentAuthorityGeneration: a.Generation, CreatedAt: a.CreatedAt, UpdatedAt: a.CreatedAt}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "execution_id"}, {Name: "operation_key"}}, DoUpdates: clause.AssignmentColumns([]string{"state", "current_authority_generation", "updated_at"})}).Create(&op).Error; err != nil {
			return err
		}
		row := attemptModel{ID: a.ID, ExecutionID: a.ExecutionID, OperationKey: a.OperationKey, AttemptNumber: a.Number, AuthorityGeneration: a.Generation, Outcome: a.Outcome, DeadlineAt: a.Deadline, CreatedAt: a.CreatedAt}
		return classify(tx.Create(&row).Error)
	})
}
