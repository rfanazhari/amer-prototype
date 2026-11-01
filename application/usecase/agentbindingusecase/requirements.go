package agentbindingusecase

import (
	"context"

	"github.com/rfanazhari/amer-prototype/application/repository/agentbindingport"
	"github.com/rfanazhari/amer-prototype/application/repository/agentconfigport"
)

// LoadRequirements restores immutable published requirements from an execution pin.
type LoadRequirements struct {
	Executions agentbindingport.Executions
	Revisions  agentconfigport.InstanceReader
}

// Execute returns the pinned revision identity and document required fields.
func (u LoadRequirements) Execute(ctx context.Context, executionID string) (Requirements, error) {
	execution, err := u.Executions.Find(ctx, executionID)
	if err != nil {
		return Requirements{}, err
	}
	revision, err := u.Revisions.FindRevision(ctx, execution.InstanceID, execution.RevisionID)
	if err != nil {
		return Requirements{}, err
	}
	return Requirements{RevisionID: execution.RevisionID, InstanceID: execution.InstanceID, RequiredFields: documentFields(revision)}, nil
}

// documentFields copies stored Document required fields; other templates have none.
func documentFields(stored agentconfigport.StoredRevision) []string {
	if stored.Document == nil {
		return nil
	}
	fields := stored.Document.Snapshot().Configuration.RequiredFields()
	out := make([]string, len(fields))
	for i, field := range fields {
		out[i] = field.String()
	}
	return out
}
