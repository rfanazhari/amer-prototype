package repository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

var ErrNotFound = errors.New("task not found")

// Task defines context-aware persistence and serialized task mutations. A failed Update callback or transaction must not persist its changes.
type Task interface {
	// Create persists a new aggregate or returns the storage failure.
	Create(context.Context, *aggregate.Task) error
	// Find returns restored history or ErrNotFound without mutating stored facts.
	Find(context.Context, string) (*aggregate.Task, error)
	// Update locks the row and commits only if the operation succeeds.
	Update(context.Context, string, func(*aggregate.Task) error) (*aggregate.Task, error)
}
