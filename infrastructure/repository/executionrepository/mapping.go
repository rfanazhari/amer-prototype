package executionrepository

import (
	"time"

	"github.com/google/uuid"
	"github.com/rfanazhari/amer-prototype/domain/aggregate/execution"
	"github.com/rfanazhari/amer-prototype/pkg/errs"
)

// utcPointer copies an optional driver timestamp into canonical UTC.
func utcPointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	canonical := value.UTC()
	return &canonical
}

// optionalString maps an empty token to a database null.
func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// fromSnapshot maps detached domain facts to the storage model for insertion.
func fromSnapshot(s execution.Snapshot) ExecutionModel {
	var progress *string
	if s.ProgressRef != nil {
		ref := string(*s.ProgressRef)
		progress = &ref
	}
	return ExecutionModel{
		ID: s.ID, InstanceID: s.Binding.InstanceID, RevisionID: s.Binding.RevisionID,
		State: string(s.State), StateVersion: s.StateVersion, ProgressRef: progress,
		ActivatedAt: utcPointer(s.ActivatedAt), TerminalAt: utcPointer(s.TerminalAt),
		TerminalReason: optionalString(s.TerminalReason),
		CreatedAt:      s.CreatedAt.UTC(), UpdatedAt: s.UpdatedAt.UTC(),
	}
}

// toSnapshot validates a stored row through the aggregate and returns a detached snapshot.
// A row that breaks the stored rules is a technical failure, never a caller input error.
// The attempt generation lives with the operation rows and is not part of this row.
func toSnapshot(m ExecutionModel) (execution.Snapshot, error) {
	var progress *execution.ProgressRef
	if m.ProgressRef != nil {
		ref := execution.ProgressRef(*m.ProgressRef)
		progress = &ref
	}
	reason := ""
	if m.TerminalReason != nil {
		reason = *m.TerminalReason
	}
	aggregate, err := execution.RestoreExecution(execution.Snapshot{
		ID: m.ID, Binding: execution.Binding{InstanceID: m.InstanceID, RevisionID: m.RevisionID},
		State: execution.State(m.State), StateVersion: m.StateVersion, ProgressRef: progress,
		ActivatedAt: utcPointer(m.ActivatedAt), TerminalAt: utcPointer(m.TerminalAt), TerminalReason: reason,
		CreatedAt: m.CreatedAt.UTC(), UpdatedAt: m.UpdatedAt.UTC(),
	})
	if err != nil {
		return execution.Snapshot{}, errs.Technical{Cause: err}
	}
	return aggregate.Snapshot(), nil
}

// fromTransition maps a transition fact to its append-only history row with a fresh identity.
func fromTransition(executionID string, f execution.TransitionFact) TransitionModel {
	var from *string
	if f.FromState != nil {
		state := string(*f.FromState)
		from = &state
	}
	return TransitionModel{
		ID: uuid.NewString(), ExecutionID: executionID, FromState: from, ToState: string(f.ToState),
		StateVersion: f.StateVersion, ActorRef: f.ActorRef, OperationKey: optionalString(f.OperationKey),
		Reason: optionalString(f.Reason), OccurredAt: f.OccurredAt.UTC(),
	}
}

// updateColumns lists the columns an update owns, using a map so nulls are written explicitly.
// The instance and revision binding is deliberately absent.
func updateColumns(s execution.Snapshot) map[string]any {
	model := fromSnapshot(s)
	columns := map[string]any{
		"state": model.State, "state_version": model.StateVersion, "updated_at": model.UpdatedAt,
		"committed_progress_ref": nil, "activated_at": nil, "terminal_at": nil, "terminal_reason": nil,
	}
	if model.ProgressRef != nil {
		columns["committed_progress_ref"] = *model.ProgressRef
	}
	if model.ActivatedAt != nil {
		columns["activated_at"] = *model.ActivatedAt
	}
	if model.TerminalAt != nil {
		columns["terminal_at"] = *model.TerminalAt
	}
	if model.TerminalReason != nil {
		columns["terminal_reason"] = *model.TerminalReason
	}
	return columns
}
