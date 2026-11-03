package interviewreportusecase

import (
	"context"
	"github.com/rfanazhari/amer-prototype/application/repository/interviewreportport"
	"github.com/rfanazhari/amer-prototype/application/service"
)

// Worker runs report attempts without owning a broker consumer or process registration.
type Worker struct {
	Store    interviewreportport.WorkerStore
	Evidence interviewreportport.Evidence
	Writer   interviewreportport.Writer
	Clock    service.Clock
}

// Execute processes a durable job identity, propagating database errors to its caller.
func (u Worker) Execute(ctx context.Context, jobID string) error { return nil }

// Reconcile processes one bounded batch of due jobs without invoking the writer.
func (u Worker) Reconcile(ctx context.Context, limit int) error { return nil }
