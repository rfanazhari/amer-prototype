package executionusecase_test

import (
	"context"
	"testing"

	"github.com/rfanazhari/amer-prototype/application/dto"
	"github.com/rfanazhari/amer-prototype/application/usecase/executionusecase"
	"github.com/stretchr/testify/assert"
)

// nopStart is a stub that satisfies StartOperation for compile-time interface checking only.
type nopStart struct{}

// Execute is the stub StartOperation implementation used only for compile-time interface checking.
func (n *nopStart) Execute(_ context.Context, _ executionusecase.StartInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopAdmit is a stub that satisfies AdmitOperation for compile-time interface checking only.
type nopAdmit struct{}

// Execute is the stub AdmitOperation implementation used only for compile-time interface checking.
func (n *nopAdmit) Execute(_ context.Context, _ executionusecase.AdmitInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopActivate is a stub that satisfies ActivateOperation for compile-time interface checking only.
type nopActivate struct{}

// Execute is the stub ActivateOperation implementation used only for compile-time interface checking.
func (n *nopActivate) Execute(_ context.Context, _ executionusecase.ActivateInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopRecordAttempt is a stub that satisfies RecordAttemptOperation for compile-time interface checking only.
type nopRecordAttempt struct{}

// Execute is the stub RecordAttemptOperation implementation used only for compile-time interface checking.
func (n *nopRecordAttempt) Execute(_ context.Context, _ executionusecase.RecordAttemptInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopCommitProgress is a stub that satisfies CommitProgressOperation for compile-time interface checking only.
type nopCommitProgress struct{}

// Execute is the stub CommitProgressOperation implementation used only for compile-time interface checking.
func (n *nopCommitProgress) Execute(_ context.Context, _ executionusecase.CommitProgressInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopComplete is a stub that satisfies CompleteOperation for compile-time interface checking only.
type nopComplete struct{}

// Execute is the stub CompleteOperation implementation used only for compile-time interface checking.
func (n *nopComplete) Execute(_ context.Context, _ executionusecase.CompleteInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopFail is a stub that satisfies FailOperation for compile-time interface checking only.
type nopFail struct{}

// Execute is the stub FailOperation implementation used only for compile-time interface checking.
func (n *nopFail) Execute(_ context.Context, _ executionusecase.FailInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopCancel is a stub that satisfies CancelOperation for compile-time interface checking only.
type nopCancel struct{}

// Execute is the stub CancelOperation implementation used only for compile-time interface checking.
func (n *nopCancel) Execute(_ context.Context, _ executionusecase.CancelInput) (*dto.ExecutionView, error) {
	return nil, nil
}

// nopCancelNonterminal is a stub that satisfies CancelNonterminalOperation for compile-time interface checking only.
type nopCancelNonterminal struct{}

// Execute is the stub CancelNonterminalOperation implementation used only for compile-time interface checking.
func (n *nopCancelNonterminal) Execute(_ context.Context, _ executionusecase.CancelNonterminalInput) error {
	return nil
}

// TestOperationInterfacesDefined verifies that all nine operation interfaces are declared in the package.
func TestOperationInterfacesDefined(t *testing.T) {
	var (
		_ executionusecase.StartOperation           = (*nopStart)(nil)
		_ executionusecase.AdmitOperation           = (*nopAdmit)(nil)
		_ executionusecase.ActivateOperation        = (*nopActivate)(nil)
		_ executionusecase.RecordAttemptOperation   = (*nopRecordAttempt)(nil)
		_ executionusecase.CommitProgressOperation  = (*nopCommitProgress)(nil)
		_ executionusecase.CompleteOperation        = (*nopComplete)(nil)
		_ executionusecase.FailOperation            = (*nopFail)(nil)
		_ executionusecase.CancelOperation          = (*nopCancel)(nil)
		_ executionusecase.CancelNonterminalOperation = (*nopCancelNonterminal)(nil)
	)
	assert.True(t, true, "all nine operation interfaces are defined")
}
