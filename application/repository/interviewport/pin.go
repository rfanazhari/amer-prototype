package interviewport

import (
	"context"
	"time"

	"github.com/rfanazhari/amer-prototype/domain/valueobject"
)

// PinnedContext carries the facts progress rows deliberately do not copy: the immutable revision pin
// and the instant the lifecycle reported RUNNING. Configuration is a detached value.
type PinnedContext struct {
	RevisionID    string
	Configuration valueobject.HRConfiguration
	RunningAt     time.Time
}

// PinReader resolves the pin and lifecycle facts that progress storage does not own.
type PinReader interface {
	// Find returns the pinned revision and running instant for one execution; absence yields ErrNotFound.
	Find(ctx context.Context, executionID string) (PinnedContext, error)
}
