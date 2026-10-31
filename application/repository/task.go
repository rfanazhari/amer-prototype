package repository

import (
	"context"
	"errors"

	"github.com/rfanazhari/amer-prototype/domain/aggregate"
)

var ErrNotFound = errors.New("task not found")

type Task interface {
	Create(context.Context, *aggregate.Task) error
	Find(context.Context, string) (*aggregate.Task, error)
	// Update locks the row and commits only if the operation succeeds.
	Update(context.Context, string, func(*aggregate.Task) error) (*aggregate.Task, error)
}
